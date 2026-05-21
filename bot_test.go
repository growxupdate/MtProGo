package mtprogo

import "testing"

func TestNewBotRequiresToken(t *testing.T) {
	if _, err := NewBot(BotConfig{}); err == nil {
		t.Fatal("expected token error")
	}
}
