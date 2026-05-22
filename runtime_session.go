package mtprogo

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/growxupdate/MtProGo/mtproto"
)

const runtimeSessionVersion = 1

// runtimeSessionBundle keeps MTProto auth metadata together with lightweight
// runtime state. It is intentionally private so the stable public portable
// string-session format remains owned by mtproto.Session.
type runtimeSessionBundle struct {
	Version   int                   `json:"version"`
	Kind      string                `json:"kind"`
	MTProto   *mtproto.Session      `json:"mtproto"`
	State     *mtproto.UpdatesState `json:"state,omitempty"`
	Peers     []mtproto.PeerRef     `json:"peers,omitempty"`
	CreatedAt int64                 `json:"created_at"`
	UpdatedAt int64                 `json:"updated_at"`
}

func marshalRuntimeSession(kind string, sess *mtproto.Session, state *mtproto.UpdatesState, peers []mtproto.PeerRef) ([]byte, error) {
	if sess == nil {
		return nil, errors.New("mtprogo: nil mtproto session")
	}
	now := time.Now().Unix()
	copySession := *sess
	copySession.AuthKey = append([]byte(nil), sess.AuthKey...)
	bundle := runtimeSessionBundle{
		Version:   runtimeSessionVersion,
		Kind:      kind,
		MTProto:   &copySession,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if state != nil {
		copyState := *state
		copyState.Raw = append([]byte(nil), state.Raw...)
		bundle.State = &copyState
	}
	if len(peers) != 0 {
		bundle.Peers = append([]mtproto.PeerRef(nil), peers...)
	}
	return json.Marshal(bundle)
}

func parseRuntimeSession(data []byte) (*runtimeSessionBundle, error) {
	var bundle runtimeSessionBundle
	if err := json.Unmarshal(data, &bundle); err == nil && bundle.MTProto != nil {
		if bundle.Version == 0 {
			bundle.Version = runtimeSessionVersion
		}
		if err := bundle.MTProto.Valid(); err != nil {
			return nil, err
		}
		bundle.MTProto.AuthKey = append([]byte(nil), bundle.MTProto.AuthKey...)
		if bundle.State != nil {
			bundle.State.Raw = append([]byte(nil), bundle.State.Raw...)
		}
		bundle.Peers = append([]mtproto.PeerRef(nil), bundle.Peers...)
		return &bundle, nil
	}
	// Backward compatibility: V13/V15 stored the bare mtproto.Session JSON.
	sess, err := mtproto.ParseSession(data)
	if err != nil {
		return nil, err
	}
	return &runtimeSessionBundle{Version: runtimeSessionVersion, Kind: sess.Kind, MTProto: sess}, nil
}
