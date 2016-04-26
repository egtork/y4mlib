# y4grab

Grab frames from a y4m video stream and saves as JPEG/PNG/TIFF images.

### Usage

    > ./y4grab -i filename [options]

Options:

      -i string
    	    input filename
      -o string
    	    output filename (defaults to input filename with appropriate image extension)
      -s int
    	    start frame (default 1)
      -n int
    	    number of frames to grab (default 1)
      -f string
    	    image format {"jpeg", "png", "tiff"} (default "jpeg")
      -jq int
    	    (JPEG only) quality [0-100] (default 75)
      -tc
    	    (TIFF only) use deflate compression
      -tp
    	    (TIFF only) use differencing predictor

### Example

Grab frames 10-14 and convert to JPEG files with quality 50

    > ./y4grab -i aspen.y4m -s 10 -n 5 -f jpeg -jq 50 -o aspen.jpg

In the case that we grab multiple frames, frame indicies are inserted between the filename and extension as follows:

    > ls *.jpg
    
    aspen10.jpg	aspen11.jpg	aspen12.jpg	
    aspen13.jpg	aspen14.jpg
