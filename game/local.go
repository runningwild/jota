package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	g2 "github.com/runningwild/magnus/gui"
	"github.com/runningwild/magnus/los"
	"math"
)

const LosMaxPlayers = 2
const LosMaxDist = 1000
const LosPlayerHorizon = 400

type personalAbilities struct {
	// All of the abilities that this player can activate.
	abilities []Ability

	// This player's active ability, if any.
	activeAbility Ability
}

type localPlayer struct {
	// This player's gid
	gid Gid

	// The device controlling this player.
	deviceIndex gin.DeviceIndex

	abs personalAbilities
}

type cameraInfo struct {
	regionPos  linear.Vec2
	regionDims linear.Vec2
	// Camera positions.  target is used for the invaders so that the camera can
	// follow the players without being too jerky.  limit is used by the architect
	// so we have a constant reference point.
	current, target, limit struct {
		mid, dims linear.Vec2
	}
	zoom         float64
	cursorHidden bool
}

type localMobaData struct {
	camera cameraInfo
}

type localInvadersData struct {
	camera cameraInfo
}

type localArchitectData struct {
	camera cameraInfo

	abs personalAbilities

	//DEPRECATED:
	place linear.Poly
}

type viewMode int

const (
	viewArchitect viewMode = iota
	viewInvaders
	viewEditor
)

type LocalData struct {
	// The engine running this game, so that the game can apply events to itself.
	engine *cgf.Engine

	mode localMode

	// All of the players controlled by humans on localhost.
	players []*localPlayer

	los struct {
		texData    [][]uint32
		texRawData []uint32
		texId      gl.Uint
	}
	back struct {
		texData    [][]uint32
		texRawData []uint32
		texId      gl.Uint
	}

	sys       system.System
	architect localArchitectData
	invaders  localInvadersData
	moba      localMobaData

	// For displaying the mana grid
	nodeTextureId      gl.Uint
	nodeTextureData    []byte
	nodeWarpingTexture gl.Uint
	nodeWarpingData    []byte
}

func (l *LocalData) DebugSwapRoles() {
	switch l.mode {
	case localModeArchitect:
		l.mode = localModeInvaders
	case localModeInvaders:
		l.mode = localModeArchitect
	}
}

type gameResponderWrapper struct {
	l *LocalData
}

func (grw *gameResponderWrapper) HandleEventGroup(group gin.EventGroup) {
	grw.l.HandleEventGroup(group)
}

func (grw *gameResponderWrapper) Think(int64) {}

type localMode int

const (
	localModeInvaders localMode = iota
	localModeArchitect
	localModeMoba
)

func NewLocalDataMoba(engine *cgf.Engine, sys system.System) *LocalData {
	return newLocalDataHelper(engine, sys, localModeMoba)
}

func NewLocalDataInvaders(engine *cgf.Engine, sys system.System) *LocalData {
	return newLocalDataHelper(engine, sys, localModeInvaders)
}

func NewLocalDataArchitect(engine *cgf.Engine, sys system.System) *LocalData {
	return newLocalDataHelper(engine, sys, localModeArchitect)
}

