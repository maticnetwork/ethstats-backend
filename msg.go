package main

import (
	"encoding/json"
	"fmt"
)

type Msg struct {
	typ string
	msg map[string]json.RawMessage
}

func DecodeMsg(message []byte) (*Msg, error) {
	var msg struct {
		Emit []json.RawMessage
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return nil, err
	}
	if len(msg.Emit) != 2 {
		return nil, fmt.Errorf("2 items expected")
	}

	// decode typename as string
	var typName string
	if err := json.Unmarshal(msg.Emit[0], &typName); err != nil {
		return nil, fmt.Errorf("failed to decode type: %v", err)
	}
	// decode data
	var data map[string]json.RawMessage
	if err := json.Unmarshal(msg.Emit[1], &data); err != nil {
		return nil, fmt.Errorf("failed to decode data: %v", err)
	}

	m := &Msg{
		typ: typName,
		msg: data,
	}
	return m, nil
}

func (m *Msg) msgType() string {
	return m.typ
}

func (m *Msg) decodeMsg(field string, out interface{}) error {
	data, ok := m.msg[field]
	if !ok {
		return fmt.Errorf("message %s not found", field)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}
