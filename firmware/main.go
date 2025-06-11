package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const (
	PORTA = iota
	PORTB
)

const (
	PortAddress   = 0x76
	ConfigureByte = 0xD1
	StackSize     = 20
	DataSize      = 10
	Version       = 2
)

type Message struct {
	read bool
	data [DataSize]byte
}

type Stack struct {
	stack    [StackSize]Message
	writePtr byte
	readPtr  byte
}

var (
	ports = []*machine.I2C{
		machine.I2C0,
		machine.I2C1,
	}
	pinSDA = []machine.Pin{
		machine.D0,
		machine.D2,
	}
	pinSCL = []machine.Pin{
		machine.D1,
		machine.D3,
	}
	err    error
	stacks [2]Stack

	led [1]color.RGBA

	versionMessage = []uint8{Version, StackSize, DataSize}
	emptyMessage   = []uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

func main() {

	time.Sleep(3 * time.Second)

	machine.WS2812.Configure(machine.PinConfig{Mode: machine.PinOutput})

	ws := ws2812.NewWS2812(machine.WS2812)

	for p := range ports {
		err = ports[p].Configure(machine.I2CConfig{
			Frequency: 2.8 * machine.MHz,
			Mode:      machine.I2CModeTarget,
			//SDA:       pinSDA[p],
			//SCL:       pinSCL[p],
		})
		if err != nil {
			led[0] = color.RGBA{0xFF, 0x00, 0x00, 0xFF}
			ws.WriteColors(led[:])
			panic("failed to config I2C0 as controller")
		}

		err = ports[p].Listen(PortAddress)
		if err != nil {
			led[0] = color.RGBA{0x00, 0x00, 0xFF, 0xFF}
			ws.WriteColors(led[:])
			panic("failed to listen as I2C target")
		}
		println("SET UP PORT", p)

		for s := range StackSize {
			stacks[p].stack[s].read = true
		}
	}
	go portListener(PORTA)
	go portListener(PORTB)
	led[0] = color.RGBA{0x00, 0xFF, 0x00, 0xFF}
	ws.WriteColors(led[:])

	for {
		time.Sleep(1 * time.Second)
	}

}

func portListener(port byte) {
	buf := make([]byte, 10)
	sendConfigInfo := false

	for {
		evt, n, err := ports[port].WaitForEvent(buf)
		if err != nil {
		}

		switch evt {
		case machine.I2CReceive: // store received message
			if n > DataSize {
				n = DataSize
			}
			if n == 1 && buf[0] == ConfigureByte {
				sendConfigInfo = true
				for s := 0; s < StackSize; s++ {
					for o := 0; o < DataSize; o++ {
						stacks[0].stack[s].data[o] = 0
						stacks[1].stack[s].data[o] = 0
					}
					stacks[1].stack[s].read = false
				}
				stacks[0].writePtr = 0
				stacks[1].writePtr = 0
			} else {
				sendConfigInfo = false
				stacks[port].writePtr = (stacks[port].writePtr + 1) % StackSize
				for o := 0; o < n; o++ {
					stacks[port].stack[stacks[port].writePtr].data[o] = buf[o]
				}
				for o := n; o < DataSize; o++ {
					stacks[port].stack[stacks[port].writePtr].data[o] = 0
				}
				stacks[port].stack[stacks[port].writePtr].read = false
			}
		case machine.I2CRequest: // return the oldest unread message
			portClient := (port + 1) % 2
			if sendConfigInfo && buf[0] == ConfigureByte {
				ports[port].Reply(versionMessage)
			} else {
				ptr := (stacks[portClient].readPtr + 1) % StackSize
				if stacks[portClient].stack[ptr].read {
					ports[port].Reply([]byte{0})
					println("REPLY 0")
					continue
				}
				stacks[portClient].readPtr = ptr
				ports[port].Reply(stacks[portClient].stack[ptr].data[:])
				println("REPLY DATA[:]", stacks[portClient].stack[ptr].data[0], "PORT", port)
				stacks[portClient].stack[ptr].read = true
			}
			sendConfigInfo = false
		case machine.I2CFinish:

		default:
		}
	}
}
