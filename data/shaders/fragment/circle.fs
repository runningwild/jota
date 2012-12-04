varying vec3 pos;

// progress ranges from 0 to 2.  [0,1] is a windup, [1,2] is a cooldown.
uniform float progress;

void main(void) {
  float x = abs(gl_TexCoord[0].x) - 0.5;
  float y = abs(gl_TexCoord[0].y) - 0.5;
  float dist = sqrt(x*x + y*y);
  if (dist <= 0.5 && dist >= 0.45) {
    if (progress <= 1.0) {
      gl_FragColor = vec4(1.0, 1.0, 0.0, 1.0) * gl_Color;
    } else {
      gl_FragColor = vec4(0.0, 0.0, 1.0, 2.0 - progress) * gl_Color;
    }
  } else {
    if (dist < 0.45) {
      if (progress <= 1.0) {
        gl_FragColor = vec4(1.0, 0.0, 0.0, progress);
      } else {
        gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
      }
    } else {
      gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
    }
  }
}

