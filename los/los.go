package los

import (
	"bytes"
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"math"
	"sync"
)

const Resolution = 4096

var bufferCache []buffer
var bufferMutex sync.Mutex
var bufferCount int

func newBuffer() buffer {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	if len(bufferCache) == 0 {
		increaseBuffer(bufferCount)
		bufferCount += len(bufferCache)
	}
	ret := bufferCache[len(bufferCache)-1]
	bufferCache = bufferCache[0 : len(bufferCache)-1]
	return ret
}
func deleteBuffer(b buffer) {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	bufferCache = append(bufferCache, b)
}
func increaseBuffer(n int) {
	for i := 0; i < n; i++ {
		bufferCache = append(bufferCache, buffer{
			ZBuffer: make([]float32, Resolution),
			SBuffer: make([]string, Resolution),
		})
	}
}
func init() {
	increaseBuffer(1)
	bufferCount = len(bufferCache)
}

type buffer struct {
	ZBuffer []float32
	SBuffer []string
}

type losInternal struct {
	Buffer  buffer
	Pos     linear.Vec2
	Horizon float32
}

var rays []linear.Seg2

func init() {
	rays = make([]linear.Seg2, Resolution)
	for i := range rays {
		rays[i] = linear.Seg2{
			linear.Vec2{0, 0},
			(linear.Vec2{1, 0}).Rotate(2 * math.Pi * (float64(i)/Resolution - 0.5)),
		}
	}
}

type Los struct {
	in losInternal
}

func Make(Horizon float64) *Los {
	var l Los
	l.in.Horizon = float32(Horizon * Horizon)
	l.in.Buffer = newBuffer()
	for i := range l.in.Buffer.ZBuffer {
		l.in.Buffer.ZBuffer[i] = l.in.Horizon
	}
	for i := range l.in.Buffer.SBuffer {
		l.in.Buffer.SBuffer[i] = ""
	}
	return &l
}
func (l *Los) ReleaseResources() {
	deleteBuffer(l.in.Buffer)
}
func (l *Los) Copy() *Los {
	var l2 Los
	l2.in.Buffer = newBuffer()
	copy(l2.in.Buffer.ZBuffer, l.in.Buffer.ZBuffer)
	copy(l2.in.Buffer.SBuffer, l.in.Buffer.SBuffer)
	l2.in.Horizon = l.in.Horizon
	l2.in.Pos = l.in.Pos
	return &l2
}
func (l *Los) WriteDepthBuffer(dst []uint32, maxDist float32) {
	for i := range dst {
		dst[i] = uint32(math.Sqrt(float64(l.in.Buffer.ZBuffer[i])) / float64(maxDist) * (1<<32 - 1))
	}
}
func (l *Los) CountSource(source string) float64 {
	count := 0.0
	for _, v := range l.in.Buffer.SBuffer {
		if v == source {
			count += 1.0
		}
	}
	return count / float64(len(l.in.Buffer.SBuffer))
}
func (l *Los) Reset(Pos linear.Vec2) {
	l.in.Pos = Pos
	for i := range l.in.Buffer.ZBuffer {
		l.in.Buffer.ZBuffer[i] = l.in.Horizon
	}
	for i := range l.in.Buffer.SBuffer {
		l.in.Buffer.SBuffer[i] = ""
	}
}
func (l *Los) DrawSeg(seg linear.Seg2, source string) {
	seg.P = seg.P.Sub(l.in.Pos)
	seg.Q = seg.Q.Sub(l.in.Pos)
	wrap := len(l.in.Buffer.ZBuffer)
	a1 := math.Atan2(seg.P.Y, seg.P.X)
	a2 := math.Atan2(seg.Q.Y, seg.Q.X)
	if a1 > a2 {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	if a2-a1 > math.Pi {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	start := int(((a1 / (2 * math.Pi)) + 0.5) * float64(len(l.in.Buffer.ZBuffer)))
	end := int(((a2 / (2 * math.Pi)) + 0.5) * float64(len(l.in.Buffer.ZBuffer)))

	for i := start % wrap; i != end%wrap; i = (i + 1) % wrap {
		dist2 := float32(rays[i].Isect(seg).Mag2())
		// dist = rays[i].Isect(seg).Mag2()

		if dist2 < l.in.Buffer.ZBuffer[i] {
			l.in.Buffer.ZBuffer[i] = dist2
			l.in.Buffer.SBuffer[i] = source
		}
	}
}

// Returns the fraction of the segment that was visible
func (l *Los) TestSeg(seg linear.Seg2) float64 {
	seg.P = seg.P.Sub(l.in.Pos)
	seg.Q = seg.Q.Sub(l.in.Pos)
	wrap := len(l.in.Buffer.ZBuffer)
	a1 := math.Atan2(seg.P.Y, seg.P.X)
	a2 := math.Atan2(seg.Q.Y, seg.Q.X)
	if a1 > a2 {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	if a2-a1 > math.Pi {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	start := int(((a1 / (2 * math.Pi)) + 0.5) * float64(len(l.in.Buffer.ZBuffer)))
	end := int(((a2 / (2 * math.Pi)) + 0.5) * float64(len(l.in.Buffer.ZBuffer)))

	count := 0.0
	visible := 0.0
	for i := start % wrap; i != end%wrap; i = (i + 1) % wrap {
		dist2 := float32(rays[i].Isect(seg).Mag2())
		if dist2 < l.in.Buffer.ZBuffer[i] {
			visible += 1.0
		}
		count += 1.0
	}
	return visible / count
}

func (l *Los) Render() {
	var v0, v1 linear.Vec2
	gl.Begin(gl.TRIANGLES)
	v1 = (linear.Vec2{-1, 0}).Scale(math.Sqrt(float64(l.in.Buffer.ZBuffer[0]))).Add(l.in.Pos)
	for i := 1; i <= len(l.in.Buffer.ZBuffer); i++ {
		dist := math.Sqrt(float64(l.in.Buffer.ZBuffer[i%len(l.in.Buffer.ZBuffer)]))
		angle := 2 * math.Pi * (float64(i%len(l.in.Buffer.ZBuffer))/float64(len(l.in.Buffer.ZBuffer)) - 0.5)
		if dist <= 0.0 {
			continue
		}
		v0 = v1
		gl.Color4d(gl.Double(1.0-dist/math.Sqrt(float64(l.in.Horizon))), 1.0, 0.0, 1.0)
		v1 = (linear.Vec2{1, 0}).Rotate(angle).Scale(dist).Add(l.in.Pos)
		gl.Vertex2d(gl.Double(l.in.Pos.X), gl.Double(l.in.Pos.Y))
		gl.Vertex2d(gl.Double(v0.X), gl.Double(v0.Y))
		gl.Vertex2d(gl.Double(v1.X), gl.Double(v1.Y))
	}
	gl.End()
}

func (l Los) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(l.in)
	return buf.Bytes(), err
}

func (l *Los) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&l.in)
	return err
}
