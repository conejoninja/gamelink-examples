package main

import (
	_ "embed"
	"image/color"
	"machine"
	"time"

	"github.com/conejoninja/gamelink"
	"tinygo.org/x/drivers/st7789"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

const (
	MENU = iota
	GAME_START
	GAME_WAIT_OTHER
	GAME_WAIT_KEY
	WIN
	LOSE
	NONE
	DEMO
	START_GL
	NO_GL
)

const (
	KEY_PRESSED = 12
	EMPTY       = 0
	CIRCLE      = 1
	CROSS       = 2
)

const (
	BLACK = iota
	WHITE
	RED
	GREEN
	TEXT
	ORANGE
	PURPLE
	BLUE
)

var colors = []color.RGBA{
	color.RGBA{0, 0, 0, 255},
	color.RGBA{255, 255, 255, 255},
	color.RGBA{250, 0, 0, 255},
	color.RGBA{0, 200, 0, 255},
	color.RGBA{160, 160, 160, 255},
	color.RGBA{255, 153, 51, 255},
	color.RGBA{153, 51, 255, 255},
	color.RGBA{0, 0, 255, 255},
}
var (
	invertRotaryPins = false
	currentLayer     = 0
	displayShowing   = MENU
	displayFrame     = 0

	state    = DEMO
	hostGame = false

	display                                       st7789.Device
	btnA, btnB, btnUp, btnLeft, btnDown, btnRight machine.Pin
	plays                                         [3][3]uint8 = [3][3]uint8{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}}
	play                                          uint8
)

