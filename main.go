package main

import (
  "bytes"
  "encoding/json"
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
  "runningwild/pnf"
  _ "runningwild/tron/ability"
  "runningwild/tron/base"
  "runningwild/tron/game"
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

  var ids []int
  var engine *pnf.Engine
  var room Room
  err = base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
  if err != nil {
    panic(err)
  }
  if IsHost() {
    sys.Think()
    var g game.Game
    g.Rng = cmwc.MakeGoodCmwc()
    g.Rng.SeedWithDevRand()
    g.Dx = 900
    g.Dy = 600
    g.Friction = 0.97
    g.Polys = room.Polys
    var p game.Player
    p.Color.R = 255
    err := json.NewDecoder(bytes.NewBuffer([]byte(`
      {
        "Base": {
          "Max_turn": 0.07,
          "Max_acc": 0.2,
          "Mass": 750,
          "Max_rate": 10,
          "Influence": 75,
          "Health": 100
        },
        "Dynamic": {
          "Health": 100
        }
      }
    `))).Decode(&p.Stats)
    if err != nil {
      panic(err)
    }
    N := 2
    p.X = float64(g.Dx-N) / 2
    p.Y = float64(g.Dy-N) / 2
    for x := 0; x < N; x++ {
      for y := 0; y < N; y++ {
        p.X += float64(x * 25)
        p.Y += float64(y * 25)
        // p.Mass += float64(x+y) * 150
        p.Processes = make(map[int]game.Process)
        temp := p
        ids = append(ids, g.AddEnt(&temp))

        // p.Mass -= float64(x+y) * 150
        p.X -= float64(x * 25)
        p.Y -= float64(y * 25)
      }
    }
    g.Ents[0], g.Ents[(N*N)/2+(1-N%2)*N/2] = g.Ents[(N*N)/2+(1-N%2)*N/2], g.Ents[0]
    g.GenerateNodes()
    engine, err = pnf.NewNetEngine(&g, 17, 120, 1194)
    if err != nil {
      panic(err.Error())
    }
  } else {
    engine, err = pnf.NewNetClientEngine(17, 120, 1194)
    if err != nil {
      panic(err.Error())
    }
  }

  // engine = pnf.NewLocalEngine(&g, 17)
  anchor := gui.MakeAnchorBox(gui.Dims{wdx, wdy})
  ui.AddChild(anchor)
  anchor.AddChild(&game.GameWindow{Engine: engine}, gui.Anchor{0.5, 0.5, 0.5, 0.5})
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

    if IsHost() {
      for i := 0; i <= 1; i++ {
        up := key_map[fmt.Sprintf("%dup", i)].FramePressAvg()
        down := key_map[fmt.Sprintf("%ddown", i)].FramePressAvg()
        left := key_map[fmt.Sprintf("%dleft", i)].FramePressAvg()
        right := key_map[fmt.Sprintf("%dright", i)].FramePressAvg()
        if up-down != 0 {
          engine.ApplyEvent(game.Accelerate{ids[i], 2 * (up - down)})
        }
        if left-right != 0 {
          engine.ApplyEvent(game.Turn{ids[i], (left - right) / 10})
        }

        if key_map[fmt.Sprintf("%d-1", i)].FramePressCount() > 0 {
          engine.ApplyEvent(game.Nitro{ids[i], 0, 20000})
        }
        if key_map[fmt.Sprintf("%d-2", i)].FramePressCount() > 0 {
          engine.ApplyEvent(game.MoonFire{ids[i], 1, 90, 150})
        }
        if key_map[fmt.Sprintf("%d-3", i)].FramePressCount() > 0 {
          engine.ApplyEvent(game.Burst{ids[i], 2, 3, 100000})
        }
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
