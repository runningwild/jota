package main

import (
  "encoding/gob"
  "fmt"
  "math"
  "path/filepath"
  "runningwild/linear"
  "runningwild/tron/base"
  "runningwild/tron/texture"
)

type Ability interface {
  Activate(player *Player, params map[string]int) Process
}

type Drain interface {
  // Request returns the most mana that this Process could use right now.
  // Some Processes can operate at any amount of mana, and some will need to
  // get all of their requested mana before they are able to do anything.
  Request() Mana

  // Supplies mana to the Process and returns the unused portion.
  Supply(Mana) Mana
}

type Thinker interface {
  Think(*Game)

  // Kills a process.  Any Killed process will return true on any future
  // calls to Complete().
  Kill(*Game)

  Complete() bool
}

type Process interface {
  Drain
  Thinker
}

// BLINK
// Causes the player to disappear for [frames] frames, where a frame is 16ms.
// Cost 50000 + [frames]^2 green mana.
func init() {
  gob.Register(blinkAbility{})
  gob.Register(&blinkProcess{})
}

type blinkAbility struct {
}

func (a *blinkAbility) Activate(player *Player, params map[string]int) Process {
  if len(params) != 1 {
    panic(fmt.Sprintf("Blink requires exactly one parameter, not %v", params))
  }
  if _, ok := params["frames"]; !ok {
    panic(fmt.Sprintf("Blink requires [frames] to be specified, not %v", params))
  }
  frames := params["frames"]
  if frames < 0 {
    panic(fmt.Sprintf("Blink requires [frames] > 0, not %d", frames))
  }
  return &blinkProcess{
    Frames:    int32(frames),
    Remaining: Mana{0, 50000 + float64(frames*frames), 0},
  }
}

type blinkProcess struct {
  Frames    int32
  Remaining Mana
  Killed    bool
  Player_id int
}

func (p *blinkProcess) Request() Mana {
  return p.Remaining
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *blinkProcess) Supply(supply Mana) Mana {
  for color := range supply {
    if p.Remaining[color] == 0 {
      continue
    }
    if supply[color] == 0 {
      continue
    }
    if supply[color] > p.Remaining[color] {
      supply[color] -= p.Remaining[color]
      p.Remaining[color] = 0
    } else {
      p.Remaining[color] -= supply[color]
      supply[color] = 0
    }
  }
  return supply
}

func (p *blinkProcess) Think(game *Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*Player)
  if p.Remaining.Magnitude() == 0 {
    player.Exile_frames += p.Frames
    p.Frames = 0
  }
}
func (p *blinkProcess) Kill(game *Game) {
  p.Killed = true
}
func (p *blinkProcess) Complete() bool {
  return p.Killed || p.Frames == 0
}

// BURST
// All nearby players are pushed radially outward from this one.  The force
// applied to each player is max(0, [max]*(1 - (x / [radius])^2)).  This fore
// is applied constantly for [frames] frames, or until the continual cost
// cannot be paid.
// Initial cost: [radius] * [force] red mana.
// Continual cost: [frames] red mana per frame.
func init() {
  gob.Register(burstAbility{})
  gob.Register(&burstProcess{})
}

type burstAbility struct {
}

func (a *burstAbility) Activate(player *Player, params map[string]int) Process {
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
  Frames            int32
  Force             float64
  Remaining_initial Mana
  Continual         Mana
  Killed            bool
  Player_id         int

  // Counting how long to cast
  count int
}

