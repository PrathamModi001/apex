package ws_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"apex/api-gateway/internal/app"
	wsinfra "apex/api-gateway/internal/infra/ws"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newConnPair creates a client/server WebSocket pair via an httptest.Server.
// Returns the client-side connection; the server-side connection is sent on serverConn.
func newConnPair(t *testing.T) (client *websocket.Conn, serverConn *websocket.Conn) {
	t.Helper()
	ch := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		ch <- conn
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	select {
	case serverConn = <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server conn")
	}
	return client, serverConn
}

func TestHub_BroadcastSendsToRegisteredClient(t *testing.T) {
	hub := wsinfra.NewHub()
	go hub.Run()

	client, serverConn := newConnPair(t)
	hub.Register(serverConn)
	// Give the hub's select loop a moment to process the registration.
	time.Sleep(10 * time.Millisecond)

	event := app.DecisionEvent{
		InvoiceID:  "inv-1",
		Decision:   "approve",
		RiskScore:  42.0,
		AuditHash:  "abc123",
		VendorName: "Acme",
	}
	hub.Broadcast(event)

	// The client should receive the JSON message.
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(msg), "inv-1") {
		t.Errorf("expected message to contain inv-1, got: %s", msg)
	}
}

func TestHub_UnregisterRemovesClient(t *testing.T) {
	hub := wsinfra.NewHub()
	go hub.Run()

	_, serverConn := newConnPair(t)
	hub.Register(serverConn)
	time.Sleep(10 * time.Millisecond)

	// Unregister closes the server conn.
	hub.Unregister(serverConn)
	time.Sleep(20 * time.Millisecond)

	// Verify the connection was closed by trying to write — should fail.
	err := serverConn.WriteMessage(websocket.TextMessage, []byte("ping"))
	if err == nil {
		t.Error("expected write to fail after unregister, but succeeded")
	}
}

func TestHub_BroadcastToNoClients(t *testing.T) {
	hub := wsinfra.NewHub()
	go hub.Run()

	// Broadcast with no clients registered should not block or panic.
	done := make(chan struct{})
	go func() {
		hub.Broadcast(app.DecisionEvent{InvoiceID: "inv-0"})
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked with no clients")
	}
}

func TestHub_MultipleClients(t *testing.T) {
	hub := wsinfra.NewHub()
	go hub.Run()

	var clients [3]*websocket.Conn
	for i := 0; i < 3; i++ {
		client, serverConn := newConnPair(t)
		clients[i] = client
		hub.Register(serverConn)
	}
	time.Sleep(20 * time.Millisecond)

	hub.Broadcast(app.DecisionEvent{InvoiceID: "inv-multi", Decision: "reject"})

	for i, client := range clients {
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := client.ReadMessage()
		if err != nil {
			t.Fatalf("client %d read: %v", i, err)
		}
		if !strings.Contains(string(msg), "inv-multi") {
			t.Errorf("client %d: expected inv-multi in message, got: %s", i, msg)
		}
	}
}

// Compile-time interface check.
var _ app.EventBus = (*wsinfra.Hub)(nil)
