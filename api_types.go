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

// UploadedFile describes a file uploaded through MTProto upload.saveFilePart/upload.saveBigFilePart.
type UploadedFile = mtproto.UploadedFile

// UploadOptions controls MTProto streaming upload chunking and progress.
type UploadOptions = mtproto.UploadOptions

// DownloadOptions controls MTProto upload.getFile chunking and progress.
type DownloadOptions = mtproto.DownloadOptions

// ProgressFunc receives completed bytes and total bytes.
type ProgressFunc = mtproto.ProgressFunc

// InputFileLocation is a raw upload.getFile location.
type InputFileLocation = mtproto.InputFileLocation

// DocumentFileLocation identifies a document/video/audio/voice file for download.
type DocumentFileLocation = mtproto.DocumentFileLocation

// PhotoFileLocation identifies a photo size for download.
type PhotoFileLocation = mtproto.PhotoFileLocation

// SendMediaOptions controls MTProto media sending.
type SendMediaOptions struct {
	Caption          string
	ReplyToMessageID int
	Silent           bool
	Background       bool
	Spoiler          bool
	ForceFile        bool
	MIMEType         string
	FileName         string
}

func toMTProtoSendMediaOptions(opts SendMediaOptions) mtproto.SendMediaOptions {
	return mtproto.SendMediaOptions{
		Caption:          opts.Caption,
		ReplyToMessageID: opts.ReplyToMessageID,
		Silent:           opts.Silent,
		Background:       opts.Background,
		Spoiler:          opts.Spoiler,
		ForceFile:        opts.ForceFile,
		MIMEType:         opts.MIMEType,
		FileName:         opts.FileName,
	}
}
