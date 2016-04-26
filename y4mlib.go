package y4m

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	streamMagicString = "YUV4MPEG2"
)

var (
	// ErrInvalidFormat can occur if file does not begin with YUV4MPEG2 signature
	ErrInvalidFormat = errors.New("not a valid YUV4MPEG stream")
)

// Stream represents a Y4M uncompressed video stream
type Stream struct {
	file               *os.File
	Width              int
	Height             int
	FrameRate          *Ratio
	Interlacing        string
	SampleAspectRatio  *Ratio
	Chroma             string
	Metadata           []string
	XSubsamplingFactor int
	YSubsamplingFactor int
	OriginalHeader     []byte
}

// Frame represents a YCbCr frame with an optional Alpha plane
type Frame struct {
	Header *FrameHeader
	Width  int
	Height int
	Chroma string
	Y      []byte
	Cb     []byte
	Cr     []byte
	Alpha  []byte
}

// FrameHeader represents a Y4M frame header.
type FrameHeader struct {
	MagicString string
	I           *IField
	Metadata    []string
	Raw         []byte
}

// IField contains the values associated with a frame header's I field
type IField struct {
	Spatial      byte
	Temporal     byte
	Presentation byte
}

// Ratio has a numerator and denomator
type Ratio struct {
	N int
	D int
}

var xSubsamplingFactor = map[string]int{
	"444":      1,
	"422":      2,
	"411":      4,
	"420jpeg":  2,
	"420mpeg2": 2,
	"420paldv": 2,
}

var ySubsamplingFactor = map[string]int{
	"444":      1,
	"422":      1,
	"411":      1,
	"420jpeg":  2,
	"420mpeg2": 2,
	"420paldv": 2,
}

// Open opens a named file for reading and parses the header.
func Open(name string) (*Stream, error) {
	var err error
	s := new(Stream)
	s.file, err = os.Open(name)
	if err != nil {
		return nil, err
	}
	err = s.IsY4M()
	if err != nil {
		return nil, err
	}
	err = s.ParseHeader()
	if err != nil {
		return nil, err
	}
	s.XSubsamplingFactor = xSubsamplingFactor[s.Chroma]
	s.YSubsamplingFactor = ySubsamplingFactor[s.Chroma]
	return s, nil
}

// IsY4M checks that the stream begins with "YUV4MPEG".
func (s *Stream) IsY4M() error {
	sb := make([]byte, len(streamMagicString))
	_, err := s.file.Read(sb)
	if err != nil {
		return err
	}
	if string(sb) != streamMagicString {
		return ErrInvalidFormat
	}
	_, err = s.file.Seek(0, 0)
	return err
}

// ParseHeader parses a Y4M stream header and stores the parsed information in the
// fields of stream s. The file read offset will be set to the end of the header.
func (s *Stream) ParseHeader() error {
	_, err := s.file.Seek(0, 0)
	r := bufio.NewReader(s.file)
	b, err := r.ReadBytes('\n')
	if err != nil {
		return err
	}
	// Store header byte sequence
	s.OriginalHeader = b
	// Set defaults
	s.Chroma = "420jpeg"
	s.Interlacing = "?"
	s.FrameRate = &Ratio{0, 0}
	s.SampleAspectRatio = &Ratio{0, 0}
	fields := bytes.Fields(b)
	for k := 0; k < len(fields); k++ {
		field := string(fields[k])
		key := field[0]
		val := field[1:]
		switch key {
		case 'Y':
			// do nothing
		case 'W':
			s.Width, err = strconv.Atoi(val)
			if err != nil {
				return err
			}
		case 'H':
			s.Height, err = strconv.Atoi(val)
			if err != nil {
				return err
			}
		case 'F':
			ratio, err := stringToRatio(val)
			if err != nil {
				return err
			}
			s.FrameRate = ratio
		case 'I':
			s.Interlacing = val
		case 'A':
			ratio, err := stringToRatio(val)
			if err != nil {
				return err
			}
			s.SampleAspectRatio = ratio
		case 'C':
			s.Chroma = val
		case 'X':
			s.Metadata = append(s.Metadata, val)
		default:
			return fmt.Errorf("Unrecognized stream header field: %c\n", key)
		}
	}
	// Seek to end of header
	_, err = s.file.Seek(int64(len(s.OriginalHeader)), 0)
	if err != nil {
		return nil
	}
	return nil
}

