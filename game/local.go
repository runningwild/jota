package game

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
)

const LosResolution = 2048 * 2
const LosMaxPlayers = 2
const LosMaxDist = 1000

type localPlayer struct {
	// This player's id
	id int

	// The device controlling this player.
	device_index gin.DeviceIndex

	// All of the abilities that this player can activate.
	abilities []Ability

	// This player's active ability, if any.
	active_ability Ability
}

type localMasterData struct {
	place linear.Poly
}

type localData struct {
	game *Game

	region gui.Region

	// The engine running this game, so that the game can apply events to itself.
	engine *cgf.Engine

	// true iff this is the computer playing the master side of the game
	isMaster bool

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

	sys    system.System
	master localMasterData
}

var local localData

func (g *Game) SetEngine(engine *cgf.Engine, isMaster bool) {
	if local.engine != nil {
		panic("Engine has already been set.")
	}
	local.engine = engine
	local.isMaster = isMaster
	gin.In().RegisterEventListener(&gameResponderWrapper{g})

	local.los.texRawData = make([]uint32, LosResolution*LosMaxPlayers)
	local.los.texData = make([][]uint32, LosMaxPlayers)
	for i := range local.los.texData {
		start := i * LosResolution
		end := (i + 1) * LosResolution
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
			LosResolution,
			LosMaxPlayers,
			0,
			gl.ALPHA,
			gl.UNSIGNED_INT,
			gl.Pointer(&local.los.texRawData[0]))
	})

	local.back.texRawData = make([]uint32, LosResolution*LosMaxPlayers)
	local.back.texData = make([][]uint32, LosMaxPlayers)
	for i := range local.back.texData {
		start := i * LosResolution
		end := (i + 1) * LosResolution
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
		900,
		600,
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
		LosResolution,
		LosMaxPlayers,
		gl.ALPHA,
		gl.UNSIGNED_INT,
		gl.Pointer(&local.los.texRawData[0]))
	base.SetUniformI("los", "tex0", 0)
	base.SetUniformF("los", "dx", 900)
	base.SetUniformF("los", "dy", 600)
	base.SetUniformF("los", "losMaxDist", LosMaxDist)
	base.SetUniformF("los", "losResolution", LosResolution)
	base.SetUniformF("los", "losMaxPlayers", LosMaxPlayers)
	if local.isMaster {
		base.SetUniformI("los", "master", 1)
	} else {
		base.SetUniformI("los", "master", 0)
	}
	var playerPos []linear.Vec2
	for i := range g.Ents {
		_, ok := g.Ents[i].(*Player)
		if !ok {
			continue
		}
		playerPos = append(playerPos, g.Ents[i].Pos())
	}
	base.SetUniformV2Array("los", "playerPos", playerPos)
	base.SetUniformI("los", "losNumPlayers", len(playerPos))
	gl.Color4d(0, 0, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0, 1)
	gl.Vertex2i(0, 0)
	gl.TexCoord2d(0, 0)
	gl.Vertex2i(0, 600)
	gl.TexCoord2d(1, 0)
	gl.Vertex2i(900, 600)
	gl.TexCoord2d(1, 1)
	gl.Vertex2i(900, 0)
	gl.End()
	base.EnableShader("")
}

func (g *Game) renderLocalInvaders(region gui.Region) {
	g.renderLosMask()
}
func (g *Game) renderLocalMaster(region gui.Region) {
	g.renderLosMask()
	gl.Disable(gl.TEXTURE_2D)
	mx, my := local.sys.GetCursorPos()
	mx -= region.X
	my -= region.Y
	dx, dy := float64(50), float64(50)
	x, y := float64(int((float64(mx)-dx/2)/10)*10), float64(int((float64(my)-dy/2)/10)*10)
	visible := false
	crosses := []linear.Seg2{
		linear.Seg2{linear.Vec2{x, y}, linear.Vec2{x + dx, y + dy}},
		linear.Seg2{linear.Vec2{x + dx, y}, linear.Vec2{x, y + dy}},
	}
	for _, ent := range g.Ents {
		p, ok := ent.(*Player)
		if !ok {
			continue
		}
		if p.Los.TestSeg(crosses[0]) > 0.0 || p.Los.TestSeg(crosses[1]) > 0.0 {
			visible = true
			break
		}
	}
	if visible {
		gl.Color4ub(255, 0, 0, 255)
	} else {
		gl.Color4ub(255, 255, 255, 255)
	}
	local.master.place = linear.Poly{
		linear.Vec2{x, y},
		linear.Vec2{x, y + dy},
		linear.Vec2{x + dx, y + dy},
		linear.Vec2{x + dx, y},
	}
	gl.Begin(gl.LINES)
	gl.Vertex2i(gl.Int(x), gl.Int(y))
	gl.Vertex2i(gl.Int(x), gl.Int(y+dy))
	gl.Vertex2i(gl.Int(x), gl.Int(y+dy))
	gl.Vertex2i(gl.Int(x+dx), gl.Int(y+dy))
	gl.Vertex2i(gl.Int(x+dx), gl.Int(y+dy))
	gl.Vertex2i(gl.Int(x+dx), gl.Int(y))
	gl.Vertex2i(gl.Int(x+dx), gl.Int(y))
	gl.Vertex2i(gl.Int(x), gl.Int(y))
	gl.End()
}

