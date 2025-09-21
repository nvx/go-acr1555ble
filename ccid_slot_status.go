package acr1555ble

import (
	"context"
	"github.com/nvx/go-rfid"
)

type ICCStatus byte

const (
	ICCStatusRunning ICCStatus = iota
	ICCStatusInactive
	ICCStatusAbsent
)

type ClockStatus byte

const (
	ClockStatusRunning ClockStatus = iota
	ClockStatusStoppedL
	ClockStatusStoppedH
	ClockStatusStoppedUnknown
)

type SlotStatus struct {
	ICCStatus   ICCStatus
	ClockStatus ClockStatus
}

func (b *ACR1555BLE) sendMessageWithSlotStatusResponse(ctx context.Context, msg ccidMessage) (_ SlotStatus, err error) {
	defer rfid.DeferWrap(ctx, &err)

	res, err := b.exchangeCCID(ctx, msg)
	if err != nil {
		return
	}

	err = ccidCheckResponseMessage(res, responseSlotStatus)
	if err != nil {
		return
	}

	return SlotStatus{
		ICCStatus:   ICCStatus(res.headerData[0] & 0x03),
		ClockStatus: ClockStatus(res.headerData[2]),
	}, nil
}
