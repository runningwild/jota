package game

import (
  "bytes"
  "encoding/gob"
  //"fmt"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/cgf"
  "github.com/runningwild/cmwc"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/util/algorithm"
  "github.com/runningwild/linear"
  "github.com/runningwild/magnus/base"
  "github.com/runningwild/magnus/stats"
  "github.com/runningwild/magnus/texture"
  "math"
  "path/filepath"
)

// type Ability func(game *Game, player *Player, params map[string]int) Process

// An Ability represents something a player can do that does not directly affect
// the game state.
type Ability interface {
  // Called when a player selects this Ability.  Returns any number of events to
  // apply, as well as a bool that is true iff this Ability should become the
  // active Ability.
  Activate(player_id int) ([]cgf.Event, bool)

  // Called when this Ability is deselected.
  Deactivate(player_id int) []cgf.Event

  // The active Ability will receive all of the events from the player.  It
  // should return true iff it consumes the event.
  Respond(player_id int, group gin.EventGroup) bool

  // Returns any number of events to apply, as well as a bool that is true iff
  // this Ability should be deactivated.  Typically this will include an event
  // that will add a Process to this player.
  Think(player_id int, game *Game) ([]cgf.Event, bool)

  // If it is the active Ability it might want to draw some Ui stuff.
  Draw(player_id int, game *Game)
}

type AbilityMaker func(params map[string]int) Ability

var ability_makers map[string]AbilityMaker

func RegisterAbility(name string, maker AbilityMaker) {
  if ability_makers == nil {
    ability_makers = make(map[string]AbilityMaker)
  }
  ability_makers[name] = maker
}

type Drain interface {
  // Supplies mana to the Process and returns the unused portion.
  Supply(Mana) Mana
}

type Phase int

const (
  // This phase is for any process that needs a ui before.  A player can only
  // have one Process in PhaseUi at a time.  If a player tries to use an ability
  // while a Process is in PhaseUi the process in PhaseUi will be killed.
  PhaseUi Phase = iota

  // Once a Process hits PhaseRunning it will remain here until it is complete.
  // A process should not reach this phase until it is done with player
  // interaction.
  PhaseRunning

  // Once a Process returns PhaseComplete it will always return PhaseComplete.
  PhaseComplete
)

type Thinker interface {
  PreThink(game *Game)
  Think(game *Game)

  // Kills a process.  Any Killed process will return true on any future
  // calls to Complete().
  Kill(game *Game)

  Phase() Phase
}

// TODO: Might want to be able to respond to events directly for Ui stuff
type Responder interface {
}

type Process interface {
  Drain
  Thinker
  Responder
  stats.Condition
  Draw(player_id int, game *Game)
}

const mana_brightness = 150
const mana_cap = 200
const mana_regen = 0.003

const node_spacing = 10

type Color int

const (
  ColorRed Color = iota
  ColorGreen
  ColorBlue
)

// One value for each color
type Mana [3]float64

func (m Mana) Magnitude() float64 {
  return m[0] + m[1] + m[2]
}

type Node struct {
  X, Y      float64
  Color     Color // TODO: Delete (used only for seeds)
  Regen     float64
  Amount    []float64
  MaxAmount []float64
}

func init() {
  gob.Register(&Node{})
}

func (n *Node) Think() {
  for i := range n.Amount {
    n.Amount[i] += n.MaxAmount[i] * n.Regen
    if n.Amount[i] > n.MaxAmount[i] {
      n.Amount[i] = n.MaxAmount[i]
    }
  }
}

type Player struct {
  Stats  stats.Inst
  X, Y   float64
  Vx, Vy float64
  Angle  float64
  Delta  struct {
    Speed float64
    Angle float64
  }
  Color struct {
    R, G, B byte
  }

  // Unique Id over all entities ever
  Gid int

  // If Exile_frames > 0 then the Player is not present in the game right now
  // and is excluded from all combat/mana/rendering/processing/etc...
  // Exile_frames is the number of frames remaining that the player is in
  // exile.
  Exile_frames int32

  // Processes contains all of the processes that this player is casting
  // right now.
  Processes map[int]Process
}

func init() {
  gob.Register(&Player{})
}

func (p *Player) Alive() bool {
  return p.Stats.HealthCur() > 0
}

func (p *Player) Exiled() bool {
  return p.Exile_frames > 0
}

func (p *Player) ApplyForce(f linear.Vec2) {
  dv := f.Scale(1 / p.Mass())
  p.Vx += dv.X
  p.Vy += dv.Y
}