func newLocalDataHelper(engine *cgf.Engine, sys system.System, mode localMode) *LocalData {
	var local LocalData
	if local.engine != nil {
		base.Error().Fatalf("Engine has already been set.")
	}
	local.engine = engine
	local.mode = mode
	if mode == localModeArchitect {
		// local.architect.abs.abilities =
		// 	append(
		// 		local.architect.abs.abilities,
		// 		ability_makers["placePoly"](map[string]int{"wall": 1}))
		// local.architect.abs.abilities =
		// 	append(
		// 		local.architect.abs.abilities,
		// 		ability_makers["placePoly"](map[string]int{"lava": 1}))
		// local.architect.abs.abilities =
		// 	append(
		// 		local.architect.abs.abilities,
		// 		ability_makers["placePoly"](map[string]int{"pests": 1}))
		// local.architect.abs.abilities = append(local.architect.abs.abilities, ability_makers["removePoly"](nil))
	}
	local.sys = sys
	gin.In().RegisterEventListener(&gameResponderWrapper{&local})

	local.los.texRawData = make([]uint32, los.Resolution*LosMaxPlayers)
	local.los.texData = make([][]uint32, LosMaxPlayers)
	for i := range local.los.texData {
		start := i * los.Resolution
		end := (i + 1) * los.Resolution
		local.los.texData[i] = local.los.texRawData[start:end]
	}
	render.Queue(func() {
		gl.GenTextures(1, &local.los.texId)
		gl.BindTexture(gl.TEXTURE_2D, local.los.texId)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexImage2D(
			gl.TEXTURE_2D,
			0,
			gl.ALPHA,
			los.Resolution,
			LosMaxPlayers,
			0,
			gl.ALPHA,
			gl.UNSIGNED_INT,
			gl.Pointer(&local.los.texRawData[0]))
	})

	local.back.texRawData = make([]uint32, los.Resolution*LosMaxPlayers)
	local.back.texData = make([][]uint32, LosMaxPlayers)
	for i := range local.back.texData {
		start := i * los.Resolution
		end := (i + 1) * los.Resolution
		local.back.texData[i] = local.back.texRawData[start:end]
	}
	render.Queue(func() {
		gl.GenTextures(1, &local.back.texId)
		gl.BindTexture(gl.TEXTURE_2D, local.back.texId)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	})
	return &local
}

func (g *Game) renderLosMask(local *LocalData) {
	base.EnableShader("los")
	gl.Enable(gl.TEXTURE_2D)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, local.los.texId)
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		los.Resolution,
		LosMaxPlayers,
		gl.ALPHA,
		gl.UNSIGNED_INT,
		gl.Pointer(&local.los.texRawData[0]))
	base.SetUniformI("los", "tex0", 0)
	// TODO: This has to not be hardcoded
	base.SetUniformF("los", "dx", float32(g.Levels[GidInvadersStart].Room.Dx))
	base.SetUniformF("los", "dy", float32(g.Levels[GidInvadersStart].Room.Dy))
	base.SetUniformF("los", "losMaxDist", LosMaxDist)
	base.SetUniformF("los", "losResolution", los.Resolution)
	base.SetUniformF("los", "losMaxPlayers", LosMaxPlayers)
	if local.mode == localModeArchitect {
		base.SetUniformI("los", "architect", 1)
	} else {
		base.SetUniformI("los", "architect", 0)
	}
	var playerPos []linear.Vec2
	g.DoForEnts(func(gid Gid, ent Ent) {
		if _, ok := ent.(*Player); ok {
			playerPos = append(playerPos, ent.Pos())
			// base.Log().Printf("Los(%d): %v", gid, ent.(*Player).Los.WriteDepthBuffer(dst, maxDist))
		}
	})
	base.SetUniformV2Array("los", "playerPos", playerPos)
	base.SetUniformI("los", "losNumPlayers", len(playerPos))
	gl.Color4d(0, 0, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0, 1)
	gl.Vertex2i(0, 0)
	gl.TexCoord2d(0, 0)
	gl.Vertex2i(0, gl.Int(g.Levels[GidInvadersStart].Room.Dy))
	gl.TexCoord2d(1, 0)
	gl.Vertex2i(gl.Int(g.Levels[GidInvadersStart].Room.Dx), gl.Int(g.Levels[GidInvadersStart].Room.Dy))
	gl.TexCoord2d(1, 1)
	gl.Vertex2i(gl.Int(g.Levels[GidInvadersStart].Room.Dx), 0)
	gl.End()
	base.EnableShader("")
}

