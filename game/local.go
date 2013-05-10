package game

import (
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/magnus/base"
)

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
}

var local localData

func (g *Game) SetEngine(engine *cgf.Engine) {
	if local.engine != nil {
		panic("Engine has already been set.")
	}
	local.engine = engine
	gin.In().RegisterEventListener(&gameResponderWrapper{g})
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
		k0 := gin.In().GetKeyFlat(gin.ControllerButton0+1, gin.DeviceTypeController, player.device_index)
		k1 := gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, player.device_index)
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
