package gui

import (
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/magnus/base"
)

type Button struct {
	Name     string
	Hover    bool
	Last     Region
	Size     int
	Callback func()
	Triggers map[gin.KeyId]struct{}
}

func (b *Button) Think(gui *Gui) {
	x, y := gui.sys.GetCursorPos()
	b.Hover = x >= b.Last.X && x < b.Last.X+b.Last.Dx &&
		y >= b.Last.Y && y < b.Last.Y+b.Last.Dy
}
func (b *Button) Respond(eventGroup gin.EventGroup) {
	// Only respond to mouse events if the mouse is hovering over the button
	if !b.Hover && eventGroup.Events[0].Key.Id().Device.Type == gin.DeviceTypeMouse {
		return
	}
	if found, event := eventGroup.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
		if b.Callback != nil {
			b.Callback()
		}
		return
	}
	for trigger := range b.Triggers {
		if found, event := eventGroup.FindEvent(trigger); found && event.Type == gin.Press {
			if b.Callback != nil {
				b.Callback()
			}
			return
		}
	}
}
func (b *Button) Draw(region Region, style StyleStack) {
	b.Last = region
	selected, ok := style.Get("selected").(bool)
	if b.Hover || (ok && selected) {
		gui.SetFontColor(1, 1, 0, 1)
	} else {
		gui.SetFontColor(1, 1, 0, 0.5)
	}
	base.GetDictionary("luxisr").RenderString(b.Name, float64(region.X), float64(region.Y), 0, float64(b.Size), gui.Left)
}
func (b *Button) RequestedDims() Dims {
	return Dims{int(base.GetDictionary("luxisr").StringWidth(b.Name, float64(b.Size))), b.Size}
}
