package service

import "testing"

func TestCloseHostUnknownIsSafe(t *testing.T) {
	hub := &AgentHub{conns: map[string]*agentConn{}}
	hub.CloseHost("missing")
}
