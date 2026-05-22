package mtproto

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
)

const (
	constructorBoolFalse                  = 0xbc799737
	constructorBoolTrue                   = 0x997275b5
	constructorUploadSaveFilePart         = 0xb304a621
	constructorUploadSaveBigFilePart      = 0xde7b673d
	constructorUploadGetFile              = 0xbe5335be
	constructorUploadFile                 = 0x096a18d5
	constructorUploadFileCdnRedirect      = 0xf18cda44
	constructorInputFile                  = 0xf52ff27f
	constructorInputFileBig               = 0xfa4f0bb5
	constructorInputMediaUploadedPhoto    = 0x1e287d04
	constructorInputMediaUploadedDocument = 0x037c9330
	constructorDocumentAttributeFilename  = 0x15590068
	constructorMessagesSendMedia          = 0xac55d9c1
	constructorInputDocumentFileLocation  = 0xbad07584
	constructorInputPhotoFileLocation     = 0x40181ffe
)

const (
	// DefaultUploadPartSize is Telegram's practical maximum upload chunk size.
	DefaultUploadPartSize = 512 * 1024
	minUploadPartSize     = 32 * 1024
	maxUploadPartSize     = 512 * 1024
	smallFileLimit        = 10 * 1024 * 1024
)

// ProgressFunc receives completed bytes and total bytes. Total can be -1 for unknown.
type ProgressFunc func(done, total int64)

// UploadOptions controls MTProto file upload memory and progress behavior.
type UploadOptions struct {
	// PartSize controls streaming memory usage. It is clamped between 32 KiB and 512 KiB.
	// Telegram requires non-final parts to use a stable part size.
	PartSize int
	// Progress is called after each successfully uploaded chunk.
	Progress ProgressFunc
}

// UploadedFile is an uploaded InputFile/InputFileBig descriptor usable by sendMedia.
type UploadedFile struct {
	ID          int64
	Parts       int
	Name        string
	Size        int64
	MD5Checksum string
	Big         bool
}

// InputFile is the raw Telegram InputFile/InputFileBig object produced by UploadFile.
type InputFile struct {
	UploadedFile
}

// DownloadOptions controls upload.getFile streaming behavior.
type DownloadOptions struct {
	PartSize     int
	Precise      bool
	CDNSupported bool
	// Limit optionally stops after this many bytes. Zero means until Telegram returns a short part.
	Limit    int64
	Progress ProgressFunc
}

// DownloadedPart is one response from upload.getFile.
type DownloadedPart struct {
	FileTypeConstructor uint32
	MTime               int32
	Bytes               []byte
	CDNRedirect         bool
}

// InputFileLocation encodes a Telegram InputFileLocation for upload.getFile.
type InputFileLocation interface {
	encodeInputFileLocation(*tlBuffer) error
}

// DocumentFileLocation identifies a document/video/audio/voice file for upload.getFile.
type DocumentFileLocation struct {
	ID            int64
	AccessHash    int64
	FileReference []byte
	ThumbSize     string
}

// PhotoFileLocation identifies a photo size for upload.getFile.
type PhotoFileLocation struct {
	ID            int64
	AccessHash    int64
	FileReference []byte
	ThumbSize     string
}

// SendMediaOptions controls messages.sendMedia flags and captions.
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

func (l DocumentFileLocation) encodeInputFileLocation(b *tlBuffer) error {
	if l.ID == 0 || l.AccessHash == 0 {
		return errors.New("document location requires id and access hash")
	}
	b.putInt(constructorInputDocumentFileLocation)
	b.putLong(uint64(l.ID))
	b.putLong(uint64(l.AccessHash))
	b.putBytes(l.FileReference)
	b.putString(l.ThumbSize)
	return nil
}

func (l PhotoFileLocation) encodeInputFileLocation(b *tlBuffer) error {
	if l.ID == 0 || l.AccessHash == 0 {
		return errors.New("photo location requires id and access hash")
	}
	b.putInt(constructorInputPhotoFileLocation)
	b.putLong(uint64(l.ID))
	b.putLong(uint64(l.AccessHash))
	b.putBytes(l.FileReference)
	b.putString(l.ThumbSize)
	return nil
}

