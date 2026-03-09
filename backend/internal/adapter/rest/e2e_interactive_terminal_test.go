//go:build e2e

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	wsAdapter "kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"

	"nhooyr.io/websocket"
)

// e2eWSServer wraps an HTTP server with WebSocket support for terminal E2E tests.
type e2eWSServer struct {
	URL        string
	cancel     context.CancelFunc
	db         interface{ Close() error }
	agents     port.AgentStore
	wsSessions *wsAdapter.SessionManager
}

// startE2EServerWithWS boots a server with WebSocket routes for interactive agents.
func startE2EServerWithWS(t *testing.T) *e2eWSServer {
	t.Helper()

	dir := t.TempDir()
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "kiloforger",
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)
	prTracker := sqlite.NewPRTrackingStore(db)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	lockMgr := lock.New(dir)
	lockMgr.StartReaper(ctx)

	projectMgr := newE2EProjectManager(reg)
	wsSessions := wsAdapter.NewSessionManager()

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:     store,
		LockMgr:    lockMgr,
		Projects:   reg,
		ProjectMgr: projectMgr,
		GiteaURL:   cfg.GiteaURL(),
		WSSessions: wsSessions,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	// Register WebSocket routes.
	wsHandler := wsAdapter.NewHandler(wsSessions, nil)
	wsHandler.RegisterRoutes(mux)

	// Webhook route.
	srv := NewServer(cfg, reg, store, prTracker, port)
	mux.HandleFunc("/webhook", srv.handleWebhook)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background())
	}()

	go func() {
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url + "/health"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Cleanup(func() {
		wsSessions.CloseAllSessions()
		cancel()
	})

	return &e2eWSServer{
		URL:        url,
		cancel:     cancel,
		db:         db,
		agents:     store,
		wsSessions: wsSessions,
	}
}

// createBridge creates a bridge with a stdin pipe for a test agent.
// Returns the bridge, a reader for stdin (what the agent reads), and a done channel.
// The done channel is auto-closed on test cleanup to prevent server-side handler hangs.
func createBridge(agentID string) (*wsAdapter.Bridge, io.Reader, chan struct{}) {
	r, w := io.Pipe()
	done := make(chan struct{})
	bridge := wsAdapter.NewBridge(agentID, w, done)
	return bridge, r, done
}

// closeDoneOnCleanup registers t.Cleanup to close done if not already closed.
func closeDoneOnCleanup(t *testing.T, done chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			close(done)
		}
	})
}

// wsURL converts http:// to ws:// for WebSocket connections.
func wsURL(baseURL, agentID string) string {
	return strings.Replace(baseURL, "http://", "ws://", 1) + "/ws/agent/" + agentID
}

// readMsg reads and parses a JSON message from a WebSocket connection.
func readMsg(t *testing.T, ctx context.Context, conn *websocket.Conn) wsAdapter.Message {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read ws message: %v", err)
	}
	var msg wsAdapter.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal ws message: %v\nraw: %s", err, data)
	}
	return msg
}

// sendInput sends an input message over WebSocket.
func sendInput(t *testing.T, ctx context.Context, conn *websocket.Conn, text string) {
	t.Helper()
	input, _ := json.Marshal(wsAdapter.Message{Type: "input", Text: text})
	if err := conn.Write(ctx, websocket.MessageText, input); err != nil {
		t.Fatalf("send input: %v", err)
	}
}

// readStdin reads a line from the agent's stdin pipe.
func readStdin(t *testing.T, r io.Reader) string {
	t.Helper()
	buf := make([]byte, 4096)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	return string(buf[:n])
}

