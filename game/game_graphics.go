// +build !nographics

package game

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/jota/base"
	g2 "github.com/runningwild/jota/gui"
	"time"
	// "github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
	"path/filepath"
)

type cameraInfo struct {
	regionDims linear.Vec2
	// Camera positions.  target is used for the invaders so that the camera can
	// follow the players without being too jerky.
	current, target struct {
		mid, dims linear.Vec2
	}
	zoom         float64
	cursorHidden bool
}

func (camera *cameraInfo) FocusRegion(g *Game, side int) {
	min := linear.Vec2{1e9, 1e9}
	max := linear.Vec2{-1e9, -1e9}
	player := g.Ents[g.local.Gid]
	if player == nil {
		min.X = 0
		min.Y = 0
		max.X = float64(g.Level.Room.Dx)
		max.Y = float64(g.Level.Room.Dy)
	} else {
		min.X = player.Pos().X - player.Stats().Vision()
		min.Y = player.Pos().Y - player.Stats().Vision()
		if min.X < 0 {
			min.X = 0
		}
		if min.Y < 0 {
			min.Y = 0
		}
		max.X = player.Pos().X + player.Stats().Vision()
		max.Y = player.Pos().Y + player.Stats().Vision()
		if max.X > float64(g.Level.Room.Dx) {
			max.X = float64(g.Level.Room.Dx)
		}
		if max.Y > float64(g.Level.Room.Dy) {
			max.Y = float64(g.Level.Room.Dy)
		}
	}

	mid := min.Add(max).Scale(0.5)
	dims := max.Sub(min)
	if dims.X/dims.Y < camera.regionDims.X/camera.regionDims.Y {
		dims.X = dims.Y * camera.regionDims.X / camera.regionDims.Y
	} else {
		dims.Y = dims.X * camera.regionDims.Y / camera.regionDims.X
	}
	camera.target.dims = dims
	camera.target.mid = mid

	camera.approachTarget()
}

// zoom == 0 fits the level exactly into the viewing region
func (camera *cameraInfo) StandardRegion(pos linear.Vec2, levelDims linear.Vec2) {
	mid := pos
	dims := levelDims
	if dims.X/dims.Y < camera.regionDims.X/camera.regionDims.Y {
		dims.X = dims.Y * camera.regionDims.X / camera.regionDims.Y
	} else {
		dims.Y = dims.X * camera.regionDims.Y / camera.regionDims.X
	}
	camera.target.dims = dims
	camera.target.mid = mid

	camera.approachTarget()
}

func (camera *cameraInfo) approachTarget() {
	if camera.current.mid.X == 0 && camera.current.mid.Y == 0 {
		// On the very first frame the current midpoint will be (0,0), which should
		// never happen after the game begins.  In this one case we'll immediately
		// set current to target so we don't start off by approaching it from the
		// origin.
		camera.current = camera.target
	} else {
		// speed is in (0, 1), the higher it is, the faster current approaches target.
		speed := 0.1
		camera.current.dims = camera.current.dims.Scale(1 - speed).Add(camera.target.dims.Scale(speed))
		camera.current.mid = camera.current.mid.Scale(1 - speed).Add(camera.target.mid.Scale(speed))
	}
}

func (g *Game) SetSystem(sys system.System) {
	g.editor.SetSystem(sys)
}

