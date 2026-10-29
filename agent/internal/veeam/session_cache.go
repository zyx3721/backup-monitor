package veeam

import (
	"strings"
	"sync"
)

var sessionDetails = struct {
	sync.Mutex
	items map[string]Session
}{items: map[string]Session{}}

// CachedSession 返回已缓存的会话详情。
func CachedSession(id string) (Session, bool) {
	key := strings.TrimSpace(id)
	if key == "" {
		return Session{}, false
	}
	sessionDetails.Lock()
	defer sessionDetails.Unlock()
	session, ok := sessionDetails.items[key]
	return session, ok
}

// RememberSession 缓存已结束会话的详情。
func RememberSession(session Session) {
	key := strings.TrimSpace(session.ID)
	if key == "" || !session.HasEnd {
		return
	}
	sessionDetails.Lock()
	defer sessionDetails.Unlock()
	sessionDetails.items[key] = session
}
