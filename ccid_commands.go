package acr1555ble

import (
	"context"
	"errors"
)

func iccPowerOnMessage(slotIsSAM bool, seq, powerSelect byte) ccidMessage {
	return ccidMessage{
		messageType: commandICCPowerOn,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		headerData:  [3]byte{powerSelect},
	}
}

// ICCPowerOn powers on the card and returns the ATR
func (b *ACR1555BLE) ICCPowerOn(ctx context.Context, sam bool, powerSelect PowerSelect) (_ []byte, err error) {
	defer deferWrap(&err)

	msg := iccPowerOnMessage(sam, b.seq, byte(powerSelect))
	b.seq++

	res, err := b.exchangeCCID(ctx, msg)
	if err != nil {
		return
	}

	err = ccidCheckResponseMessage(res, responseDataBlock)
	if err != nil {
		return
	}

	if res.headerData[2] != 0x00 {
		err = errors.New("unexpected bChainParameter for ATR response")
		return
	}

	return res.data, nil
}

func iccPowerOffMessage(slotIsSAM bool, seq byte) ccidMessage {
	return ccidMessage{
		messageType: commandICCPowerOff,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
	}
}

// ICCPowerOff powers off the card and returns the updated slot status
func (b *ACR1555BLE) ICCPowerOff(ctx context.Context, sam bool) (_ SlotStatus, err error) {
	defer deferWrap(&err)

	msg := iccPowerOffMessage(sam, b.seq)
	b.seq++

	return b.sendMessageWithSlotStatusResponse(ctx, msg)
}

func getSlotStatusMessage(slotIsSAM bool, seq byte) ccidMessage {
	return ccidMessage{
		messageType: commandGetSlotStatus,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
	}
}

func (b *ACR1555BLE) GetSlotStatus(ctx context.Context, sam bool) (_ SlotStatus, err error) {
	defer deferWrap(&err)

	msg := getSlotStatusMessage(sam, b.seq)
	b.seq++

	return b.sendMessageWithSlotStatusResponse(ctx, msg)
}

func xfrBlockMessage(slotIsSAM bool, seq, bwi byte, levelParameter uint16, data []byte) ccidMessage {
	return ccidMessage{
		messageType: commandXfrBlock,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		headerData:  [3]byte{bwi, byte(levelParameter), byte(levelParameter >> 8)},
		data:        data,
	}
}

func (b *ACR1555BLE) XfrBlock(ctx context.Context, sam bool, bwi byte, data []byte) (_ []byte, err error) {
	defer deferWrap(&err)

	var levelParameter uint16
	if len(data) > ccidMaxBlockDataSize {
		// command begins with this command and continues in the next PC_to_RDR_XfrBlock
		levelParameter = 0x0001
	}

	var responseData []byte
	for {
		if ctx.Err() != nil {
			// TODO: Send abort?
			err = context.Cause(ctx)
			return
		}

		n := min(len(data), ccidMaxBlockDataSize)

		msg := xfrBlockMessage(sam, b.seq, bwi, levelParameter, data[:n])
		b.seq++

		var res ccidMessage
		res, err = b.exchangeCCID(ctx, msg)
		if err != nil {
			return
		}

		data = data[n:]
		if len(data) == 0 {
			// empty abData, continuation of response APDU is expected in the next RDR_to_PC_DataBlock
			levelParameter = 0x0010
		} else if len(data) <= ccidMaxBlockDataSize {
			// continues a command APDU and ends the APDU command
			levelParameter = 0x0002
		} else {
			// continues a command APDU and another block is to follow
			levelParameter = 0x0003
		}

		err = ccidCheckResponseMessage(res, responseDataBlock)
		if err != nil {
			return
		}

		switch res.headerData[2] {
		case 0x00: // begins and ends in this command, no chaining
			return res.data, nil
		case 0x01: // begins in this command and continues
			responseData = res.data
		case 0x02: // continues response and ends
			responseData = append(responseData, res.data...)
			return responseData, nil
		case 0x03: // continues and another to follow
			responseData = append(responseData, res.data...)
		case 0x10: // empty
		default:
			err = errors.New("unknown bChainParameter value returned from reader")
			return
		}
	}
}

func escapeMessage(slotIsSAM bool, seq byte, data []byte) ccidMessage {
	return ccidMessage{
		messageType: commandEscape,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		data:        data,
	}
}

func (b *ACR1555BLE) Escape(ctx context.Context, sam bool, data []byte) (_ []byte, err error) {
	defer deferWrap(&err)

	msg := escapeMessage(sam, b.seq, data)
	b.seq++
	data = nil

	res, err := b.exchangeCCID(ctx, msg)
	if err != nil {
		return
	}

	err = ccidCheckResponseMessage(res, responseEscape)
	if err != nil {
		return
	}

	if res.headerData[2] != 0x00 {
		err = errors.New("unhandled bChainParameter value returned from reader during escape")
		return
	}

	return res.data, nil
}

func setParametersMessage(slotIsSAM bool, seq byte, isT1 bool, fiDi, tccks, guardTime, waiting, clockStop, ifsc byte) ccidMessage {
	var data []byte
	var protocolNum byte
	if isT1 {
		protocolNum = 1
		data = []byte{
			fiDi,
			tccks,
			guardTime,
			waiting,
			clockStop,
			ifsc,
			0x00, // nad - only 0x00 supported
		}
	} else {
		data = []byte{
			fiDi,
			tccks,
			guardTime,
			waiting,
			clockStop,
		}
	}

	return ccidMessage{
		messageType: commandSetParameters,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		headerData:  [3]byte{protocolNum},
		data:        data,
	}
}

func (b *ACR1555BLE) SetParameters(ctx context.Context, sam bool, isT1 bool, fiDi, tccks, guardTime, waiting, clockStop, ifsc byte) (err error) {
	defer deferWrap(&err)

	msg := setParametersMessage(sam, b.seq, isT1, fiDi, tccks, guardTime, waiting, clockStop, ifsc)
	b.seq++

	res, err := b.exchangeCCID(ctx, msg)
	if err != nil {
		return
	}

	err = ccidCheckResponseMessage(res, responseParameters)
	if err != nil {
		return
	}

	return nil
}
