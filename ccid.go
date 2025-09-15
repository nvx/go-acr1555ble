package acr1555ble

import (
	"encoding"
	"encoding/binary"
	"errors"
)

type ccidMessageType byte

const (
	commandSetParameters ccidMessageType = 0x61
	commandICCPowerOn    ccidMessageType = 0x62
	commandICCPowerOff   ccidMessageType = 0x63
	commandGetSlotStatus ccidMessageType = 0x65
	commandXfrBlock      ccidMessageType = 0x6F
	commandEscape        ccidMessageType = 0x6B
)

const (
	responseError      ccidMessageType = 0x53
	responseDataBlock  ccidMessageType = 0x80
	responseSlotStatus ccidMessageType = 0x81
	responseParameters ccidMessageType = 0x82
	responseEscape     ccidMessageType = 0x83
)

const (
	// in theory the docs indicate this should be 0x1E7, but in practice anything over 0xE2 fails
	ccidMaxBlockDataSize = 0xE2
)

type ccidMessage struct {
	messageType ccidMessageType
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
	buf = append(buf, byte(c.messageType))
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

	c.messageType = ccidMessageType(data[0])
	c.seq = data[6]
	c.headerData = [3]byte{data[7], data[8], data[9]}

	c.data = make([]byte, dataLen)
	copy(c.data, data[10:10+int(dataLen)])

	return nil
}
