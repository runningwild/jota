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

var all_colors = [...]Color{ColorRed, ColorGreen, ColorBlue}

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
  Alive bool
  X, Y  float64
  Angle float64
  Speed float64
  Delta struct {
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
  term := distance / float64(p.Influence)
  rate := float64(p.Max_rate) - term*term
  if rate < 0 {
    return 0
  }
  return rate
}

func (p *Player) Priority(distance float64) float64 {
  return float64(p.Dominance) * p.Rate(distance)
}

// Max distance at which this player can drain mana from a node.  At this
// distance their rate should be exactly 0.
func (p *Player) MaxDist() float64 {
  return float64(p.Influence) * math.Sqrt(float64(p.Max_rate))
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
  p.Angle += p.Delta.Angle
  p.Speed += p.Delta.Speed
  p.Speed *= g.Friction
  p.X += p.Speed * math.Cos(p.Angle)
  p.Y += p.Speed * math.Sin(p.Angle)
  for _, process := range p.Processes {
    process.Think(p, g)
  }
  algorithm.Choose(&p.Processes, func(p Process) bool { return !p.Complete() })
}

func (p *Player) Request() map[Color]float64 {
  request := make(map[Color]float64, 3)
  for _, process := range p.Processes {
    for color, value := range process.Request() {
      request[color] = request[color] + value
    }
  }
  return request
}
func (p *Player) Supply(supply map[Color]float64) map[Color]float64 {
  for _, process := range p.Processes {
    supply = process.Supply(supply)
  }
  return supply
}

type noGob struct {
  // All of the nodes on the map
  Nodes [][]*Node
}

type Game struct {
  noGob

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
  g.Nodes = make([][]*Node, 1+g.Dx/node_spacing)
  for x := 0; x < 1+g.Dx/node_spacing; x++ {
    g.Nodes[x] = make([]*Node, 1+g.Dy/node_spacing)
    for y := 0; y < 1+g.Dy/node_spacing; y++ {
      g.Nodes[x][y] = &Node{
        X:        float64(x * node_spacing),
        Y:        float64(y * node_spacing),
        Color:    Color(c.Int63() % 3),
        Capacity: 250,
        Amt:      250,
        Regen:    0.3,
      }
    }
  }
}

func (g *Game) Merge(g2 *Game) {
  frac := 0.75
  for i := range g.Players {
    g.Players[i].X = frac*g2.Players[i].X + (1-frac)*g.Players[i].X
    g.Players[i].Y = frac*g2.Players[i].Y + (1-frac)*g.Players[i].Y
    g.Players[i].Angle = frac*g2.Players[i].Angle + (1-frac)*g.Players[i].Angle
  }
}

func (g *Game) Copy() interface{} {
  var g2 Game
  // g2 = *g
  // return &g2
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
  g2.noGob = g.noGob
  // var ps [][2]float64
  // g2.Nodes.PointsInCircle([2]float64{0, 0}, 30000, &ps, &g2.Nodes)
  return &g2
}

func (g *Game) ThinkFirst() {}
func (g *Game) ThinkFinal() {}
func (g *Game) Think() {
  g.Game_thinks++
  // Advance players, check for collisions, add segments
  for i := range g.Players {
    if !g.Players[i].Alive {
      continue
    }
    g.Players[i].Think(g)
  }

  var nodes []*Node
  g.allNodesInCircle(
    g.Players[0].X,
    g.Players[0].Y,
    float64(g.Players[0].MaxDist()),
    &nodes)
  base.Log().Printf("Hits: %d", len(nodes))
  // Shuffle the nodes
  for i := range nodes {
    swap := int(g.Rng.Int63()%int64(len(nodes)-i)) + i
    nodes[i], nodes[swap] = nodes[swap], nodes[i]
  }

  supply := make(map[Color]float64)
  for _, node := range nodes {
    dx := (g.Players[0].X - node.X)
    dy := (g.Players[0].Y - node.Y)
    drain := g.Players[0].Rate(math.Sqrt(dx*dx + dy*dy))
    if drain > node.Amt {
      drain = node.Amt
    }
    supply[node.Color] += drain
  }
  used := make(map[Color]float64)
  for color, amt := range supply {
    used[color] = amt
  }
  supply = g.Players[0].Supply(supply)
  for color, amt := range supply {
    used[color] -= amt
  }
  for _, node := range nodes {
    dx := (g.Players[0].X - node.X)
    dy := (g.Players[0].Y - node.Y)
    drain := g.Players[0].Rate(math.Sqrt(dx*dx + dy*dy))
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

  for x := range g.Nodes {
    for _, node := range g.Nodes[x] {
      node.Think()
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

func (g *Game) allNodesInCircle(x, y, radius float64, nodes *[]*Node) {
  x /= node_spacing
  y /= node_spacing
  radius /= node_spacing
  minx := clamp(x-radius, 0, float64(len(g.Nodes)))
  maxx := clamp(x+radius+1, 0, float64(len(g.Nodes)))
  miny := clamp(y-radius, 0, float64(len(g.Nodes[0])))
  maxy := clamp(y+radius+1, 0, float64(len(g.Nodes[0])))
  *nodes = (*nodes)[0:0]
  for x := int(minx); x < int(maxx); x++ {
    for y := int(miny); y < int(maxy); y++ {
      *nodes = append(*nodes, g.Nodes[x][y])
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

type GameWindow struct {
  Engine *pnf.Engine
  game   *Game
  region gui.Region
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
  cur := gw.Engine.GetState().Copy().(*Game)
  if gw.game == nil {
    gw.game = cur.Copy().(*Game)
  }
  cur.Merge(gw.game)
  gw.game = cur
}
func (gw *GameWindow) Respond(g *gui.Gui, group gui.EventGroup) bool {
  return false
}
func (gw *GameWindow) Draw(region gui.Region) {
  gw.region = region
  gl.PushMatrix()
  defer gl.PopMatrix()
  gl.Translated(float64(gw.region.X), float64(gw.region.Y), 0)
  gl.Color4d(1, 1, 1, 1)
  for _, p := range gw.game.Players {
    if p.Exiled() {
      continue
    }
    var t *texture.Data
    t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
    t.RenderAdvanced(p.X-float64(t.Dx())/2, p.Y-float64(t.Dy())/2, float64(t.Dx()), float64(t.Dy()), p.Angle, false)
  }
  gl.Disable(gl.TEXTURE_2D)
  gl.Enable(gl.BLEND)
  gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
  gl.Begin(gl.POINTS)
  for x := range gw.game.Nodes {
    for _, node := range gw.game.Nodes[x] {
      alpha := node.Amt / node.Capacity
      switch node.Color {
      case ColorRed:
        gl.Color4d(1, 0.1, 0.1, alpha)
      case ColorGreen:
        gl.Color4d(0, 1, 0, alpha)
      case ColorBlue:
        gl.Color4d(0.5, 0.5, 1, alpha)
      }
      gl.Vertex2d(node.X, node.Y)
    }
  }
  gl.End()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
