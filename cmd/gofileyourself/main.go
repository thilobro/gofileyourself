package main

import (
	"flag"
	"gofileyourself/internal/explorer"
	"io"
	"log"
	"os"
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

	fe, err := explorer.NewFileExplorer()
	if err != nil {
		panic(err)
	}

	if err := fe.Run(); err != nil {
		panic(err)
	}
}
