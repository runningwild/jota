package gui

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/jota/base"
	"strings"
)

const maxLines = 30
const maxLineLength = 150
const lineHeight = 25

// A simple gui element that will display the last several lines of text from
// a log file (TODO: and also allow you to enter some basic commands).
type Console struct {
	lines   []string
	tail    base.Tailer
	xscroll float64
	dims    Dims
	dict    *gui.Dictionary
	visible bool
}

func MakeConsole(dx, dy int) *Console {
	var c Console
	c.lines = make([]string, maxLines)
	c.tail = base.GetLogTailer()
	c.dict = base.GetDictionary("luxisr")
	c.dims.Dx = dx
	c.dims.Dy = dy
	return &c
}

func (c *Console) Think(g *Gui) {
	c.visible = gin.In().GetKeyFlat(gin.EitherShift, gin.DeviceTypeAny, gin.DeviceIndexAny).IsDown()
	if c.visible {
		c.tail.GetLines(c.lines)
	}
}

func (c *Console) RequestedDims() Dims {
	return c.dims
}

func (c *Console) Respond(group gin.EventGroup) {
}

func (c *Console) Draw(region Region, stlye StyleStack) {
	if !c.visible {
		return
	}
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Color4d(0.2, 0, 0.3, 0.8)
	gl.Disable(gl.TEXTURE_2D)
	gl.Begin(gl.QUADS)
	{
		x := gl.Int(region.X)
		y := gl.Int(region.Y)
		x2 := gl.Int(region.X + region.Dx)
		y2 := gl.Int(region.Y + region.Dy)
		gl.Vertex2i(x, y)
		gl.Vertex2i(x, y2)
		gl.Vertex2i(x2, y2)
		gl.Vertex2i(x2, y)
	}
	gl.End()
	gui.SetFontColor(1, 1, 1, 1)
	startY := float64(region.Y + region.Dy - len(c.lines)*lineHeight)
	for i, line := range c.lines {
		switch {
		case strings.HasPrefix(line, "LOG"):
			gui.SetFontColor(1, 1, 1, 1)
		case strings.HasPrefix(line, "WARN"):
			gui.SetFontColor(1, 1, 0, 1)
		case strings.HasPrefix(line, "ERROR"):
			gui.SetFontColor(1, 0, 0, 1)
		default:
			gui.SetFontColor(1, 1, 1, 0.7)
		}
		c.dict.RenderString(line, float64(region.X), startY+float64(i*lineHeight), 0, lineHeight, gui.Left)
	}
}
