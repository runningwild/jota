package main

import (
  "bytes"
  "encoding/gob"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/cmwc"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/util/algorithm"
  "math"
  "path/filepath"
  "runningwild/pnf"
  "runningwild/tron/base"
  "runningwild/tron/texture"
)

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
  Color    Color
  X, Y     float64
  Capacity float64
  Amt      float64
  Regen    float64
}

func init() {
  gob.Register(&Node{})
}

func (n *Node) Think() {
  n.Amt += n.Regen
  if n.Amt > n.Capacity {
    n.Amt = n.Capacity
  }
}

type Player struct {
  Alive  bool
  Mass   float64
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

  // Max_rate and Influence determine the rate that a player can drain mana
  // from a node a distance D away:
  // Rate(D) = max(0, Max_rate - (D / Influence) ^ 2)
  Max_rate  int32
  Influence int32

  // If two players try to drain mana from the same node only the player with
  // the highest priority will be able to, where the priority is
  // Priority(D) = Max_rate(D) * Dominance
  Dominance int32

  // If Exile_frames > 0 then the Player is not present in the game right now
  // and is excluded from all combat/mana/rendering/processing/etc...
  // Exile_frames is the number of frames remaining that the player is in
  // exile.
  Exile_frames int32

  // Processes contains all of the processes that this player is casting
  // right now.
  Processes []Process
}

func (p *Player) Rate(distance float64) float64 {
  if distance < 1 {
    distance = 1
  }
  return 150 / distance
}

func (p *Player) Priority(distance float64) float64 {
  return float64(p.Dominance) * p.Rate(distance)
}

func (p *Player) Exiled() bool {
  return p.Exile_frames > 0
}

func (p *Player) Think(g *Game) {
  if p.Exile_frames > 0 {
    p.Exile_frames--
    return
  }
  if p.Delta.Speed > g.Max_acc {
    p.Delta.Speed = g.Max_acc
  }
  if p.Delta.Speed < -g.Max_acc {
    p.Delta.Speed = -g.Max_acc
  }
  if p.Delta.Angle < -g.Max_turn {
    p.Delta.Angle = -g.Max_turn
  }
  if p.Delta.Angle > g.Max_turn {
    p.Delta.Angle = g.Max_turn
  }
  p.Vx *= g.Friction
  p.Vy *= g.Friction
  p.Angle += p.Delta.Angle
  p.Vx += p.Delta.Speed * math.Cos(p.Angle)
  p.Vy += p.Delta.Speed * math.Sin(p.Angle)
  p.X += p.Vx
  p.Y += p.Vy
  for _, process := range p.Processes {
    process.Think(p, g)
  }
  algorithm.Choose(&p.Processes, func(p Process) bool { return !p.Complete() })
}

func (p *Player) Request() Mana {
  var request Mana
  for _, process := range p.Processes {
    for color, value := range process.Request() {
      request[color] = request[color] + value
    }
  }
  return request
}
func (p *Player) Supply(supply Mana) Mana {
  for _, process := range p.Processes {
    supply = process.Supply(supply)
  }
  return supply
}

type Game struct {
  // All of the nodes on the map
  Nodes [][]Node

  Rng *cmwc.Cmwc

  // Dimensions of the board
  Dx, Dy int

  Friction float64
  Max_turn float64
  Max_acc  float64
  Players  []Player

  Game_thinks int
}

func (g *Game) GenerateNodes() {
  c := cmwc.MakeCmwc(4224759397, 3)
  c.SeedWithDevRand()
  g.Nodes = make([][]Node, 1+g.Dx/node_spacing)
  for x := 0; x < 1+g.Dx/node_spacing; x++ {
    g.Nodes[x] = make([]Node, 1+g.Dy/node_spacing)
    for y := 0; y < 1+g.Dy/node_spacing; y++ {
      g.Nodes[x][y] = Node{
        X:        float64(x * node_spacing),
        Y:        float64(y * node_spacing),
        Color:    Color(c.Int63() % 3),
        Capacity: 100,
        Amt:      100,
        Regen:    0.2,
      }
    }
  }
}

func (g *Game) Merge(g2 *Game) {
  frac := 0.5
  for i := range g.Players {
    g.Players[i].X = frac*g2.Players[i].X + (1-frac)*g.Players[i].X
    g.Players[i].Y = frac*g2.Players[i].Y + (1-frac)*g.Players[i].Y
    g.Players[i].Angle = frac*g2.Players[i].Angle + (1-frac)*g.Players[i].Angle
  }
}

