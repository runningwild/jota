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
  "github.com/runningwild/cgf"
  _ "github.com/runningwild/magnus/ability"
  "github.com/runningwild/magnus/base"
  "github.com/runningwild/magnus/game"
  "os"
  "path/filepath"
  "runtime"
  // "runtime/pprof"
)

var (
  sys      system.System
  datadir  string
  ui       *gui.Gui
  wdx, wdy int
  key_map  base.KeyMap
)

func init() {
  {
    f, err := os.Create("/Users/jwills/code/src/github.com/runningwild/magnus/log.err")
    if err != nil {
      panic("shoot")
    }
    os.Stderr = f
    f, err = os.Create("/Users/jwills/code/src/github.com/runningwild/magnus/log.out")
    if err != nil {
      panic("shoot")
    }
    os.Stdout = f
  }
  runtime.LockOSThread()
  sys = system.Make(gos.GetSystemInterface())

  datadir = filepath.Join(os.Args[0], "..", "..")
  base.SetDatadir(datadir)
  base.Log().Printf("Setting datadir: %s", datadir)
  wdx = 1024
  wdy = 768

  var key_binds base.KeyBinds
  base.LoadJson(filepath.Join(datadir, "key_binds.json"), &key_binds)
  fmt.Printf("Prething: %v\n", key_binds)
  key_map = key_binds.MakeKeyMap()
  base.SetDefaultKeyMap(key_map)
}

