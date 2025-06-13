package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const (
	PORTA = 0
	PORTB = 1
)

const (
	PortAddress   = 0x76
	ConfigureByte = 0xD1
	BufferSize    = 16 // Aumentado para mejor rendimiento
	DataSize      = 5
	Version       = 2
)

// Estructura para mensajes con timestamp para debugging
type Message struct {
	data      [DataSize]byte
	length    byte
	timestamp uint32
	valid     bool
}

// Buffer circular thread-safe
type CircularBuffer struct {
	buffer   [BufferSize]Message
	head     byte
	tail     byte
	count    byte
	overflow bool
}

// Estado del puente I2C
type BridgeState struct {
	buffers         [2]CircularBuffer
	configRequested [2]bool
	lastActivity    [2]uint32
	errorCount      [2]uint32
}

var (
	ports = []*machine.I2C{
		machine.I2C0,
		machine.I2C1,
	}

	bridge BridgeState
	led    [1]color.RGBA
	ws     ws2812.Device

	versionMessage = []byte{Version, BufferSize, DataSize}
	emptyMessage   = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	// Variables para debugging
	debugCounter uint32
)

func main() {
	// Inicialización con delay para estabilidad
	time.Sleep(2 * time.Second)

	// Configurar LED de estado
	machine.WS2812.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ws = ws2812.NewWS2812(machine.WS2812)
	setStatusLED(color.RGBA{0xFF, 0xFF, 0x00, 0xFF}) // Amarillo: inicializando

	// Inicializar buffers
	initBuffers()

	// Configurar puertos I2C
	if !setupI2CPorts() {
		setStatusLED(color.RGBA{0xFF, 0x00, 0x00, 0xFF}) // Rojo: error
		panic("Failed to setup I2C ports")
	}

	// Iniciar listeners en goroutines separadas
	go portListener(PORTA)
	go portListener(PORTB)

	// LED verde: funcionando
	setStatusLED(color.RGBA{0x00, 0xFF, 0x00, 0xFF})

	// Loop principal con estadísticas periódicas
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			//printStats()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func initBuffers() {
	for p := 0; p < 2; p++ {
		bridge.buffers[p].head = 0
		bridge.buffers[p].tail = 0
		bridge.buffers[p].count = 0
		bridge.buffers[p].overflow = false
		bridge.configRequested[p] = false
		bridge.lastActivity[p] = 0
		bridge.errorCount[p] = 0

		// Limpiar todos los mensajes
		for i := 0; i < BufferSize; i++ {
			bridge.buffers[p].buffer[i].valid = false
			bridge.buffers[p].buffer[i].length = 0
		}
	}
}

func setupI2CPorts() bool {
	for p := 0; p < 2; p++ {
		// Configurar I2C como target (esclavo)
		err := ports[p].Configure(machine.I2CConfig{
			Frequency: 400 * machine.KHz, // Frecuencia más conservadora
			Mode:      machine.I2CModeTarget,
		})
		if err != nil {
			println("Error configuring I2C port", p, ":", err.Error())
			return false
		}

		// Comenzar a escuchar en la dirección especificada
		err = ports[p].Listen(PortAddress)
		if err != nil {
			println("Error listening on I2C port", p, ":", err.Error())
			return false
		}

		println("I2C Port", p, "configured successfully")
	}
	return true
}

func portListener(port byte) {
	buf := make([]byte, DataSize)

	println("Starting listener for port", port)

	for {
		// Esperar eventos I2C con timeout implícito
		evt, n, err := ports[port].WaitForEvent(buf)

		if err != nil {
			bridge.errorCount[port]++
			continue
		}

		bridge.lastActivity[port] = getTimestamp()

		switch evt {
		case machine.I2CReceive:
			handleReceive(port, buf, n)

		case machine.I2CRequest:
			handleRequest(port)

		case machine.I2CFinish:
			// Evento de finalización - no requiere acción especial

		default:
			println("Unknown I2C event:", evt, "on port", port)
		}
	}
}

