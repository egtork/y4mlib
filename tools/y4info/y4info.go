package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egtork/y4mlib"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: y4info file")
		os.Exit(1)
	}
	s, err := y4m.Open(os.Args[1])
	checkErr(err)
	defer s.Close()
	s.PrintHeaderInfo()
	nFrames, err := s.CountFrames()
	checkErr(err)
	fmt.Printf("Frames:\n  %d\n", nFrames)
	if s.FrameRate.D == 0 {
		fmt.Printf("Duration:\n  unknown (frame rate not specified)")
	} else {
		rate := float64(s.FrameRate.N) / float64(s.FrameRate.D)
		durationSeconds := float64(nFrames) / rate
		durationString := fmt.Sprintf("%.6fs", durationSeconds)
		d, err := time.ParseDuration(durationString)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Duration:\n  %s\n", d.String())
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
