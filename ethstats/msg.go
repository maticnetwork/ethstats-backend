package ethstats

import (
	"encoding/json"
	"fmt"
)

type Msg struct {
	typ string
	msg map[string]json.RawMessage
}

func (m *Msg) Set(k string, v json.RawMessage) {
	if m.msg == nil {
		m.msg = map[string]json.RawMessage{}
	}
	m.msg[k] = v
}

func (m *Msg) Copy() *Msg {
	mm := new(Msg)
	*mm = *m

	mm.msg = map[string]json.RawMessage{}
	for k, v := range m.msg {
		mm.msg[k] = append([]byte{}, v...)
	}
	return mm
}

func (m *Msg) Marshal() ([]byte, error) {
	val := map[string]interface{}{
		"emit": []interface{}{
			m.typ,
			m.msg,
		},
	}
	return json.Marshal(val)
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
