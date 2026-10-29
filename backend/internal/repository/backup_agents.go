package repository

import (
	"database/sql"

	"backup-monitor-backend/internal/domain"
)

// ListAgentProfiles 返回采集配置（优先级降序）。
func (r *BackupRepository) ListAgentProfiles() ([]domain.AgentProfile, error) {
	rows, err := r.db.Query(`SELECT id,name,hostname,COALESCE(description,''),enabled,priority,config,created_at,updated_at
		FROM agent_profiles ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AgentProfile{}
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, profile)
	}
	return out, rows.Err()
}

// GetAgentProfile 按 ID 读取采集配置。
func (r *BackupRepository) GetAgentProfile(id string) (*domain.AgentProfile, error) {
	row := r.db.QueryRow(`SELECT id,name,hostname,COALESCE(description,''),enabled,priority,config,created_at,updated_at
		FROM agent_profiles WHERE id=?`, id)
	profile, err := scanProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// CreateAgentProfile 新建采集配置。
func (r *BackupRepository) CreateAgentProfile(profile domain.AgentProfile) error {
	raw, err := marshalJSON(profile.Config)
	if err != nil {
		return err
	}
	now := nowISO()
	_, err = r.db.Exec(`INSERT INTO agent_profiles(id,name,hostname,description,enabled,priority,config,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`, newID(), profile.Name, profile.Hostname, profile.Description, boolToInt(profile.Enabled), profile.Priority, raw, now, now)
	return err
}

// UpdateAgentProfile 更新采集配置。
func (r *BackupRepository) UpdateAgentProfile(id string, profile domain.AgentProfile) error {
	raw, err := marshalJSON(profile.Config)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE agent_profiles SET name=?,hostname=?,description=?,enabled=?,priority=?,config=?,updated_at=? WHERE id=?`,
		profile.Name, profile.Hostname, profile.Description, boolToInt(profile.Enabled), profile.Priority, raw, nowISO(), id)
	return err
}

// DeleteAgentProfile 删除采集配置。
func (r *BackupRepository) DeleteAgentProfile(id string) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM agent_profiles WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// ListAgentTokens 返回全部令牌（按创建时间倒序）。
func (r *BackupRepository) ListAgentTokens() ([]domain.AgentToken, error) {
	rows, err := r.db.Query(`SELECT id,token,name,COALESCE(hostname,''),enabled,COALESCE(last_seen_at,''),created_at
		FROM agent_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AgentToken{}
	for rows.Next() {
		var t domain.AgentToken
		var enabled int
		if err := rows.Scan(&t.ID, &t.Token, &t.Name, &t.Hostname, &enabled, &t.LastSeenAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

// FindAgentToken 按令牌值查找启用中的令牌。
func (r *BackupRepository) FindAgentToken(token string) (*domain.AgentToken, error) {
	row := r.db.QueryRow(`SELECT id,token,name,COALESCE(hostname,''),enabled,COALESCE(last_seen_at,''),created_at
		FROM agent_tokens WHERE token=? AND enabled=1`, token)
	var t domain.AgentToken
	var enabled int
	if err := row.Scan(&t.ID, &t.Token, &t.Name, &t.Hostname, &enabled, &t.LastSeenAt, &t.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.Enabled = enabled == 1
	return &t, nil
}

// CreateAgentToken 新建令牌。
func (r *BackupRepository) CreateAgentToken(name, hostname, token string) error {
	_, err := r.db.Exec(`INSERT INTO agent_tokens(id,token,name,hostname,enabled,created_at) VALUES(?,?,?,?,1,?)`,
		newID(), token, name, nullable(hostname), nowISO())
	return err
}

// AgentTokenExists 判断令牌值是否已被占用（含已停用令牌）。
func (r *BackupRepository) AgentTokenExists(token string) (bool, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM agent_tokens WHERE token=?`, token).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateAgentToken 更新令牌名称、绑定主机与启用状态。
func (r *BackupRepository) UpdateAgentToken(id, name, hostname string, enabled bool) error {
	_, err := r.db.Exec(`UPDATE agent_tokens SET name=?,hostname=?,enabled=? WHERE id=?`, name, nullable(hostname), boolToInt(enabled), id)
	return err
}

// DeleteAgentToken 删除令牌。
func (r *BackupRepository) DeleteAgentToken(id string) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM agent_tokens WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// TouchAgentToken 刷新令牌最后使用时间。
func (r *BackupRepository) TouchAgentToken(id, seenAt string) error {
	_, err := r.db.Exec(`UPDATE agent_tokens SET last_seen_at=? WHERE id=?`, seenAt, id)
	return err
}

// AgentBinaryMeta Agent 二进制元数据。
type AgentBinaryMeta struct {
	Platform   string `json:"platform"`
	Arch       string `json:"arch"`
	Variant    string `json:"variant"`
	GoVersion  string `json:"goVersion"`
	Version    string `json:"version"`
	Sha256     string `json:"sha256"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploadedAt"`
}

// SaveAgentBinaryMeta 保存指定槽位的二进制元数据。
func (r *BackupRepository) SaveAgentBinaryMeta(meta AgentBinaryMeta) error {
	_, err := r.db.Exec(`INSERT INTO agent_binaries(platform,arch,variant,go_version,version,sha256,size,uploaded_at) VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(platform,arch,variant) DO UPDATE SET go_version=excluded.go_version,version=excluded.version,sha256=excluded.sha256,size=excluded.size,uploaded_at=excluded.uploaded_at`,
		meta.Platform, meta.Arch, meta.Variant, meta.GoVersion, meta.Version, meta.Sha256, meta.Size, meta.UploadedAt)
	return err
}

// GetAgentBinaryMeta 读取指定槽位的二进制元数据。
func (r *BackupRepository) GetAgentBinaryMeta(platform, arch, variant string) (*AgentBinaryMeta, error) {
	row := r.db.QueryRow(`SELECT platform,arch,variant,COALESCE(go_version,''),version,sha256,size,uploaded_at
		FROM agent_binaries WHERE platform=? AND arch=? AND variant=?`, platform, arch, variant)
	var meta AgentBinaryMeta
	if err := row.Scan(&meta.Platform, &meta.Arch, &meta.Variant, &meta.GoVersion, &meta.Version, &meta.Sha256, &meta.Size, &meta.UploadedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

// ListAgentBinaryMeta 返回全部槽位的二进制元数据。
func (r *BackupRepository) ListAgentBinaryMeta() ([]AgentBinaryMeta, error) {
	rows, err := r.db.Query(`SELECT platform,arch,variant,COALESCE(go_version,''),version,sha256,size,uploaded_at
		FROM agent_binaries ORDER BY platform,arch,variant`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AgentBinaryMeta{}
	for rows.Next() {
		var meta AgentBinaryMeta
		if err := rows.Scan(&meta.Platform, &meta.Arch, &meta.Variant, &meta.GoVersion, &meta.Version, &meta.Sha256, &meta.Size, &meta.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, meta)
	}
	return out, rows.Err()
}

type profileScanner interface {
	Scan(dest ...any) error
}

func scanProfile(scanner profileScanner) (domain.AgentProfile, error) {
	var p domain.AgentProfile
	var enabled int
	var raw string
	if err := scanner.Scan(&p.ID, &p.Name, &p.Hostname, &p.Description, &enabled, &p.Priority, &raw, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return p, err
	}
	p.Enabled = enabled == 1
	p.Config = domain.NormalizeProfileConfig(domain.ParseJSONObject(raw))
	return p, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
