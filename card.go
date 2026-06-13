package acr1555ble

import (
	"context"
	"errors"
	"github.com/nvx/go-rfid"
	"github.com/nvx/go-rfid/ccid"
)

type Card struct {
	b        *ACR1555BLE
	slot     byte
	protocol ccid.Protocol
	atr      []byte
}

var (
	_ rfid.PCSC = (*Card)(nil)
)

func (b *ACR1555BLE) Connect(ctx context.Context, protocol ccid.Protocol) (_ *Card, err error) {
	defer rfid.DeferWrap(ctx, &err)

	c := &Card{
		b:        b,
		protocol: protocol,
	}

	if protocol != ccid.ProtocolPICC {
		c.slot = 0x01
	}

	err = c.Reconnect(ctx)
	if err != nil {
		return
	}

	return c, nil
}

func (c *Card) Close() (err error) {
	ctx := context.Background()
	defer rfid.DeferWrap(ctx, &err)

	_, err = c.b.ICCPowerOff(ctx, c.slot)
	return
}

func (c *Card) Reconnect(ctx context.Context) (err error) {
	defer rfid.DeferWrap(ctx, &err)

	_, err = c.b.ICCPowerOff(ctx, c.slot)
	if err != nil {
		return
	}

	c.atr, err = c.b.ICCPowerOn(ctx, c.slot, ccid.PowerSelectAutomatic)
	if err != nil {
		return
	}

	var atr rfid.ATR
	err = atr.UnmarshalBinary(c.atr)
	if err != nil {
		return
	}

	switch c.protocol {
	case ccid.ProtocolT0:
		err = c.b.SetParametersT0(ctx, c.slot, atr.FiDi, atr.GuardTime, atr.T0WI, atr.StopClock)
		if err != nil {
			return
		}
	case ccid.ProtocolT1:
		err = c.b.SetParametersT1(ctx, c.slot, atr.FiDi, atr.GuardTime, atr.T1Waiting, atr.StopClock, atr.T1IFSC, 0x00, atr.T1CRC)
		if err != nil {
			return
		}
	default:
	}

	return nil
}

func (c *Card) ATR() ([]byte, error) {
	return c.atr, nil
}

func (c *Card) DeviceName() string {
	if c.protocol == ccid.ProtocolPICC {
		return "ACS ACR1552 1S CL Reader PICC 0"
	}

	return "ACS ACR1552 1S CL Reader SAM 0"
}

func (c *Card) Control(ctx context.Context, code uint16, data []byte) (_ []byte, err error) {
	defer rfid.DeferWrap(ctx, &err)

	switch code {
	case rfid.CtlCodePcToRdrEscape:
		return c.b.Escape(ctx, c.slot, data)
	default:
		err = errors.New("unsupported control code")
		return
	}
}

func (c *Card) Exchange(ctx context.Context, capdu []byte) (_ []byte, err error) {
	defer rfid.DeferWrap(ctx, &err)

	return c.b.XfrBlockExtendedAPDU(ctx, c.slot, 0, capdu)
}
