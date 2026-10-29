// Package wshb 维护 WebSocket 长连接、心跳、命令回执与指数退避重连。
package wshb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"backup-monitor-agent/internal/config"
	"backup-monitor-agent/internal/discover"
	"backup-monitor-agent/internal/update"
)

// Options 是 WebSocket 客户端参数。
type Options struct {
	ServerURL       string
	Token           string
	Hostname        string
	NodeID          string
	Platform        string
	Arch            string
	OSBuild         int
	AgentVersion    string
	Heartbeat       time.Duration
	BackoffMin      time.Duration
	BackoffMax      time.Duration
	Logger          *log.Logger
	Wake            <-chan struct{}
	OnConfigChanged func()
	Update          update.Options
}

// Client 是可自动重连的 WebSocket 客户端。
type Client struct {
	options Options
}

// New 创建 WebSocket 客户端并补齐默认参数。
func New(options Options) *Client {
	if options.Heartbeat <= 0 {
		options.Heartbeat = config.DefaultHeartbeatIntervalSec * time.Second
	}
	if options.BackoffMin <= 0 {
		options.BackoffMin = config.DefaultWSBackoffMinSec * time.Second
	}
	if options.BackoffMax <= 0 {
		options.BackoffMax = config.DefaultWSBackoffMaxSec * time.Second
	}
	if options.BackoffMax < options.BackoffMin {
		options.BackoffMax = options.BackoffMin
	}
	return &Client{options: options}
}

// Run 持续连接服务端，直到上下文取消。
func (c *Client) Run(ctx context.Context) {
	backoff := c.options.BackoffMin
	for {
		if ctx.Err() != nil {
			return
		}
		established, err := c.connect(ctx)
		if established {
			backoff = c.options.BackoffMin
		}
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			c.logf("[ws] 心跳断开：%v（%s 后重连）", err, backoff)
		}
		c.drainWake()
		select {
		case <-ctx.Done():
			return
		case <-c.options.Wake:
			backoff = c.options.BackoffMin
			c.logf("[ws] 上报已恢复，立即重连")
			continue
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > c.options.BackoffMax {
			backoff = c.options.BackoffMax
		}
	}
}

func (c *Client) connect(ctx context.Context) (bool, error) {
	target, err := websocketURL(c.options.ServerURL, c.options.Hostname, c.options.NodeID, c.options.Platform)
	if err != nil {
		return false, err
	}
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	header := http.Header{}
	header.Set("X-Agent-Token", c.options.Token)
	connection, response, err := dialer.DialContext(ctx, target, header)
	if err != nil {
		if response != nil {
			response.Body.Close()
			return false, fmt.Errorf("dial %s: %w (HTTP %d)", target, err, response.StatusCode)
		}
		return false, fmt.Errorf("dial %s: %w", target, err)
	}
	defer connection.Close()
	connection.SetReadLimit(1 << 20)

	connectionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-connectionCtx.Done()
		connection.Close()
	}()

	var writeMu sync.Mutex
	go c.heartbeat(connectionCtx, connection, &writeMu)
	c.logf("[ws] 心跳已连接 %s", target)

	for {
		_, data, readErr := connection.ReadMessage()
		if readErr != nil {
			return true, readErr
		}
		var message serverMessage
		if unmarshalErr := json.Unmarshal(data, &message); unmarshalErr != nil {
			c.logf("[ws] 忽略无法解析的消息：%v", unmarshalErr)
			continue
		}
		c.handle(connectionCtx, connection, &writeMu, message)
	}
}

func (c *Client) heartbeat(ctx context.Context, connection *websocket.Conn, writeMu *sync.Mutex) {
	ticker := time.NewTicker(c.options.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			writeMu.Lock()
			connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := connection.WriteMessage(websocket.TextMessage, []byte("ping"))
			writeMu.Unlock()
			if err != nil {
				c.logf("[ws] 心跳发送失败：%v", err)
				connection.Close()
				return
			}
		}
	}
}

