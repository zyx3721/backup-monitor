package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"

	"github.com/gorilla/websocket"
)

const (
	disconnectGrace  = 15 * time.Second
	heartbeatTimeout = 90 * time.Second
	setCronTimeout   = 10 * time.Second
	updateTimeout    = 20 * time.Second
)

// AgentHub 维护 Agent WebSocket 心跳连接、命令下发与在线状态。
type AgentHub struct {
	repo    *repository.BackupRepository
	svc     *BackupService
	mu      sync.Mutex
	conns   map[string]*agentConn
	online  map[string]bool
	down    map[string]bool
	timers  map[string]*time.Timer
	pending map[string]chan commandAck
}

type agentConn struct {
	conn     *websocket.Conn
	hostID   string
	lastSeen time.Time
	writeMu  sync.Mutex
}

type commandAck struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// NewAgentHub 创建连接中心。
func NewAgentHub(repo *repository.BackupRepository, svc *BackupService) *AgentHub {
	hub := &AgentHub{repo: repo, svc: svc, conns: map[string]*agentConn{}, online: map[string]bool{},
		down: map[string]bool{}, timers: map[string]*time.Timer{}, pending: map[string]chan commandAck{}}
	go hub.heartbeatLoop()
	return hub
}

// IsConnected 判断主机是否有存活心跳。
func (h *AgentHub) IsConnected(hostID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.online[hostID]
}

// IsDisconnected 判断主机是否因心跳断开被判离线。
func (h *AgentHub) IsDisconnected(hostID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.down[hostID]
}

// ClearDisconnected 清除心跳断开标记（HTTP 上报成功说明进程仍存活）。
func (h *AgentHub) ClearDisconnected(hostID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.down, hostID)
}

// PushConfigChanged 通知所有在线 Agent 重新拉取配置，返回推送连接数。
func (h *AgentHub) PushConfigChanged() int {
	payload := map[string]any{"type": "config_changed", "at": appcommon.NowTimestamp()}
	h.mu.Lock()
	conns := make([]*agentConn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.Unlock()
	sent := 0
	for _, conn := range conns {
		if err := conn.writeJSON(payload); err == nil {
			sent++
		}
	}
	return sent
}

// SendSetCron 向指定主机下发修改 crontab 命令并等待回执。
func (h *AgentHub) SendSetCron(hostID, taskKey, cron string) (bool, bool, string) {
	return h.sendCommand(hostID, map[string]any{"type": "set_cron", "requestId": newUUID(), "key": taskKey, "cron": cron}, setCronTimeout)
}

// SendUpdate 向指定主机下发自更新命令并等待回执。
func (h *AgentHub) SendUpdate(hostID, version, sha256, url string) (bool, bool, string) {
	return h.sendCommand(hostID, map[string]any{"type": "update", "requestId": newUUID(), "version": version, "sha256": sha256, "url": url}, updateTimeout)
}

func (h *AgentHub) sendCommand(hostID string, payload map[string]any, timeout time.Duration) (delivered bool, ok bool, message string) {
	h.mu.Lock()
	conn := h.conns[hostID]
	h.mu.Unlock()
	if conn == nil {
		time.Sleep(800 * time.Millisecond)
		h.mu.Lock()
		conn = h.conns[hostID]
		h.mu.Unlock()
	}
	if conn == nil {
		return false, false, "主机心跳连接离线，未下发"
	}
	requestID, _ := payload["requestId"].(string)
	ch := make(chan commandAck, 1)
	h.mu.Lock()
	if h.conns[hostID] == nil {
		h.mu.Unlock()
		return false, false, "主机心跳连接离线，未下发"
	}
	h.pending[requestID] = ch
	h.mu.Unlock()

	cleanup := func() {
		h.mu.Lock()
		delete(h.pending, requestID)
		h.mu.Unlock()
	}
	if err := conn.writeJSON(payload); err != nil {
		cleanup()
		return false, false, "下发失败：" + err.Error()
	}
	select {
	case ack := <-ch:
		cleanup()
		if ack.Message == "" {
			ack.Message = "Agent 已处理"
		}
		return true, ack.OK, ack.Message
	case <-time.After(timeout):
		cleanup()
		return true, false, "下发后等待 Agent 回执超时"
	}
}

// Handler 返回 Agent WebSocket 升级处理器。
func (h *AgentHub) Handler() http.HandlerFunc {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }, ReadBufferSize: 4096, WriteBufferSize: 4096}
	return func(w http.ResponseWriter, r *http.Request) {
		hostname := strings.TrimSpace(r.URL.Query().Get("hostname"))
		nodeID := strings.TrimSpace(r.URL.Query().Get("nodeId"))
		if hostname == "" {
			http.Error(w, "missing hostname", http.StatusBadRequest)
			return
		}
		if !h.authorize(r) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		hostID, name := h.ensureHost(nodeID, hostname)
		if hostID == "" {
			http.Error(w, "host not registered", http.StatusNotFound)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		agent := &agentConn{conn: conn, hostID: hostID, lastSeen: time.Now()}
		h.register(agent, name)
		defer h.unregister(agent)
		conn.SetReadLimit(1 << 20)
		_ = conn.SetReadDeadline(time.Now().Add(heartbeatTimeout))
		conn.SetPongHandler(func(string) error {
			agent.touch()
			return conn.SetReadDeadline(time.Now().Add(heartbeatTimeout))
		})
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			agent.touch()
			_ = conn.SetReadDeadline(time.Now().Add(heartbeatTimeout))
			h.handleMessage(data)
		}
	}
}

