package mtprogo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedFileSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := EncryptedFileSession(filepath.Join(dir, "bot.session"), "secret")
	ctx := context.Background()
	want := []byte("session-data")
	if err := store.Save(ctx, SessionData{Name: "bot", Data: want}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "bot.session"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == string(want) {
		t.Fatal("session was not encrypted")
	}
	got, ok, err := store.Load(ctx, "bot")
	if err != nil || !ok {
		t.Fatalf("load failed ok=%v err=%v", ok, err)
	}
	if string(got.Data) != string(want) {
		t.Fatalf("got %q want %q", got.Data, want)
	}
	if err := clearSession(ctx, store, "bot"); err != nil {
		t.Fatal(err)
	}
	_, ok, err = store.Load(ctx, "bot")
	if err != nil || ok {
		t.Fatalf("clear failed ok=%v err=%v", ok, err)
	}
}
