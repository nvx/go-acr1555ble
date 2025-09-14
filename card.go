package acr1555ble

import (
	"context"
	"errors"
)

const (
	CtlCodePcToRdrEscape uint16 = 3500
)

type Protocol int

const (
	ProtocolT0 Protocol = iota
	ProtocolT1
	ProtocolPICC
)

type Card struct {
	b        *ACR1555BLE
	sam      bool
	protocol Protocol
	atr      []byte
}

func (b *ACR1555BLE) Connect(ctx context.Context, protocol Protocol) (_ *Card, err error) {
	defer deferWrap(&err)

	c := &Card{
		b:        b,
		sam:      protocol != ProtocolPICC,
		protocol: protocol,
	}

	err = c.Reconnect(ctx)
	if err != nil {
		return
	}

	return c, nil
}

func (c *Card) Close() (err error) {
	defer deferWrap(&err)

	_, err = c.b.ICCPowerOff(context.Background(), c.sam)
	return
}

func (c *Card) Reconnect(ctx context.Context) (err error) {
	defer deferWrap(&err)

	_, err = c.b.ICCPowerOff(ctx, c.sam)
	if err != nil {
		return
	}

	c.atr, err = c.b.ICCPowerOn(ctx, c.sam, PowerSelectAutomatic)
	if err != nil {
		return
	}

	if c.sam {
		err = c.b.SetParameters(ctx, c.sam, c.protocol == ProtocolT1, 0x96, 0x10, 0x00, 0x55, 0x00, 0xFE)
		if err != nil {
			return
		}
	}

	return nil
}

func (c *Card) ATR() ([]byte, error) {
	return c.atr, nil
}

func (c *Card) DeviceName() string {
	if c.sam {
		return "ACS ACR1552 1S CL Reader SAM 0"
	} else {
		return "ACS ACR1552 1S CL Reader PICC 0"
	}
}

func (c *Card) Control(ctx context.Context, code uint16, data []byte) (_ []byte, err error) {
	defer deferWrap(&err)

	switch code {
	case CtlCodePcToRdrEscape:
		return c.b.Escape(ctx, c.sam, data)
	default:
		err = errors.New("unsupported control code")
		return
	}
}

func (c *Card) Exchange(ctx context.Context, capdu []byte) (_ []byte, err error) {
	defer deferWrap(&err)

	return c.b.XfrBlock(ctx, c.sam, 0, capdu)
}
