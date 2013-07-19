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
	"github.com/runningwild/magnus/los"
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

type localArchitectData struct {
	abs personalAbilities

	//DEPRECATED:
	place linear.Poly
}

type localData struct {
	regionPos  linear.Vec2
	regionDims linear.Vec2

	// The engine running this game, so that the game can apply events to itself.
	engine *cgf.Engine

	// true iff this is the computer playing the architect sgide of the game
	isArchitect bool

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

	current, target struct {
		mid, dims linear.Vec2
	}

	sys       system.System
	architect localArchitectData
}

var local localData

func IsArchitect() bool {
	return local.isArchitect
}

func SetLocalEngine(engine *cgf.Engine, sys system.System, isArchitect bool) {
	if local.engine != nil {
		panic("Engine has already been set.")
	}
	local.engine = engine
	local.isArchitect = isArchitect
	if isArchitect {
		local.architect.abs.abilities = append(local.architect.abs.abilities, ability_makers["placePoly"](nil))
		local.architect.abs.abilities = append(local.architect.abs.abilities, ability_makers["removePoly"](nil))
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
}

// This is just a placeholder for code that copies the backbuffer to a texture
func (g *Game) copyBackbuffer() {
	gl.BindTexture(gl.TEXTURE_2D, local.back.texId)
	gl.CopyTexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		0,
		0,
		gl.Sizei(g.Room.Dx),
		gl.Sizei(g.Room.Dy),
		0)
	gl.BindTexture(gl.TEXTURE_2D, local.back.texId)
	gl.Color4d(1, 1, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0, 1)
	gl.Vertex2i(0, 0)
	gl.TexCoord2d(0, 0)
	gl.Vertex2i(0, 60*3)
	gl.TexCoord2d(1, 0)
	gl.Vertex2i(90*3, 60*3)
	gl.TexCoord2d(1, 1)
	gl.Vertex2i(90*3, 0)
	gl.End()
}

func (g *Game) renderLosMask() {
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
	base.SetUniformF("los", "dx", float32(g.Room.Dx))
	base.SetUniformF("los", "dy", float32(g.Room.Dy))
	base.SetUniformF("los", "losMaxDist", LosMaxDist)
	base.SetUniformF("los", "losResolution", los.Resolution)
	base.SetUniformF("los", "losMaxPlayers", LosMaxPlayers)
	if local.isArchitect {
		base.SetUniformI("los", "architect", 1)
	} else {
		base.SetUniformI("los", "architect", 0)
	}
	var playerPos []linear.Vec2
	g.DoForEnts(func(gid Gid, ent Ent) {
		if _, ok := ent.(*Player); ok {
			playerPos = append(playerPos, ent.Pos())
		}
	})
	base.SetUniformV2Array("los", "playerPos", playerPos)
	base.SetUniformI("los", "losNumPlayers", len(playerPos))
	gl.Color4d(0, 0, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0, 1)
	gl.Vertex2i(0, 0)
	gl.TexCoord2d(0, 0)
	gl.Vertex2i(0, gl.Int(g.Room.Dy))
	gl.TexCoord2d(1, 0)
	gl.Vertex2i(gl.Int(g.Room.Dx), gl.Int(g.Room.Dy))
	gl.TexCoord2d(1, 1)
	gl.Vertex2i(gl.Int(g.Room.Dx), 0)
	gl.End()
	base.EnableShader("")
}

func (g *Game) renderLocalInvaders(region gui.Region) {
	base.GetDictionary("luxisr").RenderString("darthur is nub", 30, 10, 0, 100, gui.Left)
	g.renderLosMask()
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
	for _, wall := range g.Room.Walls {
		if linear.ConvexPolysOverlap(poly, wall) {
			return false
		}
	}

	return true
}

func (g *Game) renderLocalArchitect(region gui.Region) {
	g.renderLosMask()
	if local.architect.abs.activeAbility != nil {
		local.architect.abs.activeAbility.Draw("", g)
	}
	return
	gl.Disable(gl.TEXTURE_2D)
	mx, my := local.sys.GetCursorPos()
	mx -= region.X
	my -= region.Y
	dx, dy := float64(50), float64(50)
	x, y := float64(int((float64(mx)-dx/2)/10)*10), float64(int((float64(my)-dy/2)/10)*10)
	poly := linear.Poly{
		linear.Vec2{x, y},
		linear.Vec2{x, y + dy},
		linear.Vec2{x + dx, y + dy},
		linear.Vec2{x + dx, y},
	}
	placeable := g.IsPolyPlaceable(poly)
	if placeable {
		gl.Color4ub(255, 255, 255, 255)
	} else {
		gl.Color4ub(255, 0, 0, 255)
	}
	local.architect.place = poly
	gl.Begin(gl.LINES)
	for i := range poly {
		seg := poly.Seg(i)
		gl.Vertex2i(gl.Int(seg.P.X), gl.Int(seg.P.Y))
		gl.Vertex2i(gl.Int(seg.Q.X), gl.Int(seg.Q.Y))
	}
	gl.End()
}

