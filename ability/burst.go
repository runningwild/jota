package ability

import (
	"encoding/gob"
	"fmt"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/pnf"
	"math"
)

func makeBurst(params map[string]int) game.Ability {
	var b burst

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

func (b *burst) Activate(player_id int) ([]pnf.Event, bool) {
	event := addBurstEvent{
		Player_id: player_id,
		Frames:    3,      // b.frames,
		Force:     100000, //b.force,
	}
	return []pnf.Event{event}, false
}

type addBurstEvent struct {
	Player_id int
	Frames    int
	Force     int
}

func (e addBurstEvent) ApplyFirst(g interface{}) {}
func (e addBurstEvent) ApplyFinal(g interface{}) {}
func (e addBurstEvent) Apply(_g interface{}) {
	base.Log().Printf("A")
	g := _g.(*game.Game)
	base.Log().Printf("A")
	player := g.GetEnt(e.Player_id).(*game.Player)
	base.Log().Printf("A")
	player.Processes[100] = &burstProcess{
		Frames:            int32(e.Frames),
		Force:             float64(e.Force),
		Remaining_initial: game.Mana{math.Pow(float64(e.Force)*float64(e.Frames), 2) / 1.0e7, 0, 0},
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

func burstAbility(g *game.Game, player *game.Player, params map[string]int) game.Process {
	if len(params) != 2 {
		panic(fmt.Sprintf("Burst requires exactly two parameters, not %v", params))
	}
	for _, req := range []string{"frames", "force"} {
		if _, ok := params[req]; !ok {
			panic(fmt.Sprintf("Burst requires [%s] to be specified, not %v", req, params))
		}
	}
	frames := params["frames"]
	force := params["force"]
	if frames < 0 {
		panic(fmt.Sprintf("Burst requires [frames] > 0, not %d", frames))
	}
	if force < 0 {
		panic(fmt.Sprintf("Burst requires [force] > 0, not %d", force))
	}
	return &burstProcess{
		Frames:            int32(frames),
		Force:             float64(force),
		Remaining_initial: game.Mana{math.Pow(float64(force)*float64(frames), 2) / 1.0e7, 0, 0},
		Continual:         game.Mana{float64(force) / 50, 0, 0},
		Player_id:         player.Id(),
	}
}

type burstProcess struct {
	NoRendering
	BasicPhases
	NullCondition
	Frames            int32
	Force             float64
	Remaining_initial game.Mana
	Continual         game.Mana
	Killed            bool
	Player_id         int

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
	_player := g.GetEnt(p.Player_id)
	player := _player.(*game.Player)
	base.Log().Printf("Player %v, supplied %v", player, p.Remaining_initial)
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
