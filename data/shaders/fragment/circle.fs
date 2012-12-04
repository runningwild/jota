varying vec3 pos;

void main(void) {
  float x = abs(gl_TexCoord[0].x) - 0.5;
  float y = abs(gl_TexCoord[0].y) - 0.5;
  float dist = sqrt(x*x + y*y);
  if (dist <= 0.5 && dist >= 0.45) {
    gl_FragColor = vec4(1.0, 1.0, 0.0, 1.0) * gl_Color;
  } else {
    gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
  }
}

