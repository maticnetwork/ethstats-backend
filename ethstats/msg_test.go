package ethstats

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsg(t *testing.T) {
	data := `{
		"emit": [
			"msgtype",
			{
				"block": {"number": 1},
				"secret": "b"
			}
		]
	}`

	msg, err := DecodeMsg([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expect := &Msg{
		typ: "msgtype",
		msg: map[string]json.RawMessage{
			"block":  []byte(`{"number": 1}`),
			"secret": []byte(`"b"`),
		},
	}
	if !reflect.DeepEqual(msg, expect) {
		t.Fatal("bad")
	}

	var msg2 struct {
		Number uint64
	}
	if err := msg.decodeMsg("block", &msg2); err != nil {
		t.Fatal(err)
	}
	if msg2.Number != 1 {
		t.Fatal("expected 1")
	}

	data2, err := msg.Marshal()
	assert.NoError(t, err)
	assert.JSONEq(t, string(data), string(data2))
}
