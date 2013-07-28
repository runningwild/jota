package gui

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
)

type ThunderSubMenu struct {
	Options  []Widget
	requests map[Widget]Dims
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
	for _, option := range tsm.Options {
		region.Dy = tsm.requests[option].Dy
		option.Draw(region, style)
		region.Y += tsm.requests[option].Dy
	}
}

type ThunderMenu struct {
	Subs      map[string]*ThunderSubMenu
	menuStack []string
	inTransit float64
	request   Dims
}

func (tm *ThunderMenu) Think(gui *Gui) {
	if tm.inTransit != 0 {
		tm.inTransit *= 0.7
		threshold := 0.001
		if tm.inTransit < threshold && tm.inTransit > -threshold {
			if tm.inTransit < 0 {
				tm.menuStack = tm.menuStack[0 : len(tm.menuStack)-1]
			}
			tm.inTransit = 0
		}
		return
	}
	tsm := tm.Subs[tm.menuStack[len(tm.menuStack)-1]]
	for key := range tsm.Options {
		tsm.Options[key].Think(gui)
	}
}

func (tm *ThunderMenu) Push(target string) {
	if tm.inTransit != 0 {
		return
	}
	tm.inTransit = 1.0
	tm.menuStack = append(tm.menuStack, target)
}

func (tm *ThunderMenu) Pop() {
	if tm.inTransit != 0 {
		return
	}
	tm.inTransit = -1.0
}

func (tm *ThunderMenu) Respond(eventGroup gin.EventGroup) {
	if tm.inTransit != 0 {
		return
	}
	tsm := tm.Subs[tm.menuStack[len(tm.menuStack)-1]]
	for _, option := range tsm.Options {
		option.Respond(eventGroup)
	}
}

func (tm *ThunderMenu) Draw(region Region, style StyleStack) {
	if tm.inTransit != 0 {
		var shift float64
		if tm.inTransit < 0 {
			shift = -tm.inTransit
		} else {
			shift = 1.0 - tm.inTransit
		}
		prevRegion := region
		prevRegion.X -= int(float64(prevRegion.Dx) * shift)
		gl.Color4ub(255, 0, 0, 100)
		tm.Subs[tm.menuStack[len(tm.menuStack)-2]].Draw(prevRegion, style)
	}
	var shift float64
	if tm.inTransit < 0 {
		shift = 1.0 + tm.inTransit
	} else {
		shift = tm.inTransit
	}
	region.X += int(float64(region.Dx) * shift)
	gl.Color4ub(0, 255, 0, 100)
	tm.Subs[tm.menuStack[len(tm.menuStack)-1]].Draw(region, style)
}

func (tsm *ThunderSubMenu) RequestedDims() Dims {
	if tsm.requests == nil {
		tsm.requests = make(map[Widget]Dims)
	}
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
