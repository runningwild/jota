// +build release

// In release version we don't want to show a bright pink texture when we're
// waiting to load or have an error, so we use a completely transparent
// texture instead.
package texture

import (
  gl "github.com/chsc/gogl/gl21"
)

var error_texture gl.Uint

func makeErrorTexture() {
  gl.Enable(gl.TEXTURE_2D)
  gl.GenTextures(1, (*gl.Uint)(&error_texture))
  gl.BindTexture(gl.TEXTURE_2D, error_texture)
  gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
  transparent := []byte{0, 0, 0, 0}
  gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.BYTE, gl.Pointer(&transparent[0]))
}
