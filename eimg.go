package main

import (
	"eimg/pkg/ur-fb"
	"flag"
	"fmt"
	"github.com/BourgeoisBear/rasterm"
	"github.com/jiro4989/textimg/v3/config"
	"github.com/xplshn/a-utils/pkg/ccmd"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vector"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
)

func main() {
	inputFilePath := flag.String("input-file", "", "Input image file path")
	resizeWidth := flag.Int("resize-width", 0, "Resize width")
	resizeHeight := flag.Int("resize-height", 0, "Resize height")
	scaleFactor := flag.Float64("scale-factor", 1.0, "Scale factor")
	posX := flag.Int("pos-x", 0, "X position on framebuffer")
	posY := flag.Int("pos-y", 0, "Y position on framebuffer")
	noBounds := flag.Bool("no-bounds", false, "Disable safety feature to keep image in-bounds")

	cmdInfo := &ccmd.CmdInfo{
		Name:        "eimg",
		Repository:  "https://github.com/xplshn/eimg.git",
		Authors:     []string{"xplshn"},
		Synopsis:    "[--input-file [FILE]] <|--resize-{width,height}|--scale-factor|--pos-{x,y}|--no-bounds|>",
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

	cfg := config.Config{
		ResizeWidth:   *resizeWidth,
		ResizeHeight:  *resizeHeight,
		FileExtension: fileExtension,
		Writer:        os.Stdout, // Set the writer to stdout for terminal display
	}

	// INLINE FUNCTIONS
	displayImage := func(cfg *config.Config, inputFilePath string, posX, posY int, scaleFactor float64, noBounds bool) error {
		encodeAndDisplayImage := func(img image.Image, cfg *config.Config) error {
			if rasterm.IsKittyCapable() {
				return rasterm.KittyWriteImage(cfg.Writer, img, rasterm.KittyImgOpts{})
			} else if rasterm.IsItermCapable() {
				return rasterm.ItermWriteImage(cfg.Writer, img)
			} else if isSixelCapable, err := rasterm.IsSixelCapable(); err == nil && isSixelCapable {
				palettedImg := image.NewPaletted(img.Bounds(), nil)
				for y := 0; y < img.Bounds().Dy(); y++ {
					for x := 0; x < img.Bounds().Dx(); x++ {
						palettedImg.Set(x, y, img.At(x, y))
					}
				}
				return rasterm.SixelWriteImage(cfg.Writer, palettedImg)
			}

			return fmt.Errorf("terminal does not support any known image protocol")
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
		if err := encodeAndDisplayImage(img, cfg); err != nil {
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

	if err := displayImage(&cfg, *inputFilePath, *posX, *posY, *scaleFactor, *noBounds); err != nil {
		fmt.Println("Error:", err)
	}
	println()
}