func (p *burstProcess) Request() Mana {
  if p.Remaining_initial.Magnitude() > 0 {
    return p.Remaining_initial
  }
  return p.Continual
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

func (p *burstProcess) Think(game *Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*Player)
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
func (p *burstProcess) Kill(game *Game) {
  p.Killed = true
}
func (p *burstProcess) Complete() bool {
  return p.Killed || p.Frames <= 0
}

// NITRO
// Increases Max_acc by up to [inc]/nitro_acc_factor.
// Continual cost: up to [inc]*[inc]/nitro_mana_factor red mana per frame.
const nitro_mana_factor = 200
const nitro_acc_factor = 2500

func init() {
  gob.Register(nitroAbility{})
  gob.Register(&nitroProcess{})
}

type nitroAbility struct {
}

func (a *nitroAbility) Activate(player *Player, params map[string]int) Process {
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
  Inc       int32
  Continual Mana
  Killed    bool
  Player_id int

  Prev_delta float64
  Supplied   Mana
}

func (p *nitroProcess) Request() Mana {
  return p.Continual
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

func (p *nitroProcess) Think(game *Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*Player)
  player.Max_acc -= p.Prev_delta
  delta := math.Sqrt(p.Supplied.Magnitude()*nitro_mana_factor) / nitro_acc_factor
  delta = 0.3
  // base.Log().Printf("Delta: %.3f", delta)
  p.Supplied = Mana{}
  player.Max_acc += delta
  p.Prev_delta = delta
}
func (p *nitroProcess) Kill(game *Game) {
  _player := game.GetEnt(p.Player_id)
  player := _player.(*Player)
  p.Killed = true
  player.Max_acc -= p.Prev_delta
}
func (p *nitroProcess) Complete() bool {
  return p.Killed
}

// SHOCK
// Fires a projectile at velocity [vel] that shocks any player that comes
// within [range] of it for [power] damage, and drains that much mana from
// nearby nodes.
// Initial cost: [vel]*[range]*[range]*[power] green mana.
const shock_mana_factor = 1000
const shock_acc_factor = 2500

func init() {
  gob.Register(shockAbility{})
  gob.Register(&shockProcess{})
}

type shockAbility struct {
}

func (a *shockAbility) Activate(player *Player, params map[string]int) Process {
  if len(params) != 3 {
    panic(fmt.Sprintf("Shock requires exactly three parameters, not %v", params))
  }
  for _, req := range []string{"vel", "range", "power"} {
    if _, ok := params[req]; !ok {
      panic(fmt.Sprintf("Shock requires [%s] to be specified, not %v", req, params))
    }
  }
  vel := params["vel"]
  rng := params["range"]
  power := params["power"]
  if vel <= 0 {
    panic(fmt.Sprintf("Shock requires [vel] > 0, not %d", vel))
  }
  if rng <= 0 {
    panic(fmt.Sprintf("Shock requires [rng] > 0, not %d", rng))
  }
  if power <= 0 {
    panic(fmt.Sprintf("Shock requires [power] > 0, not %d", power))
  }
  base.Log().Printf("Lauched shock from %d", player.Id())
  return &shockProcess{
    Player_id: player.Id(),
    Remaining: Mana{float64(vel) * float64(rng) * float64(rng) * float64(power) / shock_mana_factor, 0, 0},
    Power:     float64(power),
    Range:     float64(rng),
    Velocity:  float64(vel),
  }
}

type shockState int

const (
  shockStateGather shockState = iota
  shockStateLaunched
  shockStateKilled
)

type shockProcess struct {
  Remaining    Mana
  State        shockState
  Player_id    int
  Proj_id      int
  Power        float64
  Velocity     float64
  Range        float64
  X, Y, Dx, Dy float64
}

func (p *shockProcess) Request() Mana {
  return p.Remaining
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *shockProcess) Supply(supply Mana) Mana {
  for color := range supply {
    if p.Remaining[color] == 0 {
      continue
    }
    if supply[color] == 0 {
      continue
    }
    if supply[color] > p.Remaining[color] {
      supply[color] -= p.Remaining[color]
      p.Remaining[color] = 0
    } else {
      p.Remaining[color] -= supply[color]
      supply[color] = 0
    }
  }
  return supply
}

func (p *shockProcess) Think(game *Game) {
  if p.Remaining.Magnitude() > 0 {
    return
  }
  _player := game.GetEnt(p.Player_id)
  player := _player.(*Player)
  if p.State == shockStateGather {
    p.State = shockStateLaunched
    var proj Projectile
    proj.My_pos.X = player.X
    proj.My_pos.Y = player.Y
    proj.My_vel.X = player.Vx
    proj.My_vel.Y = player.Vy
    proj.My_vel.X += math.Cos(player.Angle) * p.Velocity * 10
    proj.My_vel.Y += math.Sin(player.Angle) * p.Velocity * 10
    proj.Player_id = player.Id()
    proj.Range = p.Range
    proj.Think(game)
    p.Proj_id = game.AddEnt(&proj)
    base.Log().Printf("LAUNCH %d", p.Proj_id)
  }
}
func (p *shockProcess) Kill(game *Game) {
  p.State = shockStateKilled
}
func (p *shockProcess) Complete() bool {
  return p.State != shockStateGather
}

type Projectile struct {
  My_id          int
  My_pos, My_vel linear.Vec2
  Dead           bool
  Player_id      int
  Range          float64
}

func (p *Projectile) Draw() {
  t := texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/shock.png"))
  t.RenderAdvanced(p.Pos().X-float64(t.Dx())/2, p.Pos().Y-float64(t.Dy())/2, float64(t.Dx()), float64(t.Dy()), math.Atan2(p.Vel().Y, p.Vel().X), false)
}
func (p *Projectile) Alive() bool {
  return !p.Dead
}
func (p *Projectile) Exiled() bool {
  return false
}
func (p *Projectile) Think(game *Game) {
  p.My_pos = p.My_pos.Add(p.My_vel)
  var hits []*Player
  activate := false
  for i := range game.Ents {
    if _, ok := game.Ents[i].(*Player); !ok {
      continue
    }
    if game.Ents[i].Id() == p.Player_id {
      continue
    }
    dist := p.Pos().Sub(game.Ents[i].Pos()).Mag()
    if dist < p.Range {
      hits = append(hits, game.Ents[i].(*Player))
      if dist < p.Range/2 {
        activate = true
      }
    }
  }
  if activate {
    p.Dead = true
    for _, hit := range hits {
      hit.Dead = true
    }
    for x := range game.Nodes {
      for y := range game.Nodes[x] {
        node := &game.Nodes[x][y]
        dist := p.Pos().Sub(linear.Vec2{node.X, node.Y}).Mag()
        if dist < p.Range*2 {
          if dist < p.Range {
            node.Amt = 0
          } else {
            node.Amt *= (dist - p.Range) / p.Range
          }
        }
      }
    }
  }
}
func (p *Projectile) ApplyForce(force linear.Vec2) {
}
func (p *Projectile) Mass() float64 {
  return 1
}
func (p *Projectile) Rate(dist float64) float64 {
  return 0
}
func (p *Projectile) SetId(int) {
}
func (p *Projectile) Id() int {
  return p.My_id
}
func (p *Projectile) Pos() linear.Vec2 {
  return p.My_pos
}
func (p *Projectile) SetPos(pos linear.Vec2) {
}
func (p *Projectile) Vel() linear.Vec2 {
  return p.My_vel
}
func (p *Projectile) SetVel(vel linear.Vec2) {
}
func (p *Projectile) Supply(mana Mana) Mana {
  return mana
}
