package gui

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gos"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/jota/base"
)

type Widget interface {
	Think(gui *Gui)
	Respond(eventGroup gin.EventGroup)
	Draw(region Region, style StyleStack)
	RequestedDims() Dims
}

// Most widgets will embed a ParentWidget
type ParentWidget struct {
	Children []Widget
}

func (w *ParentWidget) Think(gui *Gui) {
	for _, child := range w.Children {
		child.Think(gui)
	}
}

type ParentResponderWidget struct {
	ParentWidget
}

func (w *ParentResponderWidget) Respond(eventGroup gin.EventGroup) {
	for _, child := range w.Children {
		child.Respond(eventGroup)
	}
}

type PosWidget struct {
	Size int
	text string
}

func (p *PosWidget) Think(gui *Gui) {
	x, y := gui.sys.GetCursorPos()
	p.text = fmt.Sprintf("(%d, %d)", x, y)
}
func (p *PosWidget) Respond(eventGroup gin.EventGroup) {}
func (p *PosWidget) Draw(region Region, style StyleStack) {
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(0, 255, 0, 255)
	gl.Begin(gl.QUADS)
	x := gl.Int(region.X)
	y := gl.Int(region.Y)
	dx := gl.Int(base.GetDictionary("luxisr").StringWidth(p.text, float64(p.Size)))
	dy := gl.Int(p.Size)
	gl.Vertex2i(x, y)
	gl.Vertex2i(x, y+dy)
	gl.Vertex2i(x+dx, y+dy)
	gl.Vertex2i(x+dx, y)
	gl.End()
	base.Log().Printf("%v %v %v %v", x, y, dx, dy)
	gl.Color4ub(255, 0, 255, 255)
	base.GetDictionary("luxisr").RenderString(p.text, float64(region.X), float64(region.Y), 0, float64(p.Size), gui.Left)
}
func (p *PosWidget) RequestedDims() Dims {
	return Dims{int(base.GetDictionary("luxisr").StringWidth(p.text, float64(p.Size))), p.Size}
}

type VTable struct {
	ParentResponderWidget
	maxWidth int
}

func (t *VTable) Think(gui *Gui) {
	t.ParentWidget.Think(gui)
	t.maxWidth = 0
	for _, child := range t.Children {
		dims := child.RequestedDims()
		if dims.Dx > t.maxWidth {
			t.maxWidth = dims.Dx
		}
	}
}
func (t *VTable) RequestedDims() Dims {
	return Dims{t.maxWidth, 30 * len(t.Children)}
}
func (t *VTable) Draw(region Region, style StyleStack) {
	dims := Dims{t.maxWidth, 30}
	for i, child := range t.Children {
		pos := Pos{region.X, region.Y + 30*i}
		child.Draw(Region{Pos: pos, Dims: dims}, style)
	}
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
	sys    system.System
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
	g.sys = system.Make(gos.GetSystemInterface())
	gin.In().RegisterEventListener(&g)
	return &g
}

func (g *Gui) StopEventListening() {
	gin.In().UnregisterEventListener(g)
}

func (g *Gui) RestartEventListening() {
	gin.In().RegisterEventListener(g)
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
	gl.Ortho(gl.Double(g.region.X), gl.Double(g.region.X+g.region.Dx), gl.Double(g.region.Y+g.region.Dy), gl.Double(g.region.Y), 1000, -1000)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	g.root.Draw(g.region, &styleStack{})
}
func (g *Gui) Think() {
	g.root.Think(g)
}
func (g *Gui) HandleEventGroup(eventGroup gin.EventGroup) {
	g.root.Respond(eventGroup)
}

type RootWidget struct {
	region   Region
	children []childPlacement
}

func (r *RootWidget) Think(gui *Gui) {
	for _, cp := range r.children {
		cp.child.Think(gui)
	}
}
func (r *RootWidget) Respond(eventGroup gin.EventGroup) {
	for _, cp := range r.children {
		cp.child.Respond(eventGroup)
	}
}
func (r *RootWidget) Draw(region Region, style StyleStack) {
	for _, cp := range r.children {
		dims := cp.child.RequestedDims()
		switch cp.place {
		case AnchorUL:
			cp.child.Draw(Region{Pos: Pos{0, 0}, Dims: dims}, style)
		case AnchorUR:
			cp.child.Draw(Region{Pos: Pos{r.region.Dx - dims.Dx, 0}, Dims: dims}, style)
		case AnchorLL:
			cp.child.Draw(Region{Pos: Pos{0, r.region.Dy - dims.Dy}, Dims: dims}, style)
		case AnchorLR:
			cp.child.Draw(Region{Pos: Pos{r.region.Dx - dims.Dx, r.region.Dy - dims.Dy}, Dims: dims}, style)
		case AnchorDeadCenter:
			cp.child.Draw(Region{Pos: Pos{r.region.X + (r.region.Dx-dims.Dx)/2, r.region.Y + (r.region.Dy-dims.Dy)/2}, Dims: dims}, style)
		}
	}
}
func (r *RootWidget) RequestedDims() Dims {
	return r.region.Dims
}

type Box struct {
	Color [4]int
	Dims  Dims
	Last  Region
	Hover bool
}

func (b *Box) Think(gui *Gui) {
	x, y := gui.sys.GetCursorPos()
	b.Hover = x >= b.Last.X && x < b.Last.X+b.Last.Dx &&
		y >= b.Last.Y && y < b.Last.Y+b.Last.Dy
}
func (b *Box) Respond(eventGroup gin.EventGroup) {}
func (b *Box) Draw(region Region, style StyleStack) {
	b.Last = region
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.TEXTURE_2D)
	if b.Hover {
		gl.Color4ub(gl.Ubyte(b.Color[0]), gl.Ubyte(b.Color[1]), gl.Ubyte(b.Color[2]), gl.Ubyte(b.Color[3]))
	} else {
		gl.Color4ub(gl.Ubyte(b.Color[0]), gl.Ubyte(b.Color[1]), gl.Ubyte(b.Color[2]), gl.Ubyte(b.Color[3])/2)
	}
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