func (g *Game) Copy() interface{} {
  var g2 Game
  buf := bytes.NewBuffer(nil)
  enc := gob.NewEncoder(buf)
  err := enc.Encode(g)
  if err != nil {
    panic(err)
  }
  err = gob.NewDecoder(buf).Decode(&g2)
  if err != nil {
    panic(err)
  }
  return &g2
}

func (g *Game) OverwriteWith(_g2 interface{}) {
  g2 := (_g2.(*Game))
  g.Rng.OverwriteWith(g2.Rng)
  g.Dx = g2.Dx
  g.Dy = g2.Dy
  g.Friction = g2.Friction
  g.Max_turn = g2.Max_turn
  g.Max_acc = g2.Max_acc

  g.Players = g.Players[0:0]
  for _, p := range g2.Players {
    g.Players = append(g.Players, p)
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

  g.Game_thinks = g2.Game_thinks
}

// Returns a mapping from player index to the list of *Nodes that that player
// has priority on.
func (g *Game) getPriorities() [][]*Node {
  r := make([][]*Node, len(g.Players))
  for x := range g.Nodes {
    for y := range g.Nodes[x] {
      var best int
      var best_dist_sq float64 = 1e9
      for j := range g.Players {
        dx := (g.Players[j].X - g.Nodes[x][y].X)
        dy := (g.Players[j].Y - g.Nodes[x][y].Y)
        dist_sq := dx*dx + dy*dy
        if dist_sq < best_dist_sq {
          best_dist_sq = dist_sq
          best = j
        }
      }
      r[best] = append(r[best], &g.Nodes[x][y])
    }
  }
  return r
}

func (g *Game) ThinkFirst() {}
func (g *Game) ThinkFinal() {}
func (g *Game) Think() {
  g.Nodes[0][0].Amt -= 1
  g.Game_thinks++
  // Advance players, check for collisions, add segments
  for i := range g.Players {
    if !g.Players[i].Alive {
      continue
    }
    g.Players[i].Think(g)
    g.Players[i].X = clamp(g.Players[i].X, 0, float64(g.Dx))
    g.Players[i].Y = clamp(g.Players[i].Y, 0, float64(g.Dy))
  }
  moved := make(map[int]bool)
  for i := range g.Players {
    for j := range g.Players {
      if i == j {
        continue
      }
      dx := g.Players[i].X - g.Players[j].X
      dy := g.Players[i].Y - g.Players[j].Y
      dist := math.Sqrt(dx*dx + dy*dy)
      if dist > 25 {
        continue
      }
      if dist <= 0.5 {
        dist = 0.5
      }
      force := 20.0 * (25 - dist)
      force /= g.Players[i].Mass
      angle := math.Atan2(dy, dx)
      g.Players[i].Vx += force * math.Cos(angle)
      g.Players[i].Vy += force * math.Sin(angle)
      moved[i] = true
    }
  }

  priorities := g.getPriorities()
  for p := range g.Players {
    player := &g.Players[p]
    nodes := priorities[p]

    for i := range nodes {
      swap := int(g.Rng.Uint32()%uint32(len(nodes)-i)) + i
      nodes[i], nodes[swap] = nodes[swap], nodes[i]
    }

    var supply Mana
    for _, node := range nodes {
      dx := (player.X - node.X)
      dy := (player.Y - node.Y)
      drain := player.Rate(math.Sqrt(dx*dx + dy*dy))
      if drain > node.Amt {
        drain = node.Amt
      }

      supply[node.Color] += drain
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
      dx := (player.X - node.X)
      dy := (player.Y - node.Y)
      drain := player.Rate(math.Sqrt(dx*dx + dy*dy))
      if drain > used[node.Color] {
        drain = used[node.Color]
      }
      if drain > node.Amt {
        drain = node.Amt
      }
      node.Amt -= drain
      used[node.Color] -= drain
      supply[node.Color] += drain
    }
  }

  for x := range g.Nodes {
    for y := range g.Nodes[x] {
      g.Nodes[x][y].Think()
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

func (g *Game) allNodesInSquare(x, y, radius float64, indexes *[]nodeIndex) {
  x /= node_spacing
  y /= node_spacing
  radius /= node_spacing
  minx := clamp(x-radius, 0, float64(len(g.Nodes)))
  maxx := clamp(x+radius+1, 0, float64(len(g.Nodes)))
  miny := clamp(y-radius, 0, float64(len(g.Nodes[0])))
  maxy := clamp(y+radius+1, 0, float64(len(g.Nodes[0])))
  *indexes = (*indexes)[0:0]
  for x := int(minx); x < int(maxx); x++ {
    for y := int(miny); y < int(maxy); y++ {
      *indexes = append(*indexes, nodeIndex{x, y})
    }
  }
}

type Turn struct {
  Player int
  Delta  float64
}

func (t Turn) ApplyFirst(g interface{}) {}
func (t Turn) ApplyFinal(g interface{}) {}
func (t Turn) Apply(_g interface{}) {
  g := _g.(*Game)
  g.Players[t.Player].Delta.Angle = t.Delta
}

type Accelerate struct {
  Player int
  Delta  float64
}

func (a Accelerate) ApplyFirst(g interface{}) {}
func (a Accelerate) ApplyFinal(g interface{}) {}
func (a Accelerate) Apply(_g interface{}) {
  g := _g.(*Game)
  g.Players[a.Player].Delta.Speed = a.Delta
}

type Blink struct {
  Player int
  Frames int
}

func (b Blink) ApplyFirst(g interface{}) {}
func (b Blink) ApplyFinal(g interface{}) {}
func (b Blink) Apply(_g interface{}) {
  g := _g.(*Game)
  player := &g.Players[b.Player]
  if !player.Alive || player.Exiled() {
    return
  }
  params := map[string]int{"frames": b.Frames}
  process := (&blinkAbility{}).Activate(player, params)
  player.Processes = append(player.Processes, process)
}

type Burst struct {
  Player int
  Frames int
  Force  int
}

func (b Burst) ApplyFirst(g interface{}) {}
func (b Burst) ApplyFinal(g interface{}) {}
func (b Burst) Apply(_g interface{}) {
  g := _g.(*Game)
  player := &g.Players[b.Player]
  if !player.Alive || player.Exiled() {
    return
  }
  params := map[string]int{"frames": b.Frames, "force": b.Force}
  process := (&burstAbility{}).Activate(player, params)
  player.Processes = append(player.Processes, process)
}

type GameWindow struct {
  Engine    *pnf.Engine
  game      *Game
  prev_game *Game
  region    gui.Region
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
    gw.game = gw.Engine.GetState().Copy().(*Game)
    gw.prev_game = gw.Engine.GetState().Copy().(*Game)
  } else {
    gw.game.OverwriteWith(gw.Engine.GetState().(*Game))
    gw.game.Merge(gw.prev_game)
    gw.prev_game.OverwriteWith(gw.game)
  }
}
func (gw *GameWindow) Respond(g *gui.Gui, group gui.EventGroup) bool {
  return false
}
func (gw *GameWindow) Draw(region gui.Region) {
  gw.region = region
  gl.PushMatrix()
  defer gl.PopMatrix()
  gl.Translated(gl.Double(gw.region.X), gl.Double(gw.region.Y), 0)
  gl.Color4d(1, 1, 1, 1)
  for i, p := range gw.game.Players {
    if p.Exiled() {
      continue
    }
    var t *texture.Data
    if i == 0 {
      t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
    } else {
      t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship2.png"))
    }
    t.RenderAdvanced(p.X-float64(t.Dx())/2, p.Y-float64(t.Dy())/2, float64(t.Dx()), float64(t.Dy()), p.Angle, false)
  }
  gl.Disable(gl.TEXTURE_2D)
  gl.Enable(gl.BLEND)
  gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
  gl.Begin(gl.POINTS)
  for x := range gw.game.Nodes {
    for _, node := range gw.game.Nodes[x] {
      alpha := gl.Double(node.Amt / node.Capacity)
      switch node.Color {
      case ColorRed:
        gl.Color4d(1, 0.1, 0.1, alpha)
      case ColorGreen:
        gl.Color4d(0, 1, 0, alpha)
      case ColorBlue:
        gl.Color4d(0.5, 0.5, 1, alpha)
      }
      gl.Vertex2d(gl.Double(node.X), gl.Double(node.Y))
    }
  }
  gl.End()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
