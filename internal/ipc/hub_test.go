package ipc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type msg struct {
	Type string `json:"type"`
	N    int    `json:"n"`
}

func dial(t *testing.T, ctx context.Context, url string) *websocket.Conn {
	t.Helper()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(url, "http"), nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(func() { c.CloseNow() })
	return c
}

func read(t *testing.T, ctx context.Context, c *websocket.Conn) msg {
	t.Helper()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m msg
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("bad json %q: %v", data, err)
	}
	return m
}

func setup(t *testing.T) (*Hub, *httptest.Server, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	h := NewHub("s3cret", func() []any { return []any{msg{Type: "hello"}} })
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return h, srv, ctx
}

func TestGreetingThenBroadcastToAllClients(t *testing.T) {
	h, srv, ctx := setup(t)
	a := dial(t, ctx, srv.URL+"/s3cret")
	b := dial(t, ctx, srv.URL+"/s3cret")
	for _, c := range []*websocket.Conn{a, b} {
		if m := read(t, ctx, c); m.Type != "hello" {
			t.Fatalf("first message = %+v, want hello", m)
		}
	}
	waitClients(t, h, 2)
	h.Broadcast(msg{Type: "chord", N: 1})
	h.Broadcast(msg{Type: "chord", N: 2})
	for _, c := range []*websocket.Conn{a, b} {
		if m := read(t, ctx, c); m.N != 1 {
			t.Errorf("got %+v", m)
		}
		if m := read(t, ctx, c); m.N != 2 {
			t.Errorf("got %+v", m)
		}
	}
}

func TestWrongTokenRejected(t *testing.T) {
	_, srv, ctx := setup(t)
	for _, path := range []string{"/", "/nope", "/s3cret/extra"} {
		_, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+path, nil)
		if err == nil {
			t.Errorf("%s: dial should fail", path)
			continue
		}
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: want 404, got %v", path, resp)
		}
	}
}

func TestStalledClientDoesNotBlockOthers(t *testing.T) {
	h, srv, ctx := setup(t)
	stalled := dial(t, ctx, srv.URL+"/s3cret") // never reads
	_ = stalled
	live := dial(t, ctx, srv.URL+"/s3cret")
	read(t, ctx, live) // hello
	waitClients(t, h, 2)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			h.Broadcast(msg{Type: "spam", N: i})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Broadcast blocked on a stalled client")
	}
	// The live client still gets messages (possibly after drops).
	if m := read(t, ctx, live); m.Type != "spam" {
		t.Errorf("got %+v", m)
	}
}

func TestClientsGoneAfterDisconnect(t *testing.T) {
	h, srv, ctx := setup(t)
	c := dial(t, ctx, srv.URL+"/s3cret")
	read(t, ctx, c)
	waitClients(t, h, 1)
	c.Close(websocket.StatusNormalClosure, "bye")
	waitClients(t, h, 0)
}

func waitClients(t *testing.T, h *Hub, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for h.Clients() != n {
		if time.Now().After(deadline) {
			t.Fatalf("clients = %d, want %d", h.Clients(), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
