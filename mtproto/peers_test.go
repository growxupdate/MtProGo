package mtproto

import (
	"encoding/binary"
	"testing"
)

func TestScanUserRefs(t *testing.T) {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorUser)
	b = binary.LittleEndian.AppendUint32(b, 1) // flags.0 access_hash
	b = binary.LittleEndian.AppendUint32(b, 0) // flags2
	b = binary.LittleEndian.AppendUint64(b, 12345)
	b = binary.LittleEndian.AppendUint64(b, 67890)

	users := scanUserRefs(b)
	if len(users) != 1 {
		t.Fatalf("users=%d", len(users))
	}
	if users[0].ID != 12345 || users[0].AccessHash != 67890 || users[0].Kind != "user" {
		t.Fatalf("unexpected user: %+v", users[0])
	}
}

func TestScanUpdatesState(t *testing.T) {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorUpdatesState)
	b = binary.LittleEndian.AppendUint32(b, 10)
	b = binary.LittleEndian.AppendUint32(b, 20)
	b = binary.LittleEndian.AppendUint32(b, 30)
	b = binary.LittleEndian.AppendUint32(b, 40)
	b = binary.LittleEndian.AppendUint32(b, 50)

	state := scanUpdatesState(b)
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.PTS != 10 || state.QTS != 20 || state.Date != 30 || state.Seq != 40 || state.UnreadCount != 50 {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestScanPeerRefsChatAndChannel(t *testing.T) {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorChat)
	b = binary.LittleEndian.AppendUint32(b, 0) // flags
	b = binary.LittleEndian.AppendUint64(b, 222)

	b = binary.LittleEndian.AppendUint32(b, constructorChannel)
	b = binary.LittleEndian.AppendUint32(b, 1<<13) // flags.13 access_hash
	b = binary.LittleEndian.AppendUint32(b, 0)     // flags2
	b = binary.LittleEndian.AppendUint64(b, 333)
	b = binary.LittleEndian.AppendUint64(b, 444)

	peers := scanPeerRefs(b)
	if len(peers) != 2 {
		t.Fatalf("peers=%d %+v", len(peers), peers)
	}
	seen := map[string]PeerRef{}
	for _, peer := range peers {
		seen[peer.Kind] = peer
	}
	if seen["chat"].ID != 222 {
		t.Fatalf("chat not found: %+v", seen)
	}
	if seen["channel"].ID != 333 || seen["channel"].AccessHash != 444 {
		t.Fatalf("channel not found: %+v", seen)
	}
}

func TestScanPeerRefsLegacyChannelConstructor(t *testing.T) {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorChannelLegacy)
	b = binary.LittleEndian.AppendUint32(b, (1<<13)|(1<<8)) // access_hash + megagroup
	b = binary.LittleEndian.AppendUint32(b, 0)              // flags2
	b = binary.LittleEndian.AppendUint64(b, 333)
	b = binary.LittleEndian.AppendUint64(b, 444)

	peers := scanPeerRefs(b)
	if len(peers) != 1 {
		t.Fatalf("peers=%d %+v", len(peers), peers)
	}
	if peers[0].ID != 333 || peers[0].AccessHash != 444 || peers[0].Kind != "supergroup" {
		t.Fatalf("unexpected legacy channel peer: %+v", peers[0])
	}
}
