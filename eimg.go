package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"strconv"

	"github.com/xplshn/a-utils/pkg/ccmd"
	"github.com/xplshn/eimg/pkg/eimg"
	"github.com/xplshn/eimg/pkg/ur-fb"
)

func main() {
	// Define command line flags
	inputFilePath := flag.String("input-file", "", "Input image file path")
	resizeFlag := flag.String("resize", "", "Resize dimensions (e.g., 800x600)")
	scaleFactor := flag.Float64("scale-factor", 1.0, "Scale factor")
	posX := flag.Int("pos-x", 0, "X position on framebuffer")
	posY := flag.Int("pos-y", 0, "Y position on framebuffer")
	noBounds := flag.Bool("no-bounds", false, "Disable safety feature to keep image in-bounds")
	useEncoding := flag.String("use-encoding", "", "Force specific encoding (kitty, iterm, sixel, ansi)")

	// Initialize command info for help and usage
	cmdInfo := &ccmd.CmdInfo{
		Name:        "eimg",
		Repository:  "https://github.com/xplshn/eimg.git",
		Authors:     []string{"xplshn"},
		Synopsis:    "[--input-file [FILE]] <|--resize|--scale-factor|--pos-{x,y}|--no-bounds|--use-encoding|--use-framebuffer|>",
		Description: "Displays images in the terminal using terminal encodings",
	}

	helpPage, err := cmdInfo.GenerateHelpPage()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating help page:", err)
		os.Exit(1)
	}
	flag.Usage = func() {
		fmt.Print(helpPage)
	}

	flag.Parse()

	// Validate input file
	if *inputFilePath == "" {
		fmt.Print(helpPage)
		fmt.Fprintln(os.Stderr, "Input file path is required")
		os.Exit(1)
	}

	// Handle image resize dimensions if provided
	resizeWidth, resizeHeight := 0, 0
	if *resizeFlag != "" {
		parts := strings.Split(*resizeFlag, "x")
		if len(parts) == 2 {
			resizeWidth, err = strconv.Atoi(parts[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Invalid width in resize flag")
				os.Exit(1)
			}
			resizeHeight, err = strconv.Atoi(parts[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Invalid height in resize flag")
				os.Exit(1)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Invalid resize format. Use WIDTHxHEIGHT (e.g., 800x600)")
			os.Exit(1)
		}
	}

	// Get terminal or framebuffer dimensions
	maxWidth, maxHeight := 80, 24 // Default terminal size
	if fbWidth, fbHeight, _, _, err := fb.FbInit(); err == nil {
		maxWidth, maxHeight = fbWidth, fbHeight
	}

	// Pass the parameters to the library, and let it handle the logic internally
	err = eimg.DisplayImage(*inputFilePath, os.Stdout, *useEncoding, maxWidth, maxHeight, *posX, *posY, *scaleFactor, *noBounds, resizeWidth, resizeHeight)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error displaying image:", err)
		os.Exit(1)
	}
	println()
}
