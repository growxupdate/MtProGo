package mtproto

import "encoding/binary"

const (
	constructorUser             = 0x020b1422
	constructorUserEmpty        = 0xd3bc4b7a
	constructorChat             = 0x41cbf256
	constructorChatForbidden    = 0x6592a1a7
	constructorChannel          = 0xfe685355
	constructorChannelLegacy    = 0x7482147e
	constructorChannelForbidden = 0x17d493d5
)

// PeerRef is a small parsed peer reference from updates users/chats vectors.
type PeerRef struct {
	ID         int64
	AccessHash int64
	Kind       string
	Title      string
	Username   string
}

// InputPeer returns the raw MTProto input peer for this peer reference.
func (p PeerRef) InputPeer() (InputPeer, bool) {
	switch p.Kind {
	case "user", "private":
		if p.ID == 0 || p.AccessHash == 0 {
			return InputPeer{}, false
		}
		return InputPeerUser(p.ID, p.AccessHash), true
	case "chat", "group":
		if p.ID == 0 {
			return InputPeer{}, false
		}
		return InputPeerChat(p.ID), true
	case "channel", "supergroup":
		if p.ID == 0 || p.AccessHash == 0 {
			return InputPeer{}, false
		}
		return InputPeerChannel(p.ID, p.AccessHash), true
	default:
		return InputPeer{}, false
	}
}

// UserAccessHash returns the access hash for a user id when it is available.
func (d *DifferenceResult) UserAccessHash(userID int64) (int64, bool) {
	if d == nil || userID == 0 {
		return 0, false
	}
	for _, peer := range d.Peers {
		if peer.ID == userID && peer.AccessHash != 0 && peer.Kind == "user" {
			return peer.AccessHash, true
		}
	}
	return 0, false
}

func scanPeerRefs(body []byte) []PeerRef {
	if len(body) < 12 {
		return nil
	}
	seen := make(map[string]PeerRef)
	put := func(peer PeerRef) {
		if peer.ID == 0 || peer.Kind == "" {
			return
		}
		key := peer.Kind + ":" + int64Key(peer.ID)
		old, ok := seen[key]
		if ok {
			if old.AccessHash == 0 && peer.AccessHash != 0 {
				old.AccessHash = peer.AccessHash
			}
			if old.Title == "" && peer.Title != "" {
				old.Title = peer.Title
			}
			if old.Username == "" && peer.Username != "" {
				old.Username = peer.Username
			}
			seen[key] = old
			return
		}
		seen[key] = peer
	}

	for off := 0; off+12 <= len(body); off += 4 {
		constructor := binary.LittleEndian.Uint32(body[off:])
		switch constructor {
		case constructorUser:
			// user#020b1422 flags:# flags2:# id:long access_hash:flags.0?long ...
			if off+20 > len(body) {
				continue
			}
			flags := binary.LittleEndian.Uint32(body[off+4:])
			id := int64(binary.LittleEndian.Uint64(body[off+12:]))
			var accessHash int64
			if flags&1 != 0 && off+28 <= len(body) {
				accessHash = int64(binary.LittleEndian.Uint64(body[off+20:]))
			}
			put(PeerRef{ID: id, AccessHash: accessHash, Kind: "user"})
		case constructorUserEmpty:
			if off+12 > len(body) {
				continue
			}
			id := int64(binary.LittleEndian.Uint64(body[off+4:]))
			put(PeerRef{ID: id, Kind: "user"})
		case constructorChat:
			// chat#41cbf256 flags:# id:long title:string ...
			if off+16 > len(body) {
				continue
			}
			id := int64(binary.LittleEndian.Uint64(body[off+8:]))
			put(PeerRef{ID: id, Kind: "chat"})
		case constructorChatForbidden:
			// chatForbidden#6592a1a7 id:long title:string
			if off+12 > len(body) {
				continue
			}
			id := int64(binary.LittleEndian.Uint64(body[off+4:]))
			put(PeerRef{ID: id, Kind: "chat"})
		case constructorChannel, constructorChannelLegacy:
			// channel#fe685355 flags:# flags2:# id:long access_hash:flags.13?long ...
			// flags.8 marks megagroup/supergroup. Plain broadcast channels do not
			// normally send bot command messages unless the bot is an admin.
			if off+20 > len(body) {
				continue
			}
			flags := binary.LittleEndian.Uint32(body[off+4:])
			id := int64(binary.LittleEndian.Uint64(body[off+12:]))
			var accessHash int64
			if flags&(1<<13) != 0 && off+28 <= len(body) {
				accessHash = int64(binary.LittleEndian.Uint64(body[off+20:]))
			}
			kind := "channel"
			if flags&(1<<8) != 0 {
				kind = "supergroup"
			}
			put(PeerRef{ID: id, AccessHash: accessHash, Kind: kind})
		case constructorChannelForbidden:
			// channelForbidden#17d493d5 flags:# id:long access_hash:long title:string ...
			if off+24 > len(body) {
				continue
			}
			flags := binary.LittleEndian.Uint32(body[off+4:])
			id := int64(binary.LittleEndian.Uint64(body[off+8:]))
			accessHash := int64(binary.LittleEndian.Uint64(body[off+16:]))
			kind := "channel"
			if flags&(1<<8) != 0 {
				kind = "supergroup"
			}
			put(PeerRef{ID: id, AccessHash: accessHash, Kind: kind})
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

func int64Key(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := v < 0
	var u uint64
	if neg {
		u = uint64(-v)
	} else {
		u = uint64(v)
	}
	for u > 0 {
		i--
		buf[i] = byte('0' + u%10)
		u /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
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

func scanUserRefs(body []byte) []PeerRef {
	peers := scanPeerRefs(body)
	if len(peers) == 0 {
		return nil
	}
	out := make([]PeerRef, 0, len(peers))
	for _, peer := range peers {
		if peer.Kind == "user" {
			out = append(out, peer)
		}
	}
	return out
}
