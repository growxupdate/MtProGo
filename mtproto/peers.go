package mtproto

import "encoding/binary"

const (
	constructorUser      = 0x020b1422
	constructorUserEmpty = 0xd3bc4b7a
)

// PeerRef is a small parsed peer reference from updates users/chats vectors.
// V12 currently focuses on user peers because private bot replies need user access hashes.
type PeerRef struct {
	ID         int64
	AccessHash int64
	Kind       string
}

// UserAccessHash returns the access hash for a user id when it is available.
func (d *DifferenceResult) UserAccessHash(userID int64) (int64, bool) {
	if d == nil || userID == 0 {
		return 0, false
	}
	for _, peer := range d.Users {
		if peer.ID == userID && peer.AccessHash != 0 {
			return peer.AccessHash, true
		}
	}
	return 0, false
}

func scanUserRefs(body []byte) []PeerRef {
	if len(body) < 24 {
		return nil
	}
	seen := make(map[int64]PeerRef)
	for off := 0; off+20 <= len(body); off += 4 {
		constructor := binary.LittleEndian.Uint32(body[off:])
		switch constructor {
		case constructorUser:
			// user#020b1422 flags:# flags2:# id:long access_hash:flags.0?long ...
			if off+20 > len(body) {
				continue
			}
			flags := binary.LittleEndian.Uint32(body[off+4:])
			id := int64(binary.LittleEndian.Uint64(body[off+12:]))
			if id == 0 {
				continue
			}
			var accessHash int64
			if flags&1 != 0 && off+28 <= len(body) {
				accessHash = int64(binary.LittleEndian.Uint64(body[off+20:]))
			}
			if accessHash != 0 {
				seen[id] = PeerRef{ID: id, AccessHash: accessHash, Kind: "user"}
			}
		case constructorUserEmpty:
			if off+12 > len(body) {
				continue
			}
			id := int64(binary.LittleEndian.Uint64(body[off+4:]))
			if id != 0 {
				if _, ok := seen[id]; !ok {
					seen[id] = PeerRef{ID: id, Kind: "user"}
				}
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]PeerRef, 0, len(seen))
	for _, peer := range seen {
		out = append(out, peer)
	}
	return out
}

func scanUpdatesState(body []byte) *UpdatesState {
	if len(body) < 24 {
		return nil
	}
	for off := len(body) - 24; off >= 0; off -= 4 {
		constructor := binary.LittleEndian.Uint32(body[off:])
		if constructor != constructorUpdatesState {
			continue
		}
		if off+24 > len(body) {
			continue
		}
		pts := int32(binary.LittleEndian.Uint32(body[off+4:]))
		qts := int32(binary.LittleEndian.Uint32(body[off+8:]))
		date := int32(binary.LittleEndian.Uint32(body[off+12:]))
		seq := int32(binary.LittleEndian.Uint32(body[off+16:]))
		unread := int32(binary.LittleEndian.Uint32(body[off+20:]))
		return &UpdatesState{PTS: pts, QTS: qts, Date: date, Seq: seq, UnreadCount: unread, Raw: append([]byte(nil), body[off:off+24]...)}
	}
	return nil
}
