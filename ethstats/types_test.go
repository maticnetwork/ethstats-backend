package ethstats

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypes_Block(t *testing.T) {
	var b Block

	num := argBigPtr(big.NewInt(4112))

	data := `{
		"difficulty": "4112",
		"totalDifficulty": "0x1010"
	}`
	assert.NoError(t, json.Unmarshal([]byte(data), &b))

	assert.Equal(t, b.Diff, num)
	assert.Equal(t, b.TotalDiff, num)
}
