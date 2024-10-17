package eimg

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"strings"

	// Support for common image formats
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vector"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/BourgeoisBear/rasterm"
	"github.com/ericpauley/go-quantize/quantize"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/xplshn/eimg/pkg/ur-fb"
)

const (
	ansiBasicBase  = 16
	ansiColorSpace = 6
	ansiForeground = "38"
	ansiReset      = "\x1b[0m"
	characters     = "01"
	defaultWidth   = 100
	proportion     = 0.46
	rgbaColorSpace = 1 << 16
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

func EncodeAndDisplayImage(img image.Image, writer *os.File, useEncoding string, posX, posY int, scaleFactor float64) error {
	switch useEncoding {
	case "kitty":
		return rasterm.KittyWriteImage(writer, img, rasterm.KittyImgOpts{})
	case "iterm":
		return rasterm.ItermWriteImage(writer, img)
	case "sixel":
		palettedImg := ConvertToPaletted(img)
		return rasterm.SixelWriteImage(writer, palettedImg)
	case "ansi":
		ansiString, err := WriteAnsiImage(img, defaultWidth)
		if err != nil {
			return err
		}
		_, err = writer.WriteString(ansiString)
		return err
	case "framebuffer":
		// Hide the cursor
		fmt.Print("\033[?25l")
		if err := fb.DrawScaledImageAt(img, posX, posY, scaleFactor); err != nil {
			return fmt.Errorf("error drawing on framebuffer: %w", err)
		}
		// Show the cursor & print a newline
		fmt.Print("\033[?25h")
		return nil
	default:
		if rasterm.IsKittyCapable() {
			return rasterm.KittyWriteImage(writer, img, rasterm.KittyImgOpts{})
		} else if rasterm.IsItermCapable() {
			return rasterm.ItermWriteImage(writer, img)
		} else if isSixelCapable, err := rasterm.IsSixelCapable(); err == nil && isSixelCapable {
			palettedImg := ConvertToPaletted(img)
			return rasterm.SixelWriteImage(writer, palettedImg)
		}

		// Fallback to ANSI
		ansiString, err := WriteAnsiImage(img, defaultWidth)
		if err != nil {
			return err
		}
		_, err = writer.WriteString(ansiString)
		return err
	}
}

// DisplayImage decodes an image from the given file path and displays it.
func DisplayImage(filePath string, writer *os.File, useEncoding string, maxWidth, maxHeight int, posX, posY int, scaleFactor float64, noBounds bool, resizeWidth, resizeHeight int) error {
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
	if resizeWidth > 0 || resizeHeight > 0 {
		img = ResizeImage(img, resizeWidth, resizeHeight)
	}

	// Scale the image if needed
	img = ScaleImage(img, scaleFactor)

	// Ensure the image is within bounds
	if !noBounds {
		img = EnsureInBounds(img, maxWidth, maxHeight)
	}

	// Encode and display the image for terminal display
	return EncodeAndDisplayImage(img, writer, useEncoding, posX, posY, scaleFactor)
}

// ---------- ANSI SUPPORT --------------

func toAnsiCode(c color.Color) string {
	r, g, b, _ := c.RGBA()
	code := ansiBasicBase + toAnsiSpace(r)*36 + toAnsiSpace(g)*6 + toAnsiSpace(b)
	if code == ansiBasicBase {
		return ansiReset
	}
	return fmt.Sprintf("\033[%s;5;%dm", ansiForeground, code)
}

func toAnsiSpace(val uint32) int {
	return int(float32(ansiColorSpace) * (float32(val) / float32(rgbaColorSpace)))
}

func WriteAnsiImage(img image.Image, width int) (string, error) {
	imgW, imgH := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	height := float32(width) * (imgH / imgW) * proportion
	resizedImg := ResizeImage(img, width, int(height))

	bounds := resizedImg.Bounds()
	var current, previous string
	var ansiString strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			current = toAnsiCode(resizedImg.At(x, y))
			if current != previous {
				ansiString.WriteString(current)
			}
			if current != ansiReset {
				char := string(characters[rand.Intn(len(characters))])
				ansiString.WriteString(char)
			} else {
				ansiString.WriteString(" ")
			}
			previous = current
		}
		ansiString.WriteString("\n")
	}
	ansiString.WriteString(ansiReset)
	return ansiString.String(), nil
}
