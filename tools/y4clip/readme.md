# y4clip

Create a new y4m video stream from an existing y4m video stream, with options to crop and truncate frames.

### Usage

    -i string
    	input file
    -o string
    	output file
    -s int
    	start frame (default 1)    	
    -e int
    	end frame; -1 for last frame of input stream (default -1)
    -h int
    	cropped height; -1 for original height (default -1)
    -w int
    	cropped width; -1 for original width (default -1)
    -x string
    	horizontal offset of cropped frame; -1 to center (default -1)
    -y string
    	vertical offset of cropped frame; -1 to center (default -1)
    -strip
    	strip header information
	
### Example

Create new video stream consisting of first 100 frames of input stream cropped to 1080 x 1080:

    > /y4clip -i aspen.y4v -o aspen-clip.y4v -w 1080 -h 1080 -s 1 -e 100