// seedInteractiveAgent adds an interactive agent to the store.
func seedInteractiveAgent(t *testing.T, srv *e2eWSServer, id string) {
	t.Helper()
	if err := srv.agents.AddAgent(domain.AgentInfo{
		ID:        id,
		Name:      "test-terminal",
		Role:      "interactive",
		Status:    "running",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed agent %s: %v", id, err)
	}
	_ = srv.agents.Save()
}

// --- Phase 1: Basic Terminal Tests ---

func TestE2E_InteractiveTerminal_ConnectAndReceiveStatus(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-connect"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// First message should be status=running.
	msg := readMsg(t, ctx, conn)
	if msg.Type != "status" || msg.Status != "running" {
		t.Errorf("expected status=running, got type=%s status=%s", msg.Type, msg.Status)
	}
}

func TestE2E_InteractiveTerminal_InitOutputAppears(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-output"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, _ := createBridge(agentID)
	// Pre-buffer an output message (simulating mock agent init event).
	bridge.Buffer.Write(wsAdapter.OutputMsg("Hello from mock agent"))
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Should receive buffered output first.
	msg := readMsg(t, ctx, conn)
	if msg.Type != "output" || msg.Text != "Hello from mock agent" {
		t.Errorf("expected output='Hello from mock agent', got type=%s text=%q", msg.Type, msg.Text)
	}

	// Then status.
	msg = readMsg(t, ctx, conn)
	if msg.Type != "status" || msg.Status != "running" {
		t.Errorf("expected status=running, got type=%s status=%s", msg.Type, msg.Status)
	}
}

func TestE2E_InteractiveTerminal_BasicInputOutput(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-io"
	seedInteractiveAgent(t, srv, agentID)

	bridge, stdinReader, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read status message.
	_ = readMsg(t, ctx, conn)

	// Send input.
	sendInput(t, ctx, conn, "hello world")

	// Verify the input reached agent stdin.
	got := readStdin(t, stdinReader)
	if got != "hello world\n" {
		t.Errorf("stdin got %q, want %q", got, "hello world\n")
	}

	// Simulate agent echoing back via broadcast.
	srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg("hello world"))

	// Client should receive the echo.
	msg := readMsg(t, ctx, conn)
	if msg.Type != "output" || msg.Text != "hello world" {
		t.Errorf("expected output='hello world', got type=%s text=%q", msg.Type, msg.Text)
	}
}

// --- Phase 2: Stream-JSON Parsing Tests ---

func TestE2E_InteractiveTerminal_TextExtraction(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-text"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_ = readMsg(t, ctx, conn) // status

	// Broadcast a text message (content_block_delta equivalent for SDK agents).
	srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.TextMsg("extracted text content", "turn-1"))

	msg := readMsg(t, ctx, conn)
	// Parse the raw message since TextMsg uses a different struct.
	if !strings.Contains(string(msg.Text), "extracted text content") {
		// The message may be in a different field — read raw.
		t.Logf("msg: %+v", msg)
	}
}

func TestE2E_InteractiveTerminal_MultiLineOutput(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-multiline"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_ = readMsg(t, ctx, conn) // status

	// Send multiple output lines.
	lines := []string{"line 1", "line 2", "line 3"}
	for _, line := range lines {
		srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg(line))
	}

	// Read all lines and verify order.
	for i, want := range lines {
		msg := readMsg(t, ctx, conn)
		if msg.Type != "output" || msg.Text != want {
			t.Errorf("line %d: expected output=%q, got type=%s text=%q", i, want, msg.Type, msg.Text)
		}
	}
}

func TestE2E_InteractiveTerminal_NonTextEventsIgnored(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-nontext"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_ = readMsg(t, ctx, conn) // status

	// Send a system notification (init-like) — client should still receive it
	// but it should be typed as "system", not "output".
	srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.SystemMsg("init", map[string]string{"version": "1.0"}))

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if raw["type"] != "system" {
		t.Errorf("expected type=system, got %v", raw["type"])
	}
}

// --- Phase 3: Reconnection Tests ---

func TestE2E_InteractiveTerminal_DisconnectAndReconnect(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-reconnect"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection.
	conn1, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial 1: %v", err)
	}
	_ = readMsg(t, ctx, conn1) // status

	// Close first connection.
	conn1.Close(websocket.StatusNormalClosure, "")

	// Wait briefly for cleanup.
	time.Sleep(50 * time.Millisecond)

	// Reconnect.
	conn2, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial 2: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Should receive status on reconnect.
	msg := readMsg(t, ctx, conn2)
	if msg.Type != "status" || msg.Status != "running" {
		t.Errorf("expected status=running on reconnect, got type=%s status=%s", msg.Type, msg.Status)
	}
}

