package fb

import (
	"fmt"
	"image"
	"os"

	"github.com/orangecms/go-framebuffer/framebuffer"
)

const fbdev = "/dev/fb0"

func DrawOnBufAt(
	buf []byte,
	img image.Image,
	posx int,
	posy int,
	stride int,
	bpp int,
) {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			offset := bpp * ((posy+y)*stride + posx + x)
			if offset+bpp > len(buf) {
				continue // Skip if offset is out of bounds
			}
			// framebuffer is BGR(A)
			buf[offset+0] = byte(b)
			buf[offset+1] = byte(g)
			buf[offset+2] = byte(r)
			if bpp >= 4 {
				buf[offset+3] = byte(a)
			}
		}
	}
}

// FbInit initializes a frambuffer by querying ioctls and returns the width and
// height in pixels, the stride, and the bytes per pixel
func FbInit() (int, int, int, int, error) {
	fbo, err := framebuffer.Init(fbdev)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	width, height := fbo.Size()
	stride := fbo.Stride()
	bpp := fbo.Bpp()
	// fmt.Fprintf(os.Stdout, "Framebuffer resolution: %v %v %v %v\n", width, height, stride, bpp)
	return width, height, stride, bpp, nil
}

func DrawImageAt(img image.Image, posx int, posy int) error {
	width, height, stride, bpp, err := FbInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Framebuffer init error: %v\n", err)
		// fallback values, probably a bad assumption
		width, height, stride, bpp = 1920, 1080, 1920*4, 4
	}
	buf := make([]byte, width*height*bpp)
	DrawOnBufAt(buf, img, posx, posy, stride, bpp)
	err = os.WriteFile(fbdev, buf, 0o600)
	if err != nil {
		return fmt.Errorf("error writing to framebuffer: %w", err)
	}
	return nil
}

func DrawScaledOnBufAt(
	buf []byte,
	img image.Image,
	posx int,
	posy int,
	scaleFactor float64,
	stride int,
	bpp int,
) {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	scaledWidth := int(float64(imgWidth) * scaleFactor)
	scaledHeight := int(float64(imgHeight) * scaleFactor)

	for y := 0; y < scaledHeight; y++ {
		for x := 0; x < scaledWidth; x++ {
			srcX := int(float64(x) / scaleFactor)
			srcY := int(float64(y) / scaleFactor)
			r, g, b, a := img.At(srcX, srcY).RGBA()
			offset := bpp * ((posy+y)*stride + posx + x)
			if offset+bpp > len(buf) {
				continue // Skip if offset is out of bounds
			}
			buf[offset+0] = byte(b)
			buf[offset+1] = byte(g)
			buf[offset+2] = byte(r)
			if bpp == 4 {
				buf[offset+3] = byte(a)
			}
		}
	}
}

func DrawScaledImageAt(img image.Image, posx int, posy int, scaleFactor float64) error {
	width, height, stride, bpp, err := FbInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Framebuffer init error: %v\n", err)
		// fallback values, probably a bad assumption
		width, height, stride, bpp = 1920, 1080, 1920*4, 4
	}
	buf := make([]byte, width*height*bpp)
	DrawScaledOnBufAt(buf, img, posx, posy, scaleFactor, stride, bpp)
	err = os.WriteFile(fbdev, buf, 0o600)
	if err != nil {
		return fmt.Errorf("error writing to framebuffer: %w", err)
	}
	return nil
}
