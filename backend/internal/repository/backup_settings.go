package repository

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
)

// LoadBackupSettings 从 basic_config 的三个业务区块读取基础参数，缺行或解析失败时回填默认值。
func (r *BackupRepository) LoadBackupSettings() (domain.AppSettings, error) {
	rows, err := r.db.Query(`SELECT name,config FROM basic_config WHERE name IN (?,?,?)`,
		domain.BaseSectionThresholds, domain.BaseSectionAgent, domain.BaseSectionAlarm)
	if err != nil {
		return domain.DefaultAppSettings(), err
	}
	defer rows.Close()
	sections := map[string]map[string]any{}
	for rows.Next() {
		var name, raw string
		if err := rows.Scan(&name, &raw); err != nil {
			return domain.DefaultAppSettings(), err
		}
		sections[name] = domain.ParseJSONObject(raw)
	}
	if err := rows.Err(); err != nil {
		return domain.DefaultAppSettings(), err
	}
	return domain.MergeAppSettingsSections(sections), nil
}

// SaveBackupSettings 把基础参数按区块写回 basic_config 对应行。
func (r *BackupRepository) SaveBackupSettings(settings domain.AppSettings) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := appcommon.NowTimestamp()
	for name, section := range settings.Sections() {
		raw, err := marshalJSON(section)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO basic_config(name,config,updated_at) VALUES(?,?,?)
			ON CONFLICT(name) DO UPDATE SET config=excluded.config,updated_at=excluded.updated_at`,
			name, raw, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
