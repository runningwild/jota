// TODO: gobEncode

package game

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/texture"
	"math/rand"
)

// One value for each color
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
	nodes   [][]node
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

func (ms *ManaSource) Init(options *ManaSourceOptions, walls []linear.Poly, lava []linear.Poly) {
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

	var allObstacles []linear.Poly
	for _, p := range walls {
		allObstacles = append(allObstacles, p)
	}
	for _, p := range lava {
		allObstacles = append(allObstacles, p)
	}

	ms.nodes = make([][]node, options.NumNodeCols)
	for col := 0; col < options.NumNodeCols; col++ {
		ms.nodes[col] = make([]node, options.NumNodeRows)
		for row := 0; row < options.NumNodeRows; row++ {
			x := options.BoardLeft + float64(col)/float64(options.NumNodeCols-1)*(options.BoardRight-options.BoardLeft)
			y := options.BoardTop + float64(row)/float64(options.NumNodeRows-1)*(options.BoardBottom-options.BoardTop)

			// all_obstacles[0] corresponds to the outer walls. We do not want to drop mana nodes for
			// being inside there.
			insideObstacle := false
			for i := 1; !insideObstacle && i < len(allObstacles); i++ {
				if vecInsideConvexPoly(linear.Vec2{x, y}, allObstacles[i]) {
					insideObstacle = true
				}
			}
			if insideObstacle {
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

	dst.nodes = make([][]node, len(src.nodes))
	for x := range src.nodes {
		dst.nodes[x] = make([]node, len(src.nodes[x]))
		for y, srcN := range src.nodes[x] {
			dst.nodes[x][y] = srcN
		}
	}
	dst.options = src.options

	return dst
}

func (dst *ManaSource) OverwriteWith(src *ManaSource) {
	for x := range src.nodes {
		for y, srcN := range src.nodes[x] {
			dst.nodes[x][y] = srcN
		}
	}
	dst.options = src.options
}

func (ms *ManaSource) getMaxDrainRate(distSquared float64) float64 {
	maxDistSquared := ms.options.MaxDrainDistance * ms.options.MaxDrainDistance
	if distSquared > maxDistSquared {
		return 0.0
	}
	distRatio := 1.0 - distSquared/maxDistSquared
	return distRatio * distRatio * ms.options.MaxDrainRate
}

func (ms *ManaSource) Think(players []Ent) {
	// Regenerate mana
	for x := range ms.nodes {
		for y := range ms.nodes[x] {
			node := &ms.nodes[x][y]
			for c := range node.Mana {
				maxRecovery := node.MaxMana[c] * node.RegenPerFrame
				scale := (node.MaxMana[c] - node.Mana[c]) / node.MaxMana[c]
				node.Mana[c] += scale * maxRecovery
			}
		}
	}

	// Drain mana.
	for _, player := range players {
		// Find the rectangle that this player can drain from.
		minX := -1
		maxX := -1
		minY := -1
		maxY := -1
		for x := range ms.nodes {
			dx := ms.nodes[x][0].X - player.Pos().X
			if dx >= -ms.options.MaxDrainDistance && minX == -1 {
				minX = x
			}
			if dx <= ms.options.MaxDrainDistance {
				maxX = x
			} else {
				break
			}
		}
		for y := range ms.nodes[0] {
			dy := ms.nodes[0][y].Y - player.Pos().Y
			if dy >= -ms.options.MaxDrainDistance && minY == -1 {
				minY = y
			}
			if dy <= ms.options.MaxDrainDistance {
				maxY = y
			} else {
				break
			}
		}

		// Do the draining.
		for x := minX; x <= maxX; x++ {
			for y := minY; y <= maxY; y++ {
				node := &ms.nodes[x][y]
				maxDrainRate := ms.getMaxDrainRate(player.Pos().Sub(linear.MakeVec2(node.X, node.Y)).Mag2())
				for c := range node.Mana {
					amountScale := node.MaxMana[c] / float64(ms.options.NodeMagnitude)
					node.Mana[c] -= maxDrainRate * amountScale
					if node.Mana[c] < 0 {
						node.Mana[c] = 0
					}
				}
			}
		}
	}
}

func (ms *ManaSource) Draw(gw *GameWindow, dx float64, dy float64) {
	if gw.nodeTextureData == nil {
		gl.Enable(gl.TEXTURE_2D)
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
			gl.Sizei(ms.options.NumNodeCols),
			gl.Sizei(ms.options.NumNodeRows),
			0,
			gl.RGB,
			gl.UNSIGNED_BYTE,
			gl.Pointer(&gw.nodeTextureData[0]))

		gl.ActiveTexture(gl.TEXTURE1)
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

	// This used to be in an else block and I think maybe causes crashed by not
	// being in one, but why?
	for x := range ms.nodes {
		for y, node := range ms.nodes[x] {
			pos := 3 * (y*ms.options.NumNodeCols + x)
			for c := 0; c < 3; c++ {
				color_frac := node.Mana[c] * 1.0 / ms.options.NodeMagnitude
				color_range := float64(ms.options.MaxNodeBrightness - ms.options.MinNodeBrightness)
				gw.nodeTextureData[pos+c] = byte(
					color_frac*color_range + float64(ms.options.MinNodeBrightness))
			}
		}
	}
	gl.Enable(gl.TEXTURE_1D)
	gl.Enable(gl.TEXTURE_2D)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, gw.nodeTextureId)
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		gl.Sizei(ms.options.NumNodeCols),
		gl.Sizei(ms.options.NumNodeRows),
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

	base.EnableShader("nodes")
	base.SetUniformI("nodes", "width", len(ms.nodes))
	base.SetUniformI("nodes", "height", len(ms.nodes[0]))
	base.SetUniformI("nodes", "drains", 1)
	base.SetUniformI("nodes", "tex0", 0)
	base.SetUniformI("nodes", "tex1", 1)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_1D, gw.nodeWarpingTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, gw.nodeTextureId)
	texture.Render(0, dy, dx, -dy)
	base.EnableShader("")
	gl.Disable(gl.TEXTURE_2D)
}
