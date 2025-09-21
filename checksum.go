package acr1555ble

func xor8(in []byte) byte {
	var out byte
	for _, b := range in {
		out ^= b
	}
	return out
}
