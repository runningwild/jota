package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	g2 "github.com/runningwild/jota/gui"
	"github.com/runningwild/linear"
	"sync"
)

type editorData struct {
	sync.RWMutex

	camera cameraInfo

	// Whether or not we're in the editor right now
	active bool
}

func (editor *editorData) Active() bool {
	editor.RLock()
	defer editor.RUnlock()
	return editor.active
}

func (editor *editorData) Toggle() {
	editor.Lock()
	defer editor.Unlock()
	editor.active = !editor.active
}

func (g *Game) HandleEventGroupEditor(group gin.EventGroup) {
	g.local.RLock()
	defer g.local.RUnlock()

	if found, event := group.FindEvent(control.editor.Id()); found && event.Type == gin.Press {
		g.editor.Toggle()
		return
	}
}

func (g *Game) RenderLocalEditor(region g2.Region) {
	g.editor.camera.regionDims = linear.Vec2{float64(region.Dims.Dx), float64(region.Dims.Dy)}
	levelDims := linear.Vec2{float64(g.Level.Room.Dx), float64(g.Level.Room.Dy)}
	g.editor.camera.StandardRegion(levelDims.Scale(0.5), levelDims)
	g.editor.camera.approachTarget()

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	defer gl.PopMatrix()

	gl.PushAttrib(gl.VIEWPORT_BIT)
	gl.Viewport(gl.Int(region.X), gl.Int(region.Y), gl.Sizei(region.Dx), gl.Sizei(region.Dy))
	defer gl.PopAttrib()

	current := &g.editor.camera.current
	gl.Ortho(
		gl.Double(current.mid.X-current.dims.X/2),
		gl.Double(current.mid.X+current.dims.X/2),
		gl.Double(current.mid.Y+current.dims.Y/2),
		gl.Double(current.mid.Y-current.dims.Y/2),
		gl.Double(1000),
		gl.Double(-1000),
	)
	defer func() {
		gl.MatrixMode(gl.PROJECTION)
		gl.PopMatrix()
		gl.MatrixMode(gl.MODELVIEW)
	}()
	gl.MatrixMode(gl.MODELVIEW)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// current := &g.local.Camera.current
	// level := g.Level
	// zoom := current.dims.X / float64(region.Dims.Dx)
	// level.ManaSource.Draw(zoom, float64(level.Room.Dx), float64(level.Room.Dy))

	g.renderWalls()
	g.renderEdges()
	g.renderBases()
	g.renderEntsAndAbilities()
	g.renderProcesses()
}
