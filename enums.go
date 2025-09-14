package acr1555ble

type PowerSelect byte

const (
	PowerSelectAutomatic PowerSelect = iota
	PowerSelect5v
	PowerSelect3v
)
