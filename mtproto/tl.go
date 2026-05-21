package mtproto

import (
	"encoding/binary"
	"errors"
)

const (
	constructorReqPQMulti        = 0xbe7e8ef1
	constructorResPQ             = 0x05162463
	constructorVector            = 0x1cb5c415
	constructorPQInnerDataDC     = 0xa9f55f95
	constructorReqDHParams       = 0xd712e4be
	constructorServerDHParamsOK  = 0xd0e8075c
	constructorServerDHParamsBad = 0x79cb045d
	constructorServerDHInnerData = 0xb5890dba
	constructorSetClientDHParams = 0xf5045f1f
	constructorClientDHInnerData = 0x6643b654
	constructorDHGenOK           = 0x3bcbf734
	constructorDHGenRetry        = 0x46dc1fb9
	constructorDHGenFail         = 0xa69dae02
)

type tlBuffer struct {
	buf []byte
}

func (b *tlBuffer) putInt(v uint32)  { b.buf = binary.LittleEndian.AppendUint32(b.buf, v) }
func (b *tlBuffer) putLong(v uint64) { b.buf = binary.LittleEndian.AppendUint64(b.buf, v) }
func (b *tlBuffer) putRaw(v []byte)  { b.buf = append(b.buf, v...) }
func (b *tlBuffer) bytes() []byte    { return b.buf }

func (b *tlBuffer) putString(v string) { b.putBytes([]byte(v)) }

func (b *tlBuffer) putBytes(v []byte) {
	n := len(v)
	header := 1
	if n < 254 {
		b.buf = append(b.buf, byte(n))
	} else {
		b.buf = append(b.buf, 254, byte(n), byte(n>>8), byte(n>>16))
		header = 4
	}
	b.buf = append(b.buf, v...)
	for (header+n)%4 != 0 {
		b.buf = append(b.buf, 0)
		header++
	}
}

type tlReader struct {
	buf []byte
	off int
}

func newTLReader(buf []byte) *tlReader { return &tlReader{buf: buf} }

func (r *tlReader) int() (uint32, error) {
	if len(r.buf)-r.off < 4 {
		return 0, errors.New("tl: not enough bytes for int")
	}
	v := binary.LittleEndian.Uint32(r.buf[r.off : r.off+4])
	r.off += 4
	return v, nil
}

func (r *tlReader) long() (uint64, error) {
	if len(r.buf)-r.off < 8 {
		return 0, errors.New("tl: not enough bytes for long")
	}
	v := binary.LittleEndian.Uint64(r.buf[r.off : r.off+8])
	r.off += 8
	return v, nil
}

func (r *tlReader) raw(n int) ([]byte, error) {
	if n < 0 || len(r.buf)-r.off < n {
		return nil, errors.New("tl: not enough bytes for raw")
	}
	v := r.buf[r.off : r.off+n]
	r.off += n
	return v, nil
}

func (r *tlReader) bytes() ([]byte, error) {
	if len(r.buf)-r.off < 1 {
		return nil, errors.New("tl: not enough bytes for bytes length")
	}
	first := int(r.buf[r.off])
	r.off++
	var n int
	header := 1
	if first < 254 {
		n = first
	} else {
		if len(r.buf)-r.off < 3 {
			return nil, errors.New("tl: not enough bytes for long bytes length")
		}
		n = int(r.buf[r.off]) | int(r.buf[r.off+1])<<8 | int(r.buf[r.off+2])<<16
		r.off += 3
		header = 4
	}
	if len(r.buf)-r.off < n {
		return nil, errors.New("tl: not enough bytes for bytes payload")
	}
	v := append([]byte(nil), r.buf[r.off:r.off+n]...)
	r.off += n
	for (header+n)%4 != 0 {
		if len(r.buf)-r.off < 1 {
			return nil, errors.New("tl: not enough bytes for bytes padding")
		}
		r.off++
		header++
	}
	return v, nil
}

func (r *tlReader) string() (string, error) {
	b, err := r.bytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *tlReader) remaining() []byte {
	if r.off >= len(r.buf) {
		return nil
	}
	return r.buf[r.off:]
}
