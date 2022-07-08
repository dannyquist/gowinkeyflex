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
	"time"
)

type K1elMessage struct {
	status string
	msg    string
	src    string
}

func k1elSerialReader(PortName string, retChan chan<- K1elMessage) {

	//serialOptions := serial.OpenOptions{
	//	PortName:        "/dev/ttyUSB0",
	//	BaudRate:        1200,
	//	DataBits:        8,
	//	StopBits:        2,
	//	ParityMode:      serial.,
	//	MinimumReadSize: 4,
	//}

	mode := &serial.Mode{
		BaudRate: 1200,
		DataBits: 8,
		StopBits: 2,
		Parity:   serial.NoParity,
	}

	log.Printf("Opening Winkeyer serial port %s", PortName)

	port, err := serial.Open(PortName, mode)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
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
		}
	}()

	_, err = port.Write([]byte{0x00, 0x02}) // Open cmd
	if err != nil {
		log.Fatalf("Could not initialize K1EL keyer: %v", err)
	}
	time.Sleep(1 * time.Second)

	// get the version
	buf := []byte{0x00}

	_, err = port.Read(buf)
	if err != nil {
		log.Fatalf("Error reading version: %v", err)
	}

	version := int(buf[0])

	retChan <- K1elMessage{status: "version", msg: fmt.Sprintf("%d", version), src: "winkeyer"}
	log.Printf("K1EL Version: %d", version)

	// Enable echo
	_, err = port.Write([]byte{0x0e, 0x40})
	if err != nil {
		log.Fatalf("Error writing to serial port: %v", err)
	}

	// Use the speed pot for all wpm settings

	_, err = port.Write([]byte{0x02, 0x00})
	if err != nil {
		log.Fatal(err)
	}

	// Make pot speed request
	_, err = port.Write([]byte{0x07})
	if err != nil {
		log.Fatalf("Could not write to serial port: %v", err)
	}

	recv := ""

	for {
		wkbyte := []byte{0x00} // Hopefully making this 1byte sets the read size

		bytesRead, err := port.Read(wkbyte)
		if err != nil {
			log.Fatalf("Could not read from serial port: %v", err)
		}

		if bytesRead == 0 {
			continue
		}

		wk := int(wkbyte[0])

		if (wk & 0xc0) == 0xc0 {
			status := wk & 0x3f

			if status == 8 {
				retChan <- K1elMessage{status: "ready", src: "winkeyer"}
			} else if status == 6 { // Keying started
				recv = ""
				retChan <- K1elMessage{status: "keystart", src: "winkeyer"}
			} else if status == 4 { // Keying stopped
				retChan <- K1elMessage{status: "keystop", msg: recv, src: "winkeyer"}
				recv = ""
			} else if status == 0 {
				// buffer ready
			} else {
				fmt.Printf("Unknown status byte received: %x", status)
			}

		} else if (wk & 0xc0) == 0x80 {
			speed := wk & 0x3f
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

	mode := &serial.Mode{
		BaudRate: 1200,
		DataBits: 8,
		StopBits: 2,
		Parity:   serial.NoParity,
	}

	log.Printf("Opening Flex serial port %s", PortName)

	port, err := serial.Open(PortName, mode)
	if err != nil {
		log.Fatalf("Flex serial.Open: %v", err)
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
		}
	}()

	_, err = port.Write([]byte{0x00, 0x02}) // Open cmd
	if err != nil {
		log.Fatalf("Could not initialize K1EL keyer: %v", err)
	}

	// get the version
	buf := []byte{0x00}

	_, err = port.Read(buf)
	if err != nil {
		log.Fatalf("Error reading version: %v", err)
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
				log.Fatalf("Could not write to flex serial: %v", err)
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
				log.Fatalf("Could not write to flex keyer: %v", err)
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
	var k1elComPort string
	var flexComPort string
	flag.StringVar(&k1elComPort, "k1el", "COM3", "COM port for K1EL WinKeyer")
	flag.StringVar(&flexComPort, "flex", "COM11", "COM port for FlexRadio WinKeyer")

	flag.Parse()

	myApp := app.New()
	myWindow := myApp.NewWindow("Box Layout")

	flexCard := widget.NewCard("Flex", flexComPort, canvas.NewText("STATUS GOES HERE", color.White))
	winkeyCard := widget.NewCard("WinKeyer", k1elComPort, canvas.NewText("STATUS GOES HERE", color.White))
	content := container.New(layout.NewHBoxLayout(), flexCard, layout.NewSpacer(), winkeyCard)

	centered := container.New(layout.NewHBoxLayout(), layout.NewSpacer())
	myWindow.SetContent(container.New(layout.NewVBoxLayout(), content, centered))

	//go func() {
	//	time.Sleep(5 * time.Second)
	//	println("Trying to update content...")
	//	flexCard.SetContent(canvas.NewText("BOINK", color.White))
	//}()

	retChan := make(chan K1elMessage, 10)
	doneChan := make(chan string, 1)
	guiChan := make(chan K1elMessage, 100)

	go k1elSerialReader(k1elComPort, retChan)
	go flexSerialWriter(flexComPort, retChan, guiChan, doneChan)
	go func() {
		for msg := range guiChan {
			println("%s: %s (%s)", msg.status, msg.msg, msg.src)
			if msg.src == "winkeyer" {
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