func (p *Player) ApplyDamage(d stats.Damage) {
  p.Stats.ApplyDamage(d)
}

func (p *Player) Mass() float64 {
  return p.Stats.Mass()
}

func (p *Player) Id() int {
  return p.Gid
}

func (p *Player) SetId(id int) {
  p.Gid = id
}

func (p *Player) Pos() linear.Vec2 {
  return linear.Vec2{p.X, p.Y}
}

func (p *Player) Vel() linear.Vec2 {
  return linear.Vec2{p.Vx, p.Vy}
}

func (p *Player) SetPos(pos linear.Vec2) {
  p.X = pos.X
  p.Y = pos.Y
}

func (p *Player) SetVel(vel linear.Vec2) {
  p.X = vel.X
  p.Y = vel.Y
}

func (p *Player) Rate(distance_squared float64) float64 {
  M := 10.0
  D := 100.0
  ret := M * (1 - distance_squared/(D*D))
  if ret < 0 {
    ret = 0
  }
  return ret
}

func (p *Player) Draw(game *Game) {
  if p.Exiled() {
    return
  }
  var t *texture.Data
  if p.Id() == 1 {
    t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
  } else if p.Id() == 2 {
    t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship3.png"))
  } else {
    t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship2.png"))
  }
  t.RenderAdvanced(p.X-float64(t.Dx())/2, p.Y-float64(t.Dy())/2, float64(t.Dx()), float64(t.Dy()), p.Angle, false)

  for _, proc := range p.Processes {
    proc.Draw(p.Id(), game)
  }
}

func (p *Player) PreThink(g *Game) {
  for _, proc := range p.Processes {
    proc.PreThink(g)
  }
}

func (p *Player) Think(g *Game) {
  if p.Exile_frames > 0 {
    p.Exile_frames--
    return
  }

  // This will clear out old conditions
  p.Stats.Think()
  var dead []int
  for i, process := range p.Processes {
    process.Think(g)
    if process.Phase() == PhaseComplete {
      dead = append(dead, i)
    }
  }
  for _, i := range dead {
    delete(p.Processes, i)
  }
  // And here we add back in all processes that are still alive.
  for _, process := range p.Processes {
    p.Stats.ApplyCondition(process)
  }

  if p.Delta.Speed > p.Stats.MaxAcc() {
    p.Delta.Speed = p.Stats.MaxAcc()
  }
  if p.Delta.Speed < -p.Stats.MaxAcc() {
    p.Delta.Speed = -p.Stats.MaxAcc()
  }
  if p.Delta.Angle < -p.Stats.MaxTurn() {
    p.Delta.Angle = -p.Stats.MaxTurn()
  }
  if p.Delta.Angle > p.Stats.MaxTurn() {
    p.Delta.Angle = p.Stats.MaxTurn()
  }

  in_lava := false
  for _, lava := range g.Room.Lava {
    if vecInsideConvexPoly(p.Pos(), lava) {
      in_lava = true
    }
  }
  if in_lava {
    p.Stats.ApplyDamage(stats.Damage{stats.DamageFire, 5})
  }

  p.Vx += p.Delta.Speed * math.Cos(p.Angle)
  p.Vy += p.Delta.Speed * math.Sin(p.Angle)
  mangle := math.Atan2(p.Vy, p.Vx)
  friction := g.Friction
  if in_lava {
    friction = g.Friction_lava
  }
  p.Vx *= math.Pow(friction, 1+3*math.Abs(math.Sin(p.Angle-mangle)))
  p.Vy *= math.Pow(friction, 1+3*math.Abs(math.Sin(p.Angle-mangle)))

  move := linear.MakeSeg2(p.X, p.Y, p.X+p.Vx, p.Y+p.Vy)
  size := 12.0
  px := p.X
  py := p.Y
  p.X += p.Vx
  p.Y += p.Vy
  for _, poly := range g.Room.Walls {
    for i := range poly {
      // First check against the leading vertex
      {
        v := poly[i]
        dist := v.DistToLine(move)
        if v.Sub(move.Q).Mag() < size {
          dist = v.Sub(move.Q).Mag()
          // Add a little extra here otherwise a player can sneak into geometry
          // through the corners
          ray := move.Q.Sub(v).Norm().Scale(size + 0.1)
          final := v.Add(ray)
          move.Q.X = final.X
          move.Q.Y = final.Y
        } else if dist < size {
          // TODO: This tries to prevent passthrough but has other problems
          // cross := move.Ray().Cross()
          // perp := linear.Seg2{v, cross.Sub(v)}
          // if perp.Left(move.P) != perp.Left(move.Q) {
          //   shift := perp.Ray().Norm().Scale(size - dist)
          //   move.Q.X += shift.X
          //   move.Q.Y += shift.Y
          // }
        }
      }

      // Now check against the segment itself
      w := poly.Seg(i)
      if w.Ray().Cross().Dot(move.Ray()) <= 0 {
        shift := w.Ray().Cross().Norm().Scale(size)
        col := linear.Seg2{shift.Add(w.P), shift.Add(w.Q)}
        if move.DoesIsect(col) {
          cross := col.Ray().Cross()
          fix := linear.Seg2{move.Q, cross.Add(move.Q)}
          isect := fix.Isect(col)
          move.Q.X = isect.X
          move.Q.Y = isect.Y
        }
      }
    }
  }
  p.X = move.Q.X
  p.Y = move.Q.Y
  p.Vx = p.X - px
  p.Vy = p.Y - py

  p.Angle += p.Delta.Angle

  p.Delta.Angle = 0
  p.Delta.Speed = 0

}

