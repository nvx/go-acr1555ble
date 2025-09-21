package acr1555ble

import (
	"encoding/hex"
	"github.com/nvx/go-rfid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPayloadUnmarshal(t *testing.T) {
	t.Parallel()

	b := rfid.Must(hex.DecodeString(`5500000F0004056B050000000000000000E000007700F7AA`))
	var p payload
	err := p.UnmarshalBinary(b)
	require.NoError(t, err)

	assert.Equal(t, payload{
		totalDataLen: 15,
		hostSeq:      4,
		readerSeq:    5,
		data:         rfid.Must(hex.DecodeString(`6B050000000000000000E000007700`)),
	}, p)
}

func TestPayloadMarshal(t *testing.T) {
	t.Parallel()

	p := payload{
		totalDataLen: 15,
		hostSeq:      4,
		readerSeq:    5,
		data:         rfid.Must(hex.DecodeString(`6B050000000000000000E000007700`)),
	}
	out, err := p.MarshalBinary()
	require.NoError(t, err)

	expected := rfid.Must(hex.DecodeString(`5500000F0004056B050000000000000000E000007700F7AA`))
	assert.Equal(t, expected, out)
}