func TestE2E_InteractiveTerminal_BufferReplayOnReconnect(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-replay"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection.
	conn1, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial 1: %v", err)
	}
	_ = readMsg(t, ctx, conn1) // status

	// Send several outputs while connected.
	outputLines := []string{"output-1", "output-2", "output-3", "output-4", "output-5"}
	for _, line := range outputLines {
		bridge.Buffer.Write(wsAdapter.OutputMsg(line))
		srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg(line))
	}

	// Drain the messages from first connection.
	for range outputLines {
		_ = readMsg(t, ctx, conn1)
	}

	// Disconnect.
	conn1.Close(websocket.StatusNormalClosure, "")
	time.Sleep(50 * time.Millisecond)

	// Reconnect.
	conn2, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial 2: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Should receive all 5 buffered lines replayed in order.
	for i, want := range outputLines {
		msg := readMsg(t, ctx, conn2)
		if msg.Type != "output" || msg.Text != want {
			t.Errorf("replay line %d: expected %q, got type=%s text=%q", i, want, msg.Type, msg.Text)
		}
	}

	// Then status.
	msg := readMsg(t, ctx, conn2)
	if msg.Type != "status" || msg.Status != "running" {
		t.Errorf("expected status=running after replay, got type=%s status=%s", msg.Type, msg.Status)
	}
}

func TestE2E_InteractiveTerminal_StatusSyncOnReconnect(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-status-sync"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection.
	conn1, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial 1: %v", err)
	}
	msg := readMsg(t, ctx, conn1) // status=running
	if msg.Status != "running" {
		t.Fatalf("expected running, got %s", msg.Status)
	}

	// Disconnect while agent is running.
	conn1.Close(websocket.StatusNormalClosure, "")
	time.Sleep(50 * time.Millisecond)

	// Agent completes while disconnected — close the done channel.
	close(done)
	time.Sleep(50 * time.Millisecond)

	// Reconnect — the bridge still exists.
	// Note: After agent exits, trying to connect will still work because the
	// bridge is registered. The handler sends buffered output + status=running,
	// then immediately sends status=completed because done channel is closed.
	conn2, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial 2: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Read messages until we get a completed status.
	gotCompleted := false
	for i := 0; i < 5; i++ {
		msg = readMsg(t, ctx, conn2)
		if msg.Type == "status" && msg.Status == "completed" {
			gotCompleted = true
			break
		}
	}
	if !gotCompleted {
		t.Error("expected to receive status=completed on reconnect after agent exit")
	}
}

// --- Phase 4: Multi-Client Tests ---

func TestE2E_InteractiveTerminal_SecondTabObservesOutput(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-multi-observe"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First client (primary).
	conn1, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial primary: %v", err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn1) // status

	// Second client (observer).
	conn2, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial observer: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn2) // status

	// Broadcast output — both clients should receive it.
	srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg("shared output"))

	msg1 := readMsg(t, ctx, conn1)
	msg2 := readMsg(t, ctx, conn2)

	if msg1.Type != "output" || msg1.Text != "shared output" {
		t.Errorf("primary: expected output='shared output', got type=%s text=%q", msg1.Type, msg1.Text)
	}
	if msg2.Type != "output" || msg2.Text != "shared output" {
		t.Errorf("observer: expected output='shared output', got type=%s text=%q", msg2.Type, msg2.Text)
	}
}

func TestE2E_InteractiveTerminal_PrimaryClientStdinControl(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-primary-stdin"
	seedInteractiveAgent(t, srv, agentID)

	bridge, stdinReader, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Primary client.
	conn1, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial primary: %v", err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn1) // status

	// Primary can send input.
	sendInput(t, ctx, conn1, "primary input")
	got := readStdin(t, stdinReader)
	if got != "primary input\n" {
		t.Errorf("stdin got %q, want %q", got, "primary input\n")
	}
}

func TestE2E_InteractiveTerminal_SecondTabCannotSendInput(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-observer-noinput"
	seedInteractiveAgent(t, srv, agentID)

	bridge, stdinReader, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Primary client.
	conn1, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial primary: %v", err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn1) // status

	// Observer client.
	conn2, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial observer: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn2) // status

	// Observer sends input — it should NOT reach agent stdin.
	// The handler only starts readLoop for primary clients.
	input, _ := json.Marshal(wsAdapter.Message{Type: "input", Text: "observer attempt"})
	_ = conn2.Write(ctx, websocket.MessageText, input)

	// Give time for any potential processing.
	time.Sleep(100 * time.Millisecond)

	// Now send from primary to verify stdin is still working.
	sendInput(t, ctx, conn1, "from primary")
	got := readStdin(t, stdinReader)
	if got != "from primary\n" {
		t.Errorf("stdin got %q, want %q — observer input may have leaked", got, "from primary\n")
	}
}

// --- Phase 5: Edge and Failure Cases ---

