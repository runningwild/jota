package main

import (
  "os"
  "runtime"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/gos"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/render"
  "github.com/runningwild/glop/system"
  "path/filepath"
)

var (
  sys      system.System
  datadir  string
  ui       *gui.Gui
  wdx, wdy int
)

func init() {
  runtime.LockOSThread()
  sys = system.Make(gos.GetSystemInterface())

  datadir = filepath.Join(os.Args[0], "..", "..")
  wdx = 1024
  wdy = 768
}

func main() {
  sys.Startup()
  err := gl.Init()
  if err != nil {
    panic(err)
  }

  render.Init()
  render.Queue(func() {
    sys.CreateWindow(10, 10, wdx, wdy)
    sys.EnableVSync(true)
    err := gl.Init()
    if err != nil {
      panic(err)
    }
  })
  runtime.GOMAXPROCS(2)
  ui, err = gui.Make(gin.In(), gui.Dims{wdx, wdy}, filepath.Join(datadir, "fonts", "skia.ttf"))
  if err != nil {
    panic(err)
  }

  // ui.AddChild(editor)
  // ui.AddChild(base.MakeConsole())
  anchor := gui.MakeAnchorBox(gui.Dims{wdx, wdy})
  ui.AddChild(anchor)
  anchor.AddChild(gui.MakeTextLine("standard", "foo", 300, 1, 1, 1, 1), gui.Anchor{0.5, 0.5, 0.5, 0.5})
  sys.Think()
  var g Game
  g.Dx = 600
  g.Dy = 400
  g.Segments = append(g.Segments, Segment{
    Axis:  X,
    Pos:   0,
    Start: 0,
    End:   g.Dy - 1,
    Color: Blue,
  })
  g.Segments = append(g.Segments, Segment{
    Axis:  X,
    Pos:   g.Dx - 1,
    Start: 0,
    End:   g.Dy - 1,
    Color: Blue,
  })
  g.Segments = append(g.Segments, Segment{
    Axis:  Y,
    Pos:   0,
    Start: 0,
    End:   g.Dx - 1,
    Color: Blue,
  })
  g.Segments = append(g.Segments, Segment{
    Axis:  Y,
    Pos:   g.Dy - 1,
    Start: 0,
    End:   g.Dx - 1,
    Color: Blue,
  })
  g.Players = append(g.Players, Player{
    Alive:     true,
    X:         150,
    Y:         150,
    Direction: Down,
    Speed:     5,
    Color:     Blue,
  })
  anchor.AddChild(&GameWindow{Game: &g}, gui.Anchor{0.5, 0.5, 0.5, 0.5})
  var v float64
  for gin.In().GetKey('q').FramePressCount() == 0 {
    sys.Think()
    render.Queue(func() {
      sys.SwapBuffers()
    })
    render.Purge()
    if gin.In().GetKey(gin.Up).FramePressCount() > 0 && g.Players[0].Direction != Down {
      g.Players[0].Direction = Up
    }
    if gin.In().GetKey(gin.Down).FramePressCount() > 0 && g.Players[0].Direction != Up {
      g.Players[0].Direction = Down
    }
    if gin.In().GetKey(gin.Left).FramePressCount() > 0 && g.Players[0].Direction != Right {
      g.Players[0].Direction = Left
    }
    if gin.In().GetKey(gin.Right).FramePressCount() > 0 && g.Players[0].Direction != Left {
      g.Players[0].Direction = Right
    }
    render.Queue(func() {
      ui.Draw()
    })
    g.Think()
    v += 0.01
  }
}
