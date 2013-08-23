package ability

import (
	"encoding/gob"
	"github.com/runningwild/cgf"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/game"
)

func makeMine(params map[string]int) game.Ability {
	var b mine
	b.id = nextAbilityId()
	b.health = float64(params["health"])
	b.damage = float64(params["damage"])
	b.trigger = float64(params["trigger"])
	b.mass = float64(params["mass"])
	return &b
}

func init() {
	game.RegisterAbility("mine", makeMine)
}

type mine struct {
	nonResponder
	nonThinker
	nonRendering

	id      int
	health  float64
	damage  float64
	trigger float64
	mass    float64
}

func (p *mine) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	if !keyPress {
		return nil, false
	}
	ret := []cgf.Event{
		addMineEvent{
			PlayerGid: gid,
			Id:        p.id,
			Health:    p.health,
			Damage:    p.damage,
			Trigger:   p.trigger,
			Mass:      p.mass,
		},
	}
	return ret, false
}

func (p *mine) Deactivate(gid game.Gid) []cgf.Event {
	return nil
}

type addMineEvent struct {
	PlayerGid game.Gid
	Id        int
	Health    float64
	Damage    float64
	Trigger   float64
	Mass      float64
}

func init() {
	gob.Register(addMineEvent{})
}

func (e addMineEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.Ents[e.PlayerGid].(*game.Player)
	pos := player.Position.Add((linear.Vec2{40, 0}).Rotate(player.Angle))
	g.MakeMine(pos, e.Health, e.Mass, e.Damage, e.Trigger)
}
