package mtproto

import (
	"context"
	"fmt"
)

const (
	constructorUpdatesGetState = 0xedd4882a
	constructorUpdatesState    = 0xa56c2a3e
)

// UpdatesState is the minimal parsed updates.State returned by updates.getState.
type UpdatesState struct {
	PTS         int32
	QTS         int32
	Date        int32
	Seq         int32
	UnreadCount int32
	Raw         []byte
}

// UpdatesGetState calls updates.getState over encrypted MTProto.
// It works after bot or user authorization.
func (c *EncryptedClient) UpdatesGetState(ctx context.Context) (*UpdatesState, *InvokeResult, error) {
	var q tlBuffer
	q.putInt(constructorUpdatesGetState)
	result, err := c.Invoke(ctx, q.bytes())
	if err != nil {
		return nil, nil, err
	}
	state, err := parseUpdatesState(result.Body)
	if err != nil {
		return nil, result, err
	}
	result.Message = "updates.getState succeeded over encrypted MTProto"
	return state, result, nil
}

func parseUpdatesState(body []byte) (*UpdatesState, error) {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, err
	}
	if constructor != constructorUpdatesState {
		return nil, fmt.Errorf("mtproto: expected updates.state, got 0x%08x", constructor)
	}
	pts, err := r.int()
	if err != nil {
		return nil, err
	}
	qts, err := r.int()
	if err != nil {
		return nil, err
	}
	date, err := r.int()
	if err != nil {
		return nil, err
	}
	seq, err := r.int()
	if err != nil {
		return nil, err
	}
	unread, err := r.int()
	if err != nil {
		return nil, err
	}
	return &UpdatesState{PTS: int32(pts), QTS: int32(qts), Date: int32(date), Seq: int32(seq), UnreadCount: int32(unread), Raw: append([]byte(nil), body...)}, nil
}