func (p *Player) Supply(supply Mana) Mana {
  for _, process := range p.Processes {
    supply = process.Supply(supply)
  }
  return supply
}

type Ent interface {
  Draw(g *Game)
  Alive() bool
  Exiled() bool
  PreThink(game *Game)
  Think(game *Game)
  ApplyForce(force linear.Vec2)
  ApplyDamage(damage stats.Damage)
  Mass() float64
  Rate(dist_sq float64) float64
  SetId(int)
  Id() int
  Pos() linear.Vec2
  SetPos(pos linear.Vec2)
  Vel() linear.Vec2
  SetVel(vel linear.Vec2)
  Supply(mana Mana) Mana
}

type nodeGrid [][]Node

func (ng *nodeGrid) GobDecode(data []byte) error {
  dec := gob.NewDecoder(bytes.NewBuffer(data))
  var dx, dy uint32
  err := dec.Decode(&dx)
  if err != nil {
    return err
  }
  err = dec.Decode(&dy)
  if err != nil {
    return err
  }
  *ng = make([][]Node, dx)
  for x := range *ng {
    (*ng)[x] = make([]Node, dy)
  }
  for x := range *ng {
    for y := range (*ng)[x] {
      err = dec.Decode(&((*ng)[x][y]))
      if err != nil {
        return err
      }
    }
  }
  return nil
}

func (ng *nodeGrid) GobEncode() ([]byte, error) {
  buf := bytes.NewBuffer(nil)
  enc := gob.NewEncoder(buf)
  err := enc.Encode(uint32(len(*ng)))
  if err != nil {
    return nil, err
  }
  err = enc.Encode(uint32(len((*ng)[0])))
  if err != nil {
    return nil, err
  }
  for x := range *ng {
    for y := range *ng {
      err = enc.Encode((*ng)[x][y])
      if err != nil {
        return nil, err
      }
    }
  }
  return buf.Bytes(), nil
}

type Game struct {
  // All of the nodes on the map
  Nodes [][]Node

  Room Room

  Rng *cmwc.Cmwc

  // Dimensions of the board
  Dx, Dy int

  Friction      float64
  Friction_lava float64

  // Last Id assigned to an entity
  Next_id int

  Ents []Ent

  Game_thinks int
}

type localData struct {
  game *Game

  // All of the abilities that this player can activate.  These are only present
  // for the local player.
  abilities []Ability

  region gui.Region

  // The engine running this game, so that the game can apply events to itself.
  engine *cgf.Engine

  // The index of the player that is being controlled by localhost
  local_player *Player

  // The local player's active ability, if any
  active_ability Ability
}

var local localData

func init() {
  gob.Register(&Game{})
}

type gameResponderWrapper struct {
  g *Game
}

func (grw *gameResponderWrapper) HandleEventGroup(group gin.EventGroup) {
  grw.g.HandleEventGroup(group)
}

func (grw *gameResponderWrapper) Think(int64) {}

func (g *Game) SetEngine(engine *cgf.Engine) {
  if local.engine != nil {
    panic("Engine has already been set.")
  }
  local.engine = engine
  gin.In().RegisterEventListener(&gameResponderWrapper{g})
}

