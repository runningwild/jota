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
	// "github.com/runningwild/linear"
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

// versions are 'standard', 'moba', 'host', 'client'
func debugHookup(version string) (*cgf.Engine, *game.LocalData) {
	if version != "standard" && version != "moba" && version != "host" && version != "client" {
		base.Log().Fatalf("Unable to handle Version() == '%s'", Version())
	}

	for false && len(sys.GetActiveDevices()[gin.DeviceTypeController]) < 2 {
		time.Sleep(time.Millisecond * 100)
		sys.Think()
	}

	var engine *cgf.Engine
	var room game.Room
	generated := generator.GenerateRoom(1024, 1024, 100, 64, 64522029961391019)
	data, err := json.Marshal(generated)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	err = json.Unmarshal(data, &room)
	// err = base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	var players []game.Gid
	var localData *game.LocalData
	var g *game.Game
	if version == "client" {
		engine, err = cgf.NewClientEngine(17, "", 50001, base.Log())
		if err != nil {
			base.Log().Printf("Unable to connect: %v", err)
			base.Error().Fatalf("%v", err.Error())
		}
		localData = game.NewLocalDataArchitect(engine, sys)
		g = engine.GetState().(*game.Game)
		for _, ent := range g.Ents {
			if _, ok := ent.(*game.Player); ok {
				players = append(players, ent.Id())
			}
		}
	} else {
		sys.Think()
		g = new(game.Game)
		g.Rng = cmwc.MakeGoodCmwc()
		g.Rng.SeedWithDevRand()
		g.Ents = make(map[game.Gid]game.Ent)
		g.Friction = 0.97
		g.Friction_lava = 0.85
		g.Levels = make(map[game.Gid]*game.Level)
		g.Levels[game.GidInvadersStart] = &game.Level{}
		g.Levels[game.GidInvadersStart].Room = room
		// g.Standard = &game.GameModeStandard{}
		g.Moba = &game.GameModeMoba{}
		ps := g.AddPlayers(1, 1)
		for _, p := range ps {
			players = append(players, p)
		}
		ps = g.AddPlayers(1, 0)
		for _, p := range ps {
			players = append(players, p)
		}

		g.Init()
		if version == "host" {
			engine, err = cgf.NewHostEngine(g, 17, "", 50001, base.Log())
		} else {
			engine, err = cgf.NewLocalEngine(g, 17, base.Log())
		}
		if err != nil {
			base.Error().Fatalf("%v", err.Error())
		}
	}
	localData = game.NewLocalDataMoba(engine, players, gin.DeviceIndexAny, sys)
	// localData = game.NewLocalDataInvaders(engine, sys)

	// Hook the players up regardless of in we're architect or not, since we can
	// switch between the two in debug mode.
	// d := sys.GetActiveDevices()
	// n := 0
	// for _, index := range d[gin.DeviceTypeController] {
	// 	localData.SetLocalPlayer(g.Ents[players[n]], index)
	// 	n++
	// 	if n > len(players) {
	// 		break
	// 	}
	// }
	// if len(d[gin.DeviceTypeController]) == 0 {
	// 	localData.SetLocalPlayer(g.Ents[players[0]], 0)
	// }

	base.Log().Printf("Engine Id: %v", engine.Id())
	base.Log().Printf("All Ids: %v", engine.Ids())
	return engine, localData
}