// Header generates a header byte sequence. It may not be identical to the stream's
// original header, since all fields are explicitly populated with default values.
func (s *Stream) Header() []byte {
	b := []byte(streamMagicString)
	b = append(b, []byte(fmt.Sprintf(" W%d", s.Width))...)
	b = append(b, []byte(fmt.Sprintf(" H%d", s.Height))...)
	b = append(b, []byte(fmt.Sprintf(" C%s", s.Chroma))...)
	b = append(b, []byte(fmt.Sprintf(" I%s", s.Interlacing))...)
	b = append(b, []byte(fmt.Sprintf(" F%v", s.FrameRate))...)
	b = append(b, []byte(fmt.Sprintf(" A%v", s.SampleAspectRatio))...)
	for k := 0; k < len(s.Metadata); k++ {
		b = append(b, []byte(fmt.Sprintf(" X%s", s.Metadata[k]))...)
	}
	b = append(b, byte('\n'))
	return b
}

// stringToRatio parses string in format "N:D" as ratio.
func stringToRatio(s string) (*Ratio, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, errors.New("Could not parse string as ratio")
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}
	d, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &Ratio{N: n, D: d}, nil
}

func (r *Ratio) String() string {
	return fmt.Sprintf("%d:%d", r.N, r.D)
}

// ToFirstFrame sets the read offset of the stream file to the beginning of the first frame.
func (s *Stream) ToFirstFrame() error {
	_, err := s.file.Seek(0, 0)
	if err != nil {
		return err
	}
	r := bufio.NewReader(s.file)
	_, err = r.ReadBytes('\x0a')
	if err != nil {
		return err
	}
	_, err = s.file.Seek(-int64(r.Buffered()), 1)
	return err
}

// SkipFrame skips to the next frame without parsing or storing data.
func (s *Stream) SkipFrame() error {
	err := s.SkipFrameHeader()
	if err != nil {
		return err
	}
	_, err = s.file.Seek(s.FrameImageDataSize(), 1)
	return err
}

// SkipFrameHeader skips past a frame header.
func (s *Stream) SkipFrameHeader() error {
	r := bufio.NewReader(s.file)
	b, err := r.ReadBytes('\x0a')
	if err != nil {
		return err
	}
	magicString := string(b[0:5])
	if magicString != "FRAME" {
		return fmt.Errorf("Did not find expected string \"FRAME\" at start of frame header, found \"%s\"\n", string(b[0:15]))
	}
	_, err = s.file.Seek(-int64(r.Buffered()), 1)
	return err
}

// ParseFrame parses frame header and planar image data and returns a Frame.
func (s *Stream) ParseFrame() (*Frame, error) {
	var err error
	frame := new(Frame)
	frame.Header, err = s.ParseFrameHeader()
	if err != nil {
		return nil, err
	}
	frame.Y, err = s.grabPlane(s.LumaPlaneSize())
	if err != nil {
		return nil, err
	}
	frame.Cb, err = s.grabPlane(s.ChromaPlaneSize())
	if err != nil {
		return nil, err
	}
	frame.Cr, err = s.grabPlane(s.ChromaPlaneSize())
	if err != nil {
		return nil, err
	}
	frame.Alpha, err = s.grabPlane(s.AlphaPlaneSize())
	if err != nil {
		return nil, err
	}
	frame.Width = s.Width
	frame.Height = s.Height
	frame.Chroma = s.Chroma
	return frame, nil
}