func (g *Game) HandleEventGroup(group gin.EventGroup) {
  k0 := gin.In().GetKeyFlat(gin.ControllerButton0+1, gin.DeviceTypeController, gin.DeviceIndexAny)
  k1 := gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, gin.DeviceIndexAny)
  if found, event := group.FindEvent(k0.Id()); found && event.Type == gin.Press {
    g.ActivateAbility(0)
    return
  }
  if found, event := group.FindEvent(k1.Id()); found && event.Type == gin.Press {
    g.ActivateAbility(1)
    return
  }
  if local.active_ability != nil {
    if local.active_ability.Respond(local.local_player.Id(), group) {
      return
    }
  }
}

func (g *Game) SetLocalPlayer(local_player *Player) {
  if local.local_player != nil {
    panic("Local player has already been set.")
  }
  local.local_player = local_player
  local.abilities = append(
    local.abilities,
    ability_makers["burst"](map[string]int{
      "frames": 2,
      "force":  200000,
    }))
  local.abilities = append(
    local.abilities,
    ability_makers["pull"](map[string]int{
      "frames": 10,
      "force":  250,
      "angle":  90,
    }))
  local.game = g
}

func (g *Game) ActivateAbility(n int) {
  active_ability := local.active_ability
  local.active_ability = nil
  if active_ability != nil {
    events := active_ability.Deactivate(local.local_player.Id())
    for _, event := range events {
      local.engine.ApplyEvent(event)
    }
    if active_ability == local.abilities[n] {
      return
    }
  }
  events, active := local.abilities[n].Activate(local.local_player.Id())
  for _, event := range events {
    local.engine.ApplyEvent(event)
  }
  if active {
    local.active_ability = local.abilities[n]
    base.Log().Printf("Setting active ability to %v", local.active_ability)
  }
}

func vecInsideConvexPoly(v linear.Vec2, p linear.Poly) bool {
  for i := range p {
    seg := p.Seg(i)
    if seg.Left(v) {
      return false
    }
  }
  return true
}

func invSquareDist(dist_sq float64) float64 {
  return 1.0 / (dist_sq + 1)
}

func getWeights(distance_squares []float64, value_sum float64, transform func(float64) float64) []float64 {
  weights := make([]float64, len(distance_squares))

  weight_sum := 0.0
  for i, dist_sq := range distance_squares {
    if dist_sq >= 0 {
      weights[i] = transform(dist_sq)
    } else {
      weights[i] = 0
    }
    weight_sum += weights[i]
  }

  for i, w := range weights {
    weights[i] = value_sum * w / weight_sum
  }
  return weights
}

func (g *Game) GenerateNodes() {
  c := cmwc.MakeGoodCmwc()
  c.SeedWithDevRand()
  g.Nodes = make([][]Node, 1+g.Dx/node_spacing)
  var primary_nodes []Node
  for i := 0; i < 9; i++ {
    x := int(c.Int63() % int64(g.Dx))
    y := int(c.Int63() % int64(g.Dy))
    color := int(c.Int63() % 3)
    primary_nodes = append(primary_nodes, Node{
      X:     float64(x),
      Y:     float64(y),
      Color: Color(color),
    })
  }
  var all_polys []linear.Poly
  for _, p := range g.Room.Walls {
    all_polys = append(all_polys, p)
  }
  for _, p := range g.Room.Lava {
    all_polys = append(all_polys, p)
  }
  for x := 0; x < 1+g.Dx/node_spacing; x++ {
    g.Nodes[x] = make([]Node, 1+g.Dy/node_spacing)
    for y := 0; y < 1+g.Dy/node_spacing; y++ {
      good := true
      for i := 1; good && i < len(all_polys); i++ {
        v := linear.Vec2{float64(x * node_spacing), float64(y * node_spacing)}
        if vecInsideConvexPoly(v, all_polys[i]) {
          good = false
        }
      }
      if !good {
        continue
      }

      nearest_by_color := []float64{-1.0, -1.0, -1.0}
      for i := range primary_nodes {
        c := primary_nodes[i].Color
        dx := float64(x*node_spacing) - primary_nodes[i].X
        dy := float64(y*node_spacing) - primary_nodes[i].Y
        dist_sq := dx*dx + dy*dy
        if nearest_by_color[c] < 0 || nearest_by_color[c] > dist_sq {
          nearest_by_color[c] = dist_sq
        }
      }

      weights := getWeights(nearest_by_color, mana_cap, invSquareDist)
      max_weights := make([]float64, len(weights))
      copy(max_weights, weights)

      g.Nodes[x][y] = Node{
        X:         float64(x * node_spacing),
        Y:         float64(y * node_spacing),
        Regen:     mana_regen,
        Amount:    weights,
        MaxAmount: max_weights,
      }
    }
  }
}

