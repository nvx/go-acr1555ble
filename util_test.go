package acr1555ble

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXOR8(t *testing.T) {
	t.Parallel()

	b := must(hex.DecodeString(`00000F0000006B050000000000000000E000007700`))
	assert.Equal(t, xor8(b), byte(0xF6))
}
