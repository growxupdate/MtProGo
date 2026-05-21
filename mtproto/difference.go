package mtproto

import (
	"context"
	"encoding/binary"
	"fmt"
)

const (
	constructorUpdatesGetDifference    = 0x19c2f763
	constructorDifferenceEmpty         = 0x5d75a138
	constructorDifference              = 0x00f49ca0
	constructorDifferenceSlice         = 0xa8fb1981
	constructorDifferenceTooLong       = 0x4afe8f6d
	constructorMessageEmpty            = 0x90a6ca84
	constructorMessage                 = 0x9815cec8
	constructorPeerUser                = 0x59511722
	constructorPeerChat                = 0x36c6019a
	constructorPeerChannel             = 0xa2a5371e
	constructorMessageEntityBotCommand = 0x6cef8ac7
)

// DifferenceResult is a minimal parsed result of updates.getDifference.
type DifferenceResult struct {
	Constructor uint32
	State       *UpdatesState
	Messages    []TextMessage
	Users       []PeerRef
	Raw         []byte
}

// TextMessage is a lightweight text message parsed from updates.Difference.
type TextMessage struct {
	ID     int32
	ChatID int64
	FromID int64
	Date   int32
	Text   string
	Raw    []byte
}

// UpdatesGetDifference calls updates.getDifference using the supplied state.
// It is a low-level polling primitive used by future high-level update loops.
func (c *EncryptedClient) UpdatesGetDifference(ctx context.Context, state UpdatesState, ptsLimit int32) (*DifferenceResult, *InvokeResult, error) {
	query := makeUpdatesGetDifferenceQuery(state, ptsLimit)
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	diff, err := parseDifferenceResult(result.Body)
	if err != nil {
		return nil, result, err
	}
	result.Message = "updates.getDifference succeeded over encrypted MTProto"
	return diff, result, nil
}

func makeUpdatesGetDifferenceQuery(state UpdatesState, ptsLimit int32) []byte {
	var q tlBuffer
	q.putInt(constructorUpdatesGetDifference)
	flags := uint32(0)
	if ptsLimit > 0 {
		flags |= 1 << 1
	}
	q.putInt(flags)
	q.putInt(uint32(state.PTS))
	if ptsLimit > 0 {
		q.putInt(uint32(ptsLimit))
	}
	q.putInt(uint32(state.Date))
	q.putInt(uint32(state.QTS))
	return q.bytes()
}

func parseDifferenceResult(body []byte) (*DifferenceResult, error) {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, err
	}
	out := &DifferenceResult{Constructor: constructor, Raw: append([]byte(nil), body...)}
	out.Users = scanUserRefs(body)
	if scannedState := scanUpdatesState(body); scannedState != nil {
		out.State = scannedState
	}
	switch constructor {
	case constructorDifferenceEmpty:
		date, err := r.int()
		if err != nil {
			return nil, err
		}
		seq, err := r.int()
		if err != nil {
			return nil, err
		}
		out.State = &UpdatesState{Date: int32(date), Seq: int32(seq)}
		return out, nil
	case constructorDifferenceTooLong:
		pts, err := r.int()
		if err != nil {
			return nil, err
		}
		out.State = &UpdatesState{PTS: int32(pts)}
		return out, nil
	case constructorDifference, constructorDifferenceSlice:
		messages, err := parseMessageVector(r)
		if err != nil {
			// Keep the raw body available even if a future Message variant is not parsed yet.
			return out, nil
		}
		out.Messages = messages
		// V12 scans the raw difference body for users/access_hash and final updates.state
		// so the high-level MTProto bot loop can reply to private messages while the
		// full generated TL parser is still being built.
		return out, nil
	default:
		return nil, fmt.Errorf("mtproto: expected updates.Difference, got 0x%08x", constructor)
	}
}

func parseMessageVector(r *tlReader) ([]TextMessage, error) {
	vector, err := r.int()
	if err != nil {
		return nil, err
	}
	if vector != constructorVector {
		return nil, fmt.Errorf("mtproto: expected vector, got 0x%08x", vector)
	}
	count, err := r.int()
	if err != nil {
		return nil, err
	}
	out := make([]TextMessage, 0, count)
	for i := 0; i < int(count); i++ {
		start := r.off
		msg, ok, err := parseTextMessage(r)
		if err != nil {
			return out, err
		}
		if ok {
			msg.Raw = append([]byte(nil), r.buf[start:r.off]...)
			out = append(out, msg)
		}
	}
	return out, nil
}

