package main

import (
  "encoding/gob"
  "fmt"
  "math"
  "runningwild/tron"
  "runningwild/tron/base"
  "runningwild/tron/game"
)

// BURST
// All nearby players are pushed radially outward from this one.  The force
// applied to each player is max(0, [max]*(1 - (x / [radius])^2)).  This fore
// is applied constantly for [frames] frames, or until the continual cost
// cannot be paid.
// Initial cost: [radius] * [force] red mana.
// Continual cost: [frames] red mana per frame.
func init() {
  game.RegisterAbility("burst", burstAbility)
  gob.Register(&burstProcess{})
}

func burstAbility(player *game.Player, params map[string]int) Process {
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
    Remaining_initial: Mana{math.Pow(float64(force)*float64(frames), 2) / 1.0e7, 0, 0},
    Continual:         Mana{float64(force) / 50, 0, 0},
    Player_id:         player.Id(),
  }
}

type burstProcess struct {
  noRendering
  Frames            int32
  Force             float64
  Remaining_initial Mana
  Continual         Mana
  Killed            bool
  Player_id         int

  // Counting how long to cast
  count int
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *burstProcess) Supply(supply Mana) Mana {
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

func (p *burstProcess) Think(game *game.Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*game.Player)
  if p.Remaining_initial.Magnitude() == 0 {
    if p.count > 0 {
      base.Log().Printf("Frames: %d", p.count)
      p.count = -1
    }
    p.Frames--
    for i := range game.Ents {
      other := game.Ents[i]
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
func (p *burstProcess) Kill(game *game.Game) {
  p.Killed = true
}
func (p *burstProcess) Complete() bool {
  return p.Killed || p.Frames <= 0
}
