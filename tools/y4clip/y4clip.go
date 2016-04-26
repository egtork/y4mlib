package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/egtork/y4mlib"
)

// start frame
// end frame

var (
	inFile       = flag.String("i", "", "input file")
	outFile      = flag.String("o", "", "output file")
	newWidth     = flag.Int("w", -1, "cropped width; -1 for original width")
	newHeight    = flag.Int("h", -1, "cropped height; -1 for original height")
	xOffset      = flag.Int("x", -1, "horizontal offset; -1 to center")
	yOffset      = flag.Int("y", -1, "vertical offset; -1 to center")
	startFrame   = flag.Int("s", 1, "start frame")
	endFrame     = flag.Int("e", -1, "end frame; -1 for last frame of input stream")
	stripHeaders = flag.Bool("strip", false, "strip header information")
)

func main() {
	flag.Parse()
	if *inFile == "" || *outFile == "" {
		flag.Usage()
	}
	sIn, err := y4m.Open(*inFile)
	checkErr(err)
	err = setAndCheckUserInputs(sIn)
	checkErr(err)
	sOut, err := y4m.NewStream(*outFile, *newWidth, *newHeight)
	checkErr(err)
	defer sOut.Close()
	sOut.Chroma = sIn.Chroma
	sOut.FrameRate = sIn.FrameRate
	sOut.Interlacing = sIn.Interlacing
	sOut.Metadata = sIn.Metadata
	sOut.SampleAspectRatio = sIn.SampleAspectRatio
	sOut.XSubsamplingFactor = sIn.XSubsamplingFactor
	sOut.YSubsamplingFactor = sIn.YSubsamplingFactor
	if !*stripHeaders {
		err = sOut.WriteHeader()
		checkErr(err)
	}
	// skip frames
	for k := 1; k < *startFrame; k++ {
		err := sIn.SkipFrame()
		checkErr(err)
	}
	// copy frames
	for k := *startFrame; *endFrame == -1 || k <= *endFrame; k++ {
		frame, err := sIn.ParseFrame()
		if err == io.EOF && *endFrame == -1 {
			break
		}
		checkErr(err)
		if sOut.Height != sIn.Height || sOut.Width != sIn.Width {
			frame.Crop(*newWidth, *newHeight, *xOffset, *yOffset)
		}
		if !*stripHeaders {
			err = sOut.WriteFrameHeader(frame)
			checkErr(err)
		}
		err = sOut.WriteFrameData(frame)
		checkErr(err)
	}
	err = sOut.Sync()
	checkErr(err)
}

func setAndCheckUserInputs(s *y4m.Stream) error {
	if *startFrame < 1 {
		return fmt.Errorf("start frame must be greater than 0")
	}
	if *endFrame == -1 {
		// do nothing
	} else if *endFrame < 1 {
		return fmt.Errorf("end frame must be -1 or greater than 0")
	}
	if *newWidth == -1 {
		*newWidth = s.Width
	} else if *newWidth < 1 {
		return fmt.Errorf("cropped width must be -1 or greater than 0")
	} else if *newWidth > s.Width {
		return fmt.Errorf("cropped width cannot exceed original width (%d)", s.Width)
	} else if *newWidth%s.XSubsamplingFactor != 0 {
		return fmt.Errorf("choose width as a multiple of %d to accomodate chroma subsampling",
			s.XSubsamplingFactor)
	}
	if *newHeight == -1 {
		*newHeight = s.Height
	} else if *newHeight < 1 {
		return fmt.Errorf("cropped height must be -1 or greater than 0")
	} else if *newHeight > s.Height {
		return fmt.Errorf("cropped height cannot exceed original height (%d)", s.Height)
	} else if *newHeight%s.YSubsamplingFactor != 0 {
		return fmt.Errorf("choose height as a multiple of %d to accomodate chroma subsampling",
			s.YSubsamplingFactor)
	}
	if *xOffset == -1 {
		*xOffset = s.XSubsamplingFactor * (s.Width - *newWidth) / 2 / s.XSubsamplingFactor
	}
	if *xOffset+*newWidth > s.Width {
		return fmt.Errorf("horizontal offset + cropped width cannot exceed original width (%d)", s.Width)
	}
	if *yOffset == -1 {
		*yOffset = s.YSubsamplingFactor * ((s.Height - *newHeight) / 2 / s.YSubsamplingFactor)
	}
	if *yOffset+*newHeight > s.Height {
		return fmt.Errorf("vertical offset + cropped height cannot exceed original height (%d)", s.Height)
	}
	return nil
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
