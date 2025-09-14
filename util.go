package acr1555ble

import (
	"encoding/hex"
	"github.com/ansel1/merry/v2"
	"log/slog"
	"strings"
)

func deferWrap(err *error) {
	if err != nil {
		*err = merry.WrapSkipping(*err, 1)
	}
}

func must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func logHex(key string, value []byte) slog.Attr {
	return slog.String(key, strings.ToUpper(hex.EncodeToString(value)))
}

func xor8(in []byte) byte {
	var out byte
	for _, b := range in {
		out ^= b
	}
	return out
}
