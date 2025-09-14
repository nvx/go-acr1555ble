package acr1555ble

import "errors"

type SlotError byte

const (
	SlotErrorCommandAborted        SlotError = 0xFF
	SlotErrorTimeout               SlotError = 0xFE
	SlotErrorParity                SlotError = 0xFD
	SlotErrorOverrun               SlotError = 0xFC
	SlotErrorHardware              SlotError = 0xFB
	SlotErrorBadATRTS              SlotError = 0xF8
	SlotErrorBadATRTCK             SlotError = 0xF7
	SlotErrorProtocolNotSupported  SlotError = 0xF6
	SlotErrorClassNotSupported     SlotError = 0xF5
	SlotErrorProcedureByteConflict SlotError = 0xF4
	SlotErrorDeactivatedProtocol   SlotError = 0xF3
	SlotErrorBusyWithAutoSequence  SlotError = 0xF2
	SlotErrorSlotBusy              SlotError = 0xE0
	SlotErrorInvalidNAD            SlotError = 0x10
	SlotErrorInvalidIFSC           SlotError = 0x0F
	SlotErrorInvalidClockStop      SlotError = 0x0E
	SlotErrorInvalidWI             SlotError = 0x0D
	SlotErrorInvalidGuardTime      SlotError = 0x0C
	SlotErrorInvalidTCCKTS         SlotError = 0x0B
	SlotErrorInvalidFIDI           SlotError = 0x0A
	SlotErrorInvalidLevelParameter SlotError = 0x08
	SlotErrorInvalidPowerSelect    SlotError = 0x07
	SlotErrorInvalidSlot           SlotError = 0x05
	SlotErrorInvalidLength         SlotError = 0x01
	SlotErrorCommandNotSupported   SlotError = 0x00
)

func (e SlotError) Error() string {
	return e.String()
}

type ErrorCode byte

const (
	ErrorCodeChecksum          ErrorCode = 0x01
	ErrorCodeTimeout           ErrorCode = 0x02
	ErrorCodeCommand           ErrorCode = 0x03
	ErrorCodeUnauthorized      ErrorCode = 0x04
	ErrorCodeUndefined         ErrorCode = 0x05
	ErrorCodeReceiveData       ErrorCode = 0x06
	ErrorCodeReceiveDataLength ErrorCode = 0x07
)

func (e ErrorCode) Error() string {
	return e.String()
}

func ccidCheckResponseMessage(res ccidMessage, expected ccidMessageType) (err error) {
	if res.messageType == responseError {
		return ErrorCode(res.headerData[1])
	}

	if res.messageType != expected {
		return errors.New("unexpected message type: " + res.messageType.String())
	}

	switch res.headerData[0] >> 6 {
	case 0:
		return nil
	case 1:
		err = SlotError(res.headerData[1])
		return
	case 2:
		err = errors.New("time extension")
		return
	default:
		err = errors.New("unexpected bmCommandStatus value")
		return
	}
}
