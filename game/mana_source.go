// TODO: gobEncode

package game

import (
	"bytes"
	"encoding/gob"
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/texture"
	"math"
	"math/rand"
	"sync"
)

type nodeCache struct {
	cache [][]node
	mutex sync.Mutex
	count int
	size  int
}

func (nc *nodeCache) newBuffer() []node {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()
	if len(nc.cache) == 0 {
		nc.increaseBuffer(nc.count)
		nc.count += len(nc.cache)
	}
	ret := nc.cache[len(nc.cache)-1]
	nc.cache = nc.cache[0 : len(nc.cache)-1]
	return ret
}
func (nc *nodeCache) deleteBuffer(nodes []node) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()
	nc.cache = append(nc.cache, nodes)
}
func (nc *nodeCache) increaseBuffer(n int) {
	for i := 0; i < n; i++ {
		nc.cache = append(nc.cache, make([]node, nc.size))
	}
}
func makeNodeCache(size int) *nodeCache {
	var nc nodeCache
	nc.size = size
	nc.increaseBuffer(1)
	nc.count = 1
	return &nc
}

var nodeCaches map[int]*nodeCache
var nodeCachesMutex sync.Mutex

func init() {
	nodeCaches = make(map[int]*nodeCache)
}
func newNodes(size int) []node {
	nodeCachesMutex.Lock()
	defer nodeCachesMutex.Unlock()
	var nc *nodeCache
	var ok bool
	nc, ok = nodeCaches[size]
	if !ok {
		nc = makeNodeCache(size)
		nodeCaches[size] = nc
	}
	return nc.newBuffer()
}
func deleteNodes(size int, nodes []node) {
	nodeCachesMutex.Lock()
	defer nodeCachesMutex.Unlock()
	nodeCaches[size].deleteBuffer(nodes)
}

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
	options ManaSourceOptions

	nodes    [][]node
	rawNodes []node // the underlying array for nodes
}

func (ms *ManaSource) GobEncode() ([]byte, error) {
	base.Log().Printf("GobEncode")
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(ms.options)
	if err == nil {
		err = enc.Encode(uint32(len(ms.nodes)))
		base.Log().Printf("Encode dx: %d", len(ms.nodes))
	}
	if err == nil {
		err = enc.Encode(uint32(len(ms.nodes[0])))
		base.Log().Printf("Encode dy: %d", len(ms.nodes[0]))
	}
	if err == nil {
		err = enc.Encode(ms.rawNodes)
	}
	return buf.Bytes(), err
}

func (ms *ManaSource) GobDecode(data []byte) error {
	base.Log().Printf("GobDecode")
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&ms.options)
	var d1, d2 int
	if err == nil {
		var d uint32
		err = dec.Decode(&d)
		d1 = int(d)
		base.Log().Printf("Decoded %d", d1)
	}
	if err == nil {
		var d uint32
		err = dec.Decode(&d)
		d2 = int(d)
		base.Log().Printf("Decoded %d", d2)
	}
	if err == nil {
		err = dec.Decode(&ms.rawNodes)
	}
	if err == nil {
		ms.nodes = make([][]node, d1)
		for i := range ms.nodes {
			ms.nodes[i] = ms.rawNodes[i*d2 : (i+1)*d2]
		}
	}
	return err
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

func (ms *ManaSource) Init(options *ManaSourceOptions) {
	ms.options = *options
	if options.NumNodeCols < 2 || options.NumNodeRows < 2 {
		panic(fmt.Sprintf("Invalid options: %v", options))
	}

	c := cmwc.MakeGoodCmwc()
	c.SeedWithDevRand()
	r := rand.New(c)

	seeds := make([]nodeSeed, options.NumSeeds)
	for i := range seeds {
		seed := &seeds[i]
		seed.x = options.BoardLeft + r.Float64()*(options.BoardRight-options.BoardLeft)
		seed.y = options.BoardTop + r.Float64()*(options.BoardBottom-options.BoardTop)
		seed.color = r.Intn(3)
	}

	ms.rawNodes = newNodes(options.NumNodeCols * options.NumNodeRows)
	// ms.rawNodes = make([]node, options.NumNodeCols*options.NumNodeRows)
	ms.nodes = make([][]node, options.NumNodeCols)
	for col := 0; col < options.NumNodeCols; col++ {
		ms.nodes[col] = ms.rawNodes[col*options.NumNodeRows : (col+1)*options.NumNodeRows]
		for row := 0; row < options.NumNodeRows; row++ {
			x := options.BoardLeft + float64(col)/float64(options.NumNodeCols-1)*(options.BoardRight-options.BoardLeft)
			y := options.BoardTop + float64(row)/float64(options.NumNodeRows-1)*(options.BoardBottom-options.BoardTop)

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

			normalizeWeights(options.NodeMagnitude, maxWeightByColor[:])
			var weightsCopy [3]float64
			copy(weightsCopy[:], maxWeightByColor[:])

			ms.nodes[col][row] = node{
				X:             x,
				Y:             y,
				RegenPerFrame: options.RegenPerFrame,
				Mana:          maxWeightByColor,
				MaxMana:       weightsCopy,
			}
		}
	}
}

func (src *ManaSource) Copy() ManaSource {
	var dst ManaSource

	dst.rawNodes = newNodes(len(src.rawNodes))
	// dst.rawNodes = make([]node, len(src.rawNodes))
	dst.nodes = make([][]node, len(src.nodes))
	for x := range src.nodes {
		dst.nodes[x] = dst.rawNodes[x*len(src.nodes[x]) : (x+1)*len(src.nodes[x])]
	}
	for i, srcNode := range src.rawNodes {
		dst.rawNodes[i] = srcNode
	}
	dst.options = src.options

	return dst
}