func main() {
	time.Sleep(1 * time.Second)
	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})

	gl := gamelink.New(i2c)
	data, err := gl.Configure()

	if err != nil {
		println("ERROR [GL]", err)
	}
	println(data[0], data[1], data[2])
	hasGameLink := (data[0] == 0x02)

	machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 8000000,
		Mode:      0,
	})

	display = st7789.New(machine.SPI0,
		machine.TFT_RST,       // TFT_RESET
		machine.TFT_WRX,       // TFT_DC
		machine.TFT_CS,        // TFT_CS
		machine.TFT_BACKLIGHT) // TFT_LITE

	display.Configure(st7789.Config{
		Rotation: st7789.ROTATION_270,
		Height:   320,
	})

	btnA = machine.BUTTON_A
	btnB = machine.BUTTON_B
	btnUp = machine.BUTTON_UP
	btnLeft = machine.BUTTON_LEFT
	btnDown = machine.BUTTON_DOWN
	btnRight = machine.BUTTON_RIGHT
	btnA.Configure(machine.PinConfig{Mode: machine.PinInput})
	btnB.Configure(machine.PinConfig{Mode: machine.PinInput})
	btnUp.Configure(machine.PinConfig{Mode: machine.PinInput})
	btnLeft.Configure(machine.PinConfig{Mode: machine.PinInput})
	btnDown.Configure(machine.PinConfig{Mode: machine.PinInput})
	btnRight.Configure(machine.PinConfig{Mode: machine.PinInput})

	black := color.RGBA{0, 0, 0, 255}
	display.FillScreen(black)

	state = NO_GL
	if hasGameLink {
		state = START_GL
	}

	for {
		switch state {
		case DEMO:
			oldI, oldJ, i, j := -1, -1, 0, 0
			pressed := false
			drawBoard(0)
			drawCircle(0, 0)
			drawCross(1, 1)
			for {
				if !pressed {
					if !btnRight.Get() {
						i++
						if i > 2 {
							i = 2
						}
						pressed = true
					}
					if !btnLeft.Get() {
						i--
						if i < 0 {
							i = 0
						}
						pressed = true
					}
					if !btnUp.Get() {
						j--
						if j < 0 {
							j = 0
						}
						pressed = true
					}
					if !btnDown.Get() {
						j++
						if j > 2 {
							j = 2
						}
						pressed = true
					}
				} else {
					if btnRight.Get() && btnLeft.Get() && btnUp.Get() && btnDown.Get() {
						pressed = false
					}
				}
				if pressed && (oldI != i || oldJ != j) {
					drawBorder(oldI, oldJ, BLACK)
					oldI = i
					oldJ = j
					drawBorder(i, j, GREEN)
				}
				time.Sleep(100 * time.Millisecond)
			}
			break
		case START_GL:

			display.FillScreen(colors[PURPLE])

			options := []string{
				"USE GAME LINK",
				"CANCEL",
			}
			selected := int16(0)
			numOpts := int16(len(options))
			for i := int16(0); i < numOpts; i++ {
				tinydraw.Circle(&display, 28, 35+20*i, 6, color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 39, 39+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 39, 40+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 39, 41+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 41+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 41, 41+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 41, 40+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 41, 39+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 39+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 40+20*i, options[i], color.RGBA{250, 250, 0, 255})
			}

			tinydraw.FilledCircle(&display, 28, 35, 4, color.RGBA{200, 200, 0, 255})

			released := true
			for {
				if released && !btnUp.Get() && selected > 0 {
					selected--
					tinydraw.FilledCircle(&display, 28, 35+20*selected, 4, color.RGBA{200, 200, 0, 255})
					tinydraw.FilledCircle(&display, 28, 35+20*(selected+1), 4, colors[PURPLE])
				}
				if released && !btnDown.Get() && selected < (numOpts-1) {
					selected++
					tinydraw.FilledCircle(&display, 28, 35+20*selected, 4, color.RGBA{200, 200, 0, 255})
					tinydraw.FilledCircle(&display, 28, 35+20*(selected-1), 4, colors[PURPLE])
				}
				if released && !btnA.Get() {
					break
				}
				if btnA.Get() && btnUp.Get() && btnDown.Get() {
					released = true
				} else {
					released = false
				}
				time.Sleep(200 * time.Millisecond)
			}

			if selected == 0 {
				state = MENU
				hostGame = true
			} else {
				state = LOSE
				hostGame = false
			}

			break

		case MENU:

			display.FillScreen(colors[PURPLE])

			options := []string{
				"HOST GAME",
				"JOIN GAME",
			}
			selected := int16(0)
			numOpts := int16(len(options))
			for i := int16(0); i < numOpts; i++ {
				tinydraw.Circle(&display, 28, 35+20*i, 6, color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 39, 39+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 39, 40+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 39, 41+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 41+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 41, 41+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 41, 40+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 41, 39+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 39+20*i, options[i], color.RGBA{0, 0, 0, 255})
				tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 40+20*i, options[i], color.RGBA{250, 250, 0, 255})
			}

			tinydraw.FilledCircle(&display, 28, 35, 4, color.RGBA{200, 200, 0, 255})

			released := true
			for {
				if released && !btnUp.Get() && selected > 0 {
					selected--
					tinydraw.FilledCircle(&display, 28, 35+20*selected, 4, color.RGBA{200, 200, 0, 255})
					tinydraw.FilledCircle(&display, 28, 35+20*(selected+1), 4, colors[PURPLE])
				}
				if released && !btnDown.Get() && selected < (numOpts-1) {
					selected++
					tinydraw.FilledCircle(&display, 28, 35+20*selected, 4, color.RGBA{200, 200, 0, 255})
					tinydraw.FilledCircle(&display, 28, 35+20*(selected-1), 4, colors[PURPLE])
				}
				if released && !btnA.Get() {
					break
				}
				if btnA.Get() && btnUp.Get() && btnDown.Get() {
					released = true
				} else {
					released = false
				}
				time.Sleep(200 * time.Millisecond)
			}

			if selected == 0 {
				state = GAME_WAIT_KEY
				hostGame = true
			} else {
				state = GAME_WAIT_OTHER
				hostGame = false
			}
			drawBoard(state)

			break
		case GAME_WAIT_KEY:
			tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 260, 20, "Press any key", colors[WHITE])

			oldI, oldJ, i, j := -1, -1, 0, 0
			pressed := false
			for {
				if !pressed {
					if !btnRight.Get() {
						i++
						if i > 2 {
							i = 2
						}
						pressed = true
					}
					if !btnLeft.Get() {
						i--
						if i < 0 {
							i = 0
						}
						pressed = true
					}
					if !btnUp.Get() {
						j--
						if j < 0 {
							j = 0
						}
						pressed = true
					}
					if !btnDown.Get() {
						j++
						if j > 2 {
							j = 2
						}
						pressed = true
					}
				} else {
					if btnRight.Get() && btnLeft.Get() && btnUp.Get() && btnDown.Get() {
						pressed = false
					}
				}
				if pressed && (oldI != i || oldJ != j) {
					drawBorder(oldI, oldJ, BLACK)
					oldI = i
					oldJ = j
					drawBorder(i, j, GREEN)
				}
				if !btnA.Get() && plays[i][j] == EMPTY {
					if hostGame {
						plays[i][j] = CIRCLE
					} else {
						plays[i][j] = CROSS
					}
					gl.Write([]uint8{KEY_PRESSED, uint8(3*i + j), uint8(3*i + j), uint8(3*i + j)})
					println("MINE", uint8(3*i+j))
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			state = GAME_WAIT_OTHER
			drawBoard(state)
			break
		case GAME_WAIT_OTHER:
			buffer, err := gl.Read()
			if err != nil {
				break
			}
			if buffer[0] == KEY_PRESSED {
				println("OTHER", buffer[0], buffer[1], buffer[2], buffer[3], len(buffer), buffer[1]/3, buffer[1]%3)
				play = getPlayFromBuffer(buffer[1], buffer[2], buffer[3])
				if hostGame {
					plays[play/3][play%3] = CROSS
				} else {
					plays[play/3][play%3] = CIRCLE
				}
				state = GAME_WAIT_KEY
				drawBoard(state)
			}

			winlose := checkTicTacToe()
			if winlose == WIN {
				state = WIN
			} else if winlose == LOSE {
				state = LOSE
			}

			break
		case WIN:
			display.FillScreen(colors[PURPLE])
			tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 50, "You WIN", colors[WHITE])
			break
		case LOSE:
			display.FillScreen(colors[PURPLE])
			tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 80, "You LOSE", colors[WHITE])
			break
		case NO_GL:
			display.FillScreen(colors[PURPLE])
			tinyfont.WriteLine(&display, &freemono.Regular12pt7b, 40, 80, "Gamelink not found", colors[WHITE])
			break
		}

		display.Display()
		time.Sleep(100 * time.Millisecond)
	}

}

