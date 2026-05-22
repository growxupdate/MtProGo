package mtprogo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/growxupdate/MtProGo/mtproto"
)

// UploadFile streams a file to Telegram over pure MTProto and returns an uploaded file handle.
func (b *MTProtoBot) UploadFile(ctx context.Context, r io.Reader, size int64, name string, opts UploadOptions) (*UploadedFile, error) {
	if b == nil || b.client == nil {
		return nil, errors.New("mtproto bot is not logged in")
	}
	var uploaded *UploadedFile
	err := b.withFloodWait(ctx, func(callCtx context.Context) error {
		f, err := b.client.UploadFile(callCtx, r, size, name, opts)
		if err == nil {
			uploaded = f
		}
		return err
	})
	return uploaded, err
}

// SendDocument uploads a local file and sends it as a document to a cached peer.
func (b *MTProtoBot) SendDocument(ctx context.Context, chatID int64, path string, opts SendMediaOptions) error {
	return b.sendLocalMedia(ctx, chatID, path, false, opts)
}

// SendPhoto uploads a local file and sends it as a photo to a cached peer.
func (b *MTProtoBot) SendPhoto(ctx context.Context, chatID int64, path string, opts SendMediaOptions) error {
	return b.sendLocalMedia(ctx, chatID, path, true, opts)
}

func (b *MTProtoBot) sendLocalMedia(ctx context.Context, chatID int64, path string, asPhoto bool, opts SendMediaOptions) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	if opts.FileName == "" {
		opts.FileName = name
	}
	if opts.MIMEType == "" {
		if mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); mt != "" {
			opts.MIMEType = mt
		}
	}
	if opts.MIMEType == "" {
		opts.MIMEType = "application/octet-stream"
	}
	uploaded, err := b.UploadFile(ctx, f, st.Size(), name, UploadOptions{Progress: nil})
	if err != nil {
		return err
	}
	if asPhoto {
		return b.SendUploadedPhoto(ctx, chatID, uploaded, opts)
	}
	return b.SendUploadedDocument(ctx, chatID, uploaded, opts)
}

// SendUploadedDocument sends an already uploaded file as a document to a cached peer.
func (b *MTProtoBot) SendUploadedDocument(ctx context.Context, chatID int64, file *UploadedFile, opts SendMediaOptions) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	peer, ok := b.getPeer(chatID)
	if !ok {
		return fmt.Errorf("mtproto: peer %d is not cached yet", chatID)
	}
	return b.withFloodWait(ctx, func(callCtx context.Context) error {
		_, err := b.client.MessagesSendUploadedDocumentToPeer(callCtx, peer, file, toMTProtoSendMediaOptions(opts))
		return err
	})
}

// SendUploadedPhoto sends an already uploaded file as a photo to a cached peer.
func (b *MTProtoBot) SendUploadedPhoto(ctx context.Context, chatID int64, file *UploadedFile, opts SendMediaOptions) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	peer, ok := b.getPeer(chatID)
	if !ok {
		return fmt.Errorf("mtproto: peer %d is not cached yet", chatID)
	}
	return b.withFloodWait(ctx, func(callCtx context.Context) error {
		_, err := b.client.MessagesSendUploadedPhotoToPeer(callCtx, peer, file, toMTProtoSendMediaOptions(opts))
		return err
	})
}

// DownloadFile streams a Telegram file location to w using upload.getFile.
func (b *MTProtoBot) DownloadFile(ctx context.Context, loc InputFileLocation, w io.Writer, opts DownloadOptions) (int64, error) {
	if b == nil || b.client == nil {
		return 0, errors.New("mtproto bot is not logged in")
	}
	var written int64
	err := b.withFloodWait(ctx, func(callCtx context.Context) error {
		n, err := b.client.DownloadFile(callCtx, loc, w, opts)
		written = n
		return err
	})
	return written, err
}

// Compile-time guard: aliases should stay backed by mtproto package types.
var _ *mtproto.UploadedFile
