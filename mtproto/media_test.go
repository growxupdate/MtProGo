package mtproto

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNormalizePartSize(t *testing.T) {
	if got := normalizePartSize(0); got != DefaultUploadPartSize {
		t.Fatalf("default=%d", got)
	}
	if got := normalizePartSize(1); got != minUploadPartSize {
		t.Fatalf("min=%d", got)
	}
	if got := normalizePartSize(9999999); got != maxUploadPartSize {
		t.Fatalf("max=%d", got)
	}
}

func TestMakeUploadSaveFilePartQuery(t *testing.T) {
	q := makeUploadSaveFilePartQuery(7, 3, []byte("abc"))
	if binary.LittleEndian.Uint32(q[:4]) != constructorUploadSaveFilePart {
		t.Fatalf("bad constructor")
	}
	if binary.LittleEndian.Uint64(q[4:12]) != 7 {
		t.Fatalf("bad id")
	}
	if binary.LittleEndian.Uint32(q[12:16]) != 3 {
		t.Fatalf("bad part")
	}
}

func TestParseTLBool(t *testing.T) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], constructorBoolTrue)
	ok, err := parseTLBool(b[:])
	if err != nil || !ok {
		t.Fatalf("true parse: %v %v", ok, err)
	}
	binary.LittleEndian.PutUint32(b[:], constructorBoolFalse)
	ok, err = parseTLBool(b[:])
	if err != nil || ok {
		t.Fatalf("false parse: %v %v", ok, err)
	}
}

func TestParseUploadFile(t *testing.T) {
	var b tlBuffer
	b.putInt(constructorUploadFile)
	b.putInt(0xaa)
	b.putInt(123)
	b.putBytes([]byte("hello"))
	part, err := parseUploadFile(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if part.FileTypeConstructor != 0xaa || part.MTime != 123 || !bytes.Equal(part.Bytes, []byte("hello")) {
		t.Fatalf("unexpected part: %+v", part)
	}
}

func TestInputMediaUploadedDocumentQuery(t *testing.T) {
	file := &UploadedFile{ID: 9, Parts: 1, Name: "a.txt", MD5Checksum: "d41d8cd98f00b204e9800998ecf8427e"}
	q, err := makeMessagesSendMediaQuery(InputPeerSelf(), inputMediaUploadedDocument{file: InputFile{UploadedFile: *file}, opts: SendMediaOptions{Caption: "cap", ForceFile: true}}, SendMediaOptions{Caption: "cap"})
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(q[:4]) != constructorMessagesSendMedia {
		t.Fatalf("bad constructor")
	}
}
