uniform int width;
uniform int height;
uniform int drains;
uniform sampler2D tex0;
uniform sampler1D tex1;
//varying vec3 pos;

float gridAttenuation1d(float f) {
  f = mod(f, 1.0);
  float band = 0.12;
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

float attAtPos(vec2 pos) {
  pos = mod(pos, 1.0);
  if (pos.x < 0.0) {
    pos.x = -pos.x;
  }
  if (pos.y < 0.0) {
    pos.y = -pos.y;
  }
  if (pos.x > 0.5) {
    pos.x = 1.0 - pos.x;
  }
  if (pos.y > 0.5) {
    pos.y = 1.0 - pos.y;
  }
  return sqrt(pos.x * pos.y);
}

void main(void) {
  vec4 drain_data = texture1D(tex1, 0.05);
  drain_data.y = -drain_data.y;
  vec2 tpos = gl_TexCoord[0].xy;
  vec2 pull = drain_data.xy;
  vec2 diff = tpos - pull;
  vec4 color = texture2D(tex0, tpos);
  vec4 grid_color = color * gridAttenuation(tpos);
  grid_color = alphaTransform(grid_color);
  vec4 grey = vec4(1.0, 1.0, 1.0, 0.1) * gridAttenuation(tpos + vec2(0.5, 0.5));
  gl_FragColor = grid_color + grey * (1.0 - pow(grid_color.a, 0.2));
  //gl_FragColor = grid_color;
}

