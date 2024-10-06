package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"strconv"

	"github.com/xplshn/a-utils/pkg/ccmd"
	"github.com/xplshn/eimg/pkg/eimg"
	"github.com/jiro4989/textimg/v3/config"
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
	useEncoding := flag.String("use-encoding", "", "Force specific encoding (kitty, iterm, sixel)")

	// Initialize command info for help and usage
	cmdInfo := &ccmd.CmdInfo{
		Name:        "eimg",
		Repository:  "https://github.com/xplshn/eimg.git",
		Authors:     []string{"xplshn"},
		Synopsis:    "[--input-file [FILE]] <|--resize|--scale-factor|--pos-{x,y}|--no-bounds|--use-encoding|>",
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

	// Create configuration for the image
	cfg := &config.Config{
		ResizeWidth:   resizeWidth,
		ResizeHeight:  resizeHeight,
		FileExtension: filepath.Ext(*inputFilePath),
		Writer:        os.Stdout, // Display output in the terminal
	}

	// Get terminal or framebuffer dimensions
	maxWidth, maxHeight := 80, 24 // Default terminal size
	if fbWidth, fbHeight, _, _, err := fb.FbInit(); err == nil {
		maxWidth, maxHeight = fbWidth, fbHeight
	}

	// Pass the configuration to the library, and let it handle framebuffer logic internally
	err = eimg.DisplayImage(*inputFilePath, cfg, *useEncoding, maxWidth, maxHeight, *posX, *posY, *scaleFactor, *noBounds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error displaying image:", err)
		os.Exit(1)
	}
	println()
}