func (g *Game) SetEngine(engine *cgf.Engine) {
	g.local.Engine = engine
	if control.up == nil {
		hatUp := gin.In().GetKeyFlat(gin.ControllerHatSwitchUp, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.hat.up = gin.In().BindDerivedKey(
			"menuUp",
			gin.In().MakeBinding(hatUp.Id(), nil, nil),
			gin.In().MakeBinding(gin.AnyKeyW, nil, nil),
			gin.In().MakeBinding(gin.AnyUp, nil, nil))

		hatDown := gin.In().GetKeyFlat(gin.ControllerHatSwitchDown, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.hat.down = gin.In().BindDerivedKey(
			"menuDown",
			gin.In().MakeBinding(hatDown.Id(), nil, nil),
			gin.In().MakeBinding(gin.AnyKeyS, nil, nil),
			gin.In().MakeBinding(gin.AnyDown, nil, nil))

		hatLeft := gin.In().GetKeyFlat(gin.ControllerHatSwitchLeft, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.hat.left = gin.In().BindDerivedKey(
			"menuLeft",
			gin.In().MakeBinding(hatLeft.Id(), nil, nil),
			gin.In().MakeBinding(gin.AnyKeyA, nil, nil),
			gin.In().MakeBinding(gin.AnyLeft, nil, nil))

		hatRight := gin.In().GetKeyFlat(gin.ControllerHatSwitchRight, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.hat.right = gin.In().BindDerivedKey(
			"menuRight",
			gin.In().MakeBinding(hatRight.Id(), nil, nil),
			gin.In().MakeBinding(gin.AnyKeyD, nil, nil),
			gin.In().MakeBinding(gin.AnyRight, nil, nil))

		control.hat.enter = gin.In().BindDerivedKey(
			"menuEnter",
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+0, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+1, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+3, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+4, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+5, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+6, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+7, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+8, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.In().GetKeyFlat(gin.ControllerButton0+9, gin.DeviceTypeController, gin.DeviceIndexAny).Id(), nil, nil),
			gin.In().MakeBinding(gin.AnyReturn, nil, nil),
		)

		// TODO: This is thread-safe, don't worry, but it is dumb.
		controllerUp := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+1, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.up = gin.In().BindDerivedKey("upKey", gin.In().MakeBinding(controllerUp.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyW, nil, nil))
		controllerDown := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+1, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.down = gin.In().BindDerivedKey("downKey", gin.In().MakeBinding(controllerDown.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyS, nil, nil))
		controllerLeft := gin.In().GetKeyFlat(gin.ControllerAxis0Negative, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.left = gin.In().BindDerivedKey("leftKey", gin.In().MakeBinding(controllerLeft.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyA, nil, nil))
		controllerRight := gin.In().GetKeyFlat(gin.ControllerAxis0Positive, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.right = gin.In().BindDerivedKey("rightKey", gin.In().MakeBinding(controllerRight.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyD, nil, nil))
		control.any = gin.In().BindDerivedKey(
			"any",
			gin.In().MakeBinding(control.up.Id(), nil, nil),
			gin.In().MakeBinding(control.down.Id(), nil, nil),
			gin.In().MakeBinding(control.left.Id(), nil, nil),
			gin.In().MakeBinding(control.right.Id(), nil, nil))

		control.editor = gin.In().GetKey(gin.AnyKeyE)
	}

	// TODO: Unregister this at some point, nub
	gin.In().RegisterEventListener(GameEventHandleWrapper{g})
}

func (game *Game) HandleEventGroupSetup(group gin.EventGroup) {
	if found, event := group.FindEvent(control.hat.up.Id()); found && event.Type == gin.Press {
		game.Setup.local.Lock()
		defer game.Setup.local.Unlock()
		if game.Setup.local.Index > 0 {
			game.Setup.local.Index--
		}
		return
	}
	if found, event := group.FindEvent(control.hat.down.Id()); found && event.Type == gin.Press {
		game.Setup.local.Lock()
		defer game.Setup.local.Unlock()
		if game.Setup.local.Index < len(game.Setup.EngineIds) {
			game.Setup.local.Index++
		}
		return
	}

	if found, event := group.FindEvent(control.hat.left.Id()); found && event.Type == gin.Press {
		game.local.Engine.ApplyEvent(SetupChampSelect{game.local.Engine.Id(), -1})
		return
	}
	if found, event := group.FindEvent(control.hat.right.Id()); found && event.Type == gin.Press {
		game.local.Engine.ApplyEvent(SetupChampSelect{game.local.Engine.Id(), 1})
		return
	}
	if found, event := group.FindEvent(control.hat.enter.Id()); found && event.Type == gin.Press {
		game.Setup.local.Lock()
		defer game.Setup.local.Unlock()
		if game.Setup.local.Index < len(game.Setup.EngineIds) {
			id := game.Setup.EngineIds[game.Setup.local.Index]
			side := (game.Setup.Players[id].Side + 1) % 2
			game.local.Engine.ApplyEvent(SetupChangeSides{id, side})
		} else {
			if game.local.Engine.Id() == game.Manager || game.local.Engine.IsHost() {
				game.local.Engine.ApplyEvent(SetupComplete{time.Now().UnixNano()})
			}
		}
		return
	}
}

// Because we don't want Think() to be called by both cgf and gin, we put a
// wrapper around Game so that the Think() method called by gin is caught and
// is just a nop.
type GameEventHandleWrapper struct {
	*Game
}

func (GameEventHandleWrapper) Think() {}

func (g *Game) HandleEventGroupGame(group gin.EventGroup) {
	g.local.RLock()
	defer g.local.RUnlock()

	if found, event := group.FindEvent(control.editor.Id()); found && event.Type == gin.Press {
		g.editor.Toggle()
		return
	}

	if found, _ := group.FindEvent(control.any.Id()); found {
		dir := getControllerDirection(gin.DeviceId{gin.DeviceTypeController, gin.DeviceIndexAny})
		g.local.Engine.ApplyEvent(&Move{
			Gid:       g.local.Gid,
			Angle:     dir.Angle(),
			Magnitude: dir.Mag(),
		})
	}

	// ability0Key := gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, gin.DeviceIndexAny)
	// abilityTrigger := gin.In().GetKeyFlat(gin.ControllerButton0+1, gin.DeviceTypeController, gin.DeviceIndexAny)
	buttons := []gin.Key{
		gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, gin.DeviceIndexAny),
		gin.In().GetKeyFlat(gin.ControllerButton0+3, gin.DeviceTypeController, gin.DeviceIndexAny),
		gin.In().GetKeyFlat(gin.ControllerButton0+4, gin.DeviceTypeController, gin.DeviceIndexAny),
		gin.In().GetKeyFlat(gin.ControllerButton0+5, gin.DeviceTypeController, gin.DeviceIndexAny),
	}
	abilityTrigger := gin.In().GetKeyFlat(gin.ControllerButton0+6, gin.DeviceTypeController, gin.DeviceIndexAny)
	for i, button := range buttons {
		foundButton, _ := group.FindEvent(button.Id())
		foundTrigger, triggerEvent := group.FindEvent(abilityTrigger.Id())
		// TODO: Check if any abilities are Active before sending events to other abilities.
		if foundButton || foundTrigger {
			g.local.Engine.ApplyEvent(UseAbility{
				Gid:     g.local.Gid,
				Index:   i,
				Button:  button.CurPressAmt(),
				Trigger: foundTrigger && triggerEvent.Type == gin.Press,
			})
		}
	}
}

func (g *Game) HandleEventGroup(group gin.EventGroup) {
	g.local.Engine.Pause()
	defer g.local.Engine.Unpause()
	switch {
	case g.Setup != nil:
		g.HandleEventGroupSetup(group)
	case g.editor.Active():
		g.HandleEventGroupEditor(group)
	case !g.editor.Active():
		g.HandleEventGroupGame(group)
	default:
		base.Error().Printf("Unexpected case in HandleEventGroup()")
	}
}

func axisControl(v float64) float64 {
	floor := 0.1
	if v < floor {
		return 0.0
	}
	v = (v - floor) / (1.0 - floor)
	v *= v
	return v
}

var control struct {
	hat struct {
		up, down, left, right, enter gin.Key
	}
	any, up, down, left, right gin.Key

	// Debug/Dev mode
	editor gin.Key
}

// Queries the input system for the direction that this controller is moving in
func getControllerDirection(controller gin.DeviceId) linear.Vec2 {
	v := linear.Vec2{
		axisControl(control.right.CurPressAmt()) - axisControl(control.left.CurPressAmt()),
		axisControl(control.down.CurPressAmt()) - axisControl(control.up.CurPressAmt()),
	}
	if v.Mag2() > 1 {
		v = v.Norm()
	}
	return v
}

func (g *Game) RenderLocalSetup(region g2.Region) {
	g.Setup.local.RLock()
	defer g.Setup.local.RUnlock()
	dict := base.GetDictionary("luxisr")
	size := 60.0
	y := 100.0
	dict.RenderString("Engines:", size, y, 0, size, gui.Left)
	for i, id := range g.Setup.EngineIds {
		y += size
		if id == g.local.Engine.Id() {
			gui.SetFontColor(0.7, 0.7, 1, 1)
		} else {
			gui.SetFontColor(0.7, 0.7, 0.7, 1)
		}
		dataStr := fmt.Sprintf("Engine %d, Side %d, %s", id, g.Setup.Players[id].Side, g.Champs[g.Setup.Players[id].ChampIndex].Name)
		dict.RenderString(dataStr, size, y, 0, size, gui.Left)
		if g.IsManaging() && i == g.Setup.local.Index {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
	y += size
	gui.SetFontColor(0.7, 0.7, 0.7, 1)
	if g.IsManaging() {
		dict.RenderString("Start!", size, y, 0, size, gui.Left)
		if g.Setup.local.Index == len(g.Setup.EngineIds) {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
}

func (g *Game) RenderLosMask() {
	ent := g.Ents[g.local.Gid]
	if ent == nil {
		return
	}
	walls := g.local.temp.VisibleWallCache.GetWalls(int(ent.Pos().X), int(ent.Pos().Y))
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(0, 0, 0, 255)
	gl.Begin(gl.TRIANGLES)
	for _, wall := range walls {
		if wall.Right(ent.Pos()) {
			continue
		}
		a := wall.P
		b := ent.Pos().Sub(wall.P).Norm().Scale(-10000.0).Add(wall.P)
		mid := wall.P.Add(wall.Q).Scale(0.5)
		c := ent.Pos().Sub(mid).Norm().Scale(-10000.0).Add(mid)
		d := ent.Pos().Sub(wall.Q).Norm().Scale(-10000.0).Add(wall.Q)
		e := wall.Q
		gl.Vertex2d(gl.Double(a.X), gl.Double(a.Y))
		gl.Vertex2d(gl.Double(b.X), gl.Double(b.Y))
		gl.Vertex2d(gl.Double(c.X), gl.Double(c.Y))

		gl.Vertex2d(gl.Double(a.X), gl.Double(a.Y))
		gl.Vertex2d(gl.Double(c.X), gl.Double(c.Y))
		gl.Vertex2d(gl.Double(d.X), gl.Double(d.Y))

		gl.Vertex2d(gl.Double(a.X), gl.Double(a.Y))
		gl.Vertex2d(gl.Double(d.X), gl.Double(d.Y))
		gl.Vertex2d(gl.Double(e.X), gl.Double(e.Y))
	}
	gl.End()
	base.EnableShader("horizon")
	base.SetUniformV2("horizon", "center", ent.Pos())
	base.SetUniformF("horizon", "horizon", float32(ent.Stats().Vision()))
	gl.Begin(gl.QUADS)
	dx := gl.Int(g.Level.Room.Dx)
	dy := gl.Int(g.Level.Room.Dy)
	gl.Vertex2i(0, 0)
	gl.Vertex2i(dx, 0)
	gl.Vertex2i(dx, dy)
	gl.Vertex2i(0, dy)
	gl.End()
	base.EnableShader("")
}

func (g *Game) renderWalls() {
	gl.Color4d(1, 1, 1, 1)
	var expandedPoly linear.Poly
	for _, poly := range g.Level.Room.Walls {
		// Don't draw counter-clockwise polys, specifically this means don't draw
		// the boundary of the level.
		if poly.IsCounterClockwise() {
			continue
		}
		// KLUDGE: This will expand the polygon slightly so that it actually shows
		// up when the los shadows are drawn over it.  Eventually there should be
		// separate los polys, colision polys, and draw polys so that this isn't
		// necessary.
		gl.Begin(gl.TRIANGLE_FAN)
		expandPoly(poly, &expandedPoly)
		for _, v := range expandedPoly {
			gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		}
		gl.End()
	}
}

func (g *Game) renderEdges() {
	// Draw edges between nodes
	for _, ent := range g.Ents {
		cp0, ok := ent.(*ControlPoint)
		if !ok {
			continue
		}
		for _, target := range cp0.Targets {
			cp1, ok := g.Ents[target].(*ControlPoint)
			if !ok {
				continue
			}
			ally := 0
			enemy := 0
			if cp0.Side() == g.local.Side {
				ally++
			} else if cp0.Side() == -1 {
				enemy++
			}
			if cp1.Side() == g.local.Side {
				ally++
			} else if cp1.Side() == -1 {
				enemy++
			}
			if ally == 2 {
				gl.Color4ub(0, 255, 0, 255)
			} else if enemy == 2 {
				gl.Color4ub(255, 0, 0, 255)
			} else if ally == 1 {
				gl.Color4ub(255, 255, 0, 255)
			} else if enemy == 1 {
				gl.Color4ub(255, 0, 0, 255)
			} else {
				gl.Color4ub(200, 200, 200, 255)
			}
			gl.Begin(gl.LINES)
			gl.Vertex2d(gl.Double(cp0.Pos().X), gl.Double(cp0.Pos().Y))
			gl.Vertex2d(gl.Double(cp1.Pos().X), gl.Double(cp1.Pos().Y))
			gl.End()
		}
	}
}

func (g *Game) renderBases() {
	gui.SetFontColor(0, 255, 0, 255)
	for side, data := range g.Level.Room.SideData {
		base.GetDictionary("luxisr").RenderString(fmt.Sprintf("S%d", side), data.Base.X, data.Base.Y, 0, 100, gui.Center)
	}
}

func (g *Game) renderEntsAndAbilities() {
	gl.Color4d(1, 1, 1, 1)
	for _, ent := range g.local.temp.AllEnts {
		ent.Draw(g)
		for _, ab := range ent.Abilities() {
			ab.Draw(ent, g)
		}
	}
}

func (g *Game) renderProcesses() {
	for _, proc := range g.Processes {
		proc.Draw(Gid(""), g.local.Gid, g)
	}
}

func (g *Game) RenderLocalGame(region g2.Region) {
	g.local.Camera.regionDims = linear.Vec2{float64(region.Dims.Dx), float64(region.Dims.Dy)}
	// func (g *Game) renderLocalHelper(region g2.Region, local *LocalData, camera *cameraInfo, side int) {
	g.local.Camera.FocusRegion(g, 0)
	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	// Set the viewport so that we only render into the region that we're supposed
	// to render to.
	// TODO: Check if this works on all graphics cards - I've heard that the opengl
	// spec doesn't actually require that viewport does any clipping.
	gl.PushAttrib(gl.VIEWPORT_BIT)
	gl.Viewport(gl.Int(region.X), gl.Int(region.Y), gl.Sizei(region.Dx), gl.Sizei(region.Dy))
	defer gl.PopAttrib()

	current := &g.local.Camera.current
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

	level := g.Level
	zoom := current.dims.X / float64(region.Dims.Dx)
	level.ManaSource.Draw(zoom, float64(level.Room.Dx), float64(level.Room.Dy))

	g.renderWalls()
	g.renderEdges()
	g.renderBases()
	g.renderEntsAndAbilities()
	g.renderProcesses()
	g.RenderLosMask()
}

// Draws everything that is relevant to the players on a computer, but not the
// players across the network.  Any ui used to determine how to place an object
// or use an ability, for example.
func (g *Game) RenderLocal(region g2.Region) {
	switch {
	case g.Setup != nil:
		g.RenderLocalSetup(region)
	case !g.editor.Active():
		g.RenderLocalGame(region)
	case g.editor.Active():
		g.RenderLocalEditor(region)
	default:
		base.Error().Printf("Unexpected case.")
	}
}

func (p *PlayerEnt) Draw(game *Game) {
	var t *texture.Data
	var alpha gl.Ubyte
	if game.local.Side == p.Side() {
		alpha = gl.Ubyte(255.0 * (1.0 - p.Stats().Cloaking()/2))
	} else {
		alpha = gl.Ubyte(255.0 * (1.0 - p.Stats().Cloaking()))
	}
	gl.Color4ub(255, 255, 255, alpha)
	// if p.Id() == 1 {
	t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
	// } else if p.Id() == 2 {
	// 	t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship3.png"))
	// } else {
	// 	t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship2.png"))
	// }
	t.RenderAdvanced(
		p.Position.X-float64(t.Dx())/2,
		p.Position.Y-float64(t.Dy())/2,
		float64(t.Dx()),
		float64(t.Dy()),
		p.Angle_,
		false)

	for _, proc := range p.Processes {
		proc.Draw(p.Id(), game.local.Gid, game)
	}
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.08)
	base.SetUniformF("status_bar", "outer", 0.09)
	base.SetUniformF("status_bar", "buffer", 0.01)

	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(125, 125, 125, alpha/2)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)

	health_frac := float32(p.Stats().HealthCur() / p.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, alpha)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, alpha)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (cp *ControlPoint) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.0)
	base.SetUniformF("status_bar", "outer", 0.5)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(50, 50, 50, 50)
	texture.Render(
		cp.Position.X-cp.Radius,
		cp.Position.Y-cp.Radius,
		2*cp.Radius,
		2*cp.Radius)

	enemyColor := []gl.Ubyte{255, 0, 0, 100}
	allyColor := []gl.Ubyte{0, 255, 0, 100}
	neutralColor := []gl.Ubyte{100, 100, 100, 100}
	var rgba []gl.Ubyte
	if cp.Controlled {
		if g.local.Side == cp.Controller {
			rgba = allyColor
		} else {
			rgba = enemyColor
		}
	} else {
		rgba = neutralColor
	}

	// The texture is flipped if this is being drawn for the controlling side.
	// This makes it look a little nicer when someone neutralizes a control point
	// because it makes the angle of the pie slice thingy continue going in the
	// same direction as it passes the neutralization point.
	gl.Color4ub(rgba[0], rgba[1], rgba[2], rgba[3])
	base.SetUniformF("status_bar", "frac", float32(cp.Control))
	texture.RenderAdvanced(
		cp.Position.X-cp.Radius,
		cp.Position.Y-cp.Radius,
		2*cp.Radius,
		2*cp.Radius,
		0,
		g.local.Side == cp.Controller)

	base.SetUniformF("status_bar", "inner", 0.45)
	base.SetUniformF("status_bar", "outer", 0.5)
	base.SetUniformF("status_bar", "frac", 1)
	gl.Color4ub(rgba[0], rgba[1], rgba[2], 255)
	texture.RenderAdvanced(
		cp.Position.X-cp.Radius,
		cp.Position.Y-cp.Radius,
		2*cp.Radius,
		2*cp.Radius,
		0,
		g.local.Side == cp.Controller)

	if !cp.Controlled {
		base.SetUniformF("status_bar", "frac", float32(cp.Control))
		if g.local.Side == cp.Controller {
			gl.Color4ub(allyColor[0], allyColor[1], allyColor[2], 255)
		} else {
			gl.Color4ub(enemyColor[0], enemyColor[1], enemyColor[2], 255)
		}
		texture.RenderAdvanced(
			cp.Position.X-cp.Radius,
			cp.Position.Y-cp.Radius,
			2*cp.Radius,
			2*cp.Radius,
			0,
			g.local.Side == cp.Controller)
	}

	base.EnableShader("")
}

func (m *HeatSeeker) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(m.Position.X-100, m.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	health_frac := float32(m.Stats().HealthCur() / m.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(m.Position.X-100, m.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (m *Mine) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(m.Position.X-100, m.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	health_frac := float32(m.Stats().HealthCur() / m.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(m.Position.X-100, m.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (c *CreepEnt) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	if c.Side() == g.local.Side {
		gl.Color4ub(100, 255, 100, 255)
	} else {
		gl.Color4ub(255, 100, 100, 255)
	}
	texture.Render(c.Position.X-100, c.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	texture.Render(c.Position.X-100, c.Position.Y-100, 200, 200)
	base.EnableShader("")
}

type manaSourceLocalData struct {
	nodeTextureId   gl.Uint
	nodeTextureData []byte
}

func (ms *ManaSource) Draw(zoom float64, dx float64, dy float64) {
	if ms.local.nodeTextureData == nil {
		//		gl.Enable(gl.TEXTURE_2D)
		ms.local.nodeTextureData = make([]byte, ms.options.NumNodeRows*ms.options.NumNodeCols*3)
		gl.GenTextures(1, &ms.local.nodeTextureId)
		gl.BindTexture(gl.TEXTURE_2D, ms.local.nodeTextureId)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexImage2D(
			gl.TEXTURE_2D,
			0,
			gl.RGB,
			gl.Sizei(ms.options.NumNodeRows),
			gl.Sizei(ms.options.NumNodeCols),
			0,
			gl.RGB,
			gl.UNSIGNED_BYTE,
			gl.Pointer(&ms.local.nodeTextureData[0]))
	}
	for i := range ms.rawNodes {
		for c := 0; c < 3; c++ {
			color_frac := ms.rawNodes[i].Mana[c] * 1.0 / ms.options.NodeMagnitude
			color_range := float64(ms.options.MaxNodeBrightness - ms.options.MinNodeBrightness)
			ms.local.nodeTextureData[i*3+c] = byte(
				color_frac*color_range + float64(ms.options.MinNodeBrightness))
		}
	}
	gl.Enable(gl.TEXTURE_2D)
	//gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, ms.local.nodeTextureId)
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		gl.Sizei(ms.options.NumNodeRows),
		gl.Sizei(ms.options.NumNodeCols),
		gl.RGB,
		gl.UNSIGNED_BYTE,
		gl.Pointer(&ms.local.nodeTextureData[0]))

	base.EnableShader("nodes")
	base.SetUniformI("nodes", "width", ms.options.NumNodeRows*3)
	base.SetUniformI("nodes", "height", ms.options.NumNodeCols*3)
	base.SetUniformI("nodes", "drains", 1)
	base.SetUniformI("nodes", "tex0", 0)
	base.SetUniformI("nodes", "tex1", 1)
	base.SetUniformF("nodes", "zoom", float32(zoom))
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, ms.local.nodeTextureId)

	// I have no idea why this value for move works, but it does.  So, hooray.
	move := (dx - dy) / 2
	texture.RenderAdvanced(move, -move, dy, dx, 3.1415926535/2, true)
	base.EnableShader("")
	gl.Disable(gl.TEXTURE_2D)
}

type GameWindow struct {
	Engine *cgf.Engine
	Dims   g2.Dims
	game   *Game
}

func (gw *GameWindow) String() string {
	return "game window"
}
func (gw *GameWindow) Expandable() (bool, bool) {
	return false, false
}
func (gw *GameWindow) Requested() g2.Dims {
	return g2.Dims{800, 600}
}
func (gw *GameWindow) Think(g *g2.Gui) {
	gw.Engine.Pause()
	// gw.Engine.GetState().(*Game)
	gw.Engine.Unpause()
}
func (gw *GameWindow) Respond(group gin.EventGroup) {
}
func (gw *GameWindow) RequestedDims() g2.Dims {
	return gw.Dims
}

func (gw *GameWindow) DrawFocused(region gui.Region) {}

func (gw *GameWindow) Draw(region g2.Region, style g2.StyleStack) {
	defer base.StackCatcher()
	defer func() {
		// gl.Translated(gl.Double(gw.region.X), gl.Double(gw.region.Y), 0)
		gl.Disable(gl.TEXTURE_2D)
		gl.Color4ub(255, 255, 255, 255)
		gl.LineWidth(3)
		gl.Begin(gl.LINES)
		bx, by := gl.Int(region.X), gl.Int(region.Y)
		bdx, bdy := gl.Int(region.Dx), gl.Int(region.Dy)
		gl.Vertex2i(bx, by)
		gl.Vertex2i(bx, by+bdy)
		gl.Vertex2i(bx, by+bdy)
		gl.Vertex2i(bx+bdx, by+bdy)
		gl.Vertex2i(bx+bdx, by+bdy)
		gl.Vertex2i(bx+bdx, by)
		gl.Vertex2i(bx+bdx, by)
		gl.Vertex2i(bx, by)
		gl.End()
		gl.LineWidth(1)
	}()

	gw.Engine.Pause()
	game := gw.Engine.GetState().(*Game)
	// Note that since we do a READER lock on game.local we cannot do any writes
	// to local data while rendering.
	game.local.RLock()
	game.RenderLocal(region)
	game.local.RUnlock()
	gw.Engine.Unpause()
}
