// TODO: gobEncode

package game

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/texture"
	"math"
	"math/rand"
)

// One value for each color
type ManaRequest [3]bool
type Mana [3]float64

func (m Mana) Magnitude() float64 {
	return m[0] + m[1] + m[2]
}

type ManaSourceOptions struct {
	NumSeeds    int
	NumNodeRows int
	NumNodeCols int

	BoardLeft   float64
	BoardTop    float64
	BoardRight  float64
	BoardBottom float64

	MaxDrainDistance float64
	MaxDrainRate     float64

	RegenPerFrame     float64
	NodeMagnitude     float64
	MinNodeBrightness int
	MaxNodeBrightness int
}

type node struct {
	X, Y          float64
	RegenPerFrame float64
	Mana          Mana
	MaxMana       Mana
}

type nodeSeed struct {
	x, y  float64
	color int
}

func (dst *node) OverwriteWith(src *node) {
	*dst = *src
}

type ManaSource struct {
	Options ManaSourceOptions

	Nodes    [][]node
	RawNodes []node // the underlying array for Nodes
}

func normalizeWeights(desiredSum float64, weights []float64) {
	sum := 0.0
	for i := range weights {
		sum += weights[i]
	}
	multiplier := desiredSum / sum
	for i := range weights {
		weights[i] *= multiplier
	}
}

func (ms *ManaSource) Init(Options *ManaSourceOptions, walls []linear.Poly, lava []linear.Poly) {
	ms.Options = *Options
	if Options.NumNodeCols < 2 || Options.NumNodeRows < 2 {
		panic(fmt.Sprintf("Invalid Options: %v", Options))
	}

	c := cmwc.MakeGoodCmwc()
	c.SeedWithDevRand()
	r := rand.New(c)

	seeds := make([]nodeSeed, Options.NumSeeds)
	for i := range seeds {
		seed := &seeds[i]
		seed.x = Options.BoardLeft + r.Float64()*(Options.BoardRight-Options.BoardLeft)
		seed.y = Options.BoardTop + r.Float64()*(Options.BoardBottom-Options.BoardTop)
		seed.color = r.Intn(3)
	}

	var allObstacles []linear.Poly
	for _, p := range walls {
		allObstacles = append(allObstacles, p)
	}
	for _, p := range lava {
		allObstacles = append(allObstacles, p)
	}

	ms.RawNodes = make([]node, Options.NumNodeCols*Options.NumNodeRows)
	ms.Nodes = make([][]node, Options.NumNodeCols)
	for col := 0; col < Options.NumNodeCols; col++ {
		ms.Nodes[col] = ms.RawNodes[col*Options.NumNodeRows : (col+1)*Options.NumNodeRows]
		for row := 0; row < Options.NumNodeRows; row++ {
			x := Options.BoardLeft + float64(col)/float64(Options.NumNodeCols-1)*(Options.BoardRight-Options.BoardLeft)
			y := Options.BoardTop + float64(row)/float64(Options.NumNodeRows-1)*(Options.BoardBottom-Options.BoardTop)

			// all_obstacles[0] corresponds to the outer walls. We do not want to drop mana Nodes for
			// being inside there.
			insideObstacle := false
			for i := 1; !insideObstacle && i < len(allObstacles); i++ {
				if vecInsideConvexPoly(linear.Vec2{x, y}, allObstacles[i]) {
					insideObstacle = true
				}
			}
			if insideObstacle {
				ms.Nodes[col][row].X = x
				ms.Nodes[col][row].Y = y
				continue
			}

			maxWeightByColor := [3]float64{0.0, 0.0, 0.0}
			for _, seed := range seeds {
				c := seed.color
				dx := x - seed.x
				dy := y - seed.y
				distSquared := dx*dx + dy*dy
				weight := 1 / (distSquared + 1.0)
				if weight > maxWeightByColor[c] {
					maxWeightByColor[c] = weight
				}
			}

			normalizeWeights(Options.NodeMagnitude, maxWeightByColor[:])
			var weightsCopy [3]float64
			copy(weightsCopy[:], maxWeightByColor[:])

			ms.Nodes[col][row] = node{
				X:             x,
				Y:             y,
				RegenPerFrame: Options.RegenPerFrame,
				Mana:          maxWeightByColor,
				MaxMana:       weightsCopy,
			}
		}
	}
}