func (g *Game) renderLocalInvaders(region g2.Region, local *LocalData) {
	local.doInvadersFocusRegion(g)
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

	current := local.invaders.camera.current
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

	level := g.Levels[GidInvadersStart]
	level.ManaSource.Draw(local, float64(level.Room.Dx), float64(level.Room.Dy))

	gl.Begin(gl.LINES)
	gl.Color4d(1, 1, 1, 1)
	for _, poly := range g.Levels[GidInvadersStart].Room.Walls {
		for i := range poly {
			seg := poly.Seg(i)
			gl.Vertex2d(gl.Double(seg.P.X), gl.Double(seg.P.Y))
			gl.Vertex2d(gl.Double(seg.Q.X), gl.Double(seg.Q.Y))
		}
	}
	gl.End()

	gl.Color4d(1, 0, 0, 1)
	for _, poly := range g.Levels[GidInvadersStart].Room.Lava {
		gl.Begin(gl.TRIANGLE_FAN)
		for _, v := range poly {
			gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		}
		gl.End()
	}

	gui.SetFontColor(0, 255, 0, 255)
	base.GetDictionary("luxisr").RenderString("Start!", g.Levels[GidInvadersStart].Room.Start.X, g.Levels[GidInvadersStart].Room.Start.Y, 0, 100, gui.Center)
	base.GetDictionary("luxisr").RenderString("End!", g.Levels[GidInvadersStart].Room.End.X, g.Levels[GidInvadersStart].Room.End.Y, 0, 100, gui.Center)

	gl.Color4d(1, 1, 1, 1)
	losCount := 0
	g.DoForEnts(func(gid Gid, ent Ent) {
		if p, ok := ent.(*Player); ok {
			p.Los.WriteDepthBuffer(local.los.texData[losCount], LosMaxDist)
			losCount++
		}
	})
	gl.Color4d(1, 1, 1, 1)
	g.DoForEnts(func(gid Gid, ent Ent) {
		ent.Draw(g)
	})
	gl.Disable(gl.TEXTURE_2D)

	g.renderLosMask(local)
	for _, p := range local.players {
		if p.abs.activeAbility != nil {
			p.abs.activeAbility.Draw(p.gid, g)
		}
	}
	gl.Color4ub(0, 0, 255, 200)
}

func (g *Game) IsExistingPolyVisible(polyIndex string) bool {
	visible := false
	g.DoForEnts(func(gid Gid, ent Ent) {
		if player, ok := ent.(*Player); ok {
			if player.Los.CountSource(polyIndex) > 0.0 {
				visible = true
			}
		}
	})
	return visible
}

func (g *Game) IsPolyPlaceable(poly linear.Poly) bool {
	placeable := true
	// Not placeable it any player can see it
	g.DoForEnts(func(gid Gid, ent Ent) {
		if player, ok := ent.(*Player); ok {
			for i := 0; i < len(poly); i++ {
				if player.Los.TestSeg(poly.Seg(i)) > 0.0 {
					placeable = false
				}
			}
		}
	})
	if !placeable {
		return false
	}

	// Not placeable if it intersects with any walls
	for _, wall := range g.Levels[GidInvadersStart].Room.Walls {
		if linear.ConvexPolysOverlap(poly, wall) {
			return false
		}
	}

	return true
}

func (l *LocalData) doArchitectFocusRegion(g *Game) {
	camera := &l.architect.camera
	if camera.limit.mid.X == 0 && camera.limit.mid.Y == 0 {
		// On the very first frame the limit midpoint will be (0,0), which should
		// never happen after the game begins.  We use this as an opportunity to
		// init the data now that we know the region we're working with.
		if camera.regionDims.X/camera.regionDims.Y > float64(g.Levels[GidInvadersStart].Room.Dx)/float64(g.Levels[GidInvadersStart].Room.Dy) {
			camera.limit.dims.Y = float64(g.Levels[GidInvadersStart].Room.Dy)
			camera.limit.dims.X = float64(g.Levels[GidInvadersStart].Room.Dx) * float64(g.Levels[GidInvadersStart].Room.Dy) / camera.regionDims.Y
		} else {
			camera.limit.dims.X = float64(g.Levels[GidInvadersStart].Room.Dx)
			camera.limit.dims.Y = float64(g.Levels[GidInvadersStart].Room.Dy) * float64(g.Levels[GidInvadersStart].Room.Dx) / camera.regionDims.X
		}
		camera.limit.mid.X = float64(g.Levels[GidInvadersStart].Room.Dx / 2)
		camera.limit.mid.Y = float64(g.Levels[GidInvadersStart].Room.Dy / 2)
		camera.current = camera.limit
		camera.zoom = 0
	}
	wheel := gin.In().GetKeyFlat(gin.MouseWheelVertical, gin.DeviceTypeAny, gin.DeviceIndexAny)
	camera.zoom += wheel.FramePressAmt() / 500
	if camera.zoom < 0 {
		camera.zoom = 0
	}
	if camera.zoom > 2 {
		camera.zoom = 2
	}
	zoom := 1 / math.Exp(camera.zoom)
	camera.current.dims = camera.limit.dims.Scale(zoom)
	if gin.In().GetKey(gin.AnySpace).CurPressAmt() > 0 {
		if !camera.cursorHidden {
			l.sys.HideCursor(true)
			camera.cursorHidden = true
		}
		x := gin.In().GetKey(gin.AnyMouseXAxis).FramePressAmt()
		y := gin.In().GetKey(gin.AnyMouseYAxis).FramePressAmt()
		camera.current.mid.X -= float64(x) * 2
		camera.current.mid.Y -= float64(y) * 2
	} else {
		if camera.cursorHidden {
			l.sys.HideCursor(false)
			camera.cursorHidden = false
		}
	}
}

