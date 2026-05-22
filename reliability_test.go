package mtprogo

import (
	"context"
	"testing"
	"time"

	"github.com/growxupdate/MtProGo/mtproto"
)

func TestReconnectBackoff(t *testing.T) {
	initial := 100 * time.Millisecond
	max := 700 * time.Millisecond
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 0},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 700 * time.Millisecond},
	}
	for _, tc := range cases {
		got := reconnectBackoff(tc.attempt, initial, max)
		if got != tc.want {
			t.Fatalf("attempt %d: got %s want %s", tc.attempt, got, tc.want)
		}
	}
}

func TestRuntimeSessionRoundTrip(t *testing.T) {
	authKey := make([]byte, 256)
	for i := range authKey {
		authKey[i] = byte(i)
	}
	sess := &mtproto.Session{DCID: 5, Address: "91.108.56.130:443", AuthKey: authKey, AuthKeyID: 123, ServerSalt: 99, Kind: "bot"}
	state := &mtproto.UpdatesState{PTS: 10, QTS: 1, Date: 2, Seq: 3}
	peers := []mtproto.PeerRef{{ID: 42, AccessHash: 77, Kind: "user"}}
	data, err := marshalRuntimeSession("bot", sess, state, peers)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := parseRuntimeSession(data)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.MTProto.DCID != 5 || bundle.State.PTS != 10 || len(bundle.Peers) != 1 || bundle.Peers[0].AccessHash != 77 {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
}

func TestRuntimeSessionBackwardCompatibleBareSession(t *testing.T) {
	authKey := make([]byte, 256)
	sess := &mtproto.Session{DCID: 2, Address: "149.154.167.50:443", AuthKey: authKey, AuthKeyID: 777, Kind: "bot"}
	data, err := sess.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := parseRuntimeSession(data)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.MTProto.AuthKeyID != 777 || bundle.State != nil {
		t.Fatalf("unexpected parsed bundle: %+v", bundle)
	}
}

func TestMTProtoBotSnapshotState(t *testing.T) {
	bot, err := NewMTProtoBot(MTProtoBotConfig{APIID: 1, APIHash: "hash", Token: "1:test"}, WithMemorySession())
	if err != nil {
		t.Fatal(err)
	}
	bot.setState(&mtproto.UpdatesState{PTS: 123})
	state := bot.snapshotState()
	if state == nil || state.PTS != 123 {
		t.Fatalf("bad snapshot: %+v", state)
	}
	if err := bot.ClearSession(context.Background()); err != nil {
		t.Fatal(err)
	}
}
