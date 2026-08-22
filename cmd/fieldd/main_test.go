package main

import (
"encoding/json"
"net"
"path/filepath"
"testing"
"time"

"emergion-sovereign-runtime/internal/reason"
"emergion-sovereign-runtime/pkg/fieldapi"
)

func TestDaemonIPCProtocol(t *testing.T) {
tmpDir := t.TempDir()
statePath := filepath.Join(tmpDir, "state")
sockPath := filepath.Join(tmpDir, "test.sock")

rt, err := fieldapi.Open(statePath, reason.GemmaFromEnv())
if err != nil {
t.Fatalf("failed to open runtime: %v", err)
}

listener, err := net.Listen("unix", sockPath)
if err != nil {
t.Fatalf("failed to listen on unix socket: %v", err)
}
defer listener.Close()

go func() {
conn, err := listener.Accept()
if err != nil {
return
}
handleConn(rt, conn)
}()

time.Sleep(50 * time.Millisecond)

conn, err := net.Dial("unix", sockPath)
if err != nil {
t.Fatalf("failed to connect to socket: %v", err)
}
defer conn.Close()

encoder := json.NewEncoder(conn)
decoder := json.NewDecoder(conn)

// 1. Test status call over IPC
req := Request{Method: "status"}
if err := encoder.Encode(req); err != nil {
t.Fatalf("failed to send status request: %v", err)
}

var resp Response
if err := decoder.Decode(&resp); err != nil {
t.Fatalf("failed to decode response: %v", err)
}

if resp.Error != "" {
t.Fatalf("unexpected daemon error: %s", resp.Error)
}

// 2. Test unknown method failure mode
badReq := Request{Method: "nonexistent_method"}
if err := encoder.Encode(badReq); err != nil {
t.Fatalf("failed to send bad request: %v", err)
}

var badResp Response
if err := decoder.Decode(&badResp); err != nil {
t.Fatalf("failed to decode error response: %v", err)
}

if badResp.Error == "" {
t.Fatal("expected error on unknown method, got nil")
}
}
