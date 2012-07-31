package texture

import (
  "image"
  "image/color"
  "github.com/runningwild/memory"
)

// NOTE: All of this code is basically ripped from the Go source, it's just
// been modified to include an alpha value

type grayAlpha uint16

func (ga grayAlpha) RGBA() (uint32, uint32, uint32, uint32) {
  v := uint32(ga & 0xff00)
  return v, v, v, uint32((ga & 0xff) << 8)
}

var GrayAlphaModel grayAlphaModel

type grayAlphaModel struct{}

func (gam grayAlphaModel) Convert(c color.Color) color.Color {
  // r, g, b, a := c.RGBA()
  // return grayAlpha{ (r + g + b) / 3, a }
  r, _, _, a := c.RGBA()
  return gam.baseConvert(r, a)
}
func (gam grayAlphaModel) baseConvert(l, a uint32) color.Color {
  return grayAlpha((l & 0xff00) | ((a & 0xff00) >> 8))
}

type GrayAlpha struct {
  // Pix holds the image's pixels, as gray values. The pixel at
  // (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*2].
  Pix []uint8
  // Stride is the Pix stride (in bytes) between vertically adjacent pixels.
  Stride int
  // Rect is the image's bounds.
  Rect image.Rectangle
}

func (p *GrayAlpha) ColorModel() color.Model { return GrayAlphaModel }

func (p *GrayAlpha) Bounds() image.Rectangle { return p.Rect }

func (p *GrayAlpha) At(x, y int) color.Color {
  if !(image.Point{x, y}.In(p.Rect)) {
    return grayAlpha(0)
  }
  i := p.PixOffset(x, y)
  return GrayAlphaModel.baseConvert(uint32(p.Pix[i]), uint32(p.Pix[i+1]))
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *GrayAlpha) PixOffset(x, y int) int {
  return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*2
}

func (p *GrayAlpha) Set(x, y int, c color.Color) {
  if !(image.Point{x, y}.In(p.Rect)) {
    return
  }
  i := p.PixOffset(x, y)
  r, _, _, a := c.RGBA()
  p.Pix[i] = byte(r >> 8)
  p.Pix[i+1] = byte(a >> 8)
}
func NewGrayAlpha(r image.Rectangle) *GrayAlpha {
  var dx, dy int
  dx = r.Dx()
  if dx%2 == 1 {
    dx++
  }
  dy = r.Dy()
  return &GrayAlpha{
    Pix:    memory.GetBlock(dx * dy * 2),
    Stride: dx * 2,
    Rect:   r,
  }
}