func (c *Client) handle(ctx context.Context, connection *websocket.Conn, writeMu *sync.Mutex, message serverMessage) {
	switch message.Type {
	case "config_changed":
		c.logf("[ws] 收到配置变更通知")
		if c.options.OnConfigChanged != nil {
			go c.options.OnConfigChanged()
		}
	case "set_cron":
		ackMessage := ""
		ok := false
		if strings.TrimSpace(message.Key) == "" {
			ackMessage = "缺少任务标识"
		} else if strings.TrimSpace(message.Cron) == "" {
			ackMessage = "计划表达式为空"
		} else {
			result, err := discover.ApplyCronByKey(ctx, message.Key, message.Cron)
			ackMessage = result
			ok = err == nil
			if err != nil {
				c.logf("[ws] 应用计划失败（key=%s）：%v", message.Key, err)
			}
		}
		c.writeResult(connection, writeMu, resultMessage{
			Type:      "set_cron_result",
			RequestID: message.RequestID,
			OK:        ok,
			Message:   ackMessage,
		})
	case "update":
		if strings.TrimSpace(message.Version) == "" || strings.TrimSpace(message.SHA256) == "" {
			c.writeResult(connection, writeMu, resultMessage{
				Type:      "update_result",
				RequestID: message.RequestID,
				OK:        false,
				Message:   "更新参数无效",
			})
			return
		}
		c.writeResult(connection, writeMu, resultMessage{
			Type:      "update_result",
			RequestID: message.RequestID,
			OK:        true,
			Message:   "开始更新",
		})
		go c.applyUpdate(message)
	}
}

func (c *Client) applyUpdate(message serverMessage) {
	options := c.options.Update
	options.Version = message.Version
	options.SHA256 = message.SHA256
	options.DownloadURL = strings.TrimSpace(message.URL)
	if options.DownloadURL == "" {
		options.DownloadURL = update.DownloadPath(c.options.Platform, c.options.Arch, c.options.OSBuild)
	}
	if strings.TrimSpace(options.ServerURL) == "" {
		options.ServerURL = c.options.ServerURL
	}
	if strings.TrimSpace(options.Token) == "" {
		options.Token = c.options.Token
	}
	if options.Logger == nil {
		options.Logger = c.options.Logger
	}
	if err := update.Apply(context.Background(), options); err != nil {
		c.logf("[ws] 应用更新失败（版本=%s）：%v", message.Version, err)
		if recordErr := update.RecordFailure(message.Version); recordErr != nil {
			c.logf("[ws] 记录更新失败状态失败：%v", recordErr)
		}
	}
}

func (c *Client) writeResult(connection *websocket.Conn, writeMu *sync.Mutex, result resultMessage) {
	data, err := json.Marshal(result)
	if err != nil {
		c.logf("[ws] 回执编码失败：%v", err)
		return
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := connection.WriteMessage(websocket.TextMessage, data); err != nil {
		c.logf("[ws] 回执发送失败：%v", err)
	}
}

func (c *Client) logf(format string, args ...any) {
	if c.options.Logger != nil {
		c.options.Logger.Printf(format, args...)
	}
}

func websocketURL(serverURL, hostname, nodeID, platform string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", fmt.Errorf("invalid server URL scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid server URL: missing host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/agent/ws"
	query := parsed.Query()
	query.Set("hostname", hostname)
	query.Set("nodeId", nodeID)
	query.Set("platform", platform)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

type serverMessage struct {
	Type      string `json:"type"`
	At        string `json:"at"`
	RequestID string `json:"requestId"`
	Key       string `json:"key"`
	Cron      string `json:"cron"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	URL       string `json:"url"`
}

type resultMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId,omitempty"`
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
}

func (c *Client) drainWake() {
	if c.options.Wake == nil {
		return
	}
	select {
	case <-c.options.Wake:
	default:
	}
}