func (f InputFile) encodeInputFile(b *tlBuffer) error {
	if f.ID == 0 || f.Parts <= 0 {
		return errors.New("input file requires id and parts")
	}
	if f.Big {
		b.putInt(constructorInputFileBig)
		b.putLong(uint64(f.ID))
		b.putInt(uint32(f.Parts))
		b.putString(f.Name)
		return nil
	}
	b.putInt(constructorInputFile)
	b.putLong(uint64(f.ID))
	b.putInt(uint32(f.Parts))
	b.putString(f.Name)
	b.putString(f.MD5Checksum)
	return nil
}

// UploadFile streams r to Telegram using upload.saveFilePart/upload.saveBigFilePart.
// It keeps at most one chunk in RAM, so larger files do not force full buffering.
func (c *EncryptedClient) UploadFile(ctx context.Context, r io.Reader, size int64, name string, opts UploadOptions) (*UploadedFile, error) {
	if c == nil {
		return nil, errors.New("mtproto: nil encrypted client")
	}
	if r == nil {
		return nil, errors.New("upload reader is nil")
	}
	if size < 0 {
		return nil, errors.New("file size is required for MTProto upload")
	}
	if strings.TrimSpace(name) == "" {
		name = "file"
	}
	partSize := normalizePartSize(opts.PartSize)
	parts := int(math.Ceil(float64(size) / float64(partSize)))
	if parts == 0 {
		parts = 1
	}
	fileID := randomNonZeroInt64()
	uploaded := &UploadedFile{ID: fileID, Parts: parts, Name: filepath.Base(name), Size: size, Big: size > smallFileLimit}

	buf := make([]byte, partSize)
	var done int64
	h := md5.New()
	for part := 0; part < parts; part++ {
		want := partSize
		remaining := size - done
		if remaining < int64(want) {
			want = int(remaining)
		}
		if want <= 0 && size == 0 {
			want = 0
		}
		chunk := buf[:want]
		if want > 0 {
			if _, err := io.ReadFull(r, chunk); err != nil {
				return nil, fmt.Errorf("read upload chunk %d: %w", part, err)
			}
			if !uploaded.Big {
				_, _ = h.Write(chunk)
			}
		}
		var q []byte
		if uploaded.Big {
			q = makeUploadSaveBigFilePartQuery(fileID, int32(part), int32(parts), chunk)
		} else {
			q = makeUploadSaveFilePartQuery(fileID, int32(part), chunk)
		}
		res, err := c.Invoke(ctx, q)
		if err != nil {
			return nil, err
		}
		ok, err := parseTLBool(res.Body)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("telegram rejected upload part %d", part)
		}
		done += int64(want)
		if opts.Progress != nil {
			opts.Progress(done, size)
		}
	}
	if !uploaded.Big {
		uploaded.MD5Checksum = hex.EncodeToString(h.Sum(nil))
	}
	return uploaded, nil
}

// UploadBytes uploads a byte slice. For large files prefer UploadFile with a streaming reader.
func (c *EncryptedClient) UploadBytes(ctx context.Context, data []byte, name string, opts UploadOptions) (*UploadedFile, error) {
	return c.UploadFile(ctx, bytes.NewReader(data), int64(len(data)), name, opts)
}

// UploadGetFilePart calls upload.getFile for one file part.
func (c *EncryptedClient) UploadGetFilePart(ctx context.Context, loc InputFileLocation, offset int64, limit int32, opts DownloadOptions) (*DownloadedPart, error) {
	if c == nil {
		return nil, errors.New("mtproto: nil encrypted client")
	}
	if loc == nil {
		return nil, errors.New("file location is nil")
	}
	if limit <= 0 {
		limit = int32(normalizePartSize(opts.PartSize))
	}
	q, err := makeUploadGetFileQuery(loc, offset, limit, opts)
	if err != nil {
		return nil, err
	}
	res, err := c.Invoke(ctx, q)
	if err != nil {
		return nil, err
	}
	return parseUploadFile(res.Body)
}

