package acr1555ble

func iccPowerOnBytes(slotIsSAM bool, seq, powerSelect byte) []byte {
	return must(ccidMessage{
		messageType: commandICCPowerOn,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		headerData:  [3]byte{powerSelect},
	}.MarshalBinary())
}

func iccPowerOffBytes(slotIsSAM bool, seq byte) []byte {
	return must(ccidMessage{
		messageType: commandICCPowerOff,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
	}.MarshalBinary())
}

func getSlotStatusBytes(slotIsSAM bool, seq byte) []byte {
	return must(ccidMessage{
		messageType: commandGetSlotStatus,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
	}.MarshalBinary())
}

func xfrBlockBytes(slotIsSAM bool, seq, bwi byte, levelParameter uint16, data []byte) []byte {
	return must(ccidMessage{
		messageType: commandGetSlotStatus,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		headerData:  [3]byte{bwi, byte(levelParameter), byte(levelParameter >> 8)},
		data:        data,
	}.MarshalBinary())
}

func escapeBytes(slotIsSAM bool, seq byte, data []byte) []byte {
	return must(ccidMessage{
		messageType: commandGetSlotStatus,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		data:        data,
	}.MarshalBinary())
}

func setParametersBytes(slotIsSAM bool, seq byte, isT1 bool, fiDi, tccks, guardTime, waiting, clockStop, ifsc byte) []byte {
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

	return must(ccidMessage{
		messageType: commandICCPowerOn,
		slotIsSAM:   slotIsSAM,
		seq:         seq,
		headerData:  [3]byte{protocolNum},
		data:        data,
	}.MarshalBinary())
}