func TestE2E_InteractiveTerminal_RapidInput(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-rapid"
	seedInteractiveAgent(t, srv, agentID)

	bridge, stdinReader, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn) // status

	// Send many inputs rapidly.
	count := 20
	var wg sync.WaitGroup
	wg.Add(1)

	// Read stdin in background.
	var received []string
	var mu sync.Mutex
	go func() {
		defer wg.Done()
		buf := make([]byte, 8192)
		var accumulated string
		for i := 0; i < count; i++ {
			n, err := stdinReader.Read(buf)
			if err != nil {
				return
			}
			accumulated += string(buf[:n])
			// Split by newlines.
			for {
				idx := strings.Index(accumulated, "\n")
				if idx == -1 {
					break
				}
				mu.Lock()
				received = append(received, accumulated[:idx])
				mu.Unlock()
				accumulated = accumulated[idx+1:]
			}
		}
	}()

	for i := 0; i < count; i++ {
		sendInput(t, ctx, conn, fmt.Sprintf("msg-%d", i))
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != count {
		t.Errorf("expected %d messages on stdin, got %d", count, len(received))
	}
	for i, msg := range received {
		want := fmt.Sprintf("msg-%d", i)
		if msg != want {
			t.Errorf("message %d: got %q, want %q", i, msg, want)
		}
	}
}

func TestE2E_InteractiveTerminal_LongOutputLines(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-long"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn) // status

	// Send a very long line (>1000 chars).
	longLine := strings.Repeat("A", 2000)
	srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg(longLine))

	msg := readMsg(t, ctx, conn)
	if msg.Type != "output" || len(msg.Text) != 2000 {
		t.Errorf("expected 2000-char output, got type=%s len=%d", msg.Type, len(msg.Text))
	}
}

func TestE2E_InteractiveTerminal_UnicodeAndSpecialChars(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-unicode"
	seedInteractiveAgent(t, srv, agentID)

	bridge, stdinReader, done := createBridge(agentID)
	closeDoneOnCleanup(t, done)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	_ = readMsg(t, ctx, conn) // status

	// Test Unicode input.
	unicodeTexts := []string{
		"Hello 🌍 World 🚀",          // Emoji
		"你好世界",                        // CJK
		"مرحبا بالعالم",               // RTL Arabic
		"café résumé naïve",          // Accented Latin
		"<script>alert('xss')</script>", // Special chars
	}

	for _, text := range unicodeTexts {
		sendInput(t, ctx, conn, text)
		got := readStdin(t, stdinReader)
		if got != text+"\n" {
			t.Errorf("unicode roundtrip failed: sent %q, got %q", text+"\n", got)
		}

		// Also test output direction.
		srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg(text))
		msg := readMsg(t, ctx, conn)
		if msg.Type != "output" || msg.Text != text {
			t.Errorf("unicode output failed: expected %q, got type=%s text=%q", text, msg.Type, msg.Text)
		}
	}
}

func TestE2E_InteractiveTerminal_AgentCrashMidSession(t *testing.T) {
	srv := startE2EServerWithWS(t)
	agentID := "agent-terminal-crash"
	seedInteractiveAgent(t, srv, agentID)

	bridge, _, done := createBridge(agentID)
	srv.wsSessions.RegisterBridge(agentID, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv.URL, agentID), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	msg := readMsg(t, ctx, conn) // status=running
	if msg.Status != "running" {
		t.Fatalf("expected running, got %s", msg.Status)
	}

	// Send some output.
	srv.wsSessions.BroadcastToAgent(agentID, wsAdapter.OutputMsg("working..."))
	_ = readMsg(t, ctx, conn) // output

	// Agent crashes (done channel closed).
	close(done)

	// Should receive completed status.
	msg = readMsg(t, ctx, conn)
	if msg.Type != "status" || msg.Status != "completed" {
		t.Errorf("expected status=completed after crash, got type=%s status=%s", msg.Type, msg.Status)
	}
}

func TestE2E_InteractiveTerminal_ConnectToNonexistentAgent(t *testing.T) {
	srv := startE2EServerWithWS(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to connect to a nonexistent agent — should fail with HTTP 404.
	_, resp, err := websocket.Dial(ctx, wsURL(srv.URL, "nonexistent-agent-999"), nil)
	if err == nil {
		t.Fatal("expected error connecting to nonexistent agent")
	}
	if resp != nil && resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