func handleReceive(port byte, buf []byte, n int) {
	if n <= 0 || n > DataSize {
		return
	}

	// Verificar si es un comando de configuración
	if n == 1 && buf[0] == ConfigureByte {
		bridge.configRequested[port] = true
		println("Config requested on port", port)
		return
	}

	bridge.configRequested[port] = false

	// Determinar el buffer de destino (puerto opuesto)
	targetPort := (port + 1) % 2

	// Agregar mensaje al buffer circular
	if addMessage(&bridge.buffers[targetPort], buf, byte(n)) {
		println("Message stored for port", targetPort, "- Length:", n)
		debugPrintBuffer(buf, n)
	} else {
		println("Buffer overflow on port", targetPort)
	}
}

func handleRequest(port byte) {
	// Si se solicitó configuración, enviar información de versión
	if bridge.configRequested[port] {
		ports[port].Reply(versionMessage)
		bridge.configRequested[port] = false
		println("Sent version info to port", port)
		return
	}

	// Intentar obtener mensaje del buffer propio
	msg, found := getMessage(&bridge.buffers[port])

	if found && msg.valid {
		// Enviar mensaje real
		reply := make([]byte, DataSize)
		for i := byte(0); i < msg.length && i < DataSize; i++ {
			reply[i] = msg.data[i]
		}
		// Rellenar con ceros si es necesario
		for i := msg.length; i < DataSize; i++ {
			reply[i] = 0x00
		}

		ports[port].Reply(reply)
		println("Sent message to port", port, "- Length:", msg.length)
		debugPrintBuffer(reply, int(msg.length))
	} else {
		// No hay mensajes, enviar respuesta vacía
		ports[port].Reply(emptyMessage)
		// println("Sent empty message to port", port)
	}
}

// Funciones del buffer circular thread-safe

func addMessage(cb *CircularBuffer, data []byte, length byte) bool {
	if cb.count >= BufferSize {
		cb.overflow = true
		return false
	}

	// Copiar datos al mensaje
	msg := &cb.buffer[cb.tail]
	msg.length = length
	msg.timestamp = getTimestamp()
	msg.valid = true

	for i := byte(0); i < length && i < DataSize; i++ {
		msg.data[i] = data[i]
	}
	println("ADD MESSAGE", msg.data[0], msg.data[1], msg.data[2], msg.data[3], length)

	// Avanzar puntero tail
	cb.tail = (cb.tail + 1) % BufferSize
	cb.count++

	return true
}

func getMessage(cb *CircularBuffer) (Message, bool) {
	if cb.count == 0 {
		return Message{}, false
	}

	// Obtener mensaje del head
	msg := cb.buffer[cb.head]

	// Marcar como leído y avanzar puntero
	cb.buffer[cb.head].valid = false
	cb.head = (cb.head + 1) % BufferSize
	cb.count--
	println("GET MESSAGE", msg.data[0], msg.data[1], msg.data[2], msg.data[3])
	return msg, true
}

// Utilidades

func setStatusLED(c color.RGBA) {
	led[0] = c
	ws.WriteColors(led[:])
}

func getTimestamp() uint32 {
	return uint32(time.Now().UnixNano() / 1000000) // Milliseconds
}

func debugPrintBuffer(buf []byte, length int) {
	print("Data: [")
	for i := 0; i < length; i++ {
		print("0x")
		if buf[i] < 16 {
			print("0")
		}
		print(buf[i])
		if i < length-1 {
			print(", ")
		}
	}
	println("]")
}

func printStats() {
	debugCounter++
	println("=== Stats ===", debugCounter)
	for p := 0; p < 2; p++ {
		println("Port", p, "- Buffer:", bridge.buffers[p].count, "/", BufferSize,
			"Errors:", bridge.errorCount[p],
			"Overflow:", bridge.buffers[p].overflow)
		bridge.buffers[p].overflow = false // Reset overflow flag
	}
	println("=============")
}
