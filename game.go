package main

import (
  "bytes"
  "encoding/gob"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/glop/gui"
  "math"
  "path/filepath"
  "runningwild/pnf"
  "runningwild/tron/base"
  "runningwild/tron/texture"
)

type playerTempData struct {
  delta struct {
    speed float64
    angle float64
  }
}

type Player struct {
  Alive bool
  X, Y  float64
  Angle float64
  Speed float64
  Color struct {
    R, G, B byte
  }
  temp playerTempData
}

type Game struct {
  // Dimensions of the board
  Dx, Dy int

  Friction float64
  Max_turn float64
  Max_acc  float64
  Players  []Player

  Game_thinks int
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
  g2 = *g
  for i := range g2.Players {
    g2.Players[i].temp = playerTempData{}
  }
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
    if p.temp.delta.speed > g.Max_acc {
      p.temp.delta.speed = g.Max_acc
    }
    if p.temp.delta.speed < -g.Max_acc {
      p.temp.delta.speed = -g.Max_acc
    }
    if p.temp.delta.angle < -g.Max_turn {
      p.temp.delta.angle = -g.Max_turn
    }
    if p.temp.delta.angle > g.Max_turn {
      p.temp.delta.angle = g.Max_turn
    }
    p.Angle += p.temp.delta.angle
    p.Speed += p.temp.delta.speed
    p.Speed *= g.Friction
    p.X += p.Speed * math.Cos(p.Angle)
    p.Y += p.Speed * math.Sin(p.Angle)
    p.temp = playerTempData{}
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
  g.Players[t.Player].temp.delta.angle += t.Delta
}

type Accelerate struct {
  Player int
  Delta  float64
}

func (a Accelerate) ApplyFirst(g interface{}) {}
func (a Accelerate) ApplyFinal(g interface{}) {}
func (a Accelerate) Apply(_g interface{}) {
  g := _g.(*Game)
  g.Players[a.Player].temp.delta.speed += a.Delta
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
  cur := gw.Engine.GetState().(*Game)
  if gw.game == nil {
    gw.game = cur.Copy().(*Game)
    return
  }
  if cur.Game_thinks == gw.game.Game_thinks {
    gw.game.Merge(cur)
    return
  }
  cur = cur.Copy().(*Game)
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
  t := texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
  base.Log().Printf("Drawing GameThink: %d", gw.game.Game_thinks)
  for _, p := range gw.game.Players {
    t.RenderAdvanced(p.X, p.Y, float64(t.Dx()), float64(t.Dy()), p.Angle, false)
  }
  // gl.Color4ub(255, 0, 0, 255)
  // gl.Begin(gl.QUADS)
  // {
  //   dx := int32(gw.region.Dx)
  //   dy := int32(gw.region.Dy)
  //   gl.Vertex2i(0, 0)
  //   gl.Vertex2i(0, dy)
  //   gl.Vertex2i(dx, dy)
  //   gl.Vertex2i(dx, 0)
  // }
  // gl.End()

  // gl.Begin(gl.LINES)
  // {
  //   for _, seg := range gw.game.Segments {
  //     switch seg.Color {
  //     case Red:
  //       gl.Color4ub(255, 0, 0, 255)
  //     case Green:
  //       gl.Color4ub(0, 255, 0, 255)
  //     case Blue:
  //       gl.Color4ub(0, 0, 255, 255)
  //     default:
  //       gl.Color4ub(255, 0, 255, 255)
  //     }
  //     switch seg.Axis {
  //     case X:
  //       gl.Vertex2i(int32(seg.Pos), int32(seg.start()))
  //       gl.Vertex2i(int32(seg.Pos), int32(seg.end()+1))
  //     case Y:
  //       gl.Vertex2i(int32(seg.start()), int32(seg.Pos))
  //       gl.Vertex2i(int32(seg.end()+1), int32(seg.Pos))
  //     }
  //   }
  // }
  // gl.End()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
