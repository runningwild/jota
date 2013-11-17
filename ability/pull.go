package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/linear"
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
func (p *pull) Draw(ent game.Ent, g *game.Game) {
	if !p.draw {
		return
	}
	player, ok := ent.(*game.PlayerEnt)
	if !ok {
		return
	}
	// TODO: Don't draw for enemies?
	gl.Color4d(1, 1, 1, 1)
	gl.Disable(gl.TEXTURE_2D)
	v1 := player.Pos()
	v2 := v1.Add(linear.Vec2{1000, 0})
	v3 := v2.RotateAround(v1, player.Angle()-p.angle/2)
	v4 := v2.RotateAround(v1, player.Angle()+p.angle/2)
	gl.Begin(gl.LINES)
	vs := []linear.Vec2{v3, v4, player.Pos()}
	for i := range vs {
		gl.Vertex2d(gl.Double(vs[i].X), gl.Double(vs[i].Y))
		gl.Vertex2d(gl.Double(vs[(i+1)%len(vs)].X), gl.Double(vs[(i+1)%len(vs)].Y))
	}
	gl.End()
}
func (p *pull) IsActive() bool {
	return false
}
