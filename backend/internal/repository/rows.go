package repository

import "database/sql"

// RowsToMaps 把 SQL 查询结果转为便于 JSON 输出的 map 切片，
// 保留原列名并将 SQLite 的 []byte 值归一为字符串。
func RowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range columns {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(columns))
		for i, column := range columns {
			row[column] = normalizeValue(values[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func normalizeValue(value interface{}) interface{} {
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return value
}
