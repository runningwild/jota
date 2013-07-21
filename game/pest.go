package game

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
)

type Pest struct {
	BaseEnt
	NonManaUser
}

func init() {
	gob.Register(&Pest{})
}

func (p *Pest) Copy() Ent {
	p2 := *p
	return &p2
}

func (p *Pest) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	health_frac := float32(p.Stats().HealthCur() / p.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.EnableShader("")
}
func (p *Pest) Alive() bool {
	return p.Stats().HealthCur() > 0
}
func (p *Pest) OnDeath(g *Game) {
	g.DoForEnts(func(gid Gid, ent Ent) {
		d := ent.Pos().Sub(p.Pos()).Mag2()
		if d < 100*100 {
			ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, 100})
			// var s Sludge = 400
			// ent.Stats().ApplyCondition(&s)
		}
	})
}
func (p *Pest) Think(g *Game) {
	p.BaseEnt.Think(g)
	var target Ent
	dist := 1.0e9
	g.DoForEnts(func(gid Gid, ent Ent) {
		player, ok := ent.(*Player)
		if !ok {
			return
		}
		d := player.Pos().Sub(p.Pos()).Mag2()
		if d < dist {
			dist = d
			target = player
		}
	})
	if target == nil {
		return
	}
	if dist < 50*50 {
		p.Stats().ApplyDamage(stats.Damage{stats.DamageFire, 1})
	}
	dir := target.Pos().Sub(p.Pos()).Norm().Scale(1.0)
	p.ApplyForce(dir.Scale(10.0))
}

type Sludge int

func init() {
	var s Sludge
	gob.Register(&s)
}

func (*Sludge) ModifyBase(b stats.Base) stats.Base {
	b.Max_acc /= 2
	return b
}
func (*Sludge) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (s *Sludge) CauseDamage() stats.Damage {
	base.Log().Printf("Sludge at %v", *s)
	(*s)--
	return stats.Damage{}
}
func (s *Sludge) Terminated() bool {
	base.Log().Printf("Checking terminated")
	return *s <= 0
}

func (g *Game) AddPest(pos linear.Vec2) Ent {
	var p Pest
	err := json.NewDecoder(bytes.NewBuffer([]byte(`
      {
        "Base": {
          "Mass": 100,
          "Health": 100
        },
        "Dynamic": {
          "Health": 100
        }
      }
    `))).Decode(&p.BaseEnt.StatsInst)
	if err != nil {
		base.Log().Fatalf("%v", err)
	}
	p.Position = pos
	p.Gid = g.NextGid()
	p.Processes = make(map[int]Process)
	g.Ents[p.Gid] = &p
	return &p
}
