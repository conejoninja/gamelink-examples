package main

import (
	_ "embed"
	"image/color"
	"machine"
	"time"

	"github.com/conejoninja/gamelink"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

const (
	START_GL = iota
	SELECT_MODE
	MAIN
)

var (
	rotaryOldValue, rotaryNewValue int
	textWhite                      = color.RGBA{255, 255, 255, 255}
	textBlack                      = color.RGBA{0, 0, 0, 255}
	mainPad                        bool
)

func main() {
	time.Sleep(3 * time.Second)
	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 2.8 * machine.MHz,
		SDA:       machine.GPIO12,
		SCL:       machine.GPIO13,
	})

	display := ssd1306.NewI2C(i2c)
	display.Configure(ssd1306.Config{
		Address:  0x3C,
		Width:    128,
		Height:   64,
		Rotation: drivers.Rotation180,
	})
	display.ClearDisplay()

	gl := gamelink.New(i2c)
	data, err := gl.Configure()

	if err != nil {
		println("ERROR", err)
	}
	println(data[0], data[1], data[2])
	hasGameLink := (data[0] == 0x02)

	enc := encoders.NewQuadratureViaInterrupt(

		machine.GPIO4,
		machine.GPIO3,
	)

	enc.Configure(encoders.QuadratureConfig{
		Precision: 4,
	})

	rotaryBtn := machine.GPIO2
	rotaryBtn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	menuOption := 0
	state := MAIN
	if hasGameLink {
		state = START_GL
	}

	display.ClearBuffer()
	display.Display()

	for state != MAIN {

		display.ClearBuffer()

		switch state {
		case START_GL:

			if rotaryNewValue = enc.Position(); rotaryNewValue != rotaryOldValue {
				println("value: ", rotaryNewValue)
				if rotaryNewValue > rotaryOldValue {
					menuOption = 1
				} else {
					menuOption = 0
				}
				rotaryOldValue = rotaryNewValue
			}

			if menuOption == 0 {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "[+] USE GAME LINK", textWhite)
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "[ ] CANCEL", textWhite)
			} else {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "[ ] USE GAME LINK", textWhite)
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "[+] CANCEL", textWhite)
			}

			if !rotaryBtn.Get() {
				println("pressed")
				if menuOption == 0 {
					state = SELECT_MODE
				} else {
					state = MAIN
				}
			}

			break

		case SELECT_MODE:

			if rotaryNewValue = enc.Position(); rotaryNewValue != rotaryOldValue {
				println("value: ", rotaryNewValue)
				if rotaryNewValue > rotaryOldValue {
					menuOption = 1
				} else {
					menuOption = 0
				}
				rotaryOldValue = rotaryNewValue
			}

			if menuOption == 0 {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "[+] MAIN PAD", textWhite)
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "[ ] SECONDARY PAD", textWhite)
			} else {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "[ ] MAIN PAD", textWhite)
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "[+] SECONDARY PAD", textWhite)
			}

			if !rotaryBtn.Get() {
				println("pressed")
				state = MAIN
				if menuOption == 0 {
					mainPad = true
				} else {
					mainPad = false
				}
			}

			break

		case MAIN:

			tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "MAIN CODE", textWhite)
			break

		}
		display.Display()
		time.Sleep(100 * time.Millisecond)
	}

}
