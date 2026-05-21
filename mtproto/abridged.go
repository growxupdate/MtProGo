package mtproto

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

func abridgedInit(conn net.Conn) error {
	_, err := conn.Write([]byte{0xef})
	return err
}

func abridgedWrite(conn net.Conn, payload []byte) error {
	if len(payload)%4 != 0 {
		return errors.New("abridged packet length must be divisible by 4")
	}
	length := len(payload) / 4
	var header []byte
	if length < 127 {
		header = []byte{byte(length)}
	} else {
		if length > 0xFFFFFF {
			return errors.New("abridged packet too large")
		}
		header = []byte{0x7f, byte(length), byte(length >> 8), byte(length >> 16)}
	}
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

func abridgedRead(conn net.Conn) ([]byte, error) {
	var first [1]byte
	if _, err := io.ReadFull(conn, first[:]); err != nil {
		return nil, err
	}
	length := int(first[0])
	if length == 0x7f {
		var extended [3]byte
		if _, err := io.ReadFull(conn, extended[:]); err != nil {
			return nil, err
		}
		length = int(extended[0]) | int(extended[1])<<8 | int(extended[2])<<16
	}
	if length <= 0 {
		return nil, errors.New("abridged invalid length")
	}
	payload := make([]byte, length*4)
	_, err := io.ReadFull(conn, payload)
	return payload, err
}

func buildUnencryptedMessage(msgID int64, body []byte) []byte {
	var out []byte
	out = binary.LittleEndian.AppendUint64(out, 0) // auth_key_id
	out = binary.LittleEndian.AppendUint64(out, uint64(msgID))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(body)))
	out = append(out, body...)
	for len(out)%4 != 0 {
		out = append(out, 0)
	}
	return out
}

func parseUnencryptedMessage(payload []byte) ([]byte, error) {
	if len(payload) < 20 {
		return nil, errors.New("mtproto: short unencrypted message")
	}
	reader := newTLReader(payload)
	authKeyID, err := reader.long()
	if err != nil {
		return nil, err
	}
	if authKeyID != 0 {
		return nil, errors.New("mtproto: expected unencrypted auth key id 0")
	}
	if _, err := reader.long(); err != nil { // msg_id
		return nil, err
	}
	length, err := reader.int()
	if err != nil {
		return nil, err
	}
	body, err := reader.raw(int(length))
	if err != nil {
		return nil, err
	}
	return body, nil
}