func checkTicTacToe() int8 {
	gamePlays := [][3]int{
		[3]int{0, 1, 2}, // VERTICAL
		[3]int{3, 4, 5},
		[3]int{6, 7, 8},

		[3]int{0, 3, 6}, // HORIZONTAL
		[3]int{1, 4, 7},
		[3]int{2, 5, 8},

		[3]int{0, 4, 8}, //DIAGONAL
		[3]int{2, 4, 6},
	}

	c1 := uint8(CROSS)
	c2 := uint8(CIRCLE)
	if hostGame {
		c1 = CIRCLE
		c2 = CROSS
	}
	for g := 0; g < len(gamePlays); g++ {
		if plays[gamePlays[g][0]/3][gamePlays[g][0]%3] == c1 &&
			plays[gamePlays[g][1]/3][gamePlays[g][1]%3] == c1 &&
			plays[gamePlays[g][2]/3][gamePlays[g][2]%3] == c1 {
			return WIN
		}
		if plays[gamePlays[g][0]/3][gamePlays[g][0]%3] == c2 &&
			plays[gamePlays[g][1]/3][gamePlays[g][1]%3] == c2 &&
			plays[gamePlays[g][2]/3][gamePlays[g][2]%3] == c2 {
			return LOSE
		}
	}

	return NONE

}

