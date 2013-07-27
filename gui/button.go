package gui

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/magnus/base"
)

type Button struct {
	Name  string
	Count int
	Hover bool
	Last  Region
}

func (b *Button) Think(gui *Gui) {
	x, y := gui.sys.GetCursorPos()
	b.Hover = x >= b.Last.X && x < b.Last.X+b.Last.Dx &&
		y >= b.Last.Y && y < b.Last.Y+b.Last.Dy
}
func (b *Button) Respond(eventGroup gin.EventGroup) {
	if !b.Hover {
		return
	}
	if found, event := eventGroup.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
		b.Count++
	}
}
func (b *Button) Draw(region Region) {
	b.Last = region
	if b.Hover {
		gl.Color4ub(255, 255, 255, 255)
	} else {
		gl.Color4ub(255, 255, 255, 128)
	}
	base.GetDictionary("luxisr").RenderString(fmt.Sprintf("%d: %s", b.Count, b.Name), float64(region.X), float64(region.Y), 0, 30, gui.Left)
}
func (b *Button) RequestedDims() Dims {
	return Dims{int(base.GetDictionary("luxisr").StringWidth(b.Name, 30)), 30}
}