// DownloadFile streams upload.getFile results into w until Telegram returns a short part or opts.Limit is reached.
func (c *EncryptedClient) DownloadFile(ctx context.Context, loc InputFileLocation, w io.Writer, opts DownloadOptions) (int64, error) {
	if w == nil {
		return 0, errors.New("download writer is nil")
	}
	partSize := normalizePartSize(opts.PartSize)
	var offset int64
	for {
		limit := int32(partSize)
		if opts.Limit > 0 && opts.Limit-offset < int64(limit) {
			limit = int32(opts.Limit - offset)
		}
		if limit <= 0 {
			return offset, nil
		}
		part, err := c.UploadGetFilePart(ctx, loc, offset, limit, opts)
		if err != nil {
			return offset, err
		}
		if part.CDNRedirect {
			return offset, errors.New("mtproto: CDN redirect downloads are not implemented yet")
		}
		if len(part.Bytes) == 0 {
			return offset, nil
		}
		n, err := w.Write(part.Bytes)
		offset += int64(n)
		if opts.Progress != nil {
			opts.Progress(offset, opts.Limit)
		}
		if err != nil {
			return offset, err
		}
		if n != len(part.Bytes) {
			return offset, io.ErrShortWrite
		}
		if len(part.Bytes) < int(limit) {
			return offset, nil
		}
	}
}

// MessagesSendUploadedDocument sends an already uploaded file as a document.
func (c *EncryptedClient) MessagesSendUploadedDocument(ctx context.Context, peer InputPeer, file *UploadedFile, opts SendMediaOptions) (*InvokeResult, error) {
	if file == nil {
		return nil, errors.New("uploaded file is nil")
	}
	media := inputMediaUploadedDocument{file: InputFile{UploadedFile: *file}, opts: opts}
	query, err := makeMessagesSendMediaQuery(peer, media, opts)
	if err != nil {
		return nil, err
	}
	res, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	res.Message = "messages.sendMedia uploaded document succeeded over encrypted MTProto"
	return res, nil
}

// MessagesSendUploadedPhoto sends an already uploaded file as a photo.
func (c *EncryptedClient) MessagesSendUploadedPhoto(ctx context.Context, peer InputPeer, file *UploadedFile, opts SendMediaOptions) (*InvokeResult, error) {
	if file == nil {
		return nil, errors.New("uploaded file is nil")
	}
	media := inputMediaUploadedPhoto{file: InputFile{UploadedFile: *file}, opts: opts}
	query, err := makeMessagesSendMediaQuery(peer, media, opts)
	if err != nil {
		return nil, err
	}
	res, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	res.Message = "messages.sendMedia uploaded photo succeeded over encrypted MTProto"
	return res, nil
}

func (c *EncryptedClient) MessagesSendUploadedDocumentToPeer(ctx context.Context, peer PeerRef, file *UploadedFile, opts SendMediaOptions) (*InvokeResult, error) {
	input, ok := peer.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: peer %d (%s) is missing access data", peer.ID, peer.Kind)
	}
	return c.MessagesSendUploadedDocument(ctx, input, file, opts)
}

func (c *EncryptedClient) MessagesSendUploadedPhotoToPeer(ctx context.Context, peer PeerRef, file *UploadedFile, opts SendMediaOptions) (*InvokeResult, error) {
	input, ok := peer.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: peer %d (%s) is missing access data", peer.ID, peer.Kind)
	}
	return c.MessagesSendUploadedPhoto(ctx, input, file, opts)
}

type inputMedia interface{ encodeInputMedia(*tlBuffer) error }

type inputMediaUploadedPhoto struct {
	file InputFile
	opts SendMediaOptions
}
type inputMediaUploadedDocument struct {
	file InputFile
	opts SendMediaOptions
}

func (m inputMediaUploadedPhoto) encodeInputMedia(b *tlBuffer) error {
	b.putInt(constructorInputMediaUploadedPhoto)
	var flags uint32
	if m.opts.Spoiler {
		flags |= 1 << 2
	}
	b.putInt(flags)
	return m.file.encodeInputFile(b)
}

