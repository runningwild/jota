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
	"github.com/runningwild/linear"
	g2 "github.com/runningwild/magnus/gui"
	"time"
	// "math"
	"encoding/json"
	"github.com/runningwild/cgf"
	_ "github.com/runningwild/magnus/ability"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/generator"
	"github.com/runningwild/magnus/texture"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
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
	fmt.Printf("Prething: %v\n", key_binds)
	key_map = key_binds.MakeKeyMap()
	base.SetDefaultKeyMap(key_map)
}

func debugHookup() *cgf.Engine {
	for false && len(sys.GetActiveDevices()[gin.DeviceTypeController]) < 2 {
		time.Sleep(time.Millisecond * 100)
		sys.Think()
	}

	var engine *cgf.Engine
	var room game.Room
	generated := generator.GenerateRoom(2000, 700, 100, 50, 64522029961391019)
	data, err := json.Marshal(generated)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	err = json.Unmarshal(data, &room)
	// err = base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	room.NextId = len(room.Lava) + len(room.Walls) + 3
	var players []game.Gid
	if Version() == "host" {
		sys.Think()
		var g game.Game
		g.Rng = cmwc.MakeGoodCmwc()
		g.Rng.SeedWithDevRand()
		g.Ents = make(map[game.Gid]game.Ent)
		g.Friction = 0.97
		g.Friction_lava = 0.85
		g.Room = room

		players = g.AddPlayers(1)
		players = append(players, g.AddPest(linear.Vec2{500, 200}).Id())

		g.Init()
		engine, err = cgf.NewHostEngine(&g, 17, "", 50001, base.Log())
		if err != nil {
			base.Error().Fatalf("%v", err.Error())
		}
		game.SetLocalEngine(engine, sys, true)
	} else if Version() == "client" {
		engine, err = cgf.NewClientEngine(17, "", 50001, base.Log())
		if err != nil {
			base.Log().Printf("Unable to connect: %v", err)
			base.Error().Fatalf("%v", err.Error())
		}
		game.SetLocalEngine(engine, sys, false)
		g := engine.CopyState().(*game.Game)
		for _, ent := range g.Ents {
			if _, ok := ent.(*game.Player); ok {
				players = append(players, ent.Id())
			}
		}
	} else {
		base.Log().Fatalf("Unable to handle Version() == '%s'", Version())
	}
	if game.IsArchitect() {

	} else {
		d := sys.GetActiveDevices()
		n := 0
		for _, index := range d[gin.DeviceTypeController] {
			game.SetLocalPlayer(players[n], index)
			n++
			if n > len(players) {
				break
			}
		}
		if len(d[gin.DeviceTypeController]) == 0 {
			game.SetLocalPlayer(players[0], 0)
		}
	}
	base.Log().Printf("Engine Id: %v", engine.Id())
	base.Log().Printf("All Ids: %v", engine.Ids())
	anchor := gui.MakeAnchorBox(gui.Dims{(wdx * 3) / 4, (wdy * 3) / 4})
	ui.AddChild(anchor)
	anchor.AddChild(&game.GameWindow{Engine: engine}, gui.Anchor{0.1, 0.5, 0.1, 0.5})
	return engine
}

func mainLoop(engine *cgf.Engine) {
	var profile_output *os.File
	var num_mem_profiles int
	// ui.AddChild(base.MakeConsole())

	ticker := time.Tick(time.Millisecond * 17)
	for {
		<-ticker
		if gin.In().GetKey(gin.AnyEscape).FramePressCount() != 0 {
			return
		}
		sys.Think()
		render.Queue(func() {
			ui.Draw()
		})
		render.Queue(func() {
			sys.SwapBuffers()
		})
		render.Purge()

		// TODO: Replace the 'P' key with an appropriate keybind
		var err error
		if gin.In().GetKey(gin.AnyKeyP).FramePressCount() > 0 {
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

		// TODO: Replace the 'M' key with an appropriate keybind
		if gin.In().GetKey(gin.AnyKeyM).FramePressCount() > 0 {
			f, err := os.Create(filepath.Join(datadir, fmt.Sprintf("mem.%d.prof", num_mem_profiles)))
			if err != nil {
				base.Error().Printf("Unable to write mem profile: %v", err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
			num_mem_profiles++
		}
	}
}

func standardHookup() *cgf.Engine {
	g := g2.Make(0, 0, wdx, wdy)
	base.Log().Printf("%d %d", wdx, wdy)
	g.AddChild(&g2.Box{Color: [4]int{255, 255, 0, 255}, Dims: g2.Dims{100, 300}}, g2.AnchorDeadCenter)
	t := texture.LoadFromPath(filepath.Join(base.GetDataDir(), "background/buttons1.jpg"))
	for gin.In().GetKey(gin.AnyEscape).FramePressCount() == 0 {
		sys.Think()
		render.Queue(func() {
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			if true {
				ratio := float64(wdx) / float64(wdy)
				t.RenderAdvanced(-1+(1-1/ratio), -1, 2/ratio, 2, 0, false)
			}
			gl.Disable(gl.TEXTURE_2D)
			base.GetDictionary("luxisr").RenderString("INvASioN!!!", 0, 0.5, 0, 0.03, gui.Center)
		})
		render.Queue(func() {
			g.Draw()
			sys.SwapBuffers()
		})
		render.Purge()
	}
	return nil

	// 1 Start with a title screen
	// 2 Option to host or join
	// 3a If host then wait for a connection
	// 3b If join then ping and connect
	// 4 Once joined up the 'game' will handle choosing sides and whatnot
}

func main() {
	fmt.Printf("%v\n", key_map)
	sys.Startup()
	err := gl.Init()
	if err != nil {
		base.Error().Fatalf("%v", err)
	}

	render.Init()
	render.Queue(func() {
		sys.CreateWindow(10, 10, wdx, wdy)
		sys.EnableVSync(true)
		err := gl.Init()
		if err != nil {
			base.Error().Fatalf("%v", err)
		}
	})
	base.InitShaders()
	runtime.GOMAXPROCS(2)
	ui, err = gui.Make(gin.In(), gui.Dims{wdx, wdy}, filepath.Join(datadir, "fonts", "skia.ttf"))
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	sys.Think()
	base.LoadAllDictionaries()

	var engine *cgf.Engine
	if Version() != "standard" {
		engine = debugHookup()
	} else {
		engine = standardHookup()
	}
	mainLoop(engine)
}
