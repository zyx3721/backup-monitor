package common

import (
	"encoding/json"
	"net/http"
)

// DecodeJSON 解析请求体（上限 2MB、拒绝未知字段），失败时写回 400 并返回 false
func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}
