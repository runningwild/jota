package gui

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/magnus/base"
)

type Widget interface {
	Think()
	Respond(eventGroup gin.EventGroup)
	Draw(region Region)
	RequestedDims() Dims
}

type Dims struct {
	Dx, Dy int
}

type Pos struct {
	X, Y int
}

type Region struct {
	Pos
	Dims
}

type Gui struct {
	root   *RootWidget
	region Region
}

type Anchor int

const (
	AnchorUL Anchor = iota
	AnchorUR
	AnchorLL
	AnchorLR
	AnchorDeadCenter
)

func Make(x, y, dx, dy int) *Gui {
	var g Gui
	g.region = Region{Pos{x, y}, Dims{dx, dy}}
	g.root = &RootWidget{region: g.region}
	gin.In().RegisterEventListener(&g)
	return &g
}

type childPlacement struct {
	child Widget
	place Anchor
}

func (g *Gui) AddChild(w Widget, anchor Anchor) {
	g.root.children = append(g.root.children, childPlacement{child: w, place: anchor})
}

func (g *Gui) Draw() {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	base.Log().Printf("Ortho: %v", g.region)
	gl.Ortho(gl.Double(g.region.X), gl.Double(g.region.X+g.region.Dx), gl.Double(g.region.Y+g.region.Dy), gl.Double(g.region.Y), 1000, -1000)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	g.root.Draw(g.region)
}
func (g *Gui) Think(t int64) {
	g.root.Think()
}
func (G *Gui) HandleEventGroup(eventGroup gin.EventGroup) {}

type RootWidget struct {
	region   Region
	children []childPlacement
}

func (r *RootWidget) Think() {
	for _, cp := range r.children {
		cp.child.Think()
	}
}
func (r *RootWidget) Respond(eventGroup gin.EventGroup) {}
func (r *RootWidget) Draw(region Region) {
	for _, cp := range r.children {
		dims := cp.child.RequestedDims()
		switch cp.place {
		case AnchorUL:
			cp.child.Draw(Region{Pos: Pos{0, 0}, Dims: dims})
		case AnchorUR:
			cp.child.Draw(Region{Pos: Pos{r.region.Dx - dims.Dx, 0}, Dims: dims})
		case AnchorLL:
			cp.child.Draw(Region{Pos: Pos{0, r.region.Dy - dims.Dy}, Dims: dims})
		case AnchorLR:
			cp.child.Draw(Region{Pos: Pos{r.region.Dx - dims.Dx, r.region.Dy - dims.Dy}, Dims: dims})
		case AnchorDeadCenter:
			cp.child.Draw(Region{Pos: Pos{(r.region.Dx - dims.Dx) / 2, (r.region.Dy - dims.Dy) / 2}, Dims: dims})
		}
	}
}
func (r *RootWidget) RequestedDims() Dims {
	return r.region.Dims
}

type Box struct {
	Color [4]int
	Dims  Dims
}

func (b *Box) Think()                            {}
func (b *Box) Respond(eventGroup gin.EventGroup) {}
func (b *Box) Draw(region Region) {
	base.Log().Printf("Rendering %v", region)
	gl.Color4ub(gl.Ubyte(b.Color[0]), gl.Ubyte(b.Color[1]), gl.Ubyte(b.Color[2]), gl.Ubyte(b.Color[3]))
	gl.Begin(gl.QUADS)
	x := gl.Int(region.X)
	y := gl.Int(region.Y)
	dx := gl.Int(region.Dx)
	dy := gl.Int(region.Dy)
	gl.Vertex2i(x, y)
	gl.Vertex2i(x+dx, y)
	gl.Vertex2i(x+dx, y+dy)
	gl.Vertex2i(x, y+dy)
	gl.End()
}
func (b *Box) RequestedDims() Dims {
	return b.Dims
}

// Focus - In order for a widget to receive events it must be on the top of the
// focus stack.  Normally there is nothing on the stack so no one receives
// events.
// Questions:

// Gui methods:
// Root()  // returns the root widget
// Focus() // returns the focus stack (supports push and pop)
// Render()
// Think()  // Think goes through everything
