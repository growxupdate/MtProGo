package filters

import (
	"testing"

	"github.com/growxupdate/MtProGo/updates"
)

func TestCommand(t *testing.T) {
	if !Command("start")(&updates.Message{Text: "/start hello"}) {
		t.Fatal("expected command to match")
	}
	if !Command("start")(&updates.Message{Text: "/start@mybot hello"}) {
		t.Fatal("expected command with bot username to match")
	}
	if Command("help")(&updates.Message{Text: "/start"}) {
		t.Fatal("unexpected command match")
	}
}