func drawCircle(i, j int) {
	tinydraw.FilledCircle(&display, 40+80*int16(i), 40+80*int16(j), 20, colors[RED])
}

func drawCross(i, j int) {
	tinydraw.Line(&display, 80*int16(i)+14, 80*int16(j)+18, 80*int16(i)+62, 80*int16(j)+66, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+15, 80*int16(j)+17, 80*int16(i)+63, 80*int16(j)+65, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+16, 80*int16(j)+16, 80*int16(i)+64, 80*int16(j)+64, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+17, 80*int16(j)+15, 80*int16(i)+65, 80*int16(j)+63, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+18, 80*int16(j)+14, 80*int16(i)+66, 80*int16(j)+62, colors[BLUE])

	tinydraw.Line(&display, 80*int16(i)+14, 80*int16(j)+62, 80*int16(i)+62, 80*int16(j)+14, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+15, 80*int16(j)+63, 80*int16(i)+63, 80*int16(j)+15, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+16, 80*int16(j)+64, 80*int16(i)+64, 80*int16(j)+16, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+17, 80*int16(j)+65, 80*int16(i)+65, 80*int16(j)+17, colors[BLUE])
	tinydraw.Line(&display, 80*int16(i)+18, 80*int16(j)+66, 80*int16(i)+66, 80*int16(j)+18, colors[BLUE])
}

func drawBorder(i, j, c int) {
	tinydraw.FilledRectangle(&display, 80*int16(i), 80*int16(j), 80, 4, colors[c])
	tinydraw.FilledRectangle(&display, 80*int16(i), 80*int16(j), 4, 80, colors[c])
	tinydraw.FilledRectangle(&display, 80*int16(i)+76, 80*int16(j), 4, 80, colors[c])
	tinydraw.FilledRectangle(&display, 80*int16(i), 80*int16(j)+76, 80, 4, colors[c])
}

func drawBoard(state int) {
	display.FillScreen(colors[WHITE])

	// BORDER
	tinydraw.FilledRectangle(&display, 0, 0, 240, 5, colors[BLACK])
	tinydraw.FilledRectangle(&display, 0, 235, 240, 5, colors[BLACK])
	tinydraw.FilledRectangle(&display, 0, 0, 5, 240, colors[BLACK])
	tinydraw.FilledRectangle(&display, 235, 0, 5, 240, colors[BLACK])
	// VERTICAL
	tinydraw.FilledRectangle(&display, 77, 0, 5, 240, colors[BLACK])
	tinydraw.FilledRectangle(&display, 157, 0, 5, 240, colors[BLACK])
	// HORIZONTAL
	tinydraw.FilledRectangle(&display, 0, 77, 240, 5, colors[BLACK])
	tinydraw.FilledRectangle(&display, 0, 157, 240, 5, colors[BLACK])

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if plays[i][j] == CIRCLE {
				drawCircle(i, j)
			} else if plays[i][j] == CROSS {
				drawCross(i, j)
			}
		}
	}

	if state == GAME_WAIT_KEY {
		tinyfont.WriteLineRotated(&display, &freemono.Regular12pt7b, 300, 220, "Press any key", colors[BLACK], tinyfont.ROTATION_270)
	} else if state == GAME_WAIT_OTHER {
		tinyfont.WriteLineRotated(&display, &freemono.Regular12pt7b, 280, 220, "Waiting for other", colors[BLACK], tinyfont.ROTATION_270)
		tinyfont.WriteLineRotated(&display, &freemono.Regular12pt7b, 300, 220, " player's move", colors[BLACK], tinyfont.ROTATION_270)
	}

}

func getPlayFromBuffer(b1, b2, b3 uint8) uint8 {
	if b1 == b2 || b1 == b3 {
		return b1
	} else if b2 == b3 {
		return b2
	}
	return b1
}