// ParseFrameHeader parses a frame header. A frame header consists of magic string "FRAME",
// any number of tagged fields preceded by ' ' separator, and '\n'.
func (s *Stream) ParseFrameHeader() (*FrameHeader, error) {
	h := new(FrameHeader)
	r := bufio.NewReader(s.file)
	hs, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	h.Raw = hs
	hf := bytes.Fields(hs)
	if len(hf) < 1 {
		return nil, errors.New("Could not parse frame header")
	}
	magicString := string(hf[0])
	if magicString == "FRAME" {
		h.MagicString = magicString
	} else {
		return nil, errors.New("Did not find expected magic string \"FRAME\" when parsing frame header")
	}
	for k := 1; k < len(hf); k++ {
		field := string(hf[k])
		key := field[0]
		val := field[1:]
		switch key {
		case 'I':
			if len(val) != 3 {
				return nil, errors.New("Frame framing/sampling field does not have expected length of 3")
			}
			x := val[0]
			if x != 't' && x != 'T' && x != 'b' && x != 'B' && x != '1' && x != '2' && x != '3' {
				return nil, fmt.Errorf("Frame presentation subfield has unexpected value %c\n", x)
			}
			y := val[1]
			if y != 'p' && y != 'i' {
				return nil, fmt.Errorf("Frame temporal sampling subfield has unexpected value %c\n", y)
			}
			z := val[2]
			if z != 'p' && z != 'i' && z != '?' {
				return nil, fmt.Errorf("Frame spatial sampling subfield has unexpected value %c\n", z)
			}
			h.I = &IField{Spatial: z, Temporal: y, Presentation: x}
		case 'X':
			h.Metadata = append(h.Metadata, val)
		}
	}
	_, err = s.file.Seek(-int64(r.Buffered()), 1)
	return h, nil
}

func (s *Stream) grabPlane(size int) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	plane := make([]byte, size)
	_, err := io.ReadFull(s.file, plane)
	if err != nil {
		return nil, err
	}
	return plane, nil
}

// LumaPlaneSize returns the size of the luma plane in octets.
func (s *Stream) LumaPlaneSize() int {
	return s.Height * s.Width
}

// ChromaPlaneSize returns the size of a single chroma plane in octets.
func (s *Stream) ChromaPlaneSize() int {
	if s.Chroma == "mono" {
		return 0
	}
	return s.Width * s.Height / s.XSubsamplingFactor / s.YSubsamplingFactor
}

// AlphaPlaneSize returns the size of the alpha plane in octets.
func (s *Stream) AlphaPlaneSize() int {
	if s.Chroma == "444alpha" {
		return s.Width * s.Height
	}
	return 0
}

// CountFrames counts the number of frames in the stream.
func (s *Stream) CountFrames() (int, error) {
	initPos, err := s.file.Seek(0, 1)
	if err != nil {
		return -1, err
	}
	_, err = s.file.Seek(0, 0)
	if err != nil {
		return -1, err
	}
	err = s.ToFirstFrame()
	if err != nil {
		return -1, err
	}
	frameCounter := 0
	for {
		err := s.SkipFrame()
		if err == io.EOF {
			break
		} else if err != nil {
			return -1, err
		}
		frameCounter++
	}
	_, err = s.file.Seek(initPos, 0)
	if err != nil {
		return -1, err
	}
	return frameCounter, nil
}

// FrameImageDataSize returns the total number of octets of planar image data per frame
func (s *Stream) FrameImageDataSize() int64 {
	return int64(s.LumaPlaneSize() + 2*s.ChromaPlaneSize() + s.AlphaPlaneSize())
}

// Crop crops the frame image to width w and height h, horizontally offset from the top-left of
// the original frame by xOffset, and vertically offset by yOffset. The frame's w and h
// fields are updated.
func (f *Frame) Crop(w, h, xOffset, yOffset int) error {
	if w+xOffset > f.Width {
		return fmt.Errorf("cropped width + x offset (%d) cannot exceed original width (%d)",
			w+xOffset, f.Width)
	}
	if h+yOffset > f.Height {
		return fmt.Errorf("cropped height + y offset (%d) cannot exceed original height (%d)",
			h+yOffset, f.Height)
	}
	newY := make([]byte, 0, w*h)
	for y := 0; y < h; y++ {
		yt := y + yOffset
		x0 := yt*f.Width + xOffset
		x1 := x0 + w
		newY = append(newY, f.Y[x0:x1]...)
	}
	f.Y = newY
	xss := xSubsamplingFactor[f.Chroma]
	yss := ySubsamplingFactor[f.Chroma]
	newCb := make([]byte, 0, w/xss*h/yss)
	newCr := make([]byte, 0, w/xss*h/yss)
	for y := 0; y < h/yss; y++ {
		yt := y + yOffset/yss
		x0 := yt*f.Width/xss + xOffset/xss
		x1 := x0 + w/xss
		newCb = append(newCb, f.Cb[x0:x1]...)
		newCr = append(newCr, f.Cr[x0:x1]...)
	}
	f.Cb = newCb
	f.Cr = newCr
	if len(f.Alpha) > 0 {
		newAlpha := make([]byte, 0, w*h)
		for y := 0; y < h; y++ {
			yt := y + yOffset
			x0 := yt*f.Width + xOffset
			x1 := x0 + w
			newAlpha = append(newAlpha, f.Alpha[x0:x1]...)
		}
		f.Alpha = newAlpha
	}
	f.Width = w
	f.Height = h
	return nil
}