func (src *ManaSource) Copy() ManaSource {
	var dst ManaSource

	dst.RawNodes = make([]node, len(src.RawNodes))
	dst.Nodes = make([][]node, len(src.Nodes))
	for x := range src.Nodes {
		dst.Nodes[x] = dst.RawNodes[x*len(src.Nodes[x]) : (x+1)*len(src.Nodes[x])]
	}
	for i, srcNode := range src.RawNodes {
		dst.RawNodes[i] = srcNode
	}
	dst.Options = src.Options

	return dst
}

func (dst *ManaSource) OverwriteWith(src *ManaSource) {
	for i, srcNode := range src.RawNodes {
		dst.RawNodes[i] = srcNode
	}
	dst.Options = src.Options
}

func (ms *ManaSource) Draw(gw *GameWindow, dx float64, dy float64) {
	if gw.nodeTextureData == nil {
		//		gl.Enable(gl.TEXTURE_2D)
		gw.nodeTextureData = make([]byte, ms.Options.NumNodeRows*ms.Options.NumNodeCols*3)
		gl.GenTextures(1, &gw.nodeTextureId)
		gl.BindTexture(gl.TEXTURE_2D, gw.nodeTextureId)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexImage2D(
			gl.TEXTURE_2D,
			0,
			gl.RGB,
			gl.Sizei(ms.Options.NumNodeRows),
			gl.Sizei(ms.Options.NumNodeCols),
			0,
			gl.RGB,
			gl.UNSIGNED_BYTE,
			gl.Pointer(&gw.nodeTextureData[0]))

		//		gl.ActiveTexture(gl.TEXTURE1)
		gl.GenTextures(1, &gw.nodeWarpingTexture)
		gl.BindTexture(gl.TEXTURE_1D, gw.nodeWarpingTexture)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_1D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_1D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_1D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_1D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gw.nodeWarpingData = make([]byte, 4*10)
		gl.TexImage1D(
			gl.TEXTURE_1D,
			0,
			gl.RGBA,
			gl.Sizei(len(gw.nodeWarpingData)/4),
			0,
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Pointer(&gw.nodeWarpingData[0]))
	}
	for i := range ms.RawNodes {
		for c := 0; c < 3; c++ {
			color_frac := ms.RawNodes[i].Mana[c] * 1.0 / ms.Options.NodeMagnitude
			color_range := float64(ms.Options.MaxNodeBrightness - ms.Options.MinNodeBrightness)
			gw.nodeTextureData[i*3+c] = byte(
				color_frac*color_range + float64(ms.Options.MinNodeBrightness))
		}
	}
	gl.Enable(gl.TEXTURE_1D)
	gl.Enable(gl.TEXTURE_2D)
	//gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, gw.nodeTextureId)
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		gl.Sizei(ms.Options.NumNodeRows),
		gl.Sizei(ms.Options.NumNodeCols),
		gl.RGB,
		gl.UNSIGNED_BYTE,
		gl.Pointer(&gw.nodeTextureData[0]))

	gl.ActiveTexture(gl.TEXTURE1)
	for i, ent := range gw.game.Ents {
		p := ent.Pos()
		gw.nodeWarpingData[3*i+0] = byte(p.X / float64(gw.game.Dx) * 255)
		gw.nodeWarpingData[3*i+1] = -byte(p.Y / float64(gw.game.Dy) * 255)
		gw.nodeWarpingData[3*i+2] = 255
	}
	gl.TexImage1D(
		gl.TEXTURE_1D,
		0,
		gl.RGBA,
		gl.Sizei(len(gw.nodeWarpingData)/4),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Pointer(&gw.nodeWarpingData[0]))

	base.EnableShader("Nodes")
	base.SetUniformI("Nodes", "width", ms.Options.NumNodeRows)
	base.SetUniformI("Nodes", "height", ms.Options.NumNodeCols)
	base.SetUniformI("Nodes", "drains", 1)
	base.SetUniformI("Nodes", "tex0", 0)
	base.SetUniformI("Nodes", "tex1", 1)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_1D, gw.nodeWarpingTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, gw.nodeTextureId)
	texture.RenderAdvanced(150, -150, dy, dx, 3.1415926535/2, true)
	base.EnableShader("")
	gl.Disable(gl.TEXTURE_2D)
	gl.Disable(gl.TEXTURE_1D)
}