func (g *Game) Merge(g2 *Game) {
  frac := 0.5
  for i := range g.Ents {
    _p1 := g.Ents[i]
    var p1 *Player
    var ok bool
    if p1, ok = _p1.(*Player); !ok {
      continue
    }
    p2, ok := g2.GetEnt(p1.Id()).(*Player)
    if p2 == nil || !ok {
      continue
    }
    p1.X = frac*p2.X + (1-frac)*p1.X
    p1.Y = frac*p2.Y + (1-frac)*p1.Y
    p1.Angle = frac*p2.Angle + (1-frac)*p1.Angle
  }
}

func (g *Game) Copy() interface{} {
  var g2 Game

  g2.Nodes = make([][]Node, len(g.Nodes))
  for x := range g2.Nodes {
    g2.Nodes[x] = make([]Node, len(g.Nodes[x]))
    for y := range g2.Nodes[x] {
      g2.Nodes[x][y] = g.Nodes[x][y]
    }
  }

  g2.Room = g.Room

  g2.Rng = g.Rng.Copy()

  g2.Dx = g.Dx
  g2.Dy = g.Dy
  g2.Friction = g.Friction
  g2.Friction_lava = g.Friction_lava
  g2.Next_id = g.Next_id
  g2.Game_thinks = g.Game_thinks

  g2.Ents = make([]Ent, len(g.Ents))
  g2.Ents = g2.Ents[0:0]
  for _, ent := range g.Ents {
    switch e := ent.(type) {
    case *Player:
      p := *e
      g2.Ents = append(g2.Ents, &p)
    }
  }

  return &g2
}

func (g *Game) OverwriteWith(_g2 interface{}) {
  g2 := _g2.(*Game)
  g.Rng.OverwriteWith(g2.Rng)
  g.Dx = g2.Dx
  g.Dy = g2.Dy
  g.Friction = g2.Friction
  g.Room.Walls = g2.Room.Walls
  g.Next_id = g2.Next_id
  g.Game_thinks = g2.Game_thinks

  g.Ents = g.Ents[0:0]
  for _, ent := range g2.Ents {
    switch e := ent.(type) {
    case *Player:
      p := *e
      g.Ents = append(g.Ents, &p)
    }
  }

  if len(g.Nodes) != len(g2.Nodes) {
    g.Nodes = make([][]Node, len(g2.Nodes))
  }
  for x := range g.Nodes {
    g.Nodes[x] = g.Nodes[x][0:0]
    for y := range g2.Nodes[x] {
      g.Nodes[x] = append(g.Nodes[x], g2.Nodes[x][y])
    }
  }
}

func (g *Game) GetEnt(id int) Ent {
  for i := range g.Ents {
    if g.Ents[i].Id() == id {
      return g.Ents[i]
    }
  }
  return nil
}

func (g *Game) AddEnt(ent Ent) int {
  g.Next_id++
  ent.SetId(g.Next_id)
  g.Ents = append(g.Ents, ent)
  return g.Ents[len(g.Ents)-1].Id()
}

// Returns a mapping from player index to the list of *Nodes that that player
// has priority on.
func (g *Game) getPriorities() [][]*Node {
  r := make([][]*Node, len(g.Ents))

  for x := range g.Nodes {
    for y := range g.Nodes[x] {
      var best int = -1
      var best_rate float64 = 0.0
      for j := range g.Ents {
        dist_sq := g.Ents[j].Pos().Sub(linear.MakeVec2(g.Nodes[x][y].X, g.Nodes[x][y].Y)).Mag2()
        rate := g.Ents[j].Rate(dist_sq)
        if rate > best_rate {
          best_rate = rate
          best = j
        }
      }
      if best == -1 {
        continue
      }
      r[best] = append(r[best], &g.Nodes[x][y])
    }
  }
  return r
}