func (h *AgentHub) authorize(r *http.Request) bool {
	settings := h.svc.Settings()
	token := strings.TrimSpace(r.Header.Get("X-Agent-Token"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token != "" {
		if found, err := h.repo.FindAgentToken(token); err == nil && found != nil {
			_ = h.repo.TouchAgentToken(found.ID, appcommon.NowTimestamp())
			return true
		}
	}
	return !settings.RequireAgentToken
}

// ensureHost 返回主机标识；首次连接尚未上报时先登记占位主机，避免握手被拒。
func (h *AgentHub) ensureHost(nodeID, hostname string) (string, string) {
	if id, name := h.findHost(nodeID, hostname); id != "" {
		return id, name
	}
	return "", ""
}

func (h *AgentHub) findHost(nodeID, hostname string) (string, string) {
	if nodeID != "" {
		if id, name, err := h.repo.FindHostByNode(nodeID); err == nil && id != "" {
			return id, name
		}
	}
	id, err := h.repo.FindHostByName(hostname)
	if err != nil || id == "" {
		return "", ""
	}
	host, err := h.repo.GetHost(id)
	if err != nil || host == nil {
		return "", ""
	}
	return id, host.Hostname
}

func (h *AgentHub) register(agent *agentConn, hostname string) {
	h.mu.Lock()
	previous := h.conns[agent.hostID]
	if timer := h.timers[agent.hostID]; timer != nil {
		timer.Stop()
		delete(h.timers, agent.hostID)
	}
	h.conns[agent.hostID] = agent
	h.online[agent.hostID] = true
	delete(h.down, agent.hostID)
	h.mu.Unlock()
	if previous != nil {
		_ = previous.conn.Close()
	}
	_ = hostname
}

func (h *AgentHub) unregister(agent *agentConn) {
	h.mu.Lock()
	current := h.conns[agent.hostID]
	if current != agent {
		h.mu.Unlock()
		return
	}
	delete(h.conns, agent.hostID)
	delete(h.online, agent.hostID)
	h.mu.Unlock()
	_ = agent.conn.Close()

	h.mu.Lock()
	h.timers[agent.hostID] = time.AfterFunc(disconnectGrace, func() {
		h.mu.Lock()
		connected := h.online[agent.hostID]
		delete(h.timers, agent.hostID)
		if !connected {
			h.down[agent.hostID] = true
		}
		h.mu.Unlock()
		if connected {
			return
		}
		_, _ = h.repo.DB().Exec(`UPDATE hosts SET status='offline' WHERE id=?`, agent.hostID)
		host, err := h.repo.GetHost(agent.hostID)
		if err == nil && host != nil {
			h.svc.AlertReportTimeout(host.ID, host.Hostname, string(domain.HostOffline), "heartbeat")
		}
	})
	h.mu.Unlock()
}

func (h *AgentHub) handleMessage(data []byte) {
	var message struct {
		Type      string `json:"type"`
		RequestID string `json:"requestId"`
		OK        bool   `json:"ok"`
		Message   string `json:"message"`
	}
	if json.Unmarshal(data, &message) != nil || message.RequestID == "" {
		return
	}
	if message.Type != "set_cron_result" && message.Type != "update_result" {
		return
	}
	h.mu.Lock()
	ch := h.pending[message.RequestID]
	h.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- commandAck{OK: message.OK, Message: message.Message}:
	default:
	}
}

// CloseHost 关闭指定主机的连接，促使 Agent 重新注册到新的主机记录。
func (h *AgentHub) CloseHost(hostID string) {
	h.mu.Lock()
	conn := h.conns[hostID]
	h.mu.Unlock()
	if conn != nil {
		_ = conn.conn.Close()
	}
}

func (h *AgentHub) CloseAll() {
	h.mu.Lock()
	conns := make([]*agentConn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.Unlock()
	for _, conn := range conns {
		_ = conn.conn.Close()
	}
}

func (h *AgentHub) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		h.mu.Lock()
		stale := []*agentConn{}
		for _, conn := range h.conns {
			if now.Sub(conn.lastSeen) > heartbeatTimeout {
				stale = append(stale, conn)
			}
		}
		h.mu.Unlock()
		for _, conn := range stale {
			_ = conn.conn.Close()
		}
	}
}

func (c *agentConn) touch() {
	c.lastSeen = time.Now()
}

func (c *agentConn) writeJSON(payload any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(payload)
}
