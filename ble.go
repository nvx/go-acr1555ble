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
}

func New(ctx context.Context, adapter *bluetooth.Adapter, address bluetooth.Address) (_ *ACR1555BLE, err error) {
	defer deferWrap(&err)

	device, err := adapter.Connect(address, bluetooth.ConnectionParams{})
	if err != nil {
		return
	}

	b := &ACR1555BLE{
		device: device,
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

func (b *ACR1555BLE) write(d []byte) (err error) {
	defer deferWrap(&err)
	_, err = b.charCommandReq.Write(d)
	return
}

func (b *ACR1555BLE) commandResCallback(ctx context.Context) func([]byte) {
	return func(d []byte) {
		slog.InfoContext(ctx, "Got command res", logHex("data", d))
	}
}
func (b *ACR1555BLE) cardNotificationCallback(ctx context.Context) func([]byte) {
	return func(d []byte) {
		slog.InfoContext(ctx, "Got card notification", logHex("data", d))
	}
}
