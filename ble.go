package acr1555ble

import (
	"context"
	"errors"
	"log/slog"
	"tinygo.org/x/bluetooth"
)

var (
	acr1555ServiceUUID          = must(bluetooth.ParseUUID(`00003970-817C-48DF-8DB2-476A8134EDE0`))
	acr1555CharCommandReqUUID   = must(bluetooth.ParseUUID(`00003971-817C-48DF-8DB2-476A8134EDE0`))
	acr1555CharCommandResUUID   = must(bluetooth.ParseUUID(`00003972-817C-48DF-8DB2-476A8134EDE0`))
	acr1555CharNotificationUUID = must(bluetooth.ParseUUID(`00003973-817C-48DF-8DB2-476A8134EDE0`))
)

type ACR1555BLE struct {
	device           bluetooth.Device
	charCommandReq   *bluetooth.DeviceCharacteristic
	charCommandRes   *bluetooth.DeviceCharacteristic
	charNotification *bluetooth.DeviceCharacteristic

	seq byte

	piccResponse, samResponse chan ccidMessage
}

func New(ctx context.Context, adapter *bluetooth.Adapter, address bluetooth.Address) (_ *ACR1555BLE, err error) {
	defer deferWrap(&err)

	device, err := adapter.Connect(address, bluetooth.ConnectionParams{})
	if err != nil {
		return
	}

	b := &ACR1555BLE{
		device:       device,
		piccResponse: make(chan ccidMessage),
		samResponse:  make(chan ccidMessage),
	}
	defer func() {
		if err != nil {
			_ = b.device.Disconnect()
		}
	}()

	svcs, err := device.DiscoverServices([]bluetooth.UUID{acr1555ServiceUUID})
	if err != nil {
		return
	}

	if len(svcs) != 1 {
		err = errors.New("unable to find service")
		return
	}

	chars, err := svcs[0].DiscoverCharacteristics([]bluetooth.UUID{
		acr1555CharCommandReqUUID,
		acr1555CharCommandResUUID,
		acr1555CharNotificationUUID,
	})
	if err != nil {
		return
	}

	if len(chars) != 3 {
		err = errors.New("unable to find characteristics")
		return
	}

	for _, char := range chars {
		switch char.UUID() {
		case acr1555CharCommandReqUUID:
			b.charCommandReq = &char
		case acr1555CharCommandResUUID:
			b.charCommandRes = &char
		case acr1555CharNotificationUUID:
			b.charNotification = &char
		}
	}

	if b.charCommandReq == nil || b.charCommandRes == nil || b.charNotification == nil {
		err = errors.New("unable to find characteristics")
		return
	}

	err = b.charCommandRes.EnableNotifications(b.commandResCallback(ctx))
	if err != nil {
		return
	}

	err = b.charNotification.EnableNotifications(b.cardNotificationCallback(ctx))
	if err != nil {
		return
	}

	return b, nil
}

func (b *ACR1555BLE) Close() (err error) {
	defer deferWrap(&err)

	return b.device.Disconnect()
}

func (b *ACR1555BLE) write(d []byte) (err error) {
	defer deferWrap(&err)
	_, err = b.charCommandReq.WriteWithoutResponse(d)
	return
}

func (b *ACR1555BLE) exchangeCCID(ctx context.Context, msg ccidMessage) (_ ccidMessage, err error) {
	defer deferWrap(&err)

	d, err := msg.MarshalBinary()
	if err != nil {
		return
	}

	p, err := payload{
		slotIsSAM:    msg.slotIsSAM,
		totalDataLen: uint16(len(d)),
		hostSeq:      msg.seq,
		readerSeq:    msg.seq,
		data:         d,
	}.MarshalBinary()
	if err != nil {
		return
	}

	err = b.write(p)
	if err != nil {
		return
	}

	return b.waitForResponse(ctx, msg.slotIsSAM)
}

func (b *ACR1555BLE) waitForResponse(ctx context.Context, sam bool) (_ ccidMessage, err error) {
	defer deferWrap(&err)

	select {
	case res := <-b.responseChan(sam):
		return res, nil
	case <-ctx.Done():
		err = context.Cause(ctx)
		return
	}
}

func (b *ACR1555BLE) responseChan(sam bool) chan ccidMessage {
	if sam {
		return b.samResponse
	}
	return b.piccResponse
}

func (b *ACR1555BLE) commandResCallback(ctx context.Context) func([]byte) {
	return func(d []byte) {
		var res payload
		err := res.UnmarshalBinary(d)
		if err != nil {
			slog.ErrorContext(ctx, "Error unmarshalling BLE response payload", slog.String("error", err.Error()))
			return
		}

		var msg ccidMessage
		err = msg.UnmarshalBinary(res.data)
		if err != nil {
			slog.ErrorContext(ctx, "Error unmarshalling CCID response message", slog.String("error", err.Error()))
			return
		}

		b.responseChan(res.slotIsSAM) <- msg
	}
}

func (b *ACR1555BLE) cardNotificationCallback(ctx context.Context) func([]byte) {
	return func(d []byte) {
		slog.InfoContext(ctx, "Got card notification", logHex("data", d))
	}
}