func (dst *ManaSource) OverwriteWith(src *ManaSource) {
	for i, srcNode := range src.rawNodes {
		dst.rawNodes[i] = srcNode
	}
	dst.options = src.options
}

func (ms *ManaSource) Draw(gw *GameWindow, dx float64, dy float64) {
	if gw.nodeTextureData == nil {
		//		gl.Enable(gl.TEXTURE_2D)
		gw.nodeTextureData = make([]byte, ms.options.NumNodeRows*ms.options.NumNodeCols*3)
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
			gl.Sizei(ms.options.NumNodeRows),
			gl.Sizei(ms.options.NumNodeCols),
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
	for i := range ms.rawNodes {
		for c := 0; c < 3; c++ {
			color_frac := ms.rawNodes[i].Mana[c] * 1.0 / ms.options.NodeMagnitude
			color_range := float64(ms.options.MaxNodeBrightness - ms.options.MinNodeBrightness)
			gw.nodeTextureData[i*3+c] = byte(
				color_frac*color_range + float64(ms.options.MinNodeBrightness))
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
		gl.Sizei(ms.options.NumNodeRows),
		gl.Sizei(ms.options.NumNodeCols),
		gl.RGB,
		gl.UNSIGNED_BYTE,
		gl.Pointer(&gw.nodeTextureData[0]))

	gl.ActiveTexture(gl.TEXTURE1)
	// TODO: Should probably extricate nodeWarpingData entirely
	// for i, ent := range gw.game.Ents {
	// 	p := ent.Pos()
	// 	gw.nodeWarpingData[3*i+0] = byte(p.X / float64(gw.game.Dx) * 255)
	// 	gw.nodeWarpingData[3*i+1] = -byte(p.Y / float64(gw.game.Dy) * 255)
	// 	gw.nodeWarpingData[3*i+2] = 255
	// }
	gl.TexImage1D(
		gl.TEXTURE_1D,
		0,
		gl.RGBA,
		gl.Sizei(len(gw.nodeWarpingData)/4),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Pointer(&gw.nodeWarpingData[0]))

	base.EnableShader("nodes")
	base.SetUniformI("nodes", "width", ms.options.NumNodeRows)
	base.SetUniformI("nodes", "height", ms.options.NumNodeCols)
	base.SetUniformI("nodes", "drains", 1)
	base.SetUniformI("nodes", "tex0", 0)
	base.SetUniformI("nodes", "tex1", 1)
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

func (p playerThinkData) isValid() bool {
	return p.minX != -1 && p.maxX != -1 && p.minY != -1 && p.maxY != -1
}

type thinkData struct {
	nodeThinkData    [][]nodeThinkData
	rawNodeThinkData []nodeThinkData // Underlying array for nodeThinkData
	playerThinkData  []playerThinkData
}

func (ms *ManaSource) regenerateMana() {
	for i := range ms.rawNodes {
		node := &ms.rawNodes[i]
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
	if len(td.nodeThinkData) != len(ms.nodes) {
		td.rawNodeThinkData = make([]nodeThinkData, len(ms.rawNodes))
		td.nodeThinkData = make([][]nodeThinkData, len(ms.nodes))
		for i := range td.nodeThinkData {
			td.nodeThinkData[i] =
				td.rawNodeThinkData[i*ms.options.NumNodeRows : (i+1)*ms.options.NumNodeRows]
		}
	}

	for i := range ms.rawNodes {
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
	maxDistSquared := ms.options.MaxDrainDistance * ms.options.MaxDrainDistance
	if distSquared > maxDistSquared {
		return 0.0
	}
	distRatio := 1.0 - distSquared/maxDistSquared
	return distRatio * distRatio * ms.options.MaxDrainRate
}

func (ms *ManaSource) getPlayerRanges(td *thinkData, players []Ent) {
	for i, player := range players {
		playerThinkData := &td.playerThinkData[i]

		playerThinkData.minX = -1
		playerThinkData.minY = -1
		playerThinkData.maxX = -1
		playerThinkData.maxY = -1

		for x := range ms.nodes {
			dx := ms.nodes[x][0].X - player.Pos().X
			if dx >= -ms.options.MaxDrainDistance && playerThinkData.minX == -1 {
				playerThinkData.minX = x
			}
			if dx <= ms.options.MaxDrainDistance {
				playerThinkData.maxX = x
			} else {
				break
			}
		}

		for y := range ms.nodes[0] {
			dy := ms.nodes[0][y].Y - player.Pos().Y
			if dy >= -ms.options.MaxDrainDistance && playerThinkData.minY == -1 {
				playerThinkData.minY = y
			}
			if dy <= ms.options.MaxDrainDistance {
				playerThinkData.maxY = y
			} else {
				break
			}
		}
	}
}

func (ms *ManaSource) setPlayerControl(td *thinkData, players []Ent) {
	maxDistSquared := ms.options.MaxDrainDistance * ms.options.MaxDrainDistance
	for i, player := range players {
		playerThinkData := &td.playerThinkData[i]
		if !playerThinkData.isValid() {
			continue
		}
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.nodes[x][y]
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

	for x := range ms.nodes {
		for y := range ms.nodes[x] {
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
		if !playerThinkData.isValid() {
			continue
		}
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.nodes[x][y]
				nodeThinkData := td.nodeThinkData[x][y]
				control := nodeThinkData.playerControl[i]
				if control > 0 {
					maxDrainRate := ms.getMaxDrainRate(nodeThinkData.playerDistSquared[i])
					for c := range node.Mana {
						amountScale := node.MaxMana[c] / float64(ms.options.NodeMagnitude)
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
		if !playerThinkData.isValid() {
			continue
		}
		drainUsed := player.Supply(playerThinkData.drain)
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.nodes[x][y]
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
