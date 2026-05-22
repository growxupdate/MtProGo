package updates

import "testing"

func TestNormalizeChatID(t *testing.T) {
	cases := []struct {
		raw  int64
		kind string
		want int64
	}{
		{12345, ChatPrivate, 12345},
		{12345, ChatGroup, -12345},
		{1251846294, ChatSupergroup, -1001251846294},
		{1251846294, ChatChannel, -1001251846294},
	}
	for _, tc := range cases {
		got := NormalizeChatID(tc.raw, tc.kind)
		if got != tc.want {
			t.Fatalf("NormalizeChatID(%d, %q) = %d, want %d", tc.raw, tc.kind, got, tc.want)
		}
		if raw := RawChatID(got); raw != tc.raw {
			t.Fatalf("RawChatID(%d) = %d, want %d", got, raw, tc.raw)
		}
	}
}

func TestLooksLikeChannelChatID(t *testing.T) {
	if !LooksLikeChannelChatID(-1001251846294) {
		t.Fatal("expected -100... id to look like channel chat id")
	}
	if LooksLikeChannelChatID(-1251846294) {
		t.Fatal("basic negative group id should not look like channel chat id")
	}
}
