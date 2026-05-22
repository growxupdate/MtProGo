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

func TestChatTypeFilters(t *testing.T) {
	if !Private(&updates.Message{ChatType: updates.ChatPrivate}) {
		t.Fatal("private filter should match private chat")
	}
	if !Group(&updates.Message{ChatType: updates.ChatGroup}) {
		t.Fatal("group filter should match group chat")
	}
	if !Channel(&updates.Message{ChatType: updates.ChatChannel}) {
		t.Fatal("channel filter should match channel chat")
	}
	if Private(&updates.Message{ChatType: updates.ChatGroup}) {
		t.Fatal("private filter should not match group chat")
	}
}