type nodeThinkData struct {
	// playerDistSquared and playerControl are both 0 if the node is not in the playerThinkData
	// range.
	playerDistSquared    []float64
	playerControl        []float64
	playerDrain          []Mana
	hasSomePlayerControl bool
}

type playerThinkData struct {
	minX  int
	maxX  int
	minY  int
	maxY  int
	drain Mana
}

type thinkData struct {
	nodeThinkData    [][]nodeThinkData
	rawNodeThinkData []nodeThinkData // Underlying array for nodeThinkData
	playerThinkData  []playerThinkData
}

func (ms *ManaSource) regenerateMana() {
	for i := range ms.RawNodes {
		node := &ms.RawNodes[i]
		for c := range node.Mana {
			if node.MaxMana[c] == 0 {
				continue
			}
			maxRecovery := node.MaxMana[c] * node.RegenPerFrame
			scale := (node.MaxMana[c] - node.Mana[c]) / node.MaxMana[c]
			node.Mana[c] += scale * maxRecovery
			if scale != scale || maxRecovery != maxRecovery {
				panic("fd")
			}
		}
	}
}

func (ms *ManaSource) initThinkData(td *thinkData, numPlayers int) {
	if len(td.nodeThinkData) != len(ms.Nodes) {
		td.rawNodeThinkData = make([]nodeThinkData, len(ms.RawNodes))
		td.nodeThinkData = make([][]nodeThinkData, len(ms.Nodes))
		for i := range td.nodeThinkData {
			td.nodeThinkData[i] =
				td.rawNodeThinkData[i*ms.Options.NumNodeRows : (i+1)*ms.Options.NumNodeRows]
		}
	}

	for i := range ms.RawNodes {
		node := &td.rawNodeThinkData[i]
		if len(node.playerDistSquared) != numPlayers {
			node.playerDistSquared = make([]float64, numPlayers)
			node.playerControl = make([]float64, numPlayers)
			node.playerDrain = make([]Mana, numPlayers)
		}
		for i := 0; i < numPlayers; i++ {
			node.playerDistSquared[i] = 0
			node.playerControl[i] = 0
			node.playerDrain[i][0] = 0
			node.playerDrain[i][1] = 0
			node.playerDrain[i][2] = 0
		}
		node.hasSomePlayerControl = false
	}

	if len(td.playerThinkData) != numPlayers {
		td.playerThinkData = make([]playerThinkData, numPlayers)
	}
	for i := 0; i < numPlayers; i++ {
		td.playerThinkData[i].drain[0] = 0
		td.playerThinkData[i].drain[1] = 0
		td.playerThinkData[i].drain[2] = 0
	}
}

func (ms *ManaSource) getMaxDrainRate(distSquared float64) float64 {
	maxDistSquared := ms.Options.MaxDrainDistance * ms.Options.MaxDrainDistance
	if distSquared > maxDistSquared {
		return 0.0
	}
	distRatio := 1.0 - distSquared/maxDistSquared
	return distRatio * distRatio * ms.Options.MaxDrainRate
}

