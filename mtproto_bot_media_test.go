package mtprogo

import (
	"testing"

	"github.com/growxupdate/MtProGo/mtproto"
)

func TestSendMediaOptionsMapping(t *testing.T) {
	got := toMTProtoSendMediaOptions(SendMediaOptions{Caption: "c", ReplyToMessageID: 4, Silent: true, Background: true, Spoiler: true, ForceFile: true, MIMEType: "text/plain", FileName: "a.txt"})
	if got.Caption != "c" || got.ReplyToMessageID != 4 || !got.Silent || !got.Background || !got.Spoiler || !got.ForceFile || got.MIMEType != "text/plain" || got.FileName != "a.txt" {
		t.Fatalf("bad mapping: %+v", got)
	}
}

func TestMediaAliases(t *testing.T) {
	f := &UploadedFile{ID: 1, Parts: 1, Name: "a"}
	var mf *mtproto.UploadedFile = f
	if mf.ID != 1 {
		t.Fatal("alias mismatch")
	}
}