func parseTextMessage(r *tlReader) (TextMessage, bool, error) {
	constructor, err := r.int()
	if err != nil {
		return TextMessage{}, false, err
	}
	switch constructor {
	case constructorMessageEmpty:
		flags, err := r.int()
		if err != nil {
			return TextMessage{}, false, err
		}
		if _, err := r.int(); err != nil {
			return TextMessage{}, false, err
		}
		if flags&1 != 0 {
			if _, err := parsePeer(r); err != nil {
				return TextMessage{}, false, err
			}
		}
		return TextMessage{}, false, nil
	case constructorMessage:
		flags, err := r.int()
		if err != nil {
			return TextMessage{}, false, err
		}
		flags2, err := r.int()
		if err != nil {
			return TextMessage{}, false, err
		}
		id, err := r.int()
		if err != nil {
			return TextMessage{}, false, err
		}
		var fromID int64
		if flags&(1<<8) != 0 {
			peer, err := parsePeer(r)
			if err != nil {
				return TextMessage{}, false, err
			}
			fromID = peer
		}
		if flags&(1<<29) != 0 {
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
		}
		chatID, err := parsePeer(r)
		if err != nil {
			return TextMessage{}, false, err
		}
		if flags&(1<<28) != 0 {
			if _, err := parsePeer(r); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<2) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: message fwd_from parsing not implemented")
		}
		if flags&(1<<11) != 0 {
			if _, err := r.long(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags2&1 != 0 {
			if _, err := r.long(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<3) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: reply_to parsing not implemented")
		}
		date, err := r.int()
		if err != nil {
			return TextMessage{}, false, err
		}
		text, err := r.string()
		if err != nil {
			return TextMessage{}, false, err
		}
		if flags&(1<<9) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: media parsing not implemented")
		}
		if flags&(1<<6) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: reply_markup parsing not implemented")
		}
		if flags&(1<<7) != 0 {
			if err := skipMessageEntities(r); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<10) != 0 {
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<23) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: replies parsing not implemented")
		}
		if flags&(1<<15) != 0 {
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<16) != 0 {
			if _, err := r.string(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<17) != 0 {
			if _, err := r.long(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<20) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: reactions parsing not implemented")
		}
		if flags&(1<<22) != 0 {
			if err := skipRawVector(r); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<25) != 0 {
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags&(1<<30) != 0 {
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags2&(1<<2) != 0 {
			if _, err := r.long(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags2&(1<<3) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: factcheck parsing not implemented")
		}
		if flags2&(1<<5) != 0 {
			if _, err := r.int(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags2&(1<<6) != 0 {
			if _, err := r.long(); err != nil {
				return TextMessage{}, false, err
			}
		}
		if flags2&(1<<7) != 0 {
			return TextMessage{}, false, fmt.Errorf("mtproto: suggested_post parsing not implemented")
		}
		return TextMessage{ID: int32(id), ChatID: chatID, FromID: fromID, Date: int32(date), Text: text}, true, nil
	default:
		return TextMessage{}, false, fmt.Errorf("mtproto: unsupported Message constructor 0x%08x", constructor)
	}
}

func parsePeer(r *tlReader) (int64, error) {
	constructor, err := r.int()
	if err != nil {
		return 0, err
	}
	id, err := r.long()
	if err != nil {
		return 0, err
	}
	v := int64(id)
	switch constructor {
	case constructorPeerUser, constructorPeerChat, constructorPeerChannel:
		return v, nil
	default:
		return 0, fmt.Errorf("mtproto: unsupported Peer constructor 0x%08x", constructor)
	}
}

func skipMessageEntities(r *tlReader) error {
	vector, err := r.int()
	if err != nil {
		return err
	}
	if vector != constructorVector {
		return fmt.Errorf("mtproto: expected entities vector, got 0x%08x", vector)
	}
	count, err := r.int()
	if err != nil {
		return err
	}
	for i := 0; i < int(count); i++ {
		constructor, err := r.int()
		if err != nil {
			return err
		}
		switch constructor {
		case 0xbb92ba95, 0xfa04579d, 0x6f635b0d, constructorMessageEntityBotCommand, 0x6ed02538, 0x64e475c2, 0xbd610bc9, 0x826f8b60, 0x28a20571, 0x9b69e34b, 0x4c4e743f, 0x9c4e7e8b, 0xbf0693d4, 0x761e6af4, 0x32ca960f:
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
		case 0x73924be0: // pre: offset length language
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.string(); err != nil {
				return err
			}
		case 0x76a6d327: // textUrl
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.string(); err != nil {
				return err
			}
		case 0x352dca58: // mentionName
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.long(); err != nil {
				return err
			}
		case 0xc8cf05f8: // customEmoji
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.long(); err != nil {
				return err
			}
		case 0x20df5d0: // blockquote flags offset length
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
			if _, err := r.int(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("mtproto: unsupported MessageEntity constructor 0x%08x", constructor)
		}
	}
	return nil
}

func skipRawVector(r *tlReader) error {
	vector, err := r.int()
	if err != nil {
		return err
	}
	if vector != constructorVector {
		return fmt.Errorf("mtproto: expected vector, got 0x%08x", vector)
	}
	count, err := r.int()
	if err != nil {
		return err
	}
	for i := 0; i < int(count); i++ {
		// Unknown object: this tiny parser cannot safely skip arbitrary TL objects.
		return fmt.Errorf("mtproto: raw vector with %d objects is not supported by the tiny V12 parser", count)
	}
	return nil
}

// encodeTestMessage is used by tests only.
func encodeTestMessage(id int32, fromID, chatID int64, text string) []byte {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorMessage)
	b = binary.LittleEndian.AppendUint32(b, 1<<8|1<<7) // from_id + entities
	b = binary.LittleEndian.AppendUint32(b, 0)         // flags2
	b = binary.LittleEndian.AppendUint32(b, uint32(id))
	b = binary.LittleEndian.AppendUint32(b, constructorPeerUser)
	b = binary.LittleEndian.AppendUint64(b, uint64(fromID))
	b = binary.LittleEndian.AppendUint32(b, constructorPeerUser)
	b = binary.LittleEndian.AppendUint64(b, uint64(chatID))
	b = binary.LittleEndian.AppendUint32(b, 123) // date
	var tb tlBuffer
	tb.buf = b
	tb.putString(text)
	tb.putInt(constructorVector)
	tb.putInt(1)
	tb.putInt(constructorMessageEntityBotCommand)
	tb.putInt(0)
	tb.putInt(uint32(len(text)))
	return tb.bytes()
}