func (g *Game) ThinkFirst() {}
func (g *Game) ThinkFinal() {}
func (g *Game) Think() {
  g.Game_thinks++

  algorithm.Choose(&g.Ents, func(e Ent) bool { return e.Alive() })

  for i := range g.Ents {
    g.Ents[i].PreThink(g)
  }
  g.nodeAndSupplyThink()

  // Advance players, check for collisions, add segments
  for i := range g.Ents {
    if !g.Ents[i].Alive() {
      continue
    }
    g.Ents[i].Think(g)
    pos := g.Ents[i].Pos()
    pos.X = clamp(pos.X, 0, float64(g.Dx))
    pos.Y = clamp(pos.Y, 0, float64(g.Dy))
    g.Ents[i].SetPos(pos)
  }
  moved := make(map[int]bool)
  for i := range g.Ents {
    for j := range g.Ents {
      if i == j || g.Ents[i].Exiled() || g.Ents[j].Exiled() {
        continue
      }
      dist := g.Ents[i].Pos().Sub(g.Ents[j].Pos()).Mag()
      if dist > 25 {
        continue
      }
      if dist < 0.01 {
        continue
      }
      if dist <= 0.5 {
        dist = 0.5
      }
      force := 20.0 * (25 - dist)
      g.Ents[i].ApplyForce(g.Ents[i].Pos().Sub(g.Ents[j].Pos()).Norm().Scale(force))
      moved[i] = true
    }
  }
}

/*
func (g *Game) getPriorities() [][]*Node {
  r := make([][]*Node, len(g.Ents))

  for x := range g.Nodes {
    for y := range g.Nodes[x] {
      var best int = -1
      var best_rate float64 = 0.0
      for j := range g.Ents {
        dist_sq := g.Ents[j].Pos().Sub(linear.MakeVec2(g.Nodes[x][y].X, g.Nodes[x][y].Y)).Mag2()
        rate := g.Ents[j].Rate(dist_sq)
        if rate > best_rate {
          best_rate = rate
          best = j
        }
      }
      if best == -1 {
        continue
      }
      r[best] = append(r[best], &g.Nodes[x][y])
    }
  }
  return r
}
*/

func (g *Game) nodeAndSupplyThink2() {
  mana_ownership_fraction := make([][][]float64, len(g.Nodes))
  mana_available := make([][]float64, 3)
  for c_index := 0; c_index < 3; c_index++ {
    mana_available[c_index] = make([]float64, len(g.Ents))
  }

  for x_index := range g.Nodes {
    mana_ownership_fraction[x_index] = make([][]float64, len(g.Nodes[x_index]))
    for y_index := range g.Nodes[x_index] {
      mana_ownership_fraction[x_index][y_index] = make([]float64, len(g.Ents))
      node := g.Nodes[x_index][y_index]
      dist_sqs := make([]float64, len(g.Ents))
      for e_index := range g.Ents {
        dist_sqs[e_index] = g.Ents[e_index].Pos().Sub(
          linear.MakeVec2(node.X, node.Y)).Mag2()
      }
      mana_ownership_fraction[x_index][y_index] = getWeights(
        dist_sqs, 1 /* value_sum */, invSquareDist)
      for e_index := range dist_sqs {
        mana_ownership_fraction[x_index][y_index][e_index] *=
          g.Ents[e_index].Rate(dist_sqs[e_index])
        for c_index := 0; c_index < len(node.Amount); c_index++ {
          mana_available[c_index][e_index] += node.Amount[c_index] *
            mana_ownership_fraction[x_index][y_index][e_index]
          // This is not the plan.
          node.Amount[c_index] = math.Max(
            0, node.Amount[c_index]*(1-mana_ownership_fraction[x_index][y_index][e_index]))
        }
      }
    }
  }

}

func (g *Game) nodeAndSupplyThink() {
  g.nodeAndSupplyThink2()
  /*priorities := g.getPriorities()
    indexes := make([]int, len(g.Ents))
    for i := range indexes {
      indexes[i] = i
    }
    for i := range indexes {
      swap := int(g.Rng.Uint32()%uint32(len(g.Ents)-i)) + i
      indexes[i], indexes[swap] = indexes[swap], indexes[i]
    }
    for _, p := range indexes {
      player := g.Ents[p]
      nodes := priorities[p]

      for i := range nodes {
        swap := int(g.Rng.Uint32()%uint32(len(nodes)-i)) + i
        nodes[i], nodes[swap] = nodes[swap], nodes[i]
      }

      var supply Mana
      for _, node := range nodes {
        drain := player.Rate(player.Pos().Sub(linear.Vec2{node.X, node.Y}).Mag())
        for i := 0; i < len(node.Amount); i++ {
          supply[i] += math.Min(node.Amount[i], drain)
        }
        if len(node.Amount) != 3 {
          fmt.Println("What teh crapz?", node)
        }
      }

      var used Mana
      for color, amt := range supply {
        used[color] = amt
      }
      supply = player.Supply(supply)
      for color, amt := range supply {
        used[color] -= amt
      }
      for _, node := range nodes {
        drain := player.Rate(player.Pos().Sub(linear.Vec2{node.X, node.Y}).Mag())
        for i := 0; i < len(node.Amount); i++ {
          drain = math.Min(math.Min(node.Amount[i], used[i]), drain)
          node.Amount[i] -= drain
          used[i] -= drain
        }
      }
    }*/

  for x := range g.Nodes {
    for y := range g.Nodes[x] {
      g.Nodes[x][y].Think()
    }
  }
}

