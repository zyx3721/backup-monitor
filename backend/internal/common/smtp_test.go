package common

import (
	"net/smtp"
	"testing"
)

func TestInsecurePlainAuthWorksOnPlainConnection(t *testing.T) {
	info := &smtp.ServerInfo{Name: "smtp.example.com"}
	if _, _, err := smtp.PlainAuth("", "user", "secret", "smtp.example.com").Start(info); err == nil {
		t.Fatal("smtp.PlainAuth should reject an unencrypted connection")
	}
	mechanism, payload, err := InsecurePlainAuth("user", "secret").Start(info)
	if err != nil {
		t.Fatalf("Start on plain connection returned error: %v", err)
	}
	if mechanism != "PLAIN" {
		t.Fatalf("mechanism = %q, want PLAIN", mechanism)
	}
	if string(payload) != "\x00user\x00secret" {
		t.Fatalf("payload = %q, want \\x00user\\x00secret", payload)
	}
}

func TestInsecurePlainAuthNextIsEmpty(t *testing.T) {
	next, err := InsecurePlainAuth("user", "secret").Next(nil, true)
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if len(next) != 0 {
		t.Fatalf("Next payload = %q, want empty", next)
	}
}
