package ability

import (
  "encoding/gob"
  "fmt"
  "math"
  "github.com/runningwild/linear"
  "github.com/runningwild/magnus/base"
  "github.com/runningwild/magnus/game"
  "github.com/runningwild/magnus/stats"
)

func init() {
  game.RegisterAbility("moonfire", moonFireAbility)
  gob.Register(&moonFireProcess{})
}

// MOONFIRE
// Params - Range, radius, damage.  Range is determined by distance to origin,
// which can change as the process works.  Radius and damage are locked
// in at the beginning, origin is locked in when the user clicks.

func moonFireAbility(g *game.Game, player *game.Player, params map[string]int) game.Process {
  if len(params) != 2 {
    panic(fmt.Sprintf("moonFire requires exactly two parameters, not %v", params))
  }
  for _, req := range []string{"radius", "damage"} {
    if _, ok := params[req]; !ok {
      panic(fmt.Sprintf("moonFire requires [%s] to be specified, not %v", req, params))
    }
  }
  damage := params["damage"]
  if damage <= 0 {
    panic(fmt.Sprintf("moonFire requires [damage] > 0, not %d", damage))
  }
  radius := params["radius"]
  if radius <= 0 {
    panic(fmt.Sprintf("moonFire requires [radius] > 0, not %d", radius))
  }
  return &moonFireProcess{
    Player_id: player.Id(),
    X:         g.Mouse.X,
    Y:         g.Mouse.Y,
    Radius:    int32(radius),
    Damage:    int32(damage),
  }
}

type moonFireProcess struct {
  basicPhases
  nullCondition

  X      float64
  Y      float64
  Radius int32
  Damage int32

  Player_id int
  Required  game.Mana
  Supplied  game.Mana
  Killed    bool
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *moonFireProcess) Supply(supply game.Mana) game.Mana {
  for color := range p.Required {
    if supply[color] < p.Required[color] {
      p.Supplied[color] += supply[color]
      supply[color] = 0
    } else {
      p.Supplied[color] += p.Required[color]
      supply[color] -= p.Required[color]
    }
  }
  return supply
}

func (p *moonFireProcess) Draw(game *game.Game) {
}

func (p *moonFireProcess) Think(g *game.Game) {
  _player := g.GetEnt(p.Player_id)
  player := _player.(*game.Player)
  dx := player.Pos().X - p.X
  dy := player.Pos().Y - p.Y
  dist := math.Sqrt(dx*dx + dy*dy)
  p.Required = game.Mana{
    0,
    0,
    float64(p.Radius) * float64(p.Damage) * dist / 100,
  }
  // base.Log().Printf("Supply %2.2f / %2.2f", p.Supplied.Magnitude(), p.Required.Magnitude())

  if p.Supplied.Magnitude() >= p.Required.Magnitude() {
    p.The_phase = game.PhaseComplete
    // Do it - for realzes
    pos := linear.Vec2{p.X, p.Y}
    for _, target := range g.Ents {
      base.Log().Printf("Check %v within %d of %v", pos, p.Radius, target.Pos())
      if target.Pos().Sub(pos).Mag() <= float64(p.Radius) {
        target.ApplyDamage(stats.Damage{Amt: float64(p.Damage)})
      }
    }
  }
}
