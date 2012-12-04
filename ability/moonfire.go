package ability

import (
  "encoding/gob"
  "fmt"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/linear"
  "github.com/runningwild/magnus/base"
  "github.com/runningwild/magnus/game"
  "github.com/runningwild/magnus/stats"
  "github.com/runningwild/magnus/texture"
  "math"
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
    Radius:    int32(radius),
    Damage:    int32(damage),
  }
}

type moonFireProcess struct {
  BasicPhases
  NullCondition

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
  if p.BasicPhases.The_phase == game.PhaseUi {
    return supply
  }
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

func (p *moonFireProcess) Draw(g *game.Game) {
  base.EnableShader("circle")
  gl.Color4ub(255, 255, 255, 255)
  texture.Render(
    float64(p.X)-float64(p.Radius),
    float64(p.Y)-float64(p.Radius),
    2*float64(p.Radius),
    2*float64(p.Radius))
  base.EnableShader("")
}

func (p *moonFireProcess) Think(g *game.Game) {
  if p.BasicPhases.The_phase == game.PhaseUi {
    if gin.In().GetKey(gin.MouseLButton).FramePressCount() > 0 {
      p.BasicPhases.The_phase = game.PhaseRunning
    } else {
      x, y := gin.In().GetCursor("Mouse").Point()
      p.X = float64(x - g.Region().Point.X)
      p.Y = float64(y - g.Region().Point.Y)
      return
    }
  }
  _player := g.GetEnt(p.Player_id)
  player := _player.(*game.Player)
  dx := player.Pos().X - p.X
  dy := player.Pos().Y - p.Y
  dist := math.Sqrt(dx*dx + dy*dy)
  p.Required = game.Mana{
    0,
    0,
    float64(p.Radius) * float64(p.Damage) * dist / 50,
  }
  // base.Log().Printf("Supply %2.2f / %2.2f", p.Supplied.Magnitude(), p.Required.Magnitude())

  if p.Supplied.Magnitude() >= p.Required.Magnitude() {
    p.The_phase = game.PhaseComplete
    // Do it - for realzes
    pos := linear.Vec2{p.X, p.Y}
    for _, target := range g.Ents {
      if target.Pos().Sub(pos).Mag() <= float64(p.Radius) {
        target.ApplyDamage(stats.Damage{Amt: float64(p.Damage)})
      }
    }
  }
}
