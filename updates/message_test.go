package updates

import (
	"context"
	"testing"
)

func TestDispatcher(t *testing.T) {
	d := NewDispatcher()
	called := false
	d.OnMessage(func(m *Message) bool { return m.Text == "/start" }, func(ctx context.Context, m *Message) error {
		called = true
		return nil
	})
	if err := d.DispatchMessage(context.Background(), &Message{Text: "/start"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}
