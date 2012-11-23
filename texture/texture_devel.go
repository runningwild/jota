// +build !release

// In devel version we want it to be clear that a texture isn't loaded or has
// failed to load, so we use a bright pink texture in both of those cases.
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
  pink := []byte{255, 0, 255, 255}
  gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.BYTE, gl.Pointer(&pink[0]))
}
