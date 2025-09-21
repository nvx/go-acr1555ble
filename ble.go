package acr1555ble

import (
	"context"
	"errors"
	"github.com/nvx/go-rfid"
	"log/slog"
	"sync"
	"tinygo.org/x/bluetooth"
)

var (
	acr1555ServiceUUID          = rfid.Must(bluetooth.ParseUUID(`00003970-817C-48DF-8DB2-476A8134EDE0`))
	acr1555CharCommandReqUUID   = rfid.Must(bluetooth.ParseUUID(`00003971-817C-48DF-8DB2-476A8134EDE0`))
	acr1555CharCommandResUUID   = rfid.Must(bluetooth.ParseUUID(`00003972-817C-48DF-8DB2-476A8134EDE0`))
	acr1555CharNotificationUUID = rfid.Must(bluetooth.ParseUUID(`00003973-817C-48DF-8DB2-476A8134EDE0`))
)

type ACR1555BLE struct {
	device           bluetooth.Device
	charCommandReq   *bluetooth.DeviceCharacteristic
	charCommandRes   *bluetooth.DeviceCharacteristic
	charNotification *bluetooth.DeviceCharacteristic
	mtu              int

	ccidSeq byte

	bleSeqLock         sync.Mutex
	hostSeq, readerSeq byte

	piccResponse, samResponse       chan ccidMessage
	piccChainingBuf, samChainingBuf []byte
}

func New(ctx context.Context, adapter *bluetooth.Adapter, address bluetooth.Address) (_ *ACR1555BLE, err error) {
	defer rfid.DeferWrap(ctx, &err)

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

	mtu, err := b.charCommandReq.GetMTU()
	if err != nil {
		return
	}

	b.mtu = int(mtu)

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
	defer rfid.DeferWrap(context.Background(), &err)

	return b.device.Disconnect()
}

func (b *ACR1555BLE) write(ctx context.Context, d []byte) (err error) {
	defer rfid.DeferWrap(ctx, &err)
	_, err = b.charCommandReq.WriteWithoutResponse(d)
	return
}

func (b *ACR1555BLE) exchangeCCID(ctx context.Context, msg ccidMessage) (_ ccidMessage, err error) {
	defer rfid.DeferWrap(ctx, &err)

	d, err := msg.MarshalBinary()
	if err != nil {
		return
	}

	if len(d) > 0xFFFF {
		err = errors.New("ccid message too large")
		return
	}

	// GATT opcode (1) handle (2) payload wrapping (9)
	maxPayloadSize := b.mtu - 1 - 2 - 9
	totalDataLen := uint16(len(d))

	for len(d) > 0 {
		b.bleSeqLock.Lock()
		hostSeq := b.hostSeq
		readerSeq := b.readerSeq
		b.hostSeq++
		b.bleSeqLock.Unlock()

		n := min(maxPayloadSize, len(d))

		var p []byte
		p, err = payload{
			slotIsSAM:    msg.slotIsSAM,
			totalDataLen: totalDataLen,
			hostSeq:      hostSeq,
			readerSeq:    readerSeq,
			data:         d[:n],
		}.MarshalBinary()
		if err != nil {
			return
		}

		d = d[n:]

		err = b.write(ctx, p)
		if err != nil {
			return
		}
	}

	return b.waitForResponse(ctx, msg.slotIsSAM)
}

func (b *ACR1555BLE) waitForResponse(ctx context.Context, sam bool) (_ ccidMessage, err error) {
	defer rfid.DeferWrap(ctx, &err)

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
			slog.ErrorContext(ctx, "Error unmarshalling BLE response payload", slog.String("error", err.Error()), rfid.LogHex("ble_payload", d))
			return
		}

		b.bleSeqLock.Lock()
		b.readerSeq++
		b.bleSeqLock.Unlock()

		data := res.data

		totalDataLen := int(res.totalDataLen)
		if totalDataLen != len(data) {
			if res.slotIsSAM {
				b.samChainingBuf = bleChain(ctx, data, b.samChainingBuf, totalDataLen)
				if len(b.samChainingBuf) != totalDataLen {
					return
				}
				data = b.samChainingBuf
				b.samChainingBuf = nil
			} else {
				b.piccChainingBuf = bleChain(ctx, data, b.piccChainingBuf, totalDataLen)
				if len(b.piccChainingBuf) != totalDataLen {
					return
				}
				data = b.piccChainingBuf
				b.piccChainingBuf = nil
			}
		}

		var msg ccidMessage
		err = msg.UnmarshalBinary(data)
		if err != nil {
			slog.ErrorContext(ctx, "Error unmarshalling CCID response message", slog.String("error", err.Error()), rfid.LogHex("ble_payload", d), rfid.LogHex("ccid_payload", data))
			return
		}

		b.responseChan(res.slotIsSAM) <- msg
	}
}

func bleChain(ctx context.Context, data, chainingBuf []byte, totalDataLen int) []byte {
	if chainingBuf == nil {
		chainingBuf = make([]byte, 0, totalDataLen)
	} else if cap(chainingBuf) != totalDataLen {
		slog.ErrorContext(ctx, "chaining total length changed")
		return nil
	}
	chainingBuf = append(chainingBuf, data...)
	if len(chainingBuf) > totalDataLen {
		slog.ErrorContext(ctx, "chaining buffer overflow")
		return nil
	}
	return chainingBuf
}

func (b *ACR1555BLE) cardNotificationCallback(ctx context.Context) func([]byte) {
	return func(d []byte) {
		slog.InfoContext(ctx, "Got card notification", rfid.LogHex("data", d))
	}
}
