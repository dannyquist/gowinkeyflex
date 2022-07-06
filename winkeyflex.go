package main

import (
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"log"
	"time"
)

func k1elSerialReader(PortName string) {
	serialOptions := serial.OpenOptions{
		PortName:        "/dev/ttyUSB0",
		BaudRate:        1200,
		DataBits:        8,
		StopBits:        2,
		ParityMode:      serial.PARITY_NONE,
		MinimumReadSize: 4,
	}

	port, err := serial.Open(serialOptions)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	defer port.Close()

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
	log.Printf("Version: %d", version)

	// Enable echo
	_, err = port.Write([]byte{0x0e, 0x40})
	if err != nil {
		log.Fatalf("Error writing to serial port: %v", err)
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
				fmt.Printf("\nReceived ready state from keyer\n")
			} else if status == 6 { // Keying started
				recv = ""
			} else if status == 4 { // Keying stopped
				fmt.Printf("\nReceived complete buffer: %s\n", recv)
				recv = ""
			} else if status == 0 {
				// buffer ready
			} else {
				fmt.Printf("Unknown status byte received: %x", status)
			}

		} else if (wk & 0xc0) == 0x80 {
			speed := wk & 0x3f
			println("Pot knob changed:", speed)
		} else {
			fmt.Printf("%c", wkbyte[0])
			recv = fmt.Sprintf("%s%c", recv, wkbyte[0])
		}

	}

}

func main() {

	k1elSerialReader("/dev/ttyUSB0")

}
