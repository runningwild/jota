package game

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/jota/base"
	g2 "github.com/runningwild/jota/gui"
	"github.com/runningwild/linear"
	"io/ioutil"
	"math"
	"path/filepath"
	"sync"
	"time"
)

type editorData struct {
	sync.RWMutex

	camera cameraInfo

	// Whether or not we're in the editor right now
	active bool

	// This is really just to get mouse position
	sys system.System

	region g2.Region

	action     editorAction
	placeBlock placeBlockData
}

type editorAction int

const (
	editorActionNone editorAction = iota
	editorActionPlaceBlock
	editorActionSave
)

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
	g.editor.Lock()
	defer g.editor.Unlock()

	if found, event := group.FindEvent(control.editor.Id()); found && event.Type == gin.Press {
		// Can't call editor.Toggle() here because we already hold the lock.
		g.editor.active = false
		g.editor.action = editorActionNone
		return
	}

	if found, event := group.FindEvent(gin.AnyKeyB); found && event.Type == gin.Press {
		g.editor.placeBlockAction()
		return
	}

	if found, event := group.FindEvent(gin.AnyKeyS); found && event.Type == gin.Press {
		g.editor.saveAction(&g.Level.Room)
		return
	}

	switch g.editor.action {
	case editorActionNone:
		return
	case editorActionPlaceBlock:
		if found, event := group.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
			g.editor.placeBlockDo(g)
			return
		}
	}
}

type placeBlockEvent struct {
	Poly linear.Poly
}

func (pbe placeBlockEvent) Apply(_g interface{}) {
	g := _g.(*Game)
	g.Level.Room.Walls[string(g.NextGid())] = pbe.Poly
	g.local.Lock()
	g.local.temp.AllWallsDirty = true
	g.local.Unlock()
}
func init() {
	gob.Register(placeBlockEvent{})
}

func (editor *editorData) placeBlockDo(g *Game) {
	g.local.Engine.ApplyEvent(placeBlockEvent{editor.getPoly(g)})
}

type placeBlockData struct {
	block  linear.Poly
	offset linear.Vec2
	grid   float64
}

func (editor *editorData) getPoly(g *Game) linear.Poly {
	x, y := editor.sys.GetCursorPos()
	pos := linear.Vec2{float64(x), float64(y)}
	regionPos := linear.Vec2{float64(editor.region.Pos.X), float64(editor.region.Pos.Y)}
	pos = pos.Sub(regionPos)
	pos = pos.Scale(float64(g.Level.Room.Dx) / float64(editor.region.Dims.Dx))
	cameraOffset := linear.Vec2{
		g.editor.camera.current.dims.X/2 - g.editor.camera.current.mid.X,
		g.editor.camera.current.dims.Y/2 - g.editor.camera.current.mid.Y,
	}
	pos = pos.Sub(cameraOffset)

	var offset linear.Vec2
	offset.X = pos.X - editor.placeBlock.offset.X
	offset.X = math.Floor(offset.X/editor.placeBlock.grid+0.5) * editor.placeBlock.grid
	offset.Y = pos.Y - editor.placeBlock.offset.Y
	offset.Y = math.Floor(offset.Y/editor.placeBlock.grid+0.5) * editor.placeBlock.grid
	block := make(linear.Poly, len(editor.placeBlock.block))
	for i := range editor.placeBlock.block {
		block[i] = editor.placeBlock.block[i].Add(offset)
	}
	return block
}

func (editor *editorData) placeBlockAction() {
	if editor.action == editorActionPlaceBlock {
		editor.action = editorActionNone
		return
	}
	editor.action = editorActionPlaceBlock
	editor.placeBlock.offset = linear.Vec2{pathingDataGrid / 2, pathingDataGrid / 2}
	editor.placeBlock.block = linear.Poly{
		linear.Vec2{0, 0},
		linear.Vec2{0, pathingDataGrid},
		linear.Vec2{pathingDataGrid, pathingDataGrid},
		linear.Vec2{pathingDataGrid, 0},
	}
	editor.placeBlock.grid = pathingDataGrid
}

func (editor *editorData) renderPlaceBlock(g *Game) {
	var expandedPoly linear.Poly
	expandPoly(editor.getPoly(g), &expandedPoly)
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4d(1, 1, 1, 1)
	gl.Begin(gl.TRIANGLE_FAN)
	for _, v := range expandedPoly {
		gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
	}
	gl.End()
}

func (editor *editorData) saveAction(room *Room) {
	data, err := json.MarshalIndent(room, "", "  ")
	if err != nil {
		base.Error().Printf("Unable to encode room to json: %v", err)
		return
	}
	name := fmt.Sprintf("save-%v.json", time.Now())
	fullPath := filepath.Join(base.GetDataDir(), name)
	err = ioutil.WriteFile(fullPath, data, 0664)
	if err != nil {
		base.Warn().Printf("Unable to write output json file: %v", err)
		return
	}
}

func (g *Game) RenderLocalEditor(region g2.Region) {
	g.editor.Lock()
	defer g.editor.Unlock()
	g.editor.region = region
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

	g.renderWalls()
	g.renderEdges()
	g.renderBases()
	g.renderEntsAndAbilities()
	g.renderProcesses()

	switch g.editor.action {
	case editorActionNone:
	case editorActionPlaceBlock:
		g.editor.renderPlaceBlock(g)
	default:
		base.Error().Printf("Unexpected editorAction: %v", g.editor.action)
	}
}
