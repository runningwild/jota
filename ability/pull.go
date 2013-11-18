package ability

import (
	"encoding/gob"
	"github.com/runningwild/jota/game"
	"math"
)

func makePull(params map[string]float64) game.Ability {
	var p pull
	p.id = NextAbilityId()
	p.force = params["force"]
	p.angle = params["angle"] * math.Pi / 180
	p.cost = params["cost"]
	return &p
}

func init() {
	game.RegisterAbility("pull", makePull)
	gob.Register(&pull{})
}

type pull struct {
	id       int
	force    float64
	angle    float64
	cost     float64
	draw     bool
	active   bool
	trigger  bool
	draining bool
}

func (p *pull) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	p.draw = pressAmt > 0
	if !trigger || pressAmt == 0.0 {
		p.trigger = false
		p.draining = false
	}
	if !p.trigger {
		p.trigger = trigger
		p.draining = true
		player := ent.(*game.PlayerEnt)
		if pressAmt == 0 {
			delete(player.Processes, p.id)
			return
		}
		_, ok := player.Processes[p.id].(*multiDrain)
		if !ok {
			player.Processes[p.id] = &multiDrain{Gid: player.Gid, Unit: game.Mana{0, 0, p.cost}}
			return
		}
	}
}
func (p *pull) Think(ent game.Ent, g *game.Game) {
	player := ent.(*game.PlayerEnt)
	proc, ok := player.Processes[p.id].(*multiDrain)
	if !ok {
		return
	}
	if p.trigger && p.draining && proc.Stored > 1 {
		proc.Stored -= 0.1
		if proc.Stored <= 1.0 {
			p.draining = false
		}
		for _, ent := range g.Ents {
			ray := ent.Pos().Sub(player.Pos())
			if ray.Mag2() < 0.1 {
				continue
			}
			target_angle := ray.Angle() - player.Angle()
			for target_angle < 0 {
				target_angle += math.Pi * 2
			}
			for target_angle > math.Pi*2 {
				target_angle -= math.Pi * 2
			}
			if target_angle > p.angle/2 && target_angle < math.Pi*2-p.angle/2 {
				continue
			}
			ray = player.Pos().Sub(ent.Pos())
			ray = ray.Norm()
			ent.ApplyForce(ray.Scale(-p.force))
			player.ApplyForce(ray.Scale(p.force).Scale(0.01))
		}
	}
}
func (p *pull) IsActive() bool {
	return false
}
