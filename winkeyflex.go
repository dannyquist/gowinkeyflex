package main

import (
	"flag"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"image/color"
	"log"
	"os"
	"strconv"
)

type K1elMessage struct {
	status string
	msg    string
	src    string
}

//const OPEN_CMD = "\x00\x02"
//const CLOSE_CMD = "\x00\x03"
//const CONFIG_ENABLE_ECHO_CMD = "\x0e\x40"
//const SET_SPEED_TO_POT_CMD = "\x02\x00"
//const GET_POT_SPEED_CMD = "\x07"

const STATUS_BITMASK = 0x3f
const STATUS_READY = 8
const STATUS_KEYSTART = 6
const STATUS_KEYSTOP = 4
const STATUS_BUFFER_READY = 0
const STATUS_MASK = 0xc0
const STATUS_TYPE_STATUS = 0xc0
const STATUS_TYPE_POT = 0x80

func k1elSerialReader(PortName string, retChan chan<- K1elMessage) {

	//serialOptions := serial.OpenOptions{
	//	PortName:        "/dev/ttyUSB0",
	//	BaudRate:        1200,
	//	DataBits:        8,
	//	StopBits:        2,
	//	ParityMode:      serial.,
	//	MinimumReadSize: 4,
	//}

	if len(PortName) == 0 {
		retChan <- K1elMessage{status: "serial", msg: "invalid port", src: "winkeyer"}
		return
	}

	mode := &serial.Mode{
		BaudRate: 1200,
		DataBits: 8,
		StopBits: 2,
		Parity:   serial.NoParity,
	}

	log.Printf("Opening Winkeyer serial port %s", PortName)

	port, err := serial.Open(PortName, mode)
	if err != nil {
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
		return
	}

	defer func() {
		_, err := port.Write([]byte{0x00, 0x03})
		if err != nil {
			log.Printf("Could not send close command to serial port: %v", err)
			return
		}

		err = port.Close()
		if err != nil {
			log.Println("Error closing port", err)
			return
		}
	}()

	_, err = port.Write([]byte{0x00, 0x02}) // Open cmd
	if err != nil {
		log.Printf("error writing to serial: %v")
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
		return
	}

	// get the version
	buf := []byte{0x00}

	_, err = port.Read(buf)
	if err != nil {
		log.Printf("Error reading version: %v", err)
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
		return
	}

	version := int(buf[0])

	retChan <- K1elMessage{status: "version", msg: fmt.Sprintf("%d", version), src: "winkeyer"}
	log.Printf("K1EL Version: %d", version)

	// Enable echo
	_, err = port.Write([]byte{0x0e, 0x40})
	if err != nil {
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
		return
	}

	// Use the speed pot for all wpm settings

	_, err = port.Write([]byte{0x02, 0x00})
	if err != nil {
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
		return
	}

	// Make pot speed request
	_, err = port.Write([]byte{0x07})
	if err != nil {
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
	}

	recv := ""

	for {
		wkbyte := []byte{0x00} // Hopefully making this 1byte sets the read size

		bytesRead, err := port.Read(wkbyte)
		if err != nil {
			retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
			return
		}

		if bytesRead == 0 {
			continue
		}

		wk := int(wkbyte[0])

		if (wk & STATUS_MASK) == STATUS_TYPE_STATUS {
			status := wk & STATUS_BITMASK

			if status == STATUS_READY {
				retChan <- K1elMessage{status: "ready", src: "winkeyer"}
			} else if status == STATUS_KEYSTART { // Keying started
				recv = ""
				retChan <- K1elMessage{status: "keystart", src: "winkeyer"}
			} else if status == STATUS_KEYSTOP { // Keying stopped
				retChan <- K1elMessage{status: "keystop", msg: recv, src: "winkeyer"}
				recv = ""
			} else if status == STATUS_BUFFER_READY {
				// buffer ready
			} else {
				fmt.Printf("Unknown status byte received: %x", status)
			}

		} else if (wk & STATUS_MASK) == STATUS_TYPE_POT {
			speed := wk & STATUS_BITMASK
			retChan <- K1elMessage{status: "pot", msg: fmt.Sprintf("%d", speed), src: "winkeyer"}
		} else {
			//fmt.Printf("%c", wkbyte[0])
			retChan <- K1elMessage{status: "echo", msg: fmt.Sprintf("%c", wkbyte[0]), src: "winkeyer"}
			recv = fmt.Sprintf("%s%c", recv, wkbyte[0])
		}

	}

}