func (g *Game) renderLocalArchitect(region g2.Region, local *LocalData) {
	local.doArchitectFocusRegion(g)
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

	current := local.architect.camera.current
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

	level := g.Levels[GidInvadersStart]
	level.ManaSource.Draw(local, float64(level.Room.Dx), float64(level.Room.Dy))

	gl.Begin(gl.LINES)
	gl.Color4d(1, 1, 1, 1)
	for _, poly := range g.Levels[GidInvadersStart].Room.Walls {
		for i := range poly {
			seg := poly.Seg(i)
			gl.Vertex2d(gl.Double(seg.P.X), gl.Double(seg.P.Y))
			gl.Vertex2d(gl.Double(seg.Q.X), gl.Double(seg.Q.Y))
		}
	}
	gl.End()

	gl.Color4d(1, 0, 0, 1)
	for _, poly := range g.Levels[GidInvadersStart].Room.Lava {
		gl.Begin(gl.TRIANGLE_FAN)
		for _, v := range poly {
			gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		}
		gl.End()
	}

	gl.Color4ub(0, 255, 0, 255)
	base.GetDictionary("luxisr").RenderString("Start!", g.Levels[GidInvadersStart].Room.Start.X, g.Levels[GidInvadersStart].Room.Start.Y-25, 0, 50, gui.Center)
	base.GetDictionary("luxisr").RenderString("End!", g.Levels[GidInvadersStart].Room.End.X, g.Levels[GidInvadersStart].Room.End.Y-25, 0, 50, gui.Center)

	gl.Color4d(1, 1, 1, 1)
	losCount := 0
	g.DoForEnts(func(gid Gid, ent Ent) {
		if p, ok := ent.(*Player); ok {
			p.Los.WriteDepthBuffer(local.los.texData[losCount], LosMaxDist)
			losCount++
		}
	})
	gl.Color4d(1, 1, 1, 1)
	g.DoForEnts(func(gid Gid, ent Ent) {
		ent.Draw(g)
	})
	gl.Disable(gl.TEXTURE_2D)

	g.renderLosMask(local)
	if local.architect.abs.activeAbility != nil {
		local.architect.abs.activeAbility.Draw("", g)
	}
}

// Draws everything that is relevant to the players on a compute, but not the
// players across the network.  Any ui used to determine how to place an object
// or use an ability, for example.
func (g *Game) RenderLocal(region g2.Region, local *LocalData) {
	var camera *cameraInfo
	switch local.mode {
	case localModeArchitect:
		camera = &local.architect.camera
	case localModeInvaders:
		camera = &local.invaders.camera
	case localModeMoba:
		camera = &local.moba.camera
	}
	camera.regionPos = linear.Vec2{float64(region.X), float64(region.Y)}
	camera.regionDims = linear.Vec2{float64(region.Dx), float64(region.Dy)}
	switch local.mode {
	case localModeArchitect:
		g.renderLocalArchitect(region, local)
	case localModeInvaders:
		g.renderLocalInvaders(region, local)
	case localModeMoba:
		// g.renderLocalInvaders(region, local)
	}
}

