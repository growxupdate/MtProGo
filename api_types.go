package mtprogo

import "github.com/growxupdate/MtProGo/mtproto"

// User is a compact Telegram user/bot profile returned by MTProto helpers.
type User = mtproto.User

// PeerRef is a compact peer reference kept in MtProGo peer caches.
type PeerRef = mtproto.PeerRef

// SendOptions controls MTProto messages.sendMessage flags.
type SendOptions struct {
	ReplyToMessageID int
	Silent           bool
	NoWebpage        bool
	Background       bool
}

// EditOptions controls MTProto messages.editMessage flags.
type EditOptions struct {
	NoWebpage bool
}

// ForwardOptions controls MTProto messages.forwardMessages flags.
type ForwardOptions struct {
	Silent            bool
	DropAuthor        bool
	DropMediaCaptions bool
}

func toMTProtoSendOptions(opts SendOptions) mtproto.SendOptions {
	return mtproto.SendOptions{ReplyToMessageID: opts.ReplyToMessageID, Silent: opts.Silent, NoWebpage: opts.NoWebpage, Background: opts.Background}
}

func toMTProtoEditOptions(opts EditOptions) mtproto.EditOptions {
	return mtproto.EditOptions{NoWebpage: opts.NoWebpage}
}

func toMTProtoForwardOptions(opts ForwardOptions) mtproto.ForwardOptions {
	return mtproto.ForwardOptions{Silent: opts.Silent, DropAuthor: opts.DropAuthor, DropMediaCaptions: opts.DropMediaCaptions}
}
