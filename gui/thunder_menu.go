package gui

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	// "github.com/runningwild/magnus/base"
	"math"
)

type ThunderSubMenu struct {
	Options  []Widget
	requests map[Widget]Dims
	selected int
	downs    []gin.KeyIndex
	ups      []gin.KeyIndex
}

func MakeThunderSubMenu(options []Widget) *ThunderSubMenu {
	var tsm ThunderSubMenu
	tsm.Options = make([]Widget, len(options))
	copy(tsm.Options, options)
	tsm.requests = make(map[Widget]Dims)
	tsm.selected = -1
	tsm.downs = []gin.KeyIndex{gin.Down, gin.ControllerHatSwitchDown}
	tsm.ups = []gin.KeyIndex{gin.Up, gin.ControllerHatSwitchUp}
	return &tsm
}

func (tsm *ThunderSubMenu) Respond(eventGroup gin.EventGroup) {
	var up, down bool
	for _, keyIndex := range tsm.downs {
		id := gin.In().GetKeyFlat(keyIndex, gin.DeviceTypeAny, gin.DeviceIndexAny).Id()
		if found, event := eventGroup.FindEvent(id); found && event.Type == gin.Press {
			down = true
		}
	}
	for _, keyIndex := range tsm.ups {
		id := gin.In().GetKeyFlat(keyIndex, gin.DeviceTypeAny, gin.DeviceIndexAny).Id()
		if found, event := eventGroup.FindEvent(id); found && event.Type == gin.Press {
			up = true
		}
	}
	if down {
		tsm.selected++
	}
	if up {
		if tsm.selected == -1 {
			tsm.selected = len(tsm.Options) - 1
		} else {
			tsm.selected--
		}
	}
	if tsm.selected >= len(tsm.Options) || tsm.selected < 0 {
		tsm.selected = -1
	}
	if eventGroup.Events[0].Key.Id().Device.Type != gin.DeviceTypeMouse {
		if tsm.selected >= 0 && tsm.selected < len(tsm.Options) {
			tsm.Options[tsm.selected].Respond(eventGroup)
		}
	} else {
		for _, option := range tsm.Options {
			option.Respond(eventGroup)
		}
	}
}

func (tsm *ThunderSubMenu) Draw(region Region, style StyleStack) {
	gl.Disable(gl.TEXTURE_2D)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)
	gl.Begin(gl.QUADS)
	x := gl.Int(region.X)
	y := gl.Int(region.Y)
	dx := gl.Int(region.Dx)
	dy := gl.Int(region.Dy)
	gl.Vertex2i(x, y)
	gl.Vertex2i(x, y+dy)
	gl.Vertex2i(x+dx, y+dy)
	gl.Vertex2i(x+dx, y)
	gl.End()
	for i, option := range tsm.Options {
		region.Dy = tsm.requests[option].Dy
		if i == tsm.selected {
			style.PushStyle(map[string]interface{}{"selected": true})
		} else {
			style.PushStyle(map[string]interface{}{"selected": false})
		}
		option.Draw(region, style)
		style.Pop()
		region.Y += tsm.requests[option].Dy
	}
}

type ThunderMenu struct {
	Subs      map[string]*ThunderSubMenu
	menuStack []string
	current   int
	delta     float64
	request   Dims
}

func (tm *ThunderMenu) Think(gui *Gui) {
	if tm.delta != 0 {
		tm.delta *= 0.7
		threshold := 0.001
		if tm.delta < threshold && tm.delta > -threshold {
			tm.delta = 0
			if tm.current < len(tm.menuStack)-1 {
				tm.menuStack = tm.menuStack[0 : len(tm.menuStack)-1]
			}
		}
	}
	tsm := tm.Subs[tm.menuStack[tm.current]]
	for key := range tsm.Options {
		tsm.Options[key].Think(gui)
	}
}

func (tm *ThunderMenu) Push(target string) {
	tm.delta -= 1.0
	tm.current++
	if len(tm.menuStack) > tm.current {
		tm.menuStack[tm.current] = target
	} else {
		tm.menuStack = append(tm.menuStack, target)
	}
}

func (tm *ThunderMenu) Pop() {
	tm.delta += 1.0
	tm.current--
}

func (tm *ThunderMenu) Respond(eventGroup gin.EventGroup) {
	tm.Subs[tm.menuStack[tm.current]].Respond(eventGroup)
}

func (tm *ThunderMenu) Draw(region Region, style StyleStack) {
	// Set clip planes
	gl.PushAttrib(gl.TRANSFORM_BIT)
	defer gl.PopAttrib()
	var eqs [4][4]gl.Double
	eqs[0][0], eqs[0][1], eqs[0][2], eqs[0][3] = 1, 0, 0, -gl.Double(region.X)
	eqs[1][0], eqs[1][1], eqs[1][2], eqs[1][3] = -1, 0, 0, gl.Double(region.X+region.Dx)
	eqs[2][0], eqs[2][1], eqs[2][2], eqs[2][3] = 0, 1, 0, -gl.Double(region.Y)
	eqs[3][0], eqs[3][1], eqs[3][2], eqs[3][3] = 0, -1, 0, gl.Double(region.Y+region.Dy)
	gl.Enable(gl.CLIP_PLANE0)
	gl.Enable(gl.CLIP_PLANE1)
	gl.Enable(gl.CLIP_PLANE2)
	gl.Enable(gl.CLIP_PLANE3)
	gl.ClipPlane(gl.CLIP_PLANE0, &eqs[0][0])
	gl.ClipPlane(gl.CLIP_PLANE1, &eqs[1][0])
	gl.ClipPlane(gl.CLIP_PLANE2, &eqs[2][0])
	gl.ClipPlane(gl.CLIP_PLANE3, &eqs[3][0])

	var start, end int
	if tm.delta <= 0 {
		start = tm.current + int(math.Floor(tm.delta))
		end = tm.current
		region.X += int(float64(region.Dx) * (float64(start-tm.current) - tm.delta))
	} else {
		start = tm.current
		end = tm.current + int(math.Ceil(tm.delta))
		region.X += int(float64(region.Dx) * (float64(end-tm.current) - tm.delta - math.Floor(tm.delta) - 1))
	}
	for i := start; i <= end; i++ {
		if i == tm.current {
			gl.Color4ub(0, 255, 0, 100)
		} else {
			gl.Color4ub(255, 0, 0, 100)
		}
		gl.Color4ub(255, 0, 0, 100)
		tm.Subs[tm.menuStack[i]].Draw(region, style)
		region.X += region.Dx
	}
}

func (tsm *ThunderSubMenu) RequestedDims() Dims {
	var dims Dims
	for _, option := range tsm.Options {
		opDims := option.RequestedDims()
		tsm.requests[option] = opDims
		if opDims.Dx > dims.Dx {
			dims.Dx = opDims.Dx
		}
		dims.Dy += opDims.Dy
	}
	return dims
}

func (tm *ThunderMenu) RequestedDims() Dims {
	tm.request.Dy = 0
	for _, sub := range tm.Subs {
		subDims := sub.RequestedDims()
		if subDims.Dy > tm.request.Dy {
			tm.request.Dy = subDims.Dy
		}
	}
	return tm.request
}

func (tm *ThunderMenu) Start(dx int) {
	tm.menuStack = []string{""}
	tm.request = Dims{Dx: dx}
}
