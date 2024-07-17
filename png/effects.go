// Package png allows for loading png images and applying
// image flitering effects on them.
package png

import (
	"image"
	"image/color"
	"math"
)

// Effects struct encapsulating the kernels for various image processing operations depending on effects.json
type Effects struct {
	S [][]float64 // Sharpen kernel
	E [][]float64 // Edge detection kernel
	B [][]float64 // Blur kernel
}

// Grayscale applies a grayscale filtering effect to the image
func (img *Image) Grayscale(start int, end int) {
	bounds := img.Out.Bounds()
	if start == 0 && end == 0 {
		start = bounds.Min.Y
		end = bounds.Max.Y
	}

	for y := start; y < end; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			//Returns the pixel (i.e., RGBA) value at a (x,y) position
			// Note: These get returned as int32 so based on the math you'll
			// be performing you'll need to do a conversion to float64(..)
			r, g, b, a := img.In.At(x, y).RGBA()

			//Note: The values for r,g,b,a for this assignment will range between [0, 65535].
			//For certain computations (i.e., convolution) the values might fall outside this
			// range so you need to clamp them between those values.
			greyC := Clamp(float64(r+g+b) / 3)

			//Note: The values need to be stored back as uint16 (I know weird..but there's valid reasons
			// for this that I won't get into right now).
			img.Out.Set(x, y, color.RGBA64{greyC, greyC, greyC, uint16(a)})
		}
	}
}

func NewEffects() Effects {
	return Effects{
		S: [][]float64{{0, -1, 0}, {-1, 5, -1}, {0, -1, 0}},
		E: [][]float64{{-1, -1, -1}, {-1, 8, -1}, {-1, -1, -1}},
		B: [][]float64{{1.0 / 9, 1.0 / 9, 1.0 / 9}, {1.0 / 9, 1.0 / 9, 1.0 / 9}, {1.0 / 9, 1.0 / 9, 1.0 / 9}},
	}
}

func (img *Image) ApplyEffects(effects []string, par bool, startY int, endY int) {
	e := NewEffects()
	for i, effect := range effects {
		if i > 0 {
			img.In = img.Out
			img.Out = image.NewRGBA64(img.In.Bounds())
		}
		switch effect {
		case "S": // Sharpen
			img.ApplyEffect(e.S, par, startY, endY)
		case "E": // Edge Detection
			img.ApplyEffect(e.E, par, startY, endY)
		case "B": // Blurx
			img.ApplyEffect(e.B, par, startY, endY)
		case "G": // Grayscale
			img.Grayscale(0, 0)
		default:
			continue
		}
	}

}

func (img *Image) ApplyEffect(kernel [][]float64, par bool, startY int, endY int) {
	bounds := img.In.Bounds()
	kernelSize := 3
	var start, end int
	if !par {
		start = bounds.Min.Y
		end = bounds.Max.Y
	} else {
		start = int(math.Max(float64(startY-1), float64(0)))
		end = int(math.Max(float64(endY+1), float64(bounds.Dy())))
	}
	for y := start; y < end; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var r, g, b, _ uint32
			var sumR, sumG, sumB float64

			for ky := 0; ky < kernelSize; ky++ {
				for kx := 0; kx < kernelSize; kx++ {
					pxX := x + kx - 1
					pxY := y + ky - 1
					// Apply zero padding for pixels outside the bounds
					if pxX < 0 || pxY < 0 {
						r, g, b = 0, 0, 0
					} else {
						r, g, b, _ = img.In.At(pxX, pxY).RGBA()
					}

					// Apply kernel
					sumR += kernel[ky][kx] * float64(r)
					sumG += kernel[ky][kx] * float64(g)
					sumB += kernel[ky][kx] * float64(b)
				}
			}

			// Clamp the summed values, scale back up to [0, 65535], and convert back to uint16
			newR := Clamp(sumR)
			newG := Clamp(sumG)
			newB := Clamp(sumB)

			_, _, _, a := img.In.At(x, y).RGBA()
			img.Out.Set(x, y, color.RGBA64{R: newR, G: newG, B: newB, A: uint16(a)})
		}
	}
}

func (img *Image) MakeChunk(startY int, endY int) *Image {
	bounds := img.In.Bounds()
	chunk := image.NewRGBA64(image.Rect(bounds.Min.X, startY, bounds.Max.X, endY))
	for y := startY; y < endY; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.In.At(x, y).RGBA()
			chunk.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), uint16(a)})
		}
	}
	return &Image{In: chunk, Out: chunk, Bounds: chunk.Bounds()}
}
