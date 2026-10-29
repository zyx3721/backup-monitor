package repository

import (
	"database/sql"
	"strings"

	"backup-monitor-backend/internal/domain"
)

// ListNotifyChannels 返回通知渠道（按创建时间升序）。
func (r *BackupRepository) ListNotifyChannels() ([]domain.NotifyChannel, error) {
	rows, err := r.db.Query(`SELECT id,name,type,event_type,level,remain_days,trigger_group,COALESCE(remark,''),
		enabled,recover,config,created_at,updated_at FROM notify_channels ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.NotifyChannel{}
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, channel)
	}
	return out, rows.Err()
}

// GetNotifyChannel 按 ID 读取通知渠道。
func (r *BackupRepository) GetNotifyChannel(id string) (*domain.NotifyChannel, error) {
	row := r.db.QueryRow(`SELECT id,name,type,event_type,level,remain_days,trigger_group,COALESCE(remark,''),
		enabled,recover,config,created_at,updated_at FROM notify_channels WHERE id=?`, id)
	channel, err := scanChannel(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// CreateNotifyChannel 新建通知渠道。
func (r *BackupRepository) CreateNotifyChannel(channel domain.NotifyChannel) (string, error) {
	configRaw, err := marshalJSON(channel.Config)
	if err != nil {
		return "", err
	}
	id := newID()
	now := nowISO()
	_, err = r.db.Exec(`INSERT INTO notify_channels(id,name,type,event_type,level,remain_days,trigger_group,remark,enabled,recover,config,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, channel.Name, channel.Type, strings.Join(channel.EventType, ","), channel.Level, channel.RemainDays,
		channel.TriggerGroup, channel.Remark, boolToInt(channel.Enabled), boolToInt(channel.Recover), configRaw, now, now)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateNotifyChannel 更新通知渠道。
func (r *BackupRepository) UpdateNotifyChannel(id string, channel domain.NotifyChannel) error {
	configRaw, err := marshalJSON(channel.Config)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE notify_channels SET name=?,event_type=?,level=?,remain_days=?,trigger_group=?,remark=?,
		enabled=?,recover=?,config=?,updated_at=? WHERE id=?`,
		channel.Name, strings.Join(channel.EventType, ","), channel.Level, channel.RemainDays, channel.TriggerGroup,
		channel.Remark, boolToInt(channel.Enabled), boolToInt(channel.Recover), configRaw, nowISO(), id)
	return err
}

// DeleteNotifyChannel 删除通知渠道。
func (r *BackupRepository) DeleteNotifyChannel(id string) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM notify_channels WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

type channelScanner interface {
	Scan(dest ...any) error
}

func scanChannel(scanner channelScanner) (domain.NotifyChannel, error) {
	var c domain.NotifyChannel
	var eventType, configRaw string
	var enabled, recover int
	if err := scanner.Scan(&c.ID, &c.Name, &c.Type, &eventType, &c.Level, &c.RemainDays, &c.TriggerGroup, &c.Remark,
		&enabled, &recover, &configRaw, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return c, err
	}
	c.Enabled = enabled == 1
	c.Recover = recover == 1
	c.Config = domain.ParseJSONObject(configRaw)
	c.EventType = splitEventTypes(eventType)
	return c, nil
}

func splitEventTypes(raw string) []string {
	out := []string{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' }) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		out = append(out, "backup_failed")
	}
	return out
}

// InsertNotification 写入一条通知发送记录。
func (r *BackupRepository) InsertNotification(record domain.NotificationRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		record.ID = newID()
	}
	if strings.TrimSpace(record.CreatedAt) == "" {
		record.CreatedAt = nowISO()
	}
	_, err := r.db.Exec(`INSERT INTO notifications(id,channel,channel_id,channel_name,type,level,recipients,subject,status,error,
		host_name,task_name,message,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.ID, record.Channel, nullable(record.ChannelID), nullable(record.ChannelName), record.Type, nullable(record.Level),
		nullable(record.Recipients), nullable(record.Subject), record.Status, nullable(record.Error),
		nullable(record.HostName), nullable(record.TaskName), nullable(record.Message), record.CreatedAt)
	return err
}

// ListNotifications 按时间倒序返回最近的发送记录，channelID 非空时只返回该渠道。
func (r *BackupRepository) ListNotifications(limit int, channelID string) ([]domain.NotificationRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const columns = `SELECT id,channel,COALESCE(channel_id,''),COALESCE(channel_name,''),type,COALESCE(level,''),COALESCE(recipients,''),
		COALESCE(subject,''),status,COALESCE(error,''),COALESCE(host_name,''),COALESCE(task_name,''),
		COALESCE(message,''),created_at FROM notifications`
	var (
		rows *sql.Rows
		err  error
	)
	if channelID == "" {
		rows, err = r.db.Query(columns+` ORDER BY created_at DESC LIMIT ?`, limit)
	} else {
		var channelName string
		if scanErr := r.db.QueryRow(`SELECT name FROM notify_channels WHERE id=?`, channelID).Scan(&channelName); scanErr != nil {
			channelName = ""
		}
		rows, err = r.db.Query(columns+` WHERE channel_id=? OR (COALESCE(channel_id,'')='' AND channel_name=?)
			ORDER BY created_at DESC LIMIT ?`, channelID, channelName, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.NotificationRecord{}
	for rows.Next() {
		var record domain.NotificationRecord
		if err := rows.Scan(&record.ID, &record.Channel, &record.ChannelID, &record.ChannelName, &record.Type, &record.Level,
			&record.Recipients, &record.Subject, &record.Status, &record.Error, &record.HostName,
			&record.TaskName, &record.Message, &record.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}
