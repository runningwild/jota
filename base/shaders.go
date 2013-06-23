package base

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/linear"
	"io/ioutil"
	"path/filepath"
	"unsafe"
)

type Shader struct {
	Defname string
	*shaderDef
}

type shaderDef struct {
	// Name of this texture as it appears in the editor, should be unique among
	// all WallTextures
	Name string

	// Paths to the vertex and fragment shaders
	Vertex_path   string
	Fragment_path string
}

// Mappings from vertex shader name, fragment shader name, and shader program
// name to their respective opengl ids.
var vertex_shaders map[string]gl.Uint
var fragment_shaders map[string]gl.Uint
var shader_progs map[string]gl.Uint

var warned_names map[string]bool

func EnableShader(name string) {
	prog_obj, ok := shader_progs[name]
	if ok {
		gl.UseProgram(prog_obj)
	} else {
		gl.UseProgram(0)
		if name != "" && !warned_names[name] {
			Warn().Printf("Tried to use unknown shader '%s'", name)
			warned_names[name] = true
		}
	}
}

func SetUniformI(shader, variable string, n int) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Warn().Printf("Tried to set a uniform in an unknown shader '%s'", shader)
			warned_names[shader] = true
		}
		return
	}
	bvariable := []byte(fmt.Sprintf("%s\x00", variable))
	loc := gl.GetUniformLocation(prog, (*gl.Char)(unsafe.Pointer(&bvariable[0])))
	gl.Uniform1i(loc, gl.Int(n))
}

func SetUniformF(shader, variable string, f float32) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Warn().Printf("Tried to set a uniform in an unknown shader '%s'", shader)
			warned_names[shader] = true
		}
		return
	}
	bvariable := []byte(fmt.Sprintf("%s\x00", variable))
	loc := gl.GetUniformLocation(prog, (*gl.Char)(unsafe.Pointer(&bvariable[0])))
	gl.Uniform1f(loc, gl.Float(f))
}

func SetUniformV2(shader, variable string, v linear.Vec2) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Warn().Printf("Tried to set a uniform in an unknown shader '%s'", shader)
			warned_names[shader] = true
		}
		return
	}
	bvariable := []byte(fmt.Sprintf("%s\x00", variable))
	loc := gl.GetUniformLocation(prog, (*gl.Char)(unsafe.Pointer(&bvariable[0])))
	gl.Uniform2f(loc, gl.Float(v.X), gl.Float(v.Y))
}

func SetUniformV2Array(shader, variable string, vs []linear.Vec2) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Warn().Printf("Tried to set a uniform in an unknown shader '%s'", shader)
			warned_names[shader] = true
		}
		return
	}
	bvariable := []byte(fmt.Sprintf("%s\x00", variable))
	loc := gl.GetUniformLocation(prog, (*gl.Char)(unsafe.Pointer(&bvariable[0])))
	var fs []gl.Float
	for i := range vs {
		fs = append(fs, gl.Float(vs[i].X))
		fs = append(fs, gl.Float(vs[i].Y))
	}
	gl.Uniform2fv(loc, gl.Sizei(len(vs)), (*gl.Float)(unsafe.Pointer(&fs[0])))
}

func InitShaders() {
	render.Queue(func() {
		vertex_shaders = make(map[string]gl.Uint)
		fragment_shaders = make(map[string]gl.Uint)
		shader_progs = make(map[string]gl.Uint)
		warned_names = make(map[string]bool)
		RemoveRegistry("shaders")
		RegisterRegistry("shaders", make(map[string]*shaderDef))
		RegisterAllObjectsInDir("shaders", filepath.Join(GetDataDir(), "shaders"), ".json", "json")
		names := GetAllNamesInRegistry("shaders")
		for _, name := range names {
			// Load the shader files
			shader := Shader{Defname: name}
			GetObject("shaders", &shader)
			vdata, err := ioutil.ReadFile(filepath.Join(GetDataDir(), shader.Vertex_path))
			if err != nil {
				Error().Printf("Unable to load vertex shader '%s': %v", shader.Vertex_path, err)
				continue
			}
			fdata, err := ioutil.ReadFile(filepath.Join(GetDataDir(), shader.Fragment_path))
			if err != nil {
				Error().Printf("Unable to load fragment shader '%s': %v", shader.Fragment_path, err)
				continue
			}

			// Create the vertex shader
			vertex_id, ok := vertex_shaders[shader.Vertex_path]
			if !ok {
				vertex_id = gl.CreateShader(gl.VERTEX_SHADER)
				pointer := &vdata[0]
				length := gl.Int(len(vdata))
				gl.ShaderSource(vertex_id, 1, (**gl.Char)(unsafe.Pointer(&pointer)), &length)
				gl.CompileShader(vertex_id)
				var param gl.Int
				gl.GetShaderiv(vertex_id, gl.COMPILE_STATUS, &param)
				if param == 0 {
					Error().Printf("Failed to compile vertex shader '%s': %v", shader.Vertex_path, param)
					continue
				}
			}

			// Create the fragment shader
			fragment_id, ok := fragment_shaders[shader.Fragment_path]
			if !ok {
				fragment_id = gl.CreateShader(gl.FRAGMENT_SHADER)
				pointer := &fdata[0]
				length := gl.Int(len(fdata))
				gl.ShaderSource(fragment_id, 1, (**gl.Char)(unsafe.Pointer(&pointer)), &length)
				gl.CompileShader(fragment_id)
				var param gl.Int
				gl.GetShaderiv(fragment_id, gl.COMPILE_STATUS, &param)
				if param == 0 {
					Error().Printf("Failed to compile fragment shader '%s': %v", shader.Fragment_path, param)
					continue
				}
			}

			// shader successfully compiled - now link
			program_id := gl.CreateProgram()
			gl.AttachShader(program_id, vertex_id)
			gl.AttachShader(program_id, fragment_id)
			gl.LinkProgram(program_id)
			var param gl.Int
			gl.GetProgramiv(program_id, gl.LINK_STATUS, &param)
			if param == 0 {
				Error().Printf("Failed to link shader '%s': %v", shader.Name, param)
				continue
			}

			vertex_shaders[shader.Vertex_path] = vertex_id
			fragment_shaders[shader.Fragment_path] = fragment_id
			shader_progs[shader.Name] = program_id
		}
	})
}