func main() {
  fmt.Printf("%v\n", key_map)
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
  base.InitShaders()
  runtime.GOMAXPROCS(2)
  ui, err = gui.Make(gin.In(), gui.Dims{wdx, wdy}, filepath.Join(datadir, "fonts", "skia.ttf"))
  if err != nil {
    panic(err)
  }

  var ids []int
  var engine *cgf.Engine
  var room game.Room
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
    g.Friction_lava = 0.85
    g.Room = room
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
          "Health": 1000
        },
        "Dynamic": {
          "Health": 1000
        }
      }
    `))).Decode(&p.Stats)
    if err != nil {
      panic(err)
    }
    Nx := 2
    Ny := 1
    p.X = float64(g.Dx-Nx)/2 - 200
    p.Y = float64(g.Dy-Ny)/2 - 200
    for x := 0; x < Nx; x++ {
      for y := 0; y < Ny; y++ {
        p.X += float64(x * 25)
        p.Y += float64(y * 25)
        p.Gid++
        // p.Mass += float64(x+y) * 150
        p.Processes = make(map[int]game.Process)
        temp := p
        ids = append(ids, g.AddEnt(&temp))

        // p.Mass -= float64(x+y) * 150
        p.X -= float64(x * 25)
        p.Y -= float64(y * 25)
      }
    }
    g.Ents[0].(*game.Player).X = 500
    g.Ents[0].(*game.Player).Y = 300
    g.Ents[1].(*game.Player).X = 550
    g.Ents[1].(*game.Player).Y = 300
    g.SetLocalPlayer(g.Ents[0].(*game.Player))
    // g.Ents[0], g.Ents[(N*N)/2+(1-N%2)*N/2] = g.Ents[(N*N)/2+(1-N%2)*N/2], g.Ents[0]
    g.GenerateNodes()
    // engine, err = cgf.NewLocalEngine(&g, 17, base.Log())
    engine, err = cgf.NewHostEngine(&g, 17, "", 1231, base.Log())
    if err != nil {
      panic(err.Error())
    }
    g.SetEngine(engine)
  } else {
    engine, err = cgf.NewClientEngine(17, "", 1231, base.Log())
    if err != nil {
      panic(err.Error())
    }
    engine.CopyState().(*game.Game).SetEngine(engine)
  }

  anchor := gui.MakeAnchorBox(gui.Dims{wdx, wdy})
  ui.AddChild(anchor)
  anchor.AddChild(&game.GameWindow{Engine: engine}, gui.Anchor{0.5, 0.5, 0.5, 0.5})
  var v float64
  // var profile_output *os.File
  // var num_mem_profiles int
  // ui.AddChild(base.MakeConsole())

  for gin.In().GetKey(gin.AnyEscape).FramePressCount() == 0 {
    sys.Think()
    render.Queue(func() {
      ui.Draw()
    })
    render.Queue(func() {
      sys.SwapBuffers()
    })
    render.Purge()
    game.LocalThink()

    if IsHost() {
      for i := 0; i <= 0; i++ {
        down_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+1, gin.DeviceTypeController, gin.DeviceIndexAny)
        up_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+1, gin.DeviceTypeController, gin.DeviceIndexAny)
        right_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+2, gin.DeviceTypeController, gin.DeviceIndexAny)
        left_axis := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+2, gin.DeviceTypeController, gin.DeviceIndexAny)
        up := key_map[fmt.Sprintf("%dup", i)].FramePressAvg()
        down := key_map[fmt.Sprintf("%ddown", i)].FramePressAvg()
        left := key_map[fmt.Sprintf("%dleft", i)].FramePressAvg()
        right := key_map[fmt.Sprintf("%dright", i)].FramePressAvg()
        up = up_axis.FramePressAmt()
        down = down_axis.FramePressAmt()
        left = left_axis.FramePressAmt()
        right = right_axis.FramePressAmt()
        fmt.Printf("Up: %v\n", up)
        if up < 0.1 {
          up = 0
        }
        if down < 0.1 {
          down = 0
        }
        if left < 0.1 {
          left = 0
        }
        if right < 0.1 {
          right = 0
        }
        if up-down != 0 {
          engine.ApplyEvent(game.Accelerate{ids[i], 2 * (up - down)})
        }
        if left-right != 0 {
          engine.ApplyEvent(game.Turn{ids[i], (left - right)})
        }

        // if key_map[fmt.Sprintf("%d-1", i)].FramePressCount() > 0 {
        // 	engine.ApplyEvent(game.Pull{ids[i], 0, 20000})
        // }
        // if key_map[fmt.Sprintf("%d-2", i)].FramePressCount() > 0 {
        // 	engine.ApplyEvent(game.MoonFire{ids[i], 1, 50, 50})
        // }
        // if gin.In().GetKeyFlat(gin.ControllerButton0, gin.DeviceTypeController, gin.DeviceTypeAny).FramePressCount() > 0 {
        // if key_map[fmt.Sprintf("%d-3", i)].FramePressCount() > 0 {
        // engine.ApplyEvent(game.Burst{ids[i], 2, 3, 100000})
        // }
      }
    }

    // if key_map["cpu profile"].FramePressCount() > 0 {
    // 	if profile_output == nil {
    // 		profile_output, err = os.Create(filepath.Join(datadir, "cpu.prof"))
    // 		if err == nil {
    // 			err = pprof.StartCPUProfile(profile_output)
    // 			if err != nil {
    // 				fmt.Printf("Unable to start CPU profile: %v\n", err)
    // 				profile_output.Close()
    // 				profile_output = nil
    // 			}
    // 			fmt.Printf("profout: %v\n", profile_output)
    // 		} else {
    // 			fmt.Printf("Unable to start CPU profile: %v\n", err)
    // 		}
    // 	} else {
    // 		pprof.StopCPUProfile()
    // 		profile_output.Close()
    // 		profile_output = nil
    // 	}
    // }

    // if key_map["mem profile"].FramePressCount() > 0 {
    // 	f, err := os.Create(filepath.Join(datadir, fmt.Sprintf("mem.%d.prof", num_mem_profiles)))
    // 	if err != nil {
    // 		base.Error().Printf("Unable to write mem profile: %v", err)
    // 	}
    // 	pprof.WriteHeapProfile(f)
    // 	f.Close()
    // 	num_mem_profiles++
    // }

    v += 0.01
  }
}
