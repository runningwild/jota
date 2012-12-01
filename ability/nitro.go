package main

import (
  "encoding/gob"
  "fmt"
  "math"
  "runningwild/tron"
  "runningwild/tron/base"
  "runningwild/tron/game"
)

// NITRO
// Increases Max_acc by up to [inc]/nitro_acc_factor.
// Continual cost: up to [inc]*[inc]/nitro_mana_factor red mana per frame.
const nitro_mana_factor = 200
const nitro_acc_factor = 2500

func init() {
  game.RegisterAbility("nitro", nitroAbility)
  gob.Register(&nitroProcess{})
}

func nitroAbility(player *game.Player, params map[string]int) Process {
  if len(params) != 1 {
    panic(fmt.Sprintf("Nitro requires exactly one parameter, not %v", params))
  }
  for _, req := range []string{"inc"} {
    if _, ok := params[req]; !ok {
      panic(fmt.Sprintf("Nitro requires [%s] to be specified, not %v", req, params))
    }
  }
  inc := params["inc"]
  if inc <= 0 {
    panic(fmt.Sprintf("Nitro requires [inc] > 0, not %d", inc))
  }
  return &nitroProcess{
    Player_id: player.Id(),
    Inc:       int32(inc),
    Continual: Mana{float64(inc) * float64(inc) / nitro_mana_factor, 0, 0},
  }
}

type nitroProcess struct {
  noRendering
  Inc       int32
  Continual Mana
  Killed    bool
  Player_id int

  Prev_delta float64
  Supplied   Mana
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *nitroProcess) Supply(supply Mana) Mana {
  for color := range p.Continual {
    if supply[color] < p.Continual[color] {
      p.Supplied[color] += supply[color]
      supply[color] = 0
    } else {
      p.Supplied[color] += p.Continual[color]
      supply[color] -= p.Continual[color]
    }
  }
  return supply
}

func (p *nitroProcess) Think(game *game.Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*game.Player)
  player.Max_acc -= p.Prev_delta
  delta := math.Sqrt(p.Supplied.Magnitude()*nitro_mana_factor) / nitro_acc_factor
  // base.Log().Printf("Delta: %.3f", delta)
  p.Supplied = Mana{}
  player.Max_acc += delta
  p.Prev_delta = delta
}
func (p *nitroProcess) Kill(game *game.Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*game.Player)
  p.Killed = true
  player.Max_acc -= p.Prev_delta
}
func (p *nitroProcess) Complete() bool {
  return p.Killed
}