// Draws everything that is relevant to the players on a compute, but not the
// players across the network.  Any ui used to determine how to place an object
// or use an ability, for example.
func (g *Game) RenderLocal(region gui.Region) {
	if local.isMaster {
		g.renderLocalMaster(region)
	} else {
		g.renderLocalInvaders(region)
	}
}

func (g *Game) SetLocalData(sys system.System) {
	local.game = g
	local.sys = sys
}

func (g *Game) SetLocalPlayer(player *Player, index gin.DeviceIndex) {
	var lp localPlayer
	lp.id = player.Id()
	lp.device_index = index
	lp.abilities = append(
		lp.abilities,
		ability_makers["burst"](map[string]int{
			"frames": 2,
			"force":  200000,
		}))
	lp.abilities = append(
		lp.abilities,
		ability_makers["pull"](map[string]int{
			"frames": 10,
			"force":  250,
			"angle":  30,
		}))
	local.players = append(local.players, &lp)
}

func (g *Game) ActivateAbility(player *localPlayer, n int) {
	active_ability := player.active_ability
	player.active_ability = nil
	if active_ability != nil {
		events := active_ability.Deactivate(player.id)
		for _, event := range events {
			local.engine.ApplyEvent(event)
		}
		if active_ability == player.abilities[n] {
			return
		}
	}
	events, active := player.abilities[n].Activate(player.id)
	for _, event := range events {
		local.engine.ApplyEvent(event)
	}
	if active {
		player.active_ability = player.abilities[n]
		base.Log().Printf("Setting active ability to %v", player.active_ability)
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

type PlacePoly struct {
	Poly linear.Poly
}

func init() {
	gob.Register(PlacePoly{})
}

func (p PlacePoly) Apply(_g interface{}) {
	g := _g.(*Game)
	g.Room.Walls = append(g.Room.Walls, p.Poly)
}

func localThinkMaster() {
	lmouse := gin.In().GetKey(gin.AnyMouseLButton)
	if lmouse.FramePressCount() > 0 {
		local.engine.ApplyEvent(PlacePoly{local.master.place})
	}
}
func localThinkInvaders() {
	for _, player := range local.players {
		if player.active_ability != nil {
			events, die := player.active_ability.Think(player.id, local.game)
			for _, event := range events {
				local.engine.ApplyEvent(event)
			}
			if die {
				more_events := player.active_ability.Deactivate(player.id)
				player.active_ability = nil
				for _, event := range more_events {
					local.engine.ApplyEvent(event)
				}
			}
		}
		down_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+1, gin.DeviceTypeController, player.device_index)
		up_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+1, gin.DeviceTypeController, player.device_index)
		right_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive, gin.DeviceTypeController, player.device_index)
		left_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative, gin.DeviceTypeController, player.device_index)
		down_axis = gin.In().GetKeyFlat(gin.KeyS, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		up_axis = gin.In().GetKeyFlat(gin.KeyW, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		right_axis = gin.In().GetKeyFlat(gin.KeyD, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		left_axis = gin.In().GetKeyFlat(gin.KeyA, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		up := axisControl(up_axis.CurPressAmt())
		down := axisControl(down_axis.CurPressAmt())
		left := axisControl(left_axis.CurPressAmt())
		right := axisControl(right_axis.CurPressAmt())
		if up-down != 0 {
			local.engine.ApplyEvent(Accelerate{player.id, 2 * (up - down)})
		}
		if left-right != 0 {
			local.engine.ApplyEvent(Turn{player.id, (left - right)})
		}
	}
}
func LocalThink() {
	if local.isMaster {
		localThinkMaster()
	} else {
		localThinkInvaders()
	}
}

func (g *Game) HandleEventGroup(group gin.EventGroup) {
	for _, player := range local.players {
		k0 := gin.In().GetKeyFlat(gin.KeyZ, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		k1 := gin.In().GetKeyFlat(gin.KeyX, gin.DeviceTypeKeyboard, gin.DeviceIndexAny)
		if found, event := group.FindEvent(k0.Id()); found && event.Type == gin.Press {
			g.ActivateAbility(player, 0)
			return
		}
		if found, event := group.FindEvent(k1.Id()); found && event.Type == gin.Press {
			g.ActivateAbility(player, 1)
			return
		}
		if player.active_ability != nil {
			if player.active_ability.Respond(player.id, group) {
				return
			}
		}
	}
}
