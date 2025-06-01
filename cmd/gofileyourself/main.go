package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/thilobro/gofileyourself/internal/display"
	"github.com/thilobro/gofileyourself/internal/explorer"
	"github.com/thilobro/gofileyourself/internal/finder"
	"github.com/thilobro/gofileyourself/internal/widget"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	cfp := flag.String("choosefiles", "", "Use as a file chooser")
	sf := flag.String("selectfile", "", "The file that was selected")

	flag.Parse()
	var chooseFilePath *string
	if *cfp != "" {
		chooseFilePath = cfp
	} else {
		chooseFilePath = nil
	}
	var selectedFilePath *string
	if *sf != "" {
		selectedFilePath = sf
	} else {
		selectedFilePath = nil
	}

	if *debug {
		logFile, err := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}

		// Configure log package to write to file
		log.SetOutput(logFile)
		// Optional: include file and line number in logs
		log.SetFlags(log.Ltime | log.Lshortfile)
	} else {
		log.SetOutput(io.Discard)
	}
	//
	// Set up the factories for each mode
	factories := map[widget.Mode]widget.Factory{
		widget.Explorer: &explorer.Factory{},
		widget.Find:     &finder.Factory{},
	}
	log.Println("Selected file path: ", selectedFilePath)

	display, err := display.NewDisplay(factories, chooseFilePath, selectedFilePath)
	if err != nil {
		panic(err)
	}

	if err := display.Run(); err != nil {
		panic(err)
	}
}
