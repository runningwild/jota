package main

import (
  "fmt"
  "bytes"
  "encoding/gob"
  "github.com/runningwild/glop/gui"
  gl "github.com/chsc/gogl/gl21"
  "runningwild/pnf"
)

type Axis int

const (
  X Axis = iota
  Y
)

type Direction int

const (
  Left Direction = iota
  Up
  Right
  Down
)

type Color int

const (
  Red Color = iota
  Blue
  Green
)

type Player struct {
  Alive     bool
  X, Y      int
  Direction Direction
  Speed     int
  Color     Color
}

func (p *Player) NextSegment() (seg Segment) {
  seg.Color = p.Color
  switch p.Direction {
  case Down:
    seg.Axis = X
    seg.Pos = p.X
    seg.Start = p.Y - 1
    seg.End = p.Y - p.Speed
  case Up:
    seg.Axis = X
    seg.Pos = p.X
    seg.Start = p.Y + 1
    seg.End = p.Y + p.Speed
  case Right:
    seg.Axis = Y
    seg.Pos = p.Y
    seg.Start = p.X + 1
    seg.End = p.X + p.Speed
  case Left:
    seg.Axis = Y
    seg.Pos = p.Y
    seg.Start = p.X - 1
    seg.End = p.X - p.Speed
  default:
    panic("wtf")
  }
  return
}

type Segment struct {
  // The Axis this Segment is parallel to
  Axis Axis

  // Pos is the coordinate on Axis that this segment lies on.
  Pos int

  // The first and last coordinate of the Segment along the other axis.
  Start, End int

  // Duh.
  Color Color
}

func (s *Segment) start() int {
  if s.Start < s.End {
    return s.Start
  }
  return s.End
}
func (s *Segment) end() int {
  if s.Start < s.End {
    return s.End
  }
  return s.Start
}

// Returns true iff the segments overlap, and if they do overlap also returns
// the portion of a that should remain after intersecting with b.
func SegmentsOverlap(a, b Segment) (bool, Segment) {
  if a.Axis == b.Axis {
    if a.Pos != b.Pos {
      return false, Segment{}
    }
    if b.start() >= a.start() && b.start() <= a.end() ||
      b.end() >= a.start() && b.end() <= a.end() {
      s := a
      if b.Start >= s.start() && b.Start <= s.end() {
        s.End = b.Start
      } else {
        s.End = b.End
      }
      return true, s
    }
    return false, Segment{}
  }
  if b.Pos >= a.start() && b.Pos <= a.end() &&
    a.Pos >= b.start() && a.Pos <= b.end() {
    s := a
    s.End = b.Pos
    return true, s
  }
  return false, Segment{}
}

type SegSlice []Segment

func (ss SegSlice) Len() int {
  return len(ss)
}
func (ss SegSlice) Less(i, j int) bool {
  if ss[i].Axis != ss[j].Axis {
    return ss[i].Axis < ss[j].Axis
  }
  if ss[i].Pos != ss[j].Pos {
    return ss[i].Pos < ss[j].Pos
  }
  if ss[i].Start != ss[j].Start {
    return ss[i].Start < ss[j].Start
  }
  return ss[i].End < ss[j].End
}
func (ss SegSlice) Swap(i, j int) {
  ss[i], ss[j] = ss[j], ss[i]
}

type Game struct {
  // Dimensions of the board
  Dx, Dy int

  Segments []Segment

  Players []Player
}

func (g *Game) Copy() interface{} {
  var g2 Game
  buf := bytes.NewBuffer(nil)
  enc := gob.NewEncoder(buf)
  err := enc.Encode(g)
  if err != nil {
    panic(err)
  }
  dec := gob.NewDecoder(buf)
  err = dec.Decode(&g2)
  if err != nil {
    panic(err)
  }
  return &g2
}
func (g *Game) ThinkFirst() {}
func (g *Game) ThinkFinal() {}
func (g *Game) Think() {
  // Advance players, check for collisions, add segments
  for i := range g.Players {
    if !g.Players[i].Alive {
      continue
    }
    seg := g.Players[i].NextSegment()
    // TODO: NEed to make sure we handle the passthrough problem, because it
    // can happen very easily
    for j := range g.Segments {
      if hit, rem := SegmentsOverlap(seg, g.Segments[j]); hit {
        // kill player
        g.Players[i].Alive = false
        seg = rem
      }
    }
    g.Segments = append(g.Segments, seg)
    fmt.Printf("append %d %d %d %d\n", seg.Axis, seg.Pos, seg.Start, seg.End)
    switch seg.Axis {
    case X:
      g.Players[i].Y = seg.End
    case Y:
      g.Players[i].X = seg.End
    }
  }
}

type TurnLeft struct {
  Player int
}

func (tl TurnLeft) ApplyFirst(g interface{}) {}
func (tl TurnLeft) ApplyFinal(g interface{}) {}
func (tl TurnLeft) Apply(_g interface{}) {
  g := _g.(*Game)
  g.Players[tl.Player].Direction = (g.Players[tl.Player].Direction + 3) % 4
}

type TurnRight struct {
  Player int
}

func (tr TurnRight) ApplyFirst(g interface{}) {}
func (tr TurnRight) ApplyFinal(g interface{}) {}
func (tr TurnRight) Apply(_g interface{}) {
  g := _g.(*Game)
  g.Players[tr.Player].Direction = (g.Players[tr.Player].Direction + 1) % 4
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
  gw.game = gw.Engine.GetState().(*Game)
}
func (gw *GameWindow) Respond(g *gui.Gui, group gui.EventGroup) bool {
  return false
}
func (gw *GameWindow) Draw(region gui.Region) {
  gw.region = region
  gl.PushMatrix()
  defer gl.PopMatrix()
  gl.Translated(float64(gw.region.X), float64(gw.region.Y), 0)
  gl.Color4ub(255, 0, 0, 255)
  gl.Begin(gl.QUADS)
  {
    dx := int32(gw.region.Dx)
    dy := int32(gw.region.Dy)
    gl.Vertex2i(0, 0)
    gl.Vertex2i(0, dy)
    gl.Vertex2i(dx, dy)
    gl.Vertex2i(dx, 0)
  }
  gl.End()

  gl.Begin(gl.LINES)
  {
    for _, seg := range gw.game.Segments {
      switch seg.Color {
      case Red:
        gl.Color4ub(255, 0, 0, 255)
      case Green:
        gl.Color4ub(0, 255, 0, 255)
      case Blue:
        gl.Color4ub(0, 0, 255, 255)
      default:
        gl.Color4ub(255, 0, 255, 255)
      }
      switch seg.Axis {
      case X:
        gl.Vertex2i(int32(seg.Pos), int32(seg.start()))
        gl.Vertex2i(int32(seg.Pos), int32(seg.end()+1))
      case Y:
        gl.Vertex2i(int32(seg.start()), int32(seg.Pos))
        gl.Vertex2i(int32(seg.end()+1), int32(seg.Pos))
      }
    }
  }
  gl.End()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