func LocalThink() {
  if local.active_ability != nil {
    events, die := local.active_ability.Think(local.local_player.Id(), local.game)
    if die {
      more_events := local.active_ability.Deactivate(local.local_player.Id())
      local.active_ability = nil
      for _, event := range more_events {
        events = append(events, event)
      }
    }
    for _, event := range events {
      local.engine.ApplyEvent(event)
    }
  }
}

func clamp(v, low, high float64) float64 {
  if v < low {
    return low
  }
  if v > high {
    return high
  }
  return v
}

type nodeIndex struct {
  x, y int
}

type Turn struct {
  Player_id int
  Delta     float64
}

func init() {
  gob.Register(Turn{})
}

func (t Turn) ApplyFirst(g interface{}) {}
func (t Turn) ApplyFinal(g interface{}) {}
func (t Turn) Apply(_g interface{}) {
  g := _g.(*Game)
  _player := g.GetEnt(t.Player_id)
  if _player == nil {
    return
  }
  player := _player.(*Player)
  player.Delta.Angle = t.Delta
}

type Accelerate struct {
  Player_id int
  Delta     float64
}

func init() {
  gob.Register(Accelerate{})
}

func (a Accelerate) ApplyFirst(g interface{}) {}
func (a Accelerate) ApplyFinal(g interface{}) {}
func (a Accelerate) Apply(_g interface{}) {
  g := _g.(*Game)
  _player := g.GetEnt(a.Player_id)
  if _player == nil {
    return
  }
  player := _player.(*Player)
  player.Delta.Speed = a.Delta / 2
}

// type Burst struct {
// 	Player_id int
// 	Id        int
// 	Frames    int
// 	Force     int
// }

// func init() {
// 	gob.Register(Burst{})
// }

// func (b Burst) ApplyFirst(g interface{}) {}
// func (b Burst) ApplyFinal(g interface{}) {}
// func (b Burst) Apply(_g interface{}) {
// 	g := _g.(*Game)
// 	player := g.GetEnt(b.Player_id).(*Player)
// 	if !player.Alive() || player.Exiled() {
// 		return
// 	}
// 	if _, ok := player.Processes[b.Id]; ok {
// 		// Already running this process
// 		return
// 	}
// 	params := map[string]int{"frames": b.Frames, "force": b.Force}
// 	process := abilities["burst"](g, player, params)
// 	player.Processes[b.Id] = process
// }

// type MoonFire struct {
// 	Player_id int
// 	Id        int
// 	Damage    int
// 	Radius    int
// }

// func init() {
// 	gob.Register(MoonFire{})
// }

// func (mf MoonFire) ApplyFirst(g interface{}) {}
// func (mf MoonFire) ApplyFinal(g interface{}) {}
// func (mf MoonFire) Apply(_g interface{}) {
// 	g := _g.(*Game)
// 	player := g.GetEnt(mf.Player_id).(*Player)
// 	if !player.Alive() || player.Exiled() {
// 		return
// 	}

// 	params := map[string]int{"damage": mf.Damage, "radius": mf.Radius}
// 	process := abilities["moonfire"](g, player, params)
// 	player.Processes[mf.Id] = process
// }

// type Nitro struct {
// 	Player_id int
// 	Id        int
// 	Inc       int
// }

// func init() {
// 	gob.Register(Nitro{})
// }

// func (n Nitro) ApplyFirst(g interface{}) {}
// func (n Nitro) ApplyFinal(g interface{}) {}
// func (n Nitro) Apply(_g interface{}) {
// 	g := _g.(*Game)
// 	player := g.GetEnt(n.Player_id).(*Player)
// 	if !player.Alive() || player.Exiled() {
// 		return
// 	}
// 	if proc, ok := player.Processes[n.Id]; ok {
// 		// Already running this process, so kill it
// 		proc.Kill(g)
// 		base.Log().Printf("Killed nitro")
// 		return
// 	}
// 	params := map[string]int{"inc": n.Inc}
// 	process := abilities["nitro"](g, player, params)
// 	player.Processes[n.Id] = process
// }

