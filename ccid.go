package acr1555ble

import (
	"encoding"
	"encoding/binary"
	"errors"
)

const (
	commandSetParameters byte = 0x61
	commandICCPowerOn    byte = 0x62
	commandICCPowerOff   byte = 0x63
	commandGetSlotStatus byte = 0x65
	commandXfrBlock      byte = 0x6F
	commandEscape        byte = 0x6B
)

const (
	responseError      byte = 0x53
	responseDataBlock  byte = 0x80
	responseSlotStatus byte = 0x81
	responseParameters byte = 0x82
	responseEscape     byte = 0x83
)

type ccidMessage struct {
	messageType byte
	slotIsSAM   bool
	seq         byte
	headerData  [3]byte
	data        []byte
}

var (
	_ encoding.BinaryMarshaler   = ccidMessage{}
	_ encoding.BinaryUnmarshaler = (*ccidMessage)(nil)
)

func (c ccidMessage) MarshalBinary() (_ []byte, err error) {
	defer deferWrap(&err)

	if len(c.data) > 0xFFFFFF {
		err = errors.New("data too large")
		return
	}

	buf := make([]byte, 0, 10+len(c.data))

	// 0
	buf = append(buf, c.messageType)
	// 1-4
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(c.data)))

	// 5
	if c.slotIsSAM {
		buf = append(buf, 0x01)
	} else {
		buf = append(buf, 0x00)
	}

	// 6
	buf = append(buf, c.seq)

	// 7-9
	buf = append(buf, c.headerData[:]...)

	if len(c.data) > 0 {
		buf = append(buf, c.data...)
	}

	return buf, nil
}

func (c *ccidMessage) UnmarshalBinary(data []byte) (err error) {
	defer deferWrap(&err)

	if len(data) < 10 {
		err = errors.New("message too short")
		return
	}

	dataLen := binary.LittleEndian.Uint32(data[1:5])
	if len(data) != 10+int(dataLen) {
		err = errors.New("corrupt payload data length")
		return
	}

	switch data[5] {
	case 0x00:
		c.slotIsSAM = false
	case 0x01:
		c.slotIsSAM = true
	default:
		err = errors.New("invalid slot")
		return
	}

	c.messageType = data[0]
	c.seq = data[6]
	c.headerData = [3]byte{data[7], data[8], data[9]}

	c.data = make([]byte, dataLen)
	copy(c.data, data[10:10+int(dataLen)])

	return nil
}