func (local *LocalData) SetLocalPlayer(gid Gid, index gin.DeviceIndex) {
	var lp localPlayer
	lp.gid = gid
	lp.deviceIndex = index
	lp.abs.abilities = append(
		lp.abs.abilities,
		ability_makers["burst"](map[string]int{
			"frames": 2,
			"force":  200000,
		}))
	lp.abs.abilities = append(
		lp.abs.abilities,
		ability_makers["pull"](map[string]int{
			"frames": 10,
			"force":  250,
			"angle":  30,
		}))
	lp.abs.abilities = append(
		lp.abs.abilities,
		ability_makers["vision"](map[string]int{
			"range":   50,
			"squeeze": 10, // 10 means 10 / 1000
		}))
	local.players = append(local.players, &lp)
}

func (l *LocalData) activateAbility(abs *personalAbilities, gid Gid, n int, keyPress bool) {
	activeAbility := abs.activeAbility
	abs.activeAbility = nil
	if activeAbility != nil {
		events := activeAbility.Deactivate(gid)
		for _, event := range events {
			l.engine.ApplyEvent(event)
		}
		if activeAbility == abs.abilities[n] {
			return
		}
	}
	events, active := abs.abilities[n].Activate(gid, keyPress)
	for _, event := range events {
		l.engine.ApplyEvent(event)
	}
	if active {
		abs.activeAbility = abs.abilities[n]
	}
}
func (l *LocalData) thinkAbility(g *Game, abs *personalAbilities, gid Gid) {
	if abs.activeAbility == nil {
		return
	}
	var mouse linear.Vec2
	if l.mode == localModeArchitect {
		mx, my := l.sys.GetCursorPos()
		mouse.X = float64(mx)
		mouse.Y = float64(my)
		mouse = mouse.Sub(l.architect.camera.regionPos)
		mouse.X /= l.architect.camera.regionDims.X
		mouse.Y /= l.architect.camera.regionDims.Y
		mouse.X *= l.architect.camera.current.dims.X
		mouse.Y *= l.architect.camera.current.dims.Y
		mouse = mouse.Sub(l.architect.camera.current.dims.Scale(0.5))
		mouse = mouse.Add(l.architect.camera.current.mid)
	}
	events, die := abs.activeAbility.Think(gid, g, mouse)
	for _, event := range events {
		l.engine.ApplyEvent(event)
	}
	if die {
		more_events := abs.activeAbility.Deactivate(gid)
		abs.activeAbility = nil
		for _, event := range more_events {
			l.engine.ApplyEvent(event)
		}
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

func (l *LocalData) localThinkArchitect(g *Game) {
	l.thinkAbility(g, &l.architect.abs, "")
}
func (l *LocalData) localThinkInvaders(g *Game) {
	for _, player := range l.players {
		l.thinkAbility(g, &player.abs, player.gid)
		down_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+1, gin.DeviceTypeController, player.deviceIndex)
		up_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+1, gin.DeviceTypeController, player.deviceIndex)
		right_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive, gin.DeviceTypeController, player.deviceIndex)
		left_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative, gin.DeviceTypeController, player.deviceIndex)
		down_axis = gin.In().GetKeyFlat(gin.KeyS, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		up_axis = gin.In().GetKeyFlat(gin.KeyW, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		right_axis = gin.In().GetKeyFlat(gin.KeyD, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		left_axis = gin.In().GetKeyFlat(gin.KeyA, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		up := axisControl(up_axis.CurPressAmt())
		down := axisControl(down_axis.CurPressAmt())
		left := axisControl(left_axis.CurPressAmt())
		right := axisControl(right_axis.CurPressAmt())
		if up-down != 0 {
			l.engine.ApplyEvent(Accelerate{player.gid, 2 * (up - down)})
		}
		if left-right != 0 {
			l.engine.ApplyEvent(Turn{player.gid, (right - left)})
		}
	}
}

func (l *LocalData) doInvadersFocusRegion(g *Game) {
	min := linear.Vec2{1e9, 1e9}
	max := linear.Vec2{-1e9, -1e9}
	g.DoForEnts(func(gid Gid, ent Ent) {
		if player, ok := ent.(*Player); ok {
			pos := player.Pos()
			if pos.X < min.X {
				min.X = pos.X
			}
			if pos.Y < min.Y {
				min.Y = pos.Y
			}
			if pos.X > max.X {
				max.X = pos.X
			}
			if pos.Y > max.Y {
				max.Y = pos.Y
			}
		}
	})
	min.X -= LosPlayerHorizon
	min.Y -= LosPlayerHorizon
	if min.X < 0 {
		min.X = 0
	}
	if min.Y < 0 {
		min.Y = 0
	}
	max.X += LosPlayerHorizon
	max.Y += LosPlayerHorizon
	if max.X > float64(g.Levels[GidInvadersStart].Room.Dx) {
		max.X = float64(g.Levels[GidInvadersStart].Room.Dx)
	}
	if max.Y > float64(g.Levels[GidInvadersStart].Room.Dy) {
		max.Y = float64(g.Levels[GidInvadersStart].Room.Dy)
	}

	mid := min.Add(max).Scale(0.5)
	dims := max.Sub(min)
	if dims.X/dims.Y < l.invaders.camera.regionDims.X/l.invaders.camera.regionDims.Y {
		dims.X = dims.Y * l.invaders.camera.regionDims.X / l.invaders.camera.regionDims.Y
	} else {
		dims.Y = dims.X * l.invaders.camera.regionDims.Y / l.invaders.camera.regionDims.X
	}
	l.invaders.camera.target.dims = dims
	l.invaders.camera.target.mid = mid

	camera := &l.invaders.camera
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

func (l *LocalData) Think(g *Game) {
	switch l.mode {
	case localModeArchitect:
		l.localThinkArchitect(g)
	case localModeInvaders:
		l.localThinkInvaders(g)
	case localModeMoba:
		//		l.localThinkInvaders(g)
	}
}

func (l *LocalData) handleEventGroupArchitect(group gin.EventGroup) {
	if found, event := group.FindEvent(gin.AnyKey1); found && event.Type == gin.Press {
		l.activateAbility(&l.architect.abs, "", 0, true)
	}
	if found, event := group.FindEvent(gin.AnyKey2); found && event.Type == gin.Press {
		l.activateAbility(&l.architect.abs, "", 1, true)
	}
	if found, event := group.FindEvent(gin.AnyKey3); found && event.Type == gin.Press {
		l.activateAbility(&l.architect.abs, "", 2, true)
	}
	if found, event := group.FindEvent(gin.AnyKey4); found && event.Type == gin.Press {
		l.activateAbility(&l.architect.abs, "", 3, true)
	}
	if l.architect.abs.activeAbility != nil {
		l.architect.abs.activeAbility.Respond("", group)
	}
}

func (l *LocalData) handleEventGroupInvaders(group gin.EventGroup) {
	for _, player := range l.players {
		k0 := gin.In().GetKeyFlat(gin.KeyU, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		k1 := gin.In().GetKeyFlat(gin.KeyI, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		k2 := gin.In().GetKeyFlat(gin.KeyO, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		if found, event := group.FindEvent(k0.Id()); found {
			l.activateAbility(&player.abs, player.gid, 0, event.Type == gin.Press)
			return
		}
		if found, event := group.FindEvent(k1.Id()); found {
			l.activateAbility(&player.abs, player.gid, 1, event.Type == gin.Press)
			return
		}
		if found, event := group.FindEvent(k2.Id()); found {
			l.activateAbility(&player.abs, player.gid, 2, event.Type == gin.Press)
			return
		}
		if player.abs.activeAbility != nil {
			if player.abs.activeAbility.Respond(player.gid, group) {
				return
			}
		}
	}
}

func (l *LocalData) HandleEventGroup(group gin.EventGroup) {
	switch l.mode {
	case localModeArchitect:
		l.handleEventGroupArchitect(group)
	case localModeInvaders:
		l.handleEventGroupInvaders(group)
	case localModeMoba:
		// l.handleEventGroupInvaders(group)
	}
}
