package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
)

const LosResolution = 2048
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

type localData struct {
	game *Game

	region gui.Region

	// The engine running this game, so that the game can apply events to itself.
	engine *cgf.Engine

	// All of the players controlled by humans on localhost.
	players []*localPlayer

	los struct {
		texData    [][]uint32
		texRawData []uint32
		texId      gl.Uint
	}
}

var local localData

func (g *Game) SetEngine(engine *cgf.Engine) {
	if local.engine != nil {
		panic("Engine has already been set.")
	}
	local.engine = engine
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
}

func (g *Game) RenderLosMask() {
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
	base.SetUniformF("los", "dx", 1200)
	base.SetUniformF("los", "dy", 800)
	base.SetUniformF("los", "losMaxDist", LosMaxDist)
	base.SetUniformF("los", "losResolution", LosResolution)
	base.SetUniformF("los", "losMaxPlayers", LosMaxPlayers)
	base.SetUniformV2Array("los", "playerPos", []linear.Vec2{
		g.Ents[0].Pos(),
		g.Ents[1].Pos(),
	})
	gl.Color4d(0, 0, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0, 1)
	gl.Vertex2i(0, 0)
	gl.TexCoord2d(0, 0)
	gl.Vertex2i(0, 800)
	gl.TexCoord2d(1, 0)
	gl.Vertex2i(1200, 800)
	gl.TexCoord2d(1, 1)
	gl.Vertex2i(1200, 0)
	gl.End()
	base.EnableShader("")
}

func (g *Game) SetLocalData() {
	local.game = g
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

func LocalThink() {
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
