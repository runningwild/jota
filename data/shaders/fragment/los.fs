uniform float losMaxDist;
uniform float losResolution;
uniform float losMaxPlayers;
uniform int losNumPlayers;
uniform float dx;
uniform float dy;
uniform sampler2D tex0;
uniform vec2 playerPos[2];
void main(void) {
  vec2 pos = vec2(gl_TexCoord[0].x * dx, dy - gl_TexCoord[0].y * dy);

  int i;
  float best = 1.0;
  for (i = 0; i < losNumPlayers; i++) {
    // Only check player 0 right now
    vec2 ray = pos - playerPos[i];
    float dist = length(ray);

    // map the angle from [-pi, pi] to [0, 1].
    float angle = atan(ray.y, ray.x);
    angle = ((angle / 3.1415926535) + 1.0) / 2.0;
    vec2 lookup = vec2(angle, (float(i) + 0.5) / losMaxPlayers);
    vec4 zvalvec = texture2D(tex0, lookup);
    float zval = zvalvec.a * losMaxDist;
    float buffer = 10.0;

    if (dist < zval) {
      gl_FragColor = vec4(0.0, 1.0, 0.0, 0.0);
      return;
    }

    if (zval < dist) {
      float current = (dist - zval) / buffer;
      if (current < best) {
        best = current;
      }
    }
  }
  gl_FragColor = vec4(0.0, 0.0, 0.0, best / 2.0);
}

