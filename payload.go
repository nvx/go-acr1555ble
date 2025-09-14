package acr1555ble

import (
	"encoding"
	"encoding/binary"
	"errors"
)

type payload struct {
	slotIsSAM    bool
	totalDataLen uint16
	hostSeq      byte
	readerSeq    byte
	data         []byte
}

var (
	_ encoding.BinaryMarshaler   = payload{}
	_ encoding.BinaryUnmarshaler = (*payload)(nil)
)

const (
	startByte byte = 0x55
	stopByte  byte = 0xAA
)

func (p payload) MarshalBinary() (_ []byte, err error) {
	defer deferWrap(&err)

	if len(p.data) > int(p.totalDataLen) {
		err = errors.New("data larger than total data length")
		return
	}

	buf := make([]byte, 0, 9+len(p.data))

	// 0
	buf = append(buf, startByte)

	// 1
	if p.slotIsSAM {
		buf = append(buf, 0x01)
	} else {
		buf = append(buf, 0x00)
	}

	// 2-3
	buf = binary.BigEndian.AppendUint16(buf, p.totalDataLen)

	buf = append(buf,
		0x00,        // 4 reserved - frame type?
		p.hostSeq,   // 5
		p.readerSeq, // 6
	)

	// 7-
	buf = append(buf, p.data...)

	buf = append(buf,
		xor8(buf[1:]), // n-1 checksum covers everything except the start/stop bytes
		stopByte,      // n
	)

	return buf, nil
}

func (p *payload) UnmarshalBinary(data []byte) (err error) {
	defer deferWrap(&err)

	if len(data) < 9 || data[0] != startByte || data[len(data)-1] != stopByte || xor8(data[1:len(data)-2]) != data[len(data)-2] {
		err = errors.New("corrupt payload")
		return
	}

	switch data[1] {
	case 0x00:
		p.slotIsSAM = false
	case 0x01:
		p.slotIsSAM = true
	default:
		err = errors.New("invalid slot")
		return
	}

	p.totalDataLen = binary.BigEndian.Uint16(data[2:4])
	if len(data) > 9+int(p.totalDataLen) {
		err = errors.New("current payload data length larger than total data length")
		return
	}

	if data[4] != 0x00 {
		err = errors.New("unknown frame type")
		return
	}

	p.hostSeq = data[5]
	p.readerSeq = data[6]

	p.data = make([]byte, len(data)-9)
	copy(p.data, data[7:7+len(p.data)])

	return nil
}
