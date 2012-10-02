package main

import (
  "bytes"
  "encoding/gob"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/cmwc"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/kdtree"
  "math"
  "path/filepath"
  "runningwild/pnf"
  "runningwild/tron/base"
  "runningwild/tron/texture"
  "sort"
)

type Color int
type Node struct {
  Color    Color
  X, Y     float64
  Capacity float64
  Amt      float64
  Regen    float64
}

func init() {
  gob.Register(Node{})
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
}

type Game struct {
  // Dimensions of the board
  Dx, Dy int

  Friction float64
  Max_turn float64
  Max_acc  float64
  Players  []Player

  Nodes    *kd.Tree2
  NodeList []*Node

  Game_thinks int
}

func (g *Game) GenerateNodes(n int) {
  c := cmwc.MakeCmwc(4224759397, 3)
  c.SeedWithDevRand()
  for i := 0; i < n; i++ {
    g.NodeList = append(g.NodeList, &Node{
      X:        float64(c.Int63() % int64(g.Dx)),
      Y:        float64(c.Int63() % int64(g.Dy)),
      Color:    Color(c.Int63() % 3),
      Capacity: 1000,
      Amt:      10,
      Regen:    1,
    })
  }

  g.Nodes = kd.MakeTree2()
  for _, n := range g.NodeList {
    g.Nodes.Add([2]float64{n.X, n.Y}, n)
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
  g2 = *g
  return &g2
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

type nodeDistArray struct {
  center [2]float64
  ps     [][2]float64
  nodes  []*Node
}

func (nda *nodeDistArray) Len() int { return len(nda.ps) }
func (nda *nodeDistArray) Less(i, j int) bool {
  dx0 := nda.center[0] - nda.ps[i][0]
  dy0 := nda.center[1] - nda.ps[i][1]
  dx1 := nda.center[0] - nda.ps[j][0]
  dy1 := nda.center[1] - nda.ps[j][1]
  return (dx0*dx0 + dy0*dy0) < (dx1*dx1 + dy1*dy1)
}
func (nda *nodeDistArray) Swap(i, j int) {
  nda.ps[i], nda.ps[j] = nda.ps[j], nda.ps[i]
  nda.nodes[i], nda.nodes[j] = nda.nodes[j], nda.nodes[i]
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
    p := &g.Players[i]
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
  }

  var ps [][2]float64
  var nodes []*Node
  center := [2]float64{g.Players[0].X, g.Players[0].Y}
  g.Nodes.PointsInCircle(center, 300, &ps, &nodes)

  sort.Sort(&nodeDistArray{center, ps, nodes})
  suck := 100.0
  for _, nd := range nodes {
    if suck <= 0 {
      break
    }
    if suck > nd.Amt {
      suck -= nd.Amt
      nd.Amt = 0
    } else {
      nd.Amt -= suck
      suck = 0
    }
  }

  for i := range g.NodeList {
    g.NodeList[i].Think()
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
    var t *texture.Data
    if p.X < 300 {
      t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship2.png"))
    } else {
      t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
    }
    t.RenderAdvanced(p.X-float64(t.Dx())/2, p.Y-float64(t.Dy())/2, float64(t.Dx()), float64(t.Dy()), p.Angle, false)
  }
  gl.Disable(gl.TEXTURE_2D)
  gl.Enable(gl.BLEND)
  gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
  gl.Begin(gl.POINTS)
  for _, node := range gw.game.NodeList {
    alpha := node.Amt / node.Capacity
    switch node.Color {
    case 0:
      gl.Color4d(1, 0, 0, alpha)
    case 1:
      gl.Color4d(0, 1, 0, alpha)
    case 2:
      gl.Color4d(0, 0, 1, alpha)
    }
    gl.Vertex2d(node.X, node.Y)
  }
  gl.End()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