// Image converts the frame planar image data into a YCbCr image. In the case that alpha
// plane is present, an NYCbCrA image is created.
func (f *Frame) Image() image.Image {
	var ssr image.YCbCrSubsampleRatio
	switch f.Chroma {
	case "444", "444alpha":
		ssr = image.YCbCrSubsampleRatio444
	case "422":
		ssr = image.YCbCrSubsampleRatio422
	case "420jpeg", "420mpeg2", "420paldv":
		ssr = image.YCbCrSubsampleRatio420
	case "411":
		ssr = image.YCbCrSubsampleRatio411
	case "mono":
		log.Fatal("Mono images should be handled by another function")
	}
	r := image.Rect(0, 0, f.Width, f.Height)
	if len(f.Alpha) > 0 {
		img := image.NewNYCbCrA(r, ssr)
		img.YCbCr.Y = f.Y
		img.YCbCr.Cb = f.Cb
		img.YCbCr.Cr = f.Cr
		img.A = f.Alpha
		return img
	} else if f.Chroma == "mono" {
		img := image.NewGray(r)
		img.Pix = f.Y
		return img
	} else {
		img := image.NewYCbCr(r, ssr)
		img.Y = f.Y
		img.Cb = f.Cb
		img.Cr = f.Cr
		return img
	}
}

// PrintHeaderInfo prints header info to stdout.
func (s *Stream) PrintHeaderInfo() {
	fmt.Println("Stream header information:")
	fmt.Printf("  Width: %d\n", s.Width)
	fmt.Printf("  Height: %d\n", s.Height)
	fmt.Printf("  Frame rate: %v\n", s.FrameRate)
	fmt.Printf("  Interlacing: %s\n", s.Interlacing)
	fmt.Printf("  SampleAspectRatio: %v\n", s.SampleAspectRatio)
	fmt.Printf("  Chroma: %s\n", s.Chroma)
	fmt.Printf("  Metadata: %v\n", s.Metadata)
}

// NewStream creates a new named stream file with width w and height h. The stream file can be
// synced with the Sync method and closed with the Close method.
func NewStream(name string, w, h int) (*Stream, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	s := new(Stream)
	s.file = f
	s.Width = w
	s.Height = h
	return s, nil
}

// WriteHeader writes a stream header byte sequence to the file stream
func (s *Stream) WriteHeader() error {
	h := s.Header()
	_, err := s.file.Write(h)
	return err
}

// WriteFrameHeader writes a frame header byte sequence to the file stream
func (s *Stream) WriteFrameHeader(frame *Frame) error {
	_, err := s.file.Write(frame.Header.Raw)
	return err
}

// WriteFrameData writes planar video data to the file stream
func (s *Stream) WriteFrameData(frame *Frame) error {
	_, err := s.file.Write(frame.Y)
	if err != nil {
		return err
	}
	_, err = s.file.Write(frame.Cb)
	if err != nil {
		return err
	}
	_, err = s.file.Write(frame.Cr)
	if err != nil {
		return err
	}
	_, err = s.file.Write(frame.Alpha)
	if err != nil {
		return err
	}
	return nil
}

// Sync commits the current contents of the stream file to stable storage
func (s *Stream) Sync() error {
	return s.file.Sync()
}

// Close closes the stream file
func (s *Stream) Close() error {
	return s.file.Close()
}
