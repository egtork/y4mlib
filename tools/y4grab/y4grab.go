package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/tiff"

	"github.com/egtork/y4mlib"
)

var inputFile = flag.String("i", "", "input filename")
var outputFile = flag.String("o", "", "output filename")
var format = flag.String("f", "jpeg", "image format {\"jpeg\", \"png\", \"tiff\"}")
var startFrame = flag.Int("s", 1, "start frame")
var frameCount = flag.Int("n", 1, "number of frames to grab")
var jpegQuality = flag.Int("jq", 75, "(JPEG only) quality [0-100]")
var compressTIFF = flag.Bool("tc", false, "(TIFF only) use deflate compression")
var predictorTIFF = flag.Bool("tp", false, "(TIFF only) use differencing predictor")

func main() {
	flag.Parse()
	if *inputFile == "" {
		flag.Usage()
		os.Exit(0)
	}
	// Open file
	s, err := y4m.Open(*inputFile)
	checkErr(err)
	defer s.Close()
	// Skip frames
	for k := 1; k < *startFrame; k++ {
		err := s.SkipFrame()
		checkErr(err)
	}
	// Grab frames
	name := filenameFormat(*inputFile, *outputFile)
	for k := 0; k < *frameCount; k++ {
		frame, err := s.ParseFrame()
		if err == io.EOF {
			checkErr(fmt.Errorf("Reached end of stream at frame %d. %d of %d frames grabbed.",
				*startFrame+k-1, k, *frameCount))
		} else {
			checkErr(err)
		}
		img := frame.Image()
		err = writeFile(img, name, *startFrame+k)
		checkErr(err)
	}
}

func filenameFormat(in, out string) string {
	var filePrefix, fileSuffix string
	if out == "" {
		// Use input file to derive output filename
		extensions := map[string]string{
			"jpeg": "jpg",
			"tiff": "tif",
			"png":  "png",
		}
		basename := filepath.Base(in)
		fileSuffix = "." + extensions[strings.ToLower(*format)]
		filePrefix = strings.TrimSuffix(basename, filepath.Ext(basename))
	} else {
		dir, file := filepath.Split(out)
		fileSuffix = filepath.Ext(file)
		filePrefix = dir + strings.TrimSuffix(file, fileSuffix)
	}
	var formatString string
	if *frameCount == 1 {
		formatString = filePrefix + fileSuffix
	} else {
		leadingZeros := int(math.Log10(float64(*startFrame+*frameCount))) + 1
		formatString = filePrefix + "%0" + strconv.Itoa(leadingZeros) + "d" + fileSuffix
	}
	return formatString
}

func writeFile(img image.Image, filenameFormat string, idx int) error {
	var f *os.File
	var err error
	if *frameCount > 1 {
		f, err = os.Create(fmt.Sprintf(filenameFormat, idx))
	} else {
		f, err = os.Create(filenameFormat)
	}
	if err != nil {
		return err
	}
	defer f.Close()
	switch *format {
	case "jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: *jpegQuality})
	case "png":
		err = png.Encode(f, img)
	case "tiff":
		compressionType := tiff.Uncompressed
		if *compressTIFF {
			compressionType = tiff.Deflate
		}
		options := &tiff.Options{
			Compression: compressionType,
			Predictor:   *predictorTIFF,
		}
		err = tiff.Encode(f, img, options)
	default:
		log.Fatalf("Unrecognized image format -- %s\n", *format)
	}
	return err
}

func checkErr(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
}
