package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/kirsle/configdir"
	"go.bug.st/serial"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"
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

type AppSettings struct {
	WinkeyPort string `json:"winkey_port"`
	FlexPort   string `json:"flex_port"`
}

func main() {

	configPath := configdir.LocalConfig("gowinkeyflex")
	err := configdir.MakePath(configPath)
	if err != nil {
		log.Printf("Could not make config path: %v", err)
		return
	}

	configFile := filepath.Join(configPath, "settings.json")
	var settings AppSettings

	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		WriteConfig(configFile, &settings)
	} else {
		ReadConfig(configFile, &settings)
	}

	log.Printf("Settings Flex: %s Winkey: %s", settings.FlexPort, settings.WinkeyPort)

	var flagWinkeyComPort string
	var flagFlexComPort string
	flag.StringVar(&flagWinkeyComPort, "winkey", "", "COM port for Hardware WinKeyer")
	flag.StringVar(&flagFlexComPort, "flex", "", "COM port for FlexRadio WinKeyer")

	flag.Parse()

	println("command line:", flagWinkeyComPort)
	println("command line:", flagFlexComPort)

	myApp := app.NewWithID("com.k1hyl.gowinkeyflex")

	//configWinkeyComPort := myApp.Preferences().String("winkey_com_port")
	//configFlexComPort := myApp.Preferences().String("flex_com_port")
	//
	//println("Preferences (winkey):", configWinkeyComPort)
	//println("Preferences (flex):", configFlexComPort)

	flexComPort := settings.FlexPort
	winkeyComPort := settings.WinkeyPort

	retChan := make(chan K1elMessage, 10)
	doneChan := make(chan string, 1)
	guiChan := make(chan K1elMessage, 100)

	comPortsList, err := serial.GetPortsList()
	if err != nil {
		log.Printf("Could not get list of serial ports")
		return
	}

	comPorts := []string{}

	for _, port := range comPortsList {
		comPorts = append(comPorts, port)
	}

	myWindow := myApp.NewWindow("gowinkeyer by K1HYL")

	flexCard := widget.NewCard("Flex", flexComPort, canvas.NewText("...", color.Black))
	winkeyCard := widget.NewCard("WinKeyer", winkeyComPort, canvas.NewText("...", color.Black))
	status := container.New(
		layout.NewVBoxLayout(),
		widget.NewButton("Start", func() {
			log.Printf("Starting Winkeyer: %s", settings.WinkeyPort)
			go k1elSerialReader(settings.WinkeyPort, retChan)
			log.Printf("Starting Flexkeyer: %s", settings.FlexPort)
			go flexSerialWriter(settings.FlexPort, retChan, guiChan, doneChan)
		}),
		widget.NewButton("Config", func() {
			configWindow := myApp.NewWindow("Configuration")

			configContainer := container.New(
				layout.NewVBoxLayout(),
				canvas.NewText("Configuration", color.Black),
				container.NewHBox(
					canvas.NewText("Flexradio Serial Port:", color.Black),
					widget.NewSelect(comPorts, func(value string) {
						log.Printf("Setting flex_com_port to %s", value)
						settings.FlexPort = value
						WriteConfig(configFile, &settings)
					})),
				container.NewHBox(
					canvas.NewText("WinKeyer Serial Port:", color.Black),
					widget.NewSelect(comPorts, func(value string) {
						log.Printf("Setting winkey_com_port to %s", value)
						settings.WinkeyPort = value
						WriteConfig(configFile, &settings)
					})))

			//configContainer := container.New(
			//	layout.NewHBoxLayout(),
			//	canvas.NewText("Flex Serial Port", color.Black),
			//	widget.NewSelect(comPorts, func(value string) {
			//		log.Printf("Selected %s", value)
			//	}),
			//)

			configWindow.SetContent(configContainer)

			configWindow.Show()
		}),
		widget.NewButton("Quit", func() {
			myApp.Quit()
		}),
	)
	content := container.New(layout.NewHBoxLayout(), flexCard, status, winkeyCard)

	logArea := widget.NewMultiLineEntry()
	logArea.Disable()
	logArea.SetText("Welcome to gowinkeyflex by K1HYL\nPress the config button if needed")

	myWindow.SetContent(container.New(layout.NewVBoxLayout(), content, logArea))

	go func() {
		for msg := range guiChan {
			fmt.Printf("%s: %s (%s)\n", msg.status, msg.msg, msg.src)

			switch msg.status {
			case "serial":
				logArea.SetText(logArea.Text + "\n" + "serial error: " + msg.msg + " " + msg.src + "\n")
				println("Received serial error", msg.msg, msg.src)
				break
			case "echo":
				logArea.SetText(logArea.Text + msg.msg)
				break
			case "keystop":
				//logArea.SetText(logArea.Text + "\n")
				break
			case "keystart":
				if strings.HasPrefix(logArea.Text, "Welcome to gowinkeyflex") {
					logArea.SetText("")
				}
				break
			default:
				log.Printf("Unknown status message: %s", msg.status)
			}

			if msg.src == "winkeyer" {
				winkeyCard.SetContent(canvas.NewText(fmt.Sprintf("%s: %s", msg.status, msg.msg), color.Black))
			} else if msg.src == "flex" {
				flexCard.SetContent(canvas.NewText(fmt.Sprintf("%s: %s", msg.status, msg.msg), color.Black))
			}
		}
	}()

	myWindow.ShowAndRun()
	//<-doneChan
	os.Exit(0)
}

func ReadConfig(configFile string, settings *AppSettings) {
	// Load the existing file
	fh, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Could not open %s", configFile)
	}
	defer fh.Close()

	decoder := json.NewDecoder(fh)
	err = decoder.Decode(settings)
	if err != nil {
		log.Fatalf("Could not decode %s: %v", configFile, err)
	}
}

func WriteConfig(configFile string, settings *AppSettings) {
	// Create the config file

	fh, err := os.Create(configFile)
	if err != nil {
		log.Fatalf("Could not create %s", configFile)
	}

	defer fh.Close()

	decoder := json.NewEncoder(fh)
	err = decoder.Encode(settings)
	if err != nil {
		log.Printf("Encoder error: %v", err)
	}

}
