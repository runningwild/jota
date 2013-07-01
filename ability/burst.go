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

	force, frames int
}

func (b *burst) Activate(player_id int) ([]cgf.Event, bool) {
	event := addBurstEvent{
		Player_id: player_id,
		Frames:    b.frames,
		Force:     b.force,
	}
	return []cgf.Event{event}, false
}

type addBurstEvent struct {
	Player_id int
	Frames    int
	Force     int
}

func init() {
	gob.Register(addBurstEvent{})
}

func (e addBurstEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.GetEnt(e.Player_id).(*game.Player)
	initial := game.Mana{math.Pow(float64(e.Force)*float64(e.Frames), 2) / 1.0e7, 0, 0}
	player.Processes[10] = &burstProcess{
		Frames:            int32(e.Frames),
		Force:             float64(e.Force),
		Initial:           initial,
		Remaining_initial: initial,
		Continual:         game.Mana{float64(e.Force) / 50, 0, 0},
		Player_id:         e.Player_id,
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
	Player_id         int

	// Counting how long to cast
	count int
}

func (p *burstProcess) Copy() game.Process {
	p2 := *p
	return &p2
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

func (p *burstProcess) PreThink(g *game.Game) {
}

func (p *burstProcess) Think(g *game.Game) {
	_player := g.GetEnt(p.Player_id)
	player := _player.(*game.Player)
	if p.Remaining_initial.Magnitude() == 0 {
		if p.count > 0 {
			p.count = -1
		}
		p.Frames--
		if p.Frames <= 0 {
			p.The_phase = game.PhaseComplete
		}
		for i := range g.Ents {
			other := g.Ents[i]
			if other == player {
				continue
			}
			dist := other.Pos().Sub(player.Pos()).Mag()
			if dist < 1 {
				dist = 1
			}
			force := p.Force / dist
			other.ApplyForce(other.Pos().Sub(player.Pos()).Norm().Scale(force))
		}
	}
}

func (p *burstProcess) Draw(player_id int, g *game.Game) {
	player := g.GetEnt(player_id).(*game.Player)
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
