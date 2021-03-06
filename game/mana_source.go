package game

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/linear"
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

func deleteNodes(nodes []node) {
	nodeCachesMutex.Lock()
	defer nodeCachesMutex.Unlock()
	nodeCaches[len(nodes)].deleteBuffer(nodes)
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

	Rng *cmwc.Cmwc
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

type ManaSource struct {
	options ManaSourceOptions

	nodes    [][]node
	rawNodes []node // the underlying array for nodes

	thinks int

	local manaSourceLocalData
}

func (ms *ManaSource) GobEncode() ([]byte, error) {
	base.Log().Printf("GobEncode")
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(ms.options)
	if err == nil {
		err = enc.Encode(uint32(ms.thinks))
		base.Log().Printf("Encode thinks: %d", ms.thinks)
	}
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
		ms.thinks = int(d)
		base.Log().Printf("Decoded %d", d)
	}
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
		base.Error().Fatalf(fmt.Sprintf("Invalid options: %v", options))
	}

	r := rand.New(options.Rng)

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

// func (src *ManaSource) ReleaseResources() {
// 	deleteNodes(src.rawNodes)
// }

type nodeThinkData struct {
	// playerDistSquared and playerControl are both 0 if the node is not in the playerThinkData
	// range.
	playerDistSquared    []float64
	playerControl        []float64
	playerDrain          []Mana
	hasSomePlayerControl bool
}

type playerThinkData struct {
	minX       int
	maxX       int
	minY       int
	maxY       int
	drain      Mana
	rateFactor float64
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
				base.Error().Fatalf("NaN showed up somewhere!")
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

func (ms *ManaSource) getPlayerRanges(td *thinkData, ents []Ent) {
	i := -1
	for _, ent := range ents {
		i++
		playerThinkData := &td.playerThinkData[i]

		playerThinkData.minX = -1
		playerThinkData.minY = -1
		playerThinkData.maxX = -1
		playerThinkData.maxY = -1

		playerThinkData.rateFactor = ent.Stats().MaxRate()

		for x := range ms.nodes {
			dx := ms.nodes[x][0].X - ent.Pos().X
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
			dy := ms.nodes[0][y].Y - ent.Pos().Y
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

func (ms *ManaSource) setPlayerControl(td *thinkData, ents []Ent) {
	maxDistSquared := ms.options.MaxDrainDistance * ms.options.MaxDrainDistance
	i := -1
	for _, ent := range ents {
		i++
		playerThinkData := &td.playerThinkData[i]
		if !playerThinkData.isValid() {
			continue
		}
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.nodes[x][y]
				nodeThinkData := &td.nodeThinkData[x][y]
				distSquared := ent.Pos().Sub(linear.MakeVec2(node.X, node.Y)).Mag2()
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
	i := -1
	for _ = range td.playerThinkData {
		i++
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
						if amountScale != amountScale {
							panic("amount")
						}
						nodeThinkData.playerDrain[i][c] =
							math.Min(amountScale*maxDrainRate*playerThinkData.rateFactor, node.Mana[c]) * control
						if nodeThinkData.playerDrain[i][c] != nodeThinkData.playerDrain[i][c] {
							panic("nodeThinkData.playerDrain[i][c]")
						}
						playerThinkData.drain[c] += nodeThinkData.playerDrain[i][c]
					}
				}
			}
		}
	}
}

func (ms *ManaSource) supplyPlayers(td *thinkData, ents []Ent) {
	i := -1
	for _, ent := range ents {
		i++
		playerThinkData := &td.playerThinkData[i]
		if !playerThinkData.isValid() {
			continue
		}
		drainUsed := ent.Supply(playerThinkData.drain)
		for x := playerThinkData.minX; x <= playerThinkData.maxX; x++ {
			for y := playerThinkData.minY; y <= playerThinkData.maxY; y++ {
				node := &ms.nodes[x][y]
				nodeThinkData := td.nodeThinkData[x][y]
				if nodeThinkData.playerControl[i] > 0 {
					for c := range node.Mana {
						if playerThinkData.drain[c] > 0 {
							usedFrac := 1.0 - drainUsed[c]/playerThinkData.drain[c]
							node.Mana[c] = math.Max(0.0, node.Mana[c]-nodeThinkData.playerDrain[i][c]*usedFrac)
						}
					}
				}
			}
		}
	}
}

var globalThinkData thinkData

func (ms *ManaSource) Think(ents map[Gid]Ent) {
	ms.thinks++
	// If regenerateMana takes too long we can just do it every other frame and
	// have mana regen at twice the rate.  Should look just as good and will save
	// on some cycles.
	if ms.thinks%1 == 0 {
		ms.regenerateMana()
	}
	var drains []Ent
	for _, ent := range ents {
		if ent.Stats().MaxRate() > 0 {
			drains = append(drains, ent)
		}
	}
	ms.initThinkData(&globalThinkData, len(drains))
	ms.getPlayerRanges(&globalThinkData, drains)
	ms.setPlayerControl(&globalThinkData, drains)
	ms.setPlayerDrain(&globalThinkData)
	ms.supplyPlayers(&globalThinkData, drains)
}