// Draws everything that is relevant to the players on a compute, but not the
// players across the network.  Any ui used to determine how to place an object
// or use an ability, for example.
func (g *Game) RenderLocal(region gui.Region) {
	local.regionPos = linear.Vec2{float64(region.X), float64(region.Y)}
	local.regionDims = linear.Vec2{float64(region.Dx), float64(region.Dy)}
	local.doPlayersFocusRegion(g)
	if local.isArchitect {
		g.renderLocalArchitect(region)
	} else {
		g.renderLocalInvaders(region)
	}
}

func SetLocalPlayer(player *Player, index gin.DeviceIndex) {
	var lp localPlayer
	lp.gid = player.Id()
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

func (l *localData) activateAbility(abs *personalAbilities, gid Gid, n int, keyPress bool) {
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
func (l *localData) thinkAbility(g *Game, abs *personalAbilities, gid Gid) {
	if abs.activeAbility == nil {
		return
	}
	mx, my := local.sys.GetCursorPos()
	mouse := linear.Vec2{float64(mx), float64(my)}
	events, die := abs.activeAbility.Think(gid, g, mouse.Sub(l.regionPos))
	for _, event := range events {
		local.engine.ApplyEvent(event)
	}
	if die {
		more_events := abs.activeAbility.Deactivate(gid)
		abs.activeAbility = nil
		for _, event := range more_events {
			local.engine.ApplyEvent(event)
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

func localThinkArchitect(g *Game) {
	local.thinkAbility(g, &local.architect.abs, "")
}
func localThinkInvaders(g *Game) {
	for _, player := range local.players {
		local.thinkAbility(g, &player.abs, player.gid)
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
			local.engine.ApplyEvent(Accelerate{player.gid, 2 * (up - down)})
		}
		if left-right != 0 {
			local.engine.ApplyEvent(Turn{player.gid, (left - right)})
		}
	}
}

func (l *localData) doPlayersFocusRegion(g *Game) {
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
	if max.X > float64(g.Room.Dx) {
		max.X = float64(g.Room.Dx)
	}
	if max.Y > float64(g.Room.Dy) {
		max.Y = float64(g.Room.Dy)
	}

	mid := min.Add(max).Scale(0.5)
	dims := max.Sub(min)
	if dims.X/dims.Y < l.regionDims.X/l.regionDims.Y {
		dims.X = dims.Y * l.regionDims.X / l.regionDims.Y
	} else {
		dims.Y = dims.X * l.regionDims.Y / l.regionDims.X
	}
	l.target.dims = dims
	l.target.mid = mid

	if l.current.mid.X == 0 && l.current.mid.Y == 0 {
		// On the very first frame the current midpoint will be (0,0), which should
		// never happen after the game begins.  In this one case we'll immediately
		// set current to target so we don't start off by approaching it from the
		// origin.
		l.current = l.target
	} else {
		// speed is in (0, 1), the higher it is, the faster current approaches target.
		speed := 0.1
		l.current.dims = l.current.dims.Scale(1 - speed).Add(l.target.dims.Scale(speed))
		l.current.mid = l.current.mid.Scale(1 - speed).Add(l.target.mid.Scale(speed))
	}
}

func localThink(g *Game) {
	if local.isArchitect {
		localThinkArchitect(g)
	} else {
		localThinkInvaders(g)
	}
}

func (l *localData) handleEventGroupArchitect(group gin.EventGroup) {
	if found, event := group.FindEvent(gin.AnyKey1); found && event.Type == gin.Press {
		l.activateAbility(&l.architect.abs, "", 0, true)
	}
	if found, event := group.FindEvent(gin.AnyKey2); found && event.Type == gin.Press {
		l.activateAbility(&l.architect.abs, "", 1, true)
	}
	if l.architect.abs.activeAbility != nil {
		l.architect.abs.activeAbility.Respond("", group)
	}
}

func (l *localData) handleEventGroupInvaders(group gin.EventGroup) {
	for _, player := range local.players {
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

func (l *localData) HandleEventGroup(group gin.EventGroup) {
	if l.isArchitect {
		l.handleEventGroupArchitect(group)
	} else {
		l.handleEventGroupInvaders(group)
	}
}
