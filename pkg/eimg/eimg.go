// pkg/eimg/eimg.go
package eimg

import (
	"fmt"
	"os"

	"image"
	"image/color"

	// Support for common image formats
	_ "image/png"
	_ "image/jpeg"
	_ "image/gif"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vector"
	_ "golang.org/x/image/webp"

	"github.com/BourgeoisBear/rasterm"
	"github.com/ericpauley/go-quantize/quantize"
	"github.com/jiro4989/textimg/v3/config"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/xplshn/eimg/pkg/ur-fb"
)

// ConvertToPaletted converts the image to a paletted format with dithering.
func ConvertToPaletted(img image.Image) *image.Paletted {
	q := quantize.MedianCutQuantizer{}
	palette := q.Quantize(make([]color.Color, 0, 256), img)

	d := dither.NewDitherer(palette)
	d.Matrix = dither.Stucki
	return d.DitherPaletted(img)
}

// ResizeImage resizes the given image to the specified width and height.
func ResizeImage(img image.Image, width, height int) image.Image {
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

// ScaleImage scales the given image by the scale factor.
func ScaleImage(img image.Image, scaleFactor float64) image.Image {
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

// EnsureInBounds resizes the image to ensure it fits within maxWidth and maxHeight while maintaining the aspect ratio.
func EnsureInBounds(img image.Image, maxWidth, maxHeight int) image.Image {
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

	return ResizeImage(img, newWidth, newHeight)
}

func EncodeAndDisplayImage(img image.Image, cfg *config.Config, useEncoding string) error {
	switch useEncoding {
	case "kitty":
		return rasterm.KittyWriteImage(cfg.Writer, img, rasterm.KittyImgOpts{})
	case "iterm":
		return rasterm.ItermWriteImage(cfg.Writer, img)
	case "sixel":
		palettedImg := ConvertToPaletted(img)
		return rasterm.SixelWriteImage(cfg.Writer, palettedImg)
	default:
		if rasterm.IsKittyCapable() {
			return rasterm.KittyWriteImage(cfg.Writer, img, rasterm.KittyImgOpts{})
		} else if rasterm.IsItermCapable() {
			return rasterm.ItermWriteImage(cfg.Writer, img)
		} else if isSixelCapable, err := rasterm.IsSixelCapable(); err == nil && isSixelCapable {
			palettedImg := ConvertToPaletted(img)
			return rasterm.SixelWriteImage(cfg.Writer, palettedImg)
		}

		return fmt.Errorf("terminal does not support any known image protocol")
	}
}

// DisplayImage decodes an image from the given file path and displays it.
func DisplayImage(filePath string, cfg *config.Config, useEncoding string, maxWidth, maxHeight int, posX, posY int, scaleFactor float64, noBounds bool) error {
	imgFile, err := os.Open(filePath)
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
		img = ResizeImage(img, cfg.ResizeWidth, cfg.ResizeHeight)
	}

	// Scale the image if needed
	img = ScaleImage(img, scaleFactor)

	// Ensure the image is within bounds
	if !noBounds {
		img = EnsureInBounds(img, maxWidth, maxHeight)
	}

	// Encode and display the image for terminal display
	if err := EncodeAndDisplayImage(img, cfg, useEncoding); err != nil {
		// Fallback to framebuffer if no terminal protocol is found
		// Hide the cursor
		fmt.Print("\033[?25l")
		if err := fb.DrawScaledImageAt(img, posX, posY, scaleFactor); err != nil {
			return fmt.Errorf("error drawing on framebuffer: %w", err)
		}
		// Show the cursor & print a newline
		fmt.Print("\033[?25h")
	}
	return nil
}
