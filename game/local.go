package game

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"time"
	// "github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	g2 "github.com/runningwild/magnus/gui"
	"github.com/runningwild/magnus/stats"
	"math"
)

const LosMaxPlayers = 32
const LosMaxDist = 1000

type personalAbilities struct {
	// All of the abilities that this player can activate.
	abilities []Ability

	// This player's active ability, if any.
	activeAbility Ability
}

type localPlayer struct {
	// This player's gid
	gid Gid

	side int

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

type localEditorData struct {
	camera cameraInfo
}

type mobaSideData struct {
	side int
}

func (lmd *localMobaData) updateLosTex(g *Game, side int) {
	cache := g.Moba.losCache
	pix := lmd.losTex.pix
	for i := range pix {
		if pix[i] < 250 {
			pix[i] += 5
			if pix[i] >= 250 {
				pix[i] = 255
			}
		}
	}

	// Calculating los is the most expensive part of the game, so the los for all
	// ents are done in parallel.
	vpchan := make(chan []visiblePos)
	count := 0
	for _, ent := range g.temp.AllEnts {
		if ent.Side() != side {
			continue
		}
		count++
		// Calls to losCache.Get() are launched in separate go routines and their
		// results are collected on vpchan.
		go func(x, y int, vision float64) {
			vpchan <- cache.Get(x, y, vision)
		}(int(ent.Pos().X), int(ent.Pos().Y), ent.Stats().Vision())
	}

	// Results from los calculations are collected here and are used to update the
	// losTex.
	losTex := lmd.losTex.Pix()
	for i := 0; i < count; i++ {
		res := <-vpchan
		for i := range res {
			cur := losTex[res[i].X][res[i].Y]
			if res[i].Val < cur {
				losTex[res[i].X][res[i].Y] = res[i].Val
			}
		}
	}
}

type mobaPlayerData struct {
	gid    Gid
	camera cameraInfo
	side   int
	abs    personalAbilities
}

type localMobaData struct {
	currentPlayer *mobaPlayerData
	currentSide   *mobaSideData
	players       []mobaPlayerData
	sides         []mobaSideData
	deviceIndex   gin.DeviceIndex
	losTex        *LosTexture
}

func (lmd *localMobaData) setCurrentPlayerByGid(gid Gid) {
	lmd.currentPlayer = nil
	for i := range lmd.players {
		if lmd.players[i].gid == gid {
			lmd.currentPlayer = &lmd.players[i]
		}
	}
	if lmd.currentPlayer == nil {
		panic(fmt.Sprintf("Didn't find a player with gid == %v.", gid))
	}
	lmd.currentSide = &lmd.sides[lmd.currentPlayer.side]
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

type localSetupData struct {
	index int
}

type LocalData struct {
	// The engine running this game, so that the game can apply events to itself.
	engine *cgf.Engine

	mode LocalMode

	setup *localSetupData

	sys       system.System
	architect localArchitectData
	invaders  localInvadersData
	moba      localMobaData
	editor    localEditorData

	// For displaying the mana grid
	nodeTextureId   gl.Uint
	nodeTextureData []byte
}

func (l *LocalData) DebugCyclePlayers() {
	if l.mode != LocalModeMoba {
		panic("Can't DebugCyclePlayers except in LocalModeMoba")
	}
	for i := range l.moba.players {
		if l.moba.currentPlayer == &l.moba.players[i] {
			l.moba.currentPlayer = &l.moba.players[(i+1)%len(l.moba.players)]
			break
		}
	}
	l.moba.currentSide = &l.moba.sides[l.moba.currentPlayer.side]
}

func (l *LocalData) DebugChangeMode(mode LocalMode) {
	l.mode = mode
}

type gameResponderWrapper struct {
	l *LocalData
}

func (grw *gameResponderWrapper) HandleEventGroup(group gin.EventGroup) {
	grw.l.HandleEventGroup(group)
}

func (grw *gameResponderWrapper) Think(int64) {}

type LocalMode int

const (
	LocalModeNone LocalMode = iota
	LocalModeInvaders
	LocalModeArchitect
	LocalModeMoba
	LocalModeEditor
)

func NewLocalDataMoba(engine *cgf.Engine, index gin.DeviceIndex, sys system.System) *LocalData {
	local := newLocalDataHelper(engine, sys, LocalModeMoba)
	local.moba.deviceIndex = index
	return local
}

func NewLocalDataInvaders(engine *cgf.Engine, sys system.System) *LocalData {
	return newLocalDataHelper(engine, sys, LocalModeInvaders)
}

func NewLocalDataArchitect(engine *cgf.Engine, sys system.System) *LocalData {
	return newLocalDataHelper(engine, sys, LocalModeArchitect)
}

func newLocalDataHelper(engine *cgf.Engine, sys system.System, mode LocalMode) *LocalData {
	var local LocalData
	if local.engine != nil {
		base.Error().Fatalf("Engine has already been set.")
	}
	local.engine = engine
	local.mode = mode
	local.setup = &localSetupData{}
	if mode == LocalModeArchitect {
		// local.architect.abs.abilities =
		// 	append(
		// 		local.architect.abs.abilities,
		// 		ability_makers["placePoly"](map[string]int{"wall": 1}))
		// local.architect.abs.abilities =
		// 	append(
		// 		local.architect.abs.abilities,
		// 		ability_makers["placePoly"](map[string]int{"pests": 1}))
		// local.architect.abs.abilities = append(local.architect.abs.abilities, ability_makers["removePoly"](nil))
	}
	local.sys = sys
	gin.In().RegisterEventListener(&gameResponderWrapper{&local})
	return &local
}

func (g *Game) renderLosMask(local *LocalData) {
	switch {
	case local.mode == LocalModeMoba:
		local.moba.updateLosTex(g, local.moba.currentSide.side)
	default:
		panic("Not implemented!!!")
	}
	local.moba.losTex.Bind()
	base.EnableShader("losgrid")
	gl.Enable(gl.TEXTURE_2D)
	gl.Color4d(1, 1, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0, 0)
	gl.Vertex2i(0, 0)
	gl.TexCoord2d(1, 0)
	// gl.Vertex2i(0, gl.Int(g.Levels[GidInvadersStart].Room.Dy))
	gl.Vertex2i(0, LosGridSize*LosTextureSize)
	gl.TexCoord2d(1, 1)
	// gl.Vertex2i(gl.Int(g.Levels[GidInvadersStart].Room.Dx), gl.Int(g.Levels[GidInvadersStart].Room.Dy))
	gl.Vertex2i(LosGridSize*LosTextureSize, LosGridSize*LosTextureSize)
	gl.TexCoord2d(0, 1)
	// gl.Vertex2i(gl.Int(g.Levels[GidInvadersStart].Room.Dx), 0)
	gl.Vertex2i(LosGridSize*LosTextureSize, 0)
	gl.End()
	gl.Disable(gl.TEXTURE_2D)
	base.EnableShader("")
}

func (g *Game) renderLocalInvaders(region g2.Region, local *LocalData) {
	panic("Need to keep track of side for local invaders")
	// g.renderLocalHelper(region, local, &local.invaders.camera)
}

func (g *Game) renderLocalMoba(region g2.Region, local *LocalData) {
	g.renderLocalHelper(region, local, &local.moba.currentPlayer.camera, local.moba.currentPlayer.side)
}

// For invaders or moba, does a lot of basic stuff common to both
func (g *Game) renderLocalHelper(region g2.Region, local *LocalData, camera *cameraInfo, side int) {
	camera.doInvadersFocusRegion(g, side)
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

	current := camera.current
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
	zoom := camera.current.dims.X / float64(region.Dims.Dx)
	level.ManaSource.Draw(local, zoom, float64(level.Room.Dx), float64(level.Room.Dy))

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

	gui.SetFontColor(0, 255, 0, 255)
	for side, pos := range g.Levels[GidInvadersStart].Room.Starts {
		base.GetDictionary("luxisr").RenderString(fmt.Sprintf("S%d", side), pos.X, pos.Y, 0, 100, gui.Center)
	}

	gl.Color4d(1, 1, 1, 1)
	for _, ent := range g.temp.AllEnts {
		ent.Draw(g, side)
	}
	gl.Disable(gl.TEXTURE_2D)

	if local.mode != LocalModeMoba {
		panic("Need to implement drawing players from standard mode data")
	}
	for i := range local.moba.players {
		p := &local.moba.players[i]
		if p.abs.activeAbility != nil {
			p.abs.activeAbility.Draw(p.gid, g, side)
		}
	}
	for _, proc := range g.Processes {
		proc.Draw(Gid(""), g, side)
	}

	gl.Color4ub(0, 0, 255, 200)
	g.renderLosMask(local)
}

func (camera *cameraInfo) doArchitectFocusRegion(g *Game, sys system.System) {
	if camera.limit.mid.X == 0 && camera.limit.mid.Y == 0 {
		// On the very first frame the limit midpoint will be (0,0), which should
		// never happen after the game begins.  We use this as an opportunity to
		// init the data now that we know the region we're working with.
		rdx := float64(g.Levels[GidInvadersStart].Room.Dx)
		rdy := float64(g.Levels[GidInvadersStart].Room.Dy)
		if camera.regionDims.X/camera.regionDims.Y > rdx/rdy {
			camera.limit.dims.Y = rdy
			camera.limit.dims.X = rdy * camera.regionDims.X / camera.regionDims.Y
		} else {
			camera.limit.dims.X = rdx
			camera.limit.dims.Y = rdx * camera.regionDims.Y / camera.regionDims.X
		}
		camera.limit.mid.X = rdx / 2
		camera.limit.mid.Y = rdy / 2
		camera.current = camera.limit
		camera.zoom = 0
		base.Log().Printf("Region Dims: %2.2v", camera.regionDims)
		base.Log().Printf("Room Dims: %2.2v %2.2v", rdx, rdy)
		base.Log().Printf("Limit Dims: %2.2v", camera.limit.dims)
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
			sys.HideCursor(true)
			camera.cursorHidden = true
		}
		x := gin.In().GetKey(gin.AnyMouseXAxis).FramePressAmt()
		y := gin.In().GetKey(gin.AnyMouseYAxis).FramePressAmt()
		camera.current.mid.X -= float64(x) * 2
		camera.current.mid.Y -= float64(y) * 2
	} else {
		if camera.cursorHidden {
			sys.HideCursor(false)
			camera.cursorHidden = false
		}
	}
}

func (g *Game) renderLocalArchitect(region g2.Region, local *LocalData) {
	local.architect.camera.doArchitectFocusRegion(g, local.sys)
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

	zoom := local.architect.camera.current.dims.X / float64(region.Dims.Dx)
	level := g.Levels[GidInvadersStart]
	level.ManaSource.Draw(local, zoom, float64(level.Room.Dx), float64(level.Room.Dy))

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

	gl.Color4ub(0, 255, 0, 255)
	for side, pos := range g.Levels[GidInvadersStart].Room.Starts {
		base.GetDictionary("luxisr").RenderString(fmt.Sprintf("S%d", side), pos.X, pos.Y, 0, 100, gui.Center)
	}

	gl.Color4d(1, 1, 1, 1)
	for _, ent := range g.temp.AllEnts {
		ent.Draw(g, -1) // TODO: Side isn't defined for architect yet
	}
	gl.Disable(gl.TEXTURE_2D)

	g.renderLosMask(local)
	if local.architect.abs.activeAbility != nil {
		local.architect.abs.activeAbility.Draw("", g, -1) // TODO: side not defined for architect
	}
}

func (g *Game) RenderLocalSetup(region g2.Region, local *LocalData) {
	dict := base.GetDictionary("luxisr")
	size := 60.0
	y := 100.0
	dict.RenderString("Engines:", size, y, 0, size, gui.Left)
	for i, id := range g.Setup.EngineIds {
		y += size
		if id == local.engine.Id() {
			gui.SetFontColor(0.7, 0.7, 1, 1)
		} else {
			gui.SetFontColor(0.7, 0.7, 0.7, 1)
		}
		dataStr := fmt.Sprintf("Engine %d, Side %d, %s", id, g.Setup.Sides[id].Side, g.Champs[g.Setup.Sides[id].Champ].Name)
		dict.RenderString(dataStr, size, y, 0, size, gui.Left)
		if local.engine.Id() == 1 && i == local.setup.index {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
	y += size
	gui.SetFontColor(0.7, 0.7, 0.7, 1)
	if local.engine.Id() == 1 {
		dict.RenderString("Start!", size, y, 0, size, gui.Left)
		if local.setup.index == len(g.Setup.EngineIds) {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
	ids := local.engine.Ids()
	if len(ids) > 0 {
		// This is the host engine - so update the list of ids in case it's changed
		local.engine.ApplyEvent(SetupSetEngineIds{ids})
	}

}

// Draws everything that is relevant to the players on a computer, but not the
// players across the network.  Any ui used to determine how to place an object
// or use an ability, for example.
func (g *Game) RenderLocal(region g2.Region, local *LocalData) {
	if g.Setup != nil {
		g.RenderLocalSetup(region, local)
		return
	}
	switch {
	case local.mode == LocalModeMoba:
	default:
		panic("Not implemented!!!")
	}
	if local.moba.losTex == nil {
		// This can happen because the function to set g.Setup to nil is called in
		// an event, and the function that makes the losTex is called in Think(), so
		// we might get here (inside the Draw() function) between the event and the
		// Think().
		return
	}
	local.moba.losTex.Remap()
	var camera *cameraInfo
	switch local.mode {
	case LocalModeArchitect:
		camera = &local.architect.camera
	case LocalModeInvaders:
		camera = &local.invaders.camera
	case LocalModeMoba:
		camera = &local.moba.currentPlayer.camera
	case LocalModeEditor:
		camera = &local.editor.camera
	}
	camera.regionPos = linear.Vec2{float64(region.X), float64(region.Y)}
	camera.regionDims = linear.Vec2{float64(region.Dx), float64(region.Dy)}
	switch local.mode {
	case LocalModeArchitect:
		g.renderLocalArchitect(region, local)
	case LocalModeInvaders:
		g.renderLocalInvaders(region, local)
	case LocalModeMoba:
		g.renderLocalMoba(region, local)
	}
}

func (l *LocalData) activateAbility(abs *personalAbilities, gid Gid, n int, keyPress bool) {
	if len(abs.abilities) <= n {
		return
	}
	activeAbility := abs.activeAbility
	abs.activeAbility = nil

	events, active := abs.abilities[n].Activate(gid, keyPress)
	for _, event := range events {
		l.engine.ApplyEvent(event)
	}

	if active && activeAbility != nil && activeAbility != abs.abilities[n] {
		base.Log().Printf("Deactivate on keypress")
		events := activeAbility.Deactivate(gid)
		for _, event := range events {
			l.engine.ApplyEvent(event)
		}
		if activeAbility == abs.abilities[n] {
			return
		}
	}
	if active {
		abs.activeAbility = abs.abilities[n]
	}
	if abs.activeAbility == nil {
		abs.activeAbility = activeAbility
	}
}
func (l *LocalData) thinkAbility(g *Game, abs *personalAbilities, gid Gid) {
	if abs.activeAbility == nil {
		return
	}
	var mouse linear.Vec2
	if l.mode == LocalModeArchitect {
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
		base.Log().Printf("Deactivate on die")
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

func (local *LocalData) setupMobaData(g *Game) {
	sidesSet := make(map[int]bool)
	for _, ent := range g.Ents {
		p, ok := ent.(*Player)
		if !ok {
			continue
		}
		var pd mobaPlayerData
		pd.gid = p.Gid
		pd.side = p.Side()
		sidesSet[pd.side] = true
		// pd.abs.abilities = append(
		// 	pd.abs.abilities,
		// 	ability_makers["mine"](map[string]int{
		// 		"health":  10,
		// 		"damage":  100,
		// 		"trigger": 100,
		// 		"mass":    300,
		// 	}))
		if p.Champ < 0 || p.Champ > len(g.Champs) {
			panic(fmt.Sprintf("p.Champ == %d, ouside of range of champ, %d", p.Champ, len(g.Champs)))
		}
		for _, ability := range g.Champs[p.Champ].Abilities {
			pd.abs.abilities = append(
				pd.abs.abilities,
				ability_makers[ability.Name](ability.Params))
		}
		local.moba.players = append(local.moba.players, pd)
	}
	for _ = range sidesSet {
		var sd mobaSideData
		local.moba.sides = append(local.moba.sides, sd)
	}
	for i := range local.moba.sides {
		local.moba.sides[i].side = i
	}
	local.moba.losTex = MakeLosTexture()
	pix := local.moba.losTex.pix
	for i := range pix {
		pix[i] = 255
	}
	local.moba.setCurrentPlayerByGid(Gid(fmt.Sprintf("Engine:%d", local.engine.Id())))
	local.setup = nil
}

func (l *LocalData) localThinkInvaders(g *Game) {
	if l.mode != LocalModeMoba {
		panic("Need to implement controls for multiple players on a single screen")
	}
	if l.setup != nil {
		l.setupMobaData(g)
	}
	l.thinkAbility(g, &l.moba.currentPlayer.abs, l.moba.currentPlayer.gid)
	down_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+1, gin.DeviceTypeController, l.moba.deviceIndex)
	up_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+1, gin.DeviceTypeController, l.moba.deviceIndex)
	right_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive, gin.DeviceTypeController, l.moba.deviceIndex)
	left_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative, gin.DeviceTypeController, l.moba.deviceIndex)
	down_axis = gin.In().GetKeyFlat(gin.KeyS, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	up_axis = gin.In().GetKeyFlat(gin.KeyW, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	right_axis = gin.In().GetKeyFlat(gin.KeyD, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	left_axis = gin.In().GetKeyFlat(gin.KeyA, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	up := axisControl(up_axis.CurPressAmt())
	down := axisControl(down_axis.CurPressAmt())
	left := axisControl(left_axis.CurPressAmt())
	right := axisControl(right_axis.CurPressAmt())
	if up-down != 0 {
		l.engine.ApplyEvent(Accelerate{l.moba.currentPlayer.gid, 300 * (up - down)})
	}
	if left-right != 0 {
		l.engine.ApplyEvent(Turn{l.moba.currentPlayer.gid, (right - left)})
	}
}

func (camera *cameraInfo) doInvadersFocusRegion(g *Game, side int) {
	min := linear.Vec2{1e9, 1e9}
	max := linear.Vec2{-1e9, -1e9}
	for _, ent := range g.temp.AllEnts {
		if ent.Side() != side {
			continue
		}
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
	}
	min.X -= stats.LosPlayerHorizon
	min.Y -= stats.LosPlayerHorizon
	if min.X < 0 {
		min.X = 0
	}
	if min.Y < 0 {
		min.Y = 0
	}
	max.X += stats.LosPlayerHorizon
	max.Y += stats.LosPlayerHorizon
	if max.X > float64(g.Levels[GidInvadersStart].Room.Dx) {
		max.X = float64(g.Levels[GidInvadersStart].Room.Dx)
	}
	if max.Y > float64(g.Levels[GidInvadersStart].Room.Dy) {
		max.Y = float64(g.Levels[GidInvadersStart].Room.Dy)
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

func (l *LocalData) Setup(g *Game) {
	if len(l.engine.Ids()) > 0 {
		if gin.In().GetKey(gin.AnyUp).FramePressCount() > 0 {
			l.setup.index--
			if l.setup.index < 0 {
				l.setup.index = 0
			}
		}
		if gin.In().GetKey(gin.AnyDown).FramePressCount() > 0 {
			l.setup.index++
			if l.setup.index > len(g.Setup.EngineIds) {
				l.setup.index = len(g.Setup.EngineIds)
			}
		}
	} else {
		for i, v := range g.Setup.EngineIds {
			if v == l.engine.Id() {
				l.setup.index = i
			}
		}
	}
	if gin.In().GetKey(gin.AnyLeft).FramePressCount() > 0 {
		l.engine.ApplyEvent(SetupChampSelect{l.engine.Id(), -1})
	}
	if gin.In().GetKey(gin.AnyRight).FramePressCount() > 0 {
		l.engine.ApplyEvent(SetupChampSelect{l.engine.Id(), 1})
	}
	if gin.In().GetKey(gin.AnyReturn).FramePressCount() > 0 {
		if l.setup.index < len(g.Setup.EngineIds) {
			id := g.Setup.EngineIds[l.setup.index]
			side := (g.Setup.Sides[id].Side + 1) % 2
			l.engine.ApplyEvent(SetupChangeSides{id, side})
		} else {
			if len(l.engine.Ids()) > 0 {
				l.engine.ApplyEvent(SetupComplete{time.Now().UnixNano()})
			}
		}
	}
}

func (l *LocalData) Think(g *Game) {
	if g.Setup != nil {
		l.Setup(g)
		return
	}
	switch l.mode {
	case LocalModeArchitect:
		l.localThinkArchitect(g)
	case LocalModeInvaders:
		l.localThinkInvaders(g)
	case LocalModeMoba:
		l.localThinkInvaders(g)
	}
}

func (l *LocalData) handleEventGroupArchitect(group gin.EventGroup) {
	keys := []gin.KeyId{gin.AnyKey1, gin.AnyKey2, gin.AnyKey3, gin.AnyKey4, gin.AnyKey5, gin.AnyKey6, gin.AnyKey7, gin.AnyKey8, gin.AnyKey9}
	for i := range l.architect.abs.abilities {
		if found, event := group.FindEvent(keys[i]); found && event.Type == gin.Press {
			l.activateAbility(&l.architect.abs, "", i, true)
		}
	}
	if l.architect.abs.activeAbility != nil {
		l.architect.abs.activeAbility.Respond("", group)
	}
}

func (l *LocalData) handleEventGroupInvaders(group gin.EventGroup) {
	if l.mode != LocalModeMoba {
		panic("Need to implement controls for multiple players on a single screen")
	}
	k0 := gin.In().GetKeyFlat(gin.Key6, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	k1 := gin.In().GetKeyFlat(gin.Key7, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	k2 := gin.In().GetKeyFlat(gin.Key8, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	k3 := gin.In().GetKeyFlat(gin.Key9, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
	if found, event := group.FindEvent(k0.Id()); found {
		l.activateAbility(&l.moba.currentPlayer.abs, l.moba.currentPlayer.gid, 0, event.Type == gin.Press)
		return
	}
	if found, event := group.FindEvent(k1.Id()); found {
		l.activateAbility(&l.moba.currentPlayer.abs, l.moba.currentPlayer.gid, 1, event.Type == gin.Press)
		return
	}
	if found, event := group.FindEvent(k2.Id()); found {
		l.activateAbility(&l.moba.currentPlayer.abs, l.moba.currentPlayer.gid, 2, event.Type == gin.Press)
		return
	}
	if found, event := group.FindEvent(k3.Id()); found {
		l.activateAbility(&l.moba.currentPlayer.abs, l.moba.currentPlayer.gid, 3, event.Type == gin.Press)
		return
	}
	if l.moba.currentPlayer.abs.activeAbility != nil {
		if l.moba.currentPlayer.abs.activeAbility.Respond(l.moba.currentPlayer.gid, group) {
			return
		}
	}
}

func (l *LocalData) HandleEventGroup(group gin.EventGroup) {
	// TODO: Should probably handle event groups and do proper even handling on setup
	if l.setup != nil {
		return
	}
	switch l.mode {
	case LocalModeArchitect:
		l.handleEventGroupArchitect(group)
	case LocalModeInvaders:
		l.handleEventGroupInvaders(group)
	case LocalModeMoba:
		l.handleEventGroupInvaders(group)
	}
}
