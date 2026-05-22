package filters

import (
	"regexp"
	"strings"

	"github.com/growxupdate/MtProGo/updates"
)

// Filter checks whether a message matches.
type Filter = updates.MessageFilter

// All matches every message.
func All(m *updates.Message) bool { return true }

// Text matches messages with non-empty text.
func Text(m *updates.Message) bool { return m != nil && m.Text != "" }

// Private matches one-to-one user/bot chats.
func Private(m *updates.Message) bool { return m != nil && m.ChatType == updates.ChatPrivate }

// Group matches basic groups and supergroups.
func Group(m *updates.Message) bool {
	return m != nil && (m.ChatType == updates.ChatGroup || m.ChatType == updates.ChatSupergroup)
}

// Supergroup matches Telegram supergroups represented by MTProto PeerChannel.
func Supergroup(m *updates.Message) bool { return m != nil && m.ChatType == updates.ChatSupergroup }

// Channel matches broadcast channels. For compatibility it also returns true for
// supergroups, because early MtProGo versions exposed PeerChannel as channel.
func Channel(m *updates.Message) bool {
	return m != nil && (m.ChatType == updates.ChatChannel || m.ChatType == updates.ChatSupergroup)
}

// ChatType matches an exact normalized chat type.
func ChatType(kind string) Filter {
	return func(m *updates.Message) bool { return m != nil && m.ChatType == kind }
}

// Command matches /command and /command@bot forms.
func Command(command string) Filter {
	command = strings.TrimPrefix(strings.TrimSpace(command), "/")
	return func(m *updates.Message) bool {
		if m == nil || command == "" || !strings.HasPrefix(m.Text, "/") {
			return false
		}
		first := strings.Fields(m.Text)
		if len(first) == 0 {
			return false
		}
		cmd := strings.TrimPrefix(first[0], "/")
		if at := strings.IndexByte(cmd, '@'); at >= 0 {
			cmd = cmd[:at]
		}
		return strings.EqualFold(cmd, command)
	}
}

// ChatID matches a specific chat.
func ChatID(chatID int64) Filter {
	return func(m *updates.Message) bool { return m != nil && m.ChatID == chatID }
}

// FromID matches a specific sender.
func FromID(userID int64) Filter {
	return func(m *updates.Message) bool { return m != nil && m.FromID == userID }
}

// Regex matches message text with a compiled regular expression.
func Regex(pattern string) Filter {
	re := regexp.MustCompile(pattern)
	return func(m *updates.Message) bool { return m != nil && re.MatchString(m.Text) }
}

// And combines filters with logical AND.
func And(filters ...Filter) Filter {
	return func(m *updates.Message) bool {
		for _, filter := range filters {
			if filter != nil && !filter(m) {
				return false
			}
		}
		return true
	}
}

// Or combines filters with logical OR.
func Or(filters ...Filter) Filter {
	return func(m *updates.Message) bool {
		for _, filter := range filters {
			if filter != nil && filter(m) {
				return true
			}
		}
		return false
	}
}