func mainLoop(engine *cgf.Engine, local *game.LocalData, mode string) {
	defer engine.Kill()
	var profile_output *os.File
	var contention_output *os.File
	var num_mem_profiles int
	// ui.AddChild(base.MakeConsole())

	ticker := time.Tick(time.Millisecond * 17)
	ui := g2.Make(0, 0, wdx, wdy)
	ui.AddChild(&game.GameWindow{Engine: engine, Local: local, Dims: g2.Dims{wdx, wdy}}, g2.AnchorDeadCenter)
	ui.AddChild(g2.MakeConsole(wdx, wdy), g2.AnchorDeadCenter)
	side0Index := gin.In().BindDerivedKeyFamily("Side0", gin.In().MakeBindingFamily(gin.Key1, []gin.KeyIndex{gin.EitherControl}, []bool{true}))
	side1Index := gin.In().BindDerivedKeyFamily("Side1", gin.In().MakeBindingFamily(gin.Key2, []gin.KeyIndex{gin.EitherControl}, []bool{true}))
	side2Index := gin.In().BindDerivedKeyFamily("Side2", gin.In().MakeBindingFamily(gin.Key3, []gin.KeyIndex{gin.EitherControl}, []bool{true}))
	side0Key := gin.In().GetKeyFlat(side0Index, gin.DeviceTypeAny, gin.DeviceIndexAny)
	side1Key := gin.In().GetKeyFlat(side1Index, gin.DeviceTypeAny, gin.DeviceIndexAny)
	side2Key := gin.In().GetKeyFlat(side2Index, gin.DeviceTypeAny, gin.DeviceIndexAny)
	defer ui.StopEventListening()
	for {
		<-ticker
		if gin.In().GetKey(gin.AnyEscape).FramePressCount() != 0 {
			return
		}
		if mode == "moba" {
			if side0Key.FramePressCount() > 0 {
				local.DebugCyclePlayers()
			}
			// if side0Key.FramePressCount() > 0 {
			// 	local.DebugSetSide(0)
			// }
			// if side1Key.FramePressCount() > 0 {
			// 	local.DebugSetSide(1)
			// }
		}
		if mode == "standard" {
			if side0Key.FramePressCount() > 0 {
				local.DebugChangeMode(game.LocalModeInvaders)
			}
			if side1Key.FramePressCount() > 0 {
				local.DebugChangeMode(game.LocalModeArchitect)
			}
			if side2Key.FramePressCount() > 0 {
				local.DebugChangeMode(game.LocalModeEditor)
			}
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
						base.Log().Printf("Unable to start CPU profile: %v\n", err)
						profile_output.Close()
						profile_output = nil
					}
					base.Log().Printf("cpu prof: %v\n", profile_output)
				} else {
					base.Log().Printf("Unable to start CPU profile: %v\n", err)
				}
			} else {
				pprof.StopCPUProfile()
				profile_output.Close()
				profile_output = nil
			}
		}

		if gin.In().GetKey(gin.AnyKeyL).FramePressCount() > 0 {
			if contention_output == nil {
				contention_output, err = os.Create(filepath.Join(datadir, "contention.prof"))
				if err == nil {
					runtime.SetBlockProfileRate(1)
					base.Log().Printf("contention prof: %v\n", contention_output)
				} else {
					base.Log().Printf("Unable to start contention profile: %v\n", err)
				}
			} else {
				pprof.Lookup("block").WriteTo(contention_output, 0)
				contention_output.Close()
				contention_output = nil
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

func standardHookup() {
	g := g2.Make(0, 0, wdx, wdy)
	var tm g2.ThunderMenu
	tm.Subs = make(map[string]*g2.ThunderSubMenu)
	triggers := map[gin.KeyId]struct{}{
		gin.AnyReturn: struct{}{},
		gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, gin.DeviceIndexAny).Id(): struct{}{},
	}
	action := ""
	tm.Subs[""] = g2.MakeThunderSubMenu(
		[]g2.Widget{
			&g2.Button{Size: 50, Triggers: triggers, Name: "Debug", Callback: func() { tm.Push("debug") }},
			&g2.Button{Size: 50, Triggers: triggers, Name: "Host LAN game", Callback: func() { base.Log().Printf("HOST"); print("HOST\n") }},
			&g2.Button{Size: 50, Triggers: triggers, Name: "Join LAN game", Callback: func() { base.Log().Printf("JOIN"); print("JOIN\n") }},
			&g2.Button{Size: 50, Triggers: triggers, Name: "Quit", Callback: func() { action = "Quit" }},
		})

	tm.Subs["debug"] = g2.MakeThunderSubMenu(
		[]g2.Widget{
			&g2.Button{Size: 50, Triggers: triggers, Name: "Standard", Callback: func() { action = "standard" }},
			&g2.Button{Size: 50, Triggers: triggers, Name: "Moba", Callback: func() { action = "moba" }},
			&g2.Button{Size: 50, Triggers: triggers, Name: "Back", Callback: func() { tm.Pop() }},
		})

	tm.Start(500)
	g.AddChild(&tm, g2.AnchorDeadCenter)
	g.AddChild(g2.MakeConsole(wdx, wdy), g2.AnchorDeadCenter)

	t := texture.LoadFromPath(filepath.Join(base.GetDataDir(), "background/buttons1.jpg"))
	for {
		sys.Think()
		if action == "Quit" {
			return
		}
		if action == "standard" || action == "moba" {
			g.StopEventListening()
			engine, local := debugHookup(action)
			mainLoop(engine, local, action)
			g.RestartEventListening()
			action = ""
		}
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
	// 1 Start with a title screen
	// 2 Option to host or join
	// 3a If host then wait for a connection
	// 3b If join then ping and connect
	// 4 Once joined up the 'game' will handle choosing sides and whatnot
}

func main() {
	defer base.StackCatcher()
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
	sys.Think()
	base.LoadAllDictionaries()

	if Version() != "standard" {
		engine, local := debugHookup(Version())
		mainLoop(engine, local, "standard")
	} else {
		standardHookup()
	}
}
