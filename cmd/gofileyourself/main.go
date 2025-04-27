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
	flag.Parse()

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
	factories := map[display.Mode]widget.Factory{
		display.Explorer: &explorer.Factory{},
		display.Find:     &finder.Factory{},
	}

	display, err := display.NewDisplay(factories)
	if err != nil {
		panic(err)
	}

	if err := display.Run(); err != nil {
		panic(err)
	}
}
