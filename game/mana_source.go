// TODO: add Width, Height functions
//       gobEncode

package game

import (
	"fmt"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/linear"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/texture"
	"math/rand"
)

type ManaSourceOptions struct {
	NumSeeds int
	NumNodeRows int
	NumNodeCols int

	BoardLeft float64
	BoardTop float64
	BoardRight float64
	BoardBottom float64

	RegenPerFrame float64
	NodeAmount float64
	MinNodeBrightness int
	MaxNodeBrightness int
}

type node struct {
	X, Y float64
	RegenPerFrame float64
	Amount []float64
	MaxAmount []float64
}

type nodeSeed struct {
	x, y float64
	color int
}

func (dst *node) OverwriteWith(src *node) {
	*dst = *src
	copy(dst.Amount, src.Amount)
	copy(dst.MaxAmount, src.MaxAmount)
}

type ManaSource struct {
	options ManaSourceOptions
	nodes [][]node
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
	for i := range(seeds) {
		seed := &seeds[i]
		seed.x = options.BoardLeft + r.Float64() * (options.BoardRight - options.BoardLeft)
		seed.y = options.BoardTop + r.Float64() * (options.BoardBottom - options.BoardTop)
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
			x := options.BoardLeft + float64(col) / float64(options.NumNodeCols - 1) * (
				options.BoardRight - options.BoardLeft)
			y := options.BoardTop + float64(row) / float64(options.NumNodeRows - 1) * (
				options.BoardBottom - options.BoardTop)

			// all_obstacles[0] corresponds to the outer walls. We do not want to drop mana nodes for
			// being inside there.
			insideObstacle := false
			for i := 1; !insideObstacle && i < len(allObstacles); i++ {
				if vecInsideConvexPoly(linear.Vec2{x, y}, allObstacles[i]) {
					insideObstacle = true
				}
		  }
			if (insideObstacle) {
			  continue
			}

			maxWeightByColor := []float64{0.0, 0.0, 0.0}
			for _, seed := range seeds {
				c := seed.color
				dx := x - seed.x
				dy := y - seed.y
				dist_sq := dx * dx + dy * dy
				weight := 1 / (dist_sq + 1.0)
				if weight > maxWeightByColor[c] {
					maxWeightByColor[c] = weight
				}
			}

			normalizeWeights(options.NodeAmount, maxWeightByColor)
			weightsCopy := make([]float64, len(maxWeightByColor))
			copy(weightsCopy, maxWeightByColor)

			ms.nodes[col][row] = node{
				X: x,
				Y: y,
				RegenPerFrame: options.RegenPerFrame,
				Amount: maxWeightByColor,
				MaxAmount: weightsCopy,
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
			dstN := &dst.nodes[x][y]
			dstN.Amount = make([]float64, len(srcN.Amount))
			dstN.MaxAmount = make([]float64, len(srcN.MaxAmount))
			dstN.OverwriteWith(&srcN)
		}
	}
	dst.options = src.options

	return dst
}

func (dst *ManaSource) OverwriteWith(src *ManaSource) {
	for x := range src.nodes {
		for y, srcN := range src.nodes[x] {
			dstN := &src.nodes[x][y]
			dstN.OverwriteWith(&srcN)
		}
	}
	dst.options = src.options
}

func (ms *ManaSource) Think() {
	for _, nodeList := range ms.nodes {
		for _, node := range nodeList {
			for c := range node.Amount {
				node.Amount[c] += node.MaxAmount[c] * node.RegenPerFrame
				if node.Amount[c] > node.MaxAmount[c] {
					node.Amount[c] = node.MaxAmount[c]
				}
			}
		}
	}
}

func (ms *ManaSource) Draw(gw *GameWindow, dx float64, dy float64) {
	// Nodes
	if gw.nodeTextureData == nil {
		gw.nodeTextureData = make([]byte, len(ms.nodes) * len(ms.nodes[0]) * 4)
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
			gl.RGBA,
			gl.Sizei(len(ms.nodes)),
			gl.Sizei(len(ms.nodes[0])),
			0,
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Pointer(&gw.nodeTextureData[0]))
	} else {
		// TODO: Switch to RGB? Move outside of else?
		for x := range ms.nodes {
			for y, node := range ms.nodes[x] {
				pos := 4 * (y * len(ms.nodes) + x)
				for c := 0; c < 3; c++ {
					if len(node.Amount) > c {
						color_frac := node.Amount[c] * 1.0 / ms.options.NodeAmount
						color_range := float64(ms.options.MaxNodeBrightness - ms.options.MinNodeBrightness)
						gw.nodeTextureData[pos + c] = byte(
							color_frac * color_range + float64(ms.options.MinNodeBrightness))
					}
					gw.nodeTextureData[pos + 3] = 255
				}
			}
		}
		gl.Enable(gl.TEXTURE_2D)
		gl.BindTexture(gl.TEXTURE_2D, gw.nodeTextureId)
	}
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		gl.Sizei(len(ms.nodes)),
		gl.Sizei(len(ms.nodes[0])),
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Pointer(&gw.nodeTextureData[0]))

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	base.EnableShader("nodes")
	base.SetUniformI("nodes", "width", len(ms.nodes))
	base.SetUniformI("nodes", "height", len(ms.nodes[0]))
	texture.Render(0, dy, dx, -dy)
	base.EnableShader("")
	gl.Disable(gl.TEXTURE_2D)
}
