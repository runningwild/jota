package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/magnus/texture"
	// "fmt"
	"github.com/runningwild/cgf"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"math"
)

func makeBurst(params map[string]int) game.Ability {
	var b burst
	b.id = nextAbilityId()
	b.force = params["force"]
	b.frames = params["frames"]
	return &b
}

func init() {
	game.RegisterAbility("burst", makeBurst)
}

type burst struct {
	neverActive
	nonResponder
	nonThinker
	nonRendering

	id            int
	force, frames int
}

func (b *burst) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	event := addBurstEvent{
		PlayerGid: gid,
		Id:        b.id,
		Frames:    b.frames,
		Force:     b.force,
	}
	return []cgf.Event{event}, false
}

type addBurstEvent struct {
	PlayerGid game.Gid
	Id        int
	Frames    int
	Force     int
}

func init() {
	gob.Register(addBurstEvent{})
}

func (e addBurstEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.Ents[e.PlayerGid].(*game.Player)
	initial := game.Mana{math.Pow(float64(e.Force)*float64(e.Frames), 2) / 1.0e7, 0, 0}
	player.Processes[100+e.Id] = &burstProcess{
		Frames:            int32(e.Frames),
		Force:             float64(e.Force),
		Initial:           initial,
		Remaining_initial: initial,
		Continual:         game.Mana{float64(e.Force) / 50, 0, 0},
		PlayerGid:         e.PlayerGid,
	}
}

// BURST
// All nearby players are pushed radially outward from this one.  The force
// applied to each player is max(0, [max]*(1 - (x / [radius])^2)).  This fore
// is applied constantly for [frames] frames, or until the continual cost
// cannot be paid.
// Initial cost: [radius] * [force] red mana.
// Continual cost: [frames] red mana per frame.
func init() {
	gob.Register(&burstProcess{})
}

type burstProcess struct {
	BasicPhases
	NullCondition
	Frames            int32
	Force             float64
	Initial           game.Mana
	Remaining_initial game.Mana
	Continual         game.Mana
	Killed            bool
	PlayerGid         game.Gid

	// Counting how long to cast
	count int
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *burstProcess) Supply(supply game.Mana) game.Mana {
	if p.Remaining_initial.Magnitude() > 0 {
		p.count++
		for color := range supply {
			if p.Remaining_initial[color] == 0 {
				continue
			}
			if supply[color] == 0 {
				continue
			}
			if supply[color] > p.Remaining_initial[color] {
				supply[color] -= p.Remaining_initial[color]
				p.Remaining_initial[color] = 0
			} else {
				p.Remaining_initial[color] -= supply[color]
				supply[color] = 0
			}
		}
	} else {
		for color := range p.Continual {
			if supply[color] < p.Continual[color] {
				p.Frames = 0
				return supply
			}
		}
		for color := range p.Continual {
			supply[color] -= p.Continual[color]
		}
	}
	return supply
}

func (p *burstProcess) Think(g *game.Game) {
	player := g.Ents[p.PlayerGid].(*game.Player)
	if p.Remaining_initial.Magnitude() == 0 {
		if p.count > 0 {
			p.count = -1
		}
		p.Frames--
		if p.Frames <= 0 {
			p.The_phase = game.PhaseComplete
		}
		g.DoForEnts(func(gid game.Gid, other game.Ent) {
			if other == player {
				return
			}
			dist := other.Pos().Sub(player.Pos()).Mag()
			if dist < 1 {
				dist = 1
			}
			force := p.Force / dist
			other.ApplyForce(other.Pos().Sub(player.Pos()).Norm().Scale(force))
		})
	}
}

func (p *burstProcess) Draw(gid game.Gid, g *game.Game) {
	player := g.Ents[p.PlayerGid].(*game.Player)
	base.EnableShader("circle")
	prog := p.Remaining_initial.Magnitude() / p.Initial.Magnitude()
	base.SetUniformF("circle", "progress", 1-float32(prog))
	gl.Color4ub(255, 255, 255, 255)
	radius := 40.0
	texture.Render(
		player.Position.X-radius,
		player.Position.Y-radius,
		2*radius,
		2*radius)
	base.EnableShader("")
}
