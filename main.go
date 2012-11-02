package main

import (
  "fmt"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/cmwc"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/gos"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/render"
  "github.com/runningwild/glop/system"
  // "math"
  "os"
  "path/filepath"
  "runningwild/linear"
  "runningwild/pnf"
  "runningwild/tron/base"
  "runtime"
  "runtime/pprof"
)

var (
  sys      system.System
  datadir  string
  ui       *gui.Gui
  wdx, wdy int
  key_map  base.KeyMap
)

func init() {
  runtime.LockOSThread()
  sys = system.Make(gos.GetSystemInterface())

  datadir = filepath.Join(os.Args[0], "..", "..")
  base.SetDatadir(datadir)
  base.Log().Printf("Setting datadir: %s", datadir)
  wdx = 1024
  wdy = 768

  var key_binds base.KeyBinds
  base.LoadJson(filepath.Join(datadir, "key_binds.json"), &key_binds)
  key_map = key_binds.MakeKeyMap()
  base.SetDefaultKeyMap(key_map)
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

  anchor := gui.MakeAnchorBox(gui.Dims{wdx, wdy})
  ui.AddChild(anchor)
  sys.Think()
  var g Game
  g.Rng = cmwc.MakeCmwc(4224759397, 3)
  g.Rng.SeedWithDevRand()
  g.Dx = 900
  g.Dy = 600
  g.Friction = 0.97
  g.Polys = []linear.Poly{
    linear.Poly{
      linear.Vec2{600, 300},
      linear.Vec2{600, 400},
      linear.Vec2{700, 400},
      linear.Vec2{700, 300},
    },
    linear.Poly{
      linear.Vec2{200, 300},
      linear.Vec2{100, 400},
      linear.Vec2{500, 350},
    },
    linear.Poly{
      linear.Vec2{0, 0},
      linear.Vec2{float64(g.Dx), 0},
      linear.Vec2{float64(g.Dx), float64(g.Dy)},
      linear.Vec2{0, float64(g.Dy)},
    },
  }
  var p Player
  p.Alive = true
  p.Max_turn = 0.07
  p.Max_acc = 0.1
  p.Mass = 750 // who knows
  p.Color.R = 255
  p.Max_rate = 10
  p.Influence = 75
  p.Dominance = 10
  N := 2
  p.X = float64(g.Dx-N) / 2
  p.Y = float64(g.Dy-N) / 2
  for x := 0; x < N; x++ {
    for y := 0; y < N; y++ {
      p.X += float64(x * 25)
      p.Y += float64(y * 25)
      // p.Mass += float64(x+y) * 150
      p.Processes = make(map[int]Process)
      g.Players = append(g.Players, p)
      // p.Mass -= float64(x+y) * 150
      p.X -= float64(x * 25)
      p.Y -= float64(y * 25)
    }
  }
  g.Players[0], g.Players[(N*N)/2+(1-N%2)*N/2] = g.Players[(N*N)/2+(1-N%2)*N/2], g.Players[0]
  // g.Players[1].Mass = math.Inf(1)
  g.GenerateNodes()
  var engine *pnf.Engine
  engine = pnf.NewLocalEngine(&g, 17)
  anchor.AddChild(&GameWindow{Engine: engine}, gui.Anchor{0.5, 0.5, 0.5, 0.5})
  var v float64
  var profile_output *os.File
  var num_mem_profiles int
  ui.AddChild(base.MakeConsole())
  for key_map["quit"].FramePressCount() == 0 {
    sys.Think()
    render.Queue(func() {
      ui.Draw()
    })
    render.Queue(func() {
      sys.SwapBuffers()
    })
    render.Purge()

    for i := 0; i <= 1; i++ {
      up := key_map[fmt.Sprintf("%dup", i)].FramePressAvg()
      down := key_map[fmt.Sprintf("%ddown", i)].FramePressAvg()
      left := key_map[fmt.Sprintf("%dleft", i)].FramePressAvg()
      right := key_map[fmt.Sprintf("%dright", i)].FramePressAvg()
      engine.ApplyEvent(Accelerate{i, 2 * (up - down)})
      engine.ApplyEvent(Turn{i, (left - right) / 10})

      if key_map[fmt.Sprintf("%d-1", i)].FramePressCount() > 0 {
        engine.ApplyEvent(Nitro{i, 0, 100000})
      }
      // if key_map[fmt.Sprintf("%d-1", i)].FramePressCount() > 0 {
      //   engine.ApplyEvent(Blink{i, 0, 50})
      // }
      if key_map[fmt.Sprintf("%d-2", i)].FramePressCount() > 0 {
        engine.ApplyEvent(Burst{i, 1, 100, 10000})
      }
      if key_map[fmt.Sprintf("%d-3", i)].FramePressCount() > 0 {
        engine.ApplyEvent(Burst{i, 2, 3, 100000})
      }
    }

    if key_map["cpu profile"].FramePressCount() > 0 {
      if profile_output == nil {
        profile_output, err = os.Create(filepath.Join(datadir, "cpu.prof"))
        if err == nil {
          err = pprof.StartCPUProfile(profile_output)
          if err != nil {
            fmt.Printf("Unable to start CPU profile: %v\n", err)
            profile_output.Close()
            profile_output = nil
          }
          fmt.Printf("profout: %v\n", profile_output)
        } else {
          fmt.Printf("Unable to start CPU profile: %v\n", err)
        }
      } else {
        pprof.StopCPUProfile()
        profile_output.Close()
        profile_output = nil
      }
    }

    if key_map["mem profile"].FramePressCount() > 0 {
      f, err := os.Create(filepath.Join(datadir, fmt.Sprintf("mem.%d.prof", num_mem_profiles)))
      if err != nil {
        base.Error().Printf("Unable to write mem profile: %v", err)
      }
      pprof.WriteHeapProfile(f)
      f.Close()
      num_mem_profiles++
    }

    v += 0.01
  }
}
