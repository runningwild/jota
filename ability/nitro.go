package ability

import (
  "encoding/gob"
  "fmt"
  "github.com/runningwild/magnus/base"
  "github.com/runningwild/magnus/game"
  "github.com/runningwild/magnus/stats"
  "math"
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

func nitroAbility(g *game.Game, player *game.Player, params map[string]int) game.Process {
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
    Continual: game.Mana{float64(inc) * float64(inc) / nitro_mana_factor, 0, 0},
  }
}

type nitroProcess struct {
  noRendering
  basicPhases
  Inc       int32
  Continual game.Mana
  Killed    bool
  Player_id int

  Prev_delta float64
  Supplied   game.Mana
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *nitroProcess) Supply(supply game.Mana) game.Mana {
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

func (p *nitroProcess) Think(g *game.Game) {
  p.Prev_delta = math.Sqrt(p.Supplied.Magnitude()*nitro_mana_factor) / nitro_acc_factor
  p.Supplied = game.Mana{}
}
func (p *nitroProcess) ModifyBase(b stats.Base) stats.Base {
  b.Max_acc += p.Prev_delta
  base.Log().Printf("Delta: %2.2f", p.Prev_delta)
  return b
}
func (*nitroProcess) ModifyDamage(damage stats.Damage) stats.Damage {
  return damage
}
func (*nitroProcess) CauseDamage() stats.Damage {
  return stats.Damage{}
}