func flexSerialWriter(PortName string, k1elChan <-chan K1elMessage, retChan chan<- K1elMessage, doneChan chan<- string) {
	//serialOptions := serial.OpenOptions{
	//	PortName:        "/dev/ttyUSB0",
	//	BaudRate:        1200,
	//	DataBits:        8,
	//	StopBits:        2,
	//	ParityMode:      serial.,
	//	MinimumReadSize: 4,
	//}

	if len(PortName) == 0 {
		retChan <- K1elMessage{status: "serial", msg: "invalid port name", src: "flex"}
		return
	}

	mode := &serial.Mode{
		BaudRate: 1200,
		DataBits: 8,
		StopBits: 2,
		Parity:   serial.NoParity,
	}

	log.Printf("Opening Flex serial port %s", PortName)

	port, err := serial.Open(PortName, mode)
	if err != nil {
		log.Printf("Flex serial.Open: %v", err)
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "flex"}
	}

	defer func() {
		_, err := port.Write([]byte{0x00, 0x03}) // close cmd
		if err != nil {
			log.Printf("Could not send close command to serial port: %v", err)
			return
		}

		err = port.Close()
		if err != nil {
			log.Println("Error closing port", err)
			return
		}
	}()

	_, err = port.Write([]byte{0x00, 0x02}) // Open cmd
	if err != nil {
		log.Printf("Could not initialize flex keyer: %v", err)
	}

	// get the version
	buf := []byte{0x00}

	_, err = port.Read(buf)
	if err != nil {
		log.Printf("Error reading version: %v", err)
		retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "flex"}
	}

	version := int(buf[0])

	//k1elChan <- K1elMessage{status: "version", msg: fmt.Sprintf("%d", version)}
	retChan <- K1elMessage{status: "version", msg: fmt.Sprintf("%d", version), src: "flex"}
	log.Printf("Flex Version: %d", version)

	for msg := range k1elChan {
		switch msg.status {
		case "echo":
			_, err := port.Write([]byte(msg.msg))
			if err != nil {
				retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
				return
			}
			break
		case "pot":
			speed, err := strconv.Atoi(msg.msg)
			if err != nil {
				log.Printf("Could not convert %s to string", msg.msg)
			}

			fmt.Printf("Speed change: %d\n", speed)

			// Change the Flex speed
			cmd := []byte{0x02, 0x00}
			cmd[1] = byte(speed)

			_, err = port.Write(cmd)
			if err != nil {
				retChan <- K1elMessage{status: "serial", msg: fmt.Sprintf("error: %v", err), src: "winkeyer"}
				return
			}
			break
		default:
			break
		}

		retChan <- msg
	}
	doneChan <- "done"
}

func main() {

	var flagWinkeyComPort string
	var flagFlexComPort string
	flag.StringVar(&flagWinkeyComPort, "winkey", "", "COM port for Hardware WinKeyer")
	flag.StringVar(&flagFlexComPort, "flex", "", "COM port for FlexRadio WinKeyer")

	flag.Parse()

	println("command line:", flagWinkeyComPort)
	println("command line:", flagFlexComPort)

	myApp := app.NewWithID("com.k1hyl.gowinkeyflex")

	configWinkeyComPort := myApp.Preferences().String("winkey_com_port")
	configFlexComPort := myApp.Preferences().String("flex_com_port")

	println("Preferences (winkey):", configWinkeyComPort)
	println("Preferences (flex):", configFlexComPort)

	var flexComPort string
	var winkeyComPort string

	// flags win priority
	if len(flagWinkeyComPort) > 0 {
		winkeyComPort = flagWinkeyComPort
	} else if len(configWinkeyComPort) > 0 {
		winkeyComPort = configWinkeyComPort
	}

	if len(flagFlexComPort) > 0 {
		flexComPort = flagFlexComPort
	} else if len(configWinkeyComPort) > 0 {
		flexComPort = configFlexComPort
	}

	if len(flexComPort) == 0 || len(winkeyComPort) == 0 {
		log.Printf("No configured serial port, getting list")
		comPorts, err := serial.GetPortsList()
		if err != nil {
			log.Printf("Could not get list of serial ports")
			return
		}

		for port := range comPorts {
			println(port)
		}
	}

	myWindow := myApp.NewWindow("gowinkeyer by K1HYL")

	flexCard := widget.NewCard("Flex", flexComPort, canvas.NewText("...", color.White))
	winkeyCard := widget.NewCard("WinKeyer", winkeyComPort, canvas.NewText("...", color.White))
	content := container.New(layout.NewHBoxLayout(), flexCard, layout.NewSpacer(), winkeyCard)

	centered := container.New(layout.NewHBoxLayout(), layout.NewSpacer())
	myWindow.SetContent(container.New(layout.NewVBoxLayout(), content, centered))

	retChan := make(chan K1elMessage, 10)
	doneChan := make(chan string, 1)
	guiChan := make(chan K1elMessage, 100)

	if len(winkeyComPort) > 0 {
		go k1elSerialReader(winkeyComPort, retChan)
	}

	if len(flexComPort) > 0 {
		go flexSerialWriter(flexComPort, retChan, guiChan, doneChan)
	}

	go func() {
		for msg := range guiChan {
			fmt.Printf("%s: %s (%s)", msg.status, msg.msg, msg.src)
			if msg.status == "serial" {
				println("Received serial error, popping config", msg.msg, msg.src)
			} else if msg.src == "winkeyer" {
				winkeyCard.SetContent(canvas.NewText(msg.status, color.White))
			} else if msg.src == "flex" {
				flexCard.SetContent(canvas.NewText(msg.status, color.White))
			}
		}
	}()

	myWindow.ShowAndRun()
	//<-doneChan
	os.Exit(0)
}