func (m inputMediaUploadedDocument) encodeInputMedia(b *tlBuffer) error {
	b.putInt(constructorInputMediaUploadedDocument)
	var flags uint32
	if m.opts.ForceFile {
		flags |= 1 << 4
	}
	if m.opts.Spoiler {
		flags |= 1 << 5
	}
	b.putInt(flags)
	if err := m.file.encodeInputFile(b); err != nil {
		return err
	}
	mimeType := strings.TrimSpace(m.opts.MIMEType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	b.putString(mimeType)
	// attributes: vector<documentAttributeFilename>
	name := strings.TrimSpace(m.opts.FileName)
	if name == "" {
		name = m.file.Name
	}
	if name == "" {
		name = "file"
	}
	b.putInt(constructorVector)
	b.putInt(1)
	b.putInt(constructorDocumentAttributeFilename)
	b.putString(filepath.Base(name))
	return nil
}

func makeUploadSaveFilePartQuery(fileID int64, part int32, data []byte) []byte {
	var b tlBuffer
	b.putInt(constructorUploadSaveFilePart)
	b.putLong(uint64(fileID))
	b.putInt(uint32(part))
	b.putBytes(data)
	return b.bytes()
}

func makeUploadSaveBigFilePartQuery(fileID int64, part, total int32, data []byte) []byte {
	var b tlBuffer
	b.putInt(constructorUploadSaveBigFilePart)
	b.putLong(uint64(fileID))
	b.putInt(uint32(part))
	b.putInt(uint32(total))
	b.putBytes(data)
	return b.bytes()
}

func makeUploadGetFileQuery(loc InputFileLocation, offset int64, limit int32, opts DownloadOptions) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorUploadGetFile)
	var flags uint32
	if opts.Precise {
		flags |= 1 << 0
	}
	if opts.CDNSupported {
		flags |= 1 << 1
	}
	b.putInt(flags)
	if err := loc.encodeInputFileLocation(&b); err != nil {
		return nil, err
	}
	b.putLong(uint64(offset))
	b.putInt(uint32(limit))
	return b.bytes(), nil
}

func makeMessagesSendMediaQuery(peer InputPeer, media inputMedia, opts SendMediaOptions) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorMessagesSendMedia)
	var flags uint32
	if opts.ReplyToMessageID > 0 {
		flags |= 1 << 0
	}
	if opts.Silent {
		flags |= 1 << 5
	}
	if opts.Background {
		flags |= 1 << 6
	}
	b.putInt(flags)
	if err := peer.encode(&b); err != nil {
		return nil, err
	}
	if opts.ReplyToMessageID > 0 {
		b.putInt(constructorInputReplyToMessage)
		b.putInt(0)
		b.putInt(uint32(int32(opts.ReplyToMessageID)))
	}
	if err := media.encodeInputMedia(&b); err != nil {
		return nil, err
	}
	b.putString(opts.Caption)
	b.putLong(uint64(randomInt64()))
	return b.bytes(), nil
}

func parseTLBool(body []byte) (bool, error) {
	if len(body) < 4 {
		return false, errors.New("tl bool response too short")
	}
	switch binary.LittleEndian.Uint32(body[:4]) {
	case constructorBoolTrue:
		return true, nil
	case constructorBoolFalse:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected bool constructor 0x%08x", binary.LittleEndian.Uint32(body[:4]))
	}
}

func parseUploadFile(body []byte) (*DownloadedPart, error) {
	if len(body) < 4 {
		return nil, errors.New("upload.file response too short")
	}
	constructor := binary.LittleEndian.Uint32(body[:4])
	switch constructor {
	case constructorUploadFile:
		r := newTLReader(body[4:])
		typ, err := r.int()
		if err != nil {
			return nil, err
		}
		mtime, err := r.int()
		if err != nil {
			return nil, err
		}
		data, err := r.bytes()
		if err != nil {
			return nil, err
		}
		return &DownloadedPart{FileTypeConstructor: typ, MTime: int32(mtime), Bytes: data}, nil
	case constructorUploadFileCdnRedirect:
		return &DownloadedPart{CDNRedirect: true}, nil
	default:
		return nil, fmt.Errorf("unexpected upload.File constructor 0x%08x", constructor)
	}
}

func normalizePartSize(v int) int {
	if v <= 0 {
		v = DefaultUploadPartSize
	}
	if v < minUploadPartSize {
		v = minUploadPartSize
	}
	if v > maxUploadPartSize {
		v = maxUploadPartSize
	}
	// Keep part sizes friendly to Telegram and memory pools.
	const step = 1024
	v = (v / step) * step
	if v <= 0 {
		v = minUploadPartSize
	}
	return v
}

func randomNonZeroInt64() int64 {
	var buf [8]byte
	for {
		if _, err := rand.Read(buf[:]); err == nil {
			v := int64(binary.LittleEndian.Uint64(buf[:]))
			if v != 0 {
				return v
			}
		}
		v := randomInt64()
		if v != 0 {
			return v
		}
	}
}
