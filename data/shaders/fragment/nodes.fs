uniform int width;
uniform int height;
uniform sampler2D sampler;
//varying vec3 pos;

float gridAttenuation1d(float f) {
  f = mod(f, 1.0);
  float band = 0.1;
  float blur = 0.09;
  if (f < 0.5) {
    f = smoothstep(band + blur, band - blur, f);
  } else {
    f = smoothstep(1.0 - band - blur, 1.0 - band + blur, f);
  }
  return f;
}

float gridAttenuation(vec2 pos) {
  float attx = gridAttenuation1d(pos.x * float(width));
  float atty = gridAttenuation1d(pos.y * float(height));
  return 1.0 - (1.0 - attx) * (1.0 - atty);
}

vec4 alphaTransform(vec4 color) {
  float m = max(color.r, max(color.g, color.b));
  if (m == 0.0) {
    return vec4(0.0, 0.0, 0.0, 0.0);
  }
  color = color / m;
  color.a = m;
  return color;
}

void main(void) {
  vec2 pull = vec2(0.3, -0.5);
  vec2 tpos = gl_TexCoord[0].xy;
  vec2 diff = tpos - pull;
  float d = 10.0 * length(diff);
  float d2 = d * d;
  tpos = tpos + diff / (d2 + 1.0);
  vec4 color = texture2D(sampler, tpos);
  vec4 grid_color = color * gridAttenuation(tpos);
  grid_color = alphaTransform(grid_color);
  vec4 grey = vec4(1.0, 1.0, 1.0, 0.1) * gridAttenuation(tpos + vec2(0.5, 0.5));
  gl_FragColor = grid_color + grey * (1.0 - pow(grid_color.a, 0.1));
  //gl_FragColor = grid_color;
}

