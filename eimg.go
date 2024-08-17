// eimg.go - Converts your images into encodings that your terminal&framebuffer can/may understand
package main

import (
	"flag"
	"fmt"
	"github.com/BourgeoisBear/rasterm"
	"github.com/ericpauley/go-quantize/quantize"
	"github.com/jiro4989/textimg/v3/config"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/xplshn/a-utils/pkg/ccmd"
	"github.com/xplshn/eimg/pkg/ur-fb"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vector"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	inputFilePath := flag.String("input-file", "", "Input image file path")
	resizeFlag := flag.String("resize", "", "Resize dimensions (e.g., 800x600)")
	scaleFactor := flag.Float64("scale-factor", 1.0, "Scale factor")
	posX := flag.Int("pos-x", 0, "X position on framebuffer")
	posY := flag.Int("pos-y", 0, "Y position on framebuffer")
	noBounds := flag.Bool("no-bounds", false, "Disable safety feature to keep image in-bounds")
	useEncoding := flag.String("use-encoding", "", "Force specific encoding (kitty, iterm, sixel)")

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

	if *inputFilePath == "" {
		fmt.Print(helpPage)
		fmt.Fprintln(os.Stderr, "Input file path is required")
		os.Exit(1)
	}

	fileExtension := filepath.Ext(*inputFilePath)
	if fileExtension == "" {
		fmt.Print(helpPage)
		fmt.Fprintln(os.Stderr, "Unable to determine file format")
		os.Exit(1)
	}

	if *scaleFactor < 0 {
		fmt.Print(helpPage)
		fmt.Fprintln(os.Stderr, "Scale factor cannot be negative")
		os.Exit(1)
	}

	if *scaleFactor == 0 {
		fmt.Print(helpPage)
		fmt.Fprintln(os.Stderr, "Scale factor cannot be 0")
		os.Exit(1)
	}

	resizeWidth, resizeHeight := 0, 0
	if *resizeFlag != "" {
		parts := strings.Split(*resizeFlag, "x")
		if len(parts) != 2 {
			fmt.Print(helpPage)
			fmt.Fprintln(os.Stderr, "Invalid resize format. Use WIDTHxHEIGHT (e.g., 800x600)")
			os.Exit(1)
		}
		var err error
		resizeWidth, err = strconv.Atoi(parts[0])
		if err != nil {
			fmt.Print(helpPage)
			fmt.Fprintln(os.Stderr, "Invalid width value")
			os.Exit(1)
		}
		resizeHeight, err = strconv.Atoi(parts[1])
		if err != nil {
			fmt.Print(helpPage)
			fmt.Fprintln(os.Stderr, "Invalid height value")
			os.Exit(1)
		}
	}

	cfg := config.Config{
		ResizeWidth:   resizeWidth,
		ResizeHeight:  resizeHeight,
		FileExtension: fileExtension,
		Writer:        os.Stdout, // Set the writer to stdout for terminal display
	}

	// INLINE FUNCTIONS
	convertToPaletted := func(img image.Image) *image.Paletted {
		if palettedImg, ok := img.(*image.Paletted); ok && len(palettedImg.Palette) == 256 {
			// The image is already paletted with a depth of 256, no need to convert.
			return palettedImg
		}

		// Create a palette with 256 colors
		q := quantize.MedianCutQuantizer{}
		palette := q.Quantize(make([]color.Color, 0, 256), img)

		d := dither.NewDitherer(palette)
		d.Matrix = dither.Stucki
		palettedImg := d.DitherPaletted(img)

		return palettedImg
	}
	resizeImage := func(img image.Image, width, height int) image.Image {
		if width == 0 && height == 0 {
			return img
		}

		bounds := img.Bounds()
		imgWidth := bounds.Dx()
		imgHeight := bounds.Dy()

		if width == 0 {
			width = int(float64(imgWidth) * (float64(height) / float64(imgHeight)))
		} else if height == 0 {
			height = int(float64(imgHeight) * (float64(width) / float64(imgWidth)))
		}

		resizedImg := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				srcX := x * imgWidth / width
				srcY := y * imgHeight / height
				resizedImg.Set(x, y, img.At(srcX, srcY))
			}
		}

		return resizedImg
	}
	scaleImage := func(img image.Image, scaleFactor float64) image.Image {
		if scaleFactor == 1.0 {
			return img
		}

		bounds := img.Bounds()
		imgWidth := bounds.Dx()
		imgHeight := bounds.Dy()

		scaledWidth := int(float64(imgWidth) * scaleFactor)
		scaledHeight := int(float64(imgHeight) * scaleFactor)

		scaledImg := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
		for y := 0; y < scaledHeight; y++ {
			for x := 0; x < scaledWidth; x++ {
				srcX := int(float64(x) / scaleFactor)
				srcY := int(float64(y) / scaleFactor)
				scaledImg.Set(x, y, img.At(srcX, srcY))
			}
		}

		return scaledImg
	}
	ensureInBounds := func(img image.Image, maxWidth, maxHeight int) image.Image {
		bounds := img.Bounds()
		imgWidth := bounds.Dx()
		imgHeight := bounds.Dy()

		if imgWidth <= maxWidth && imgHeight <= maxHeight {
			return img
		}

		newWidth := imgWidth
		newHeight := imgHeight

		if imgWidth > maxWidth {
			newWidth = maxWidth
			newHeight = int(float64(imgHeight) * (float64(maxWidth) / float64(imgWidth)))
		}

		if newHeight > maxHeight {
			newHeight = maxHeight
			newWidth = int(float64(imgWidth) * (float64(maxHeight) / float64(imgHeight)))
		}

		return resizeImage(img, newWidth, newHeight)
	}
	displayImage := func(cfg *config.Config, inputFilePath string, posX, posY int, scaleFactor float64, noBounds bool, useEncoding string) error {
		encodeAndDisplayImage := func(img image.Image, cfg *config.Config, useEncoding string) error {
			switch useEncoding {
			case "kitty":
				return rasterm.KittyWriteImage(cfg.Writer, img, rasterm.KittyImgOpts{})
			case "iterm":
				return rasterm.ItermWriteImage(cfg.Writer, img)
			case "sixel":
				palettedImg := convertToPaletted(img)
				return rasterm.SixelWriteImage(cfg.Writer, palettedImg)
			default:
				if rasterm.IsKittyCapable() {
					return rasterm.KittyWriteImage(cfg.Writer, img, rasterm.KittyImgOpts{})
				} else if rasterm.IsItermCapable() {
					return rasterm.ItermWriteImage(cfg.Writer, img)
				} else if isSixelCapable, err := rasterm.IsSixelCapable(); err == nil && isSixelCapable {
					palettedImg := convertToPaletted(img)
					return rasterm.SixelWriteImage(cfg.Writer, palettedImg)
				}

				return fmt.Errorf("terminal does not support any known image protocol")
			}
		}
		// E-OF-INLINE-FUNCTIONS

		imgFile, err := os.Open(inputFilePath)
		if err != nil {
			return err
		}
		defer imgFile.Close()

		img, _, err := image.Decode(imgFile)
		if err != nil {
			return err
		}

		// Resize the image if needed
		if cfg.ResizeWidth > 0 || cfg.ResizeHeight > 0 {
			img = resizeImage(img, cfg.ResizeWidth, cfg.ResizeHeight)
		}

		// Scale the image if needed
		img = scaleImage(img, scaleFactor)

		// Get terminal or framebuffer dimensions
		maxWidth, maxHeight := 80, 24 // Default terminal size
		if fbWidth, fbHeight, _, _, err := fb.FbInit(); err == nil {
			maxWidth, maxHeight = fbWidth, fbHeight
		}

		// Ensure the image is within bounds
		if !noBounds {
			img = ensureInBounds(img, maxWidth, maxHeight)
		}

		// Encode and display the image for terminal display
		if err := encodeAndDisplayImage(img, cfg, useEncoding); err != nil {
			// Fallback to framebuffer if no terminal protocol is found
			// Hide the cursor
			fmt.Print("\033[?25l")
			if err := fb.DrawScaledImageAt(img, posX, posY, scaleFactor); err != nil {
				return fmt.Errorf("error drawing on framebuffer: %w", err)
			}
			// Show the cursor
			fmt.Print("\033[?25h")
		}
		return nil
	}
	// E-OF-INLINE-FUNCTIONS

	if err := displayImage(&cfg, *inputFilePath, *posX, *posY, *scaleFactor, *noBounds, *useEncoding); err != nil {
		fmt.Println("Error:", err)
	}
	println()
}
