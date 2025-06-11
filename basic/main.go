package main

import (
	_ "embed"
	"image/color"
	"machine"
	"time"

	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"

	"github.com/conejoninja/gamelink"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

const (
	SCREENSAVER = iota
	LAYER
)

const (
	MENU = iota
	GAME_START
	GAME_WAIT_OTHER
	GAME_WAIT_KEY
	WIN
	LOSE
	NONE
)

const (
	KEY_PRESSED = 12
)

var (
	invertRotaryPins = false
	currentLayer     = 0
	displayShowing   = SCREENSAVER
	displayFrame     = 0

	textWhite = color.RGBA{255, 255, 255, 255}
	textBlack = color.RGBA{0, 0, 0, 255}

	rotaryOldValue, rotaryNewValue int

	state    = MENU
	hostGame = false

	colPins = []machine.Pin{
		machine.GPIO5,
		machine.GPIO6,
		machine.GPIO7,
		machine.GPIO8,
	}

	rowPins = []machine.Pin{
		machine.GPIO9,
		machine.GPIO10,
		machine.GPIO11,
	}

	matrixBtn [12]bool
	colors    []uint32
)

const (
	white = 0x3F3F3FFF
	red   = 0x00FF00FF
	green = 0xFF0000FF
	blue  = 0x0000FFFF
	black = 0x000000FF
)

type WS2812B struct {
	Pin machine.Pin
	ws  *piolib.WS2812B
}

func NewWS2812B(pin machine.Pin) *WS2812B {
	s, _ := pio.PIO0.ClaimStateMachine()
	ws, _ := piolib.NewWS2812B(s, pin)
	ws.EnableDMA(true)
	return &WS2812B{
		ws: ws,
	}
}

func (ws *WS2812B) WriteRaw(rawGRB []uint32) error {
	return ws.ws.WriteRaw(rawGRB)
}

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
	gl.Configure()

	enc := encoders.NewQuadratureViaInterrupt(
		machine.GPIO4,
		machine.GPIO3,
	)

	enc.Configure(encoders.QuadratureConfig{
		Precision: 4,
	})
	rotaryBtn := machine.GPIO2
	rotaryBtn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	for _, c := range colPins {
		c.Configure(machine.PinConfig{Mode: machine.PinOutput})
		c.Low()
	}

	for _, c := range rowPins {
		c.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}

	colors = []uint32{
		black, black, black, black,
		black, black, black, black,
		black, black, black, black,
	}
	ws := NewWS2812B(machine.GPIO1)

	menuOption := 0
	pressed := -1

	for {
		display.ClearBuffer()

		getMatrixState()

		switch state {
		case MENU:

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
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "[+] HOST GAME", textWhite)
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "[ ] JOIN GAME", textWhite)
			} else {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "[ ] HOST GAME", textWhite)
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "[+] JOIN GAME", textWhite)
			}

			if !rotaryBtn.Get() {
				println("pressed")
				if menuOption == 0 {
					state = GAME_WAIT_KEY
					hostGame = true
				} else {
					state = GAME_WAIT_OTHER
					hostGame = false
				}
			}

			break
		case GAME_WAIT_KEY:
			pressed = -1
			for i := 0; i < 12; i++ {
				if matrixBtn[i] {
					pressed = i
					break
				}
			}
			if pressed == -1 {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "Press any key", textWhite)
				break
			}

			if colors[pressed] != black {
				tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "Invalid key", textWhite)
				break
			}

			if hostGame {
				colors[pressed] = red
			} else {
				colors[pressed] = blue
			}
			gl.Write([]uint8{KEY_PRESSED, uint8(pressed)})
			state = GAME_WAIT_OTHER
			break
		case GAME_WAIT_OTHER:
			buffer, err := gl.Read()
			if err != nil {
				break
			}
			if buffer[0] == KEY_PRESSED {
				if hostGame {
					colors[buffer[1]] = blue
				} else {
					colors[buffer[1]] = red
				}
				state = GAME_WAIT_KEY
			}
			tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "Waiting for other", textWhite)
			tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 34, "player's move", textWhite)

			winlose := checkTicTacToe()
			if winlose == WIN {
				state = WIN
			} else if winlose == LOSE {
				state = LOSE
			}

			break
		case WIN:
			tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "You WIN", textWhite)
			break
		case LOSE:
			tinyfont.WriteLine(&display, &proggy.TinySZ8pt7b, 10, 20, "You LOSE", textWhite)
			break
		}

		ws.WriteRaw(colors)
		display.Display()
		time.Sleep(100 * time.Millisecond)
	}

}

func getMatrixState() {
	colPins[0].High()
	colPins[1].Low()
	colPins[2].Low()
	colPins[3].Low()
	time.Sleep(1 * time.Millisecond)

	matrixBtn[0] = rowPins[0].Get()
	matrixBtn[1] = rowPins[1].Get()
	matrixBtn[2] = rowPins[2].Get()

	// COL2
	colPins[0].Low()
	colPins[1].High()
	colPins[2].Low()
	colPins[3].Low()
	time.Sleep(1 * time.Millisecond)

	matrixBtn[3] = rowPins[0].Get()
	matrixBtn[4] = rowPins[1].Get()
	matrixBtn[5] = rowPins[2].Get()

	// COL3
	colPins[0].Low()
	colPins[1].Low()
	colPins[2].High()
	colPins[3].Low()
	time.Sleep(1 * time.Millisecond)

	matrixBtn[6] = rowPins[0].Get()
	matrixBtn[7] = rowPins[1].Get()
	matrixBtn[8] = rowPins[2].Get()

	// COL4
	colPins[0].Low()
	colPins[1].Low()
	colPins[2].Low()
	colPins[3].High()
	time.Sleep(1 * time.Millisecond)

	matrixBtn[9] = rowPins[0].Get()
	matrixBtn[10] = rowPins[1].Get()
	matrixBtn[11] = rowPins[2].Get()
}

func checkTicTacToe() int8 {
	gamePlays := [][3]int{
		[3]int{0, 1, 2}, // VERTICAL
		[3]int{3, 4, 5},
		[3]int{6, 7, 8},
		[3]int{0, 10, 11},

		[3]int{0, 3, 6}, // HORIZONTAL
		[3]int{3, 6, 9},
		[3]int{1, 4, 7},
		[3]int{4, 7, 10},
		[3]int{2, 5, 8},
		[3]int{5, 8, 11},

		[3]int{0, 4, 8}, //DIAGONAL
		[3]int{3, 7, 11},
		[3]int{2, 4, 6},
		[3]int{5, 7, 9},
	}

	c1 := uint32(blue)
	c2 := uint32(red)
	if hostGame {
		c1 = red
		c2 = blue
	}
	for g := 0; g < len(gamePlays); g++ {
		if colors[gamePlays[g][0]] == c1 && colors[gamePlays[g][1]] == c1 && colors[gamePlays[g][2]] == c1 {
			return WIN
		}
		if colors[gamePlays[g][0]] == c2 && colors[gamePlays[g][1]] == c2 && colors[gamePlays[g][2]] == c2 {
			return LOSE
		}
	}

	return NONE

}