type GameWindow struct {
  Engine    *cgf.Engine
  game      *Game
  prev_game *Game
  region    gui.Region

  node_texture      gl.Uint
  node_texture_data []byte
}

func (gw *GameWindow) String() string {
  return "game window"
}
func (gw *GameWindow) Expandable() (bool, bool) {
  return false, false
}
func (gw *GameWindow) Requested() gui.Dims {
  if gw.game == nil {
    return gui.Dims{}
  }
  return gui.Dims{gw.game.Dx, gw.game.Dy}
}
func (gw *GameWindow) Rendered() gui.Region {
  return gw.region
}
func (gw *GameWindow) Think(g *gui.Gui, t int64) {
  if gw.game == nil {
    gw.game = gw.Engine.CopyState().(*Game)
    gw.prev_game = gw.game.Copy().(*Game)
  } else {
    gw.Engine.UpdateState(gw.game)
    gw.game.Merge(gw.prev_game)
    gw.prev_game.OverwriteWith(gw.game)
  }
}
func (gw *GameWindow) Respond(g *gui.Gui, group gui.EventGroup) bool {
  return false
}

var latest_region gui.Region

// Returns the most recent region used when rendering the game.
func (g *Game) Region() gui.Region {
  return latest_region
}

func (gw *GameWindow) Draw(region gui.Region) {
  gw.region = region
  latest_region = region
  gl.PushMatrix()
  defer gl.PopMatrix()
  gl.Translated(gl.Double(gw.region.X), gl.Double(gw.region.Y), 0)

  // Nodes
  if gw.node_texture_data == nil {
    gw.node_texture_data = make([]byte, len(gw.game.Nodes)*len(gw.game.Nodes[0])*4)
    gl.GenTextures(1, &gw.node_texture)
    gl.BindTexture(gl.TEXTURE_2D, gw.node_texture)
    gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
    gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
    gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
    gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
    gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
    gl.TexImage2D(
      gl.TEXTURE_2D,
      0,
      gl.RGBA,
      gl.Sizei(len(gw.game.Nodes)),
      gl.Sizei(len(gw.game.Nodes[0])),
      0,
      gl.RGBA,
      gl.UNSIGNED_BYTE,
      gl.Pointer(&gw.node_texture_data[0]))
  } else {
    for x := range gw.game.Nodes {
      for y, node := range gw.game.Nodes[x] {
        pos := 4 * (y*len(gw.game.Nodes) + x)
        for c := 0; c < 3; c++ {
          if len(node.Amount) > c {
            gw.node_texture_data[pos+c] = byte(
              node.Amount[c] * mana_brightness * 1.0 / mana_cap)
          }
          gw.node_texture_data[pos+3] = 255
        }
      }
    }
    gl.Enable(gl.TEXTURE_2D)
    gl.BindTexture(gl.TEXTURE_2D, gw.node_texture)
  }
  gl.TexSubImage2D(
    gl.TEXTURE_2D,
    0,
    0,
    0,
    gl.Sizei(len(gw.game.Nodes)),
    gl.Sizei(len(gw.game.Nodes[0])),
    gl.RGBA,
    gl.UNSIGNED_BYTE,
    gl.Pointer(&gw.node_texture_data[0]))

  gl.Enable(gl.BLEND)
  gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
  texture.Render(0, float64(gw.game.Dy), float64(gw.game.Dx), -float64(gw.game.Dy))

  gl.Disable(gl.TEXTURE_2D)
  gl.Begin(gl.LINES)
  gl.Color4d(1, 1, 1, 1)
  for _, poly := range gw.game.Room.Walls {
    for i := range poly {
      seg := poly.Seg(i)
      gl.Vertex2d(gl.Double(seg.P.X), gl.Double(seg.P.Y))
      gl.Vertex2d(gl.Double(seg.Q.X), gl.Double(seg.Q.Y))
    }
  }
  gl.End()

  gl.Begin(gl.TRIANGLE_FAN)
  gl.Color4d(1, 0, 0, 1)
  for _, poly := range gw.game.Room.Lava {
    for _, v := range poly {
      gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
    }
  }
  gl.End()

  gl.Color4d(1, 1, 1, 1)
  for _, ent := range gw.game.Ents {
    ent.Draw(gw.game)
  }
  gl.Disable(gl.TEXTURE_2D)

  if local.active_ability != nil {
    local.active_ability.Draw(local.local_player.Id(), gw.game)
  }
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
