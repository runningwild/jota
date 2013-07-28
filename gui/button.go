package gui

import (
	"fmt"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/magnus/base"
)

type Button struct {
	Name         string
	Count        int
	Hover        bool
	Last         Region
	Size         int
	renderString string
	Callback     func()
}

func (b *Button) Think(gui *Gui) {
	x, y := gui.sys.GetCursorPos()
	b.Hover = x >= b.Last.X && x < b.Last.X+b.Last.Dx &&
		y >= b.Last.Y && y < b.Last.Y+b.Last.Dy
	b.renderString = fmt.Sprintf("%d: %s", b.Count, b.Name)
}
func (b *Button) Respond(eventGroup gin.EventGroup) {
	if !b.Hover {
		return
	}
	if found, event := eventGroup.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
		if b.Callback != nil {
			b.Callback()
		} else {
			b.Count++
		}
	}
}
func (b *Button) Draw(region Region) {
	b.Last = region
	if b.Hover {
		gui.SetFontColor(1, 1, 0, 1)
	} else {
		gui.SetFontColor(1, 1, 0, 0.5)
	}
	base.GetDictionary("luxisr").RenderString(b.renderString, float64(region.X), float64(region.Y), 0, float64(b.Size), gui.Left)
}
func (b *Button) RequestedDims() Dims {
	return Dims{int(base.GetDictionary("luxisr").StringWidth(b.renderString, float64(b.Size))), b.Size}
}