func (ms *ManaSource) getPlayerRanges(td *thinkData, players []Ent) {
	for i, player := range players {
		playerThinkData := &td.playerThinkData[i]

		playerThinkData.minX = -1
		playerThinkData.minY = -1
		playerThinkData.maxX = -1
		playerThinkData.maxY = -1

		for x := range ms.Nodes {
			dx := ms.Nodes[x][0].X - player.Pos().X
			if dx >= -ms.Options.MaxDrainDistance && playerThinkData.minX == -1 {
				playerThinkData.minX = x
			}
			if dx <= ms.Options.MaxDrainDistance {
				playerThinkData.maxX = x
			} else {
				break
			}
		}

		for y := range ms.Nodes[0] {
			dy := ms.Nodes[0][y].Y - player.Pos().Y
			if dy >= -ms.Options.MaxDrainDistance && playerThinkData.minY == -1 {
				playerThinkData.minY = y
			}
			if dy <= ms.Options.MaxDrainDistance {
				playerThinkData.maxY = y
			} else {
				break
			}
		}
	}
}

func (ms *ManaSource) setPlayerControl(td *thinkData, players []Ent) {
	maxDistSquared := ms.Options.MaxDrainDistance * ms.Options.MaxDrainDistance
	for i, player := range players {
		playerThinkData := &td.playerThinkData[i]
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.Nodes[x][y]
				nodeThinkData := &td.nodeThinkData[x][y]
				distSquared := player.Pos().Sub(linear.MakeVec2(node.X, node.Y)).Mag2()
				if distSquared <= maxDistSquared {
					nodeThinkData.playerDistSquared[i] = distSquared
					nodeThinkData.playerControl[i] = 1.0 / (distSquared + 1.0)
					nodeThinkData.hasSomePlayerControl = true
				}
			}
		}
	}

	for x := range ms.Nodes {
		for y := range ms.Nodes[x] {
			nodeThinkData := &td.nodeThinkData[x][y]
			if nodeThinkData.hasSomePlayerControl {
				normalizeWeights(1.0, nodeThinkData.playerControl)
			}
		}
	}
}

func (ms *ManaSource) setPlayerDrain(td *thinkData) {
	for i := range td.playerThinkData {
		playerThinkData := &td.playerThinkData[i]
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.Nodes[x][y]
				nodeThinkData := td.nodeThinkData[x][y]
				control := nodeThinkData.playerControl[i]
				if control > 0 {
					maxDrainRate := ms.getMaxDrainRate(nodeThinkData.playerDistSquared[i])
					for c := range node.Mana {
						amountScale := node.MaxMana[c] / float64(ms.Options.NodeMagnitude)
						nodeThinkData.playerDrain[i][c] =
							math.Min(amountScale*maxDrainRate, node.Mana[c]) * control
						playerThinkData.drain[c] += nodeThinkData.playerDrain[i][c]
					}
				}
			}
		}
	}
}

func (ms *ManaSource) supplyPlayers(td *thinkData, players []Ent) {
	for i, player := range players {
		playerThinkData := &td.playerThinkData[i]
		drainUsed := player.Supply(playerThinkData.drain)
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.Nodes[x][y]
				nodeThinkData := td.nodeThinkData[x][y]
				if nodeThinkData.playerControl[i] > 0 {
					for c := range node.Mana {
						usedFrac := 1.0 - drainUsed[c]/playerThinkData.drain[c]
						node.Mana[c] = math.Max(0.0, node.Mana[c]-nodeThinkData.playerDrain[i][c]*usedFrac)
					}
				}
			}
		}
	}
}

var globalThinkData thinkData

func (ms *ManaSource) Think(players []Ent) {
	ms.regenerateMana()
	ms.initThinkData(&globalThinkData, len(players))
	ms.getPlayerRanges(&globalThinkData, players)
	ms.setPlayerControl(&globalThinkData, players)
	ms.setPlayerDrain(&globalThinkData)
	ms.supplyPlayers(&globalThinkData, players)
}
