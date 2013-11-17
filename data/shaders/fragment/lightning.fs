varying vec3 pos;
uniform vec2 dir;
uniform vec2 bolt_root;
uniform float bolt_thickness;
uniform float rand_offset;

vec3 mod289(vec3 x) {
  return x - floor(x * (1.0 / 289.0)) * 289.0;
}

vec2 mod289(vec2 x) {
  return x - floor(x * (1.0 / 289.0)) * 289.0;
}

vec3 permute(vec3 x) {
  return mod289(((x*34.0)+1.0)*x);
}

float snoise(vec2 v) {
  const vec4 C = vec4(0.211324865405187,  // (3.0-sqrt(3.0))/6.0
                      0.366025403784439,  // 0.5*(sqrt(3.0)-1.0)
                     -0.577350269189626,  // -1.0 + 2.0 * C.x
                      0.024390243902439); // 1.0 / 41.0
// First corner
  vec2 i  = floor(v + dot(v, C.yy) );
  vec2 x0 = v -   i + dot(i, C.xx);

// Other corners
  vec2 i1;
  //i1.x = step( x0.y, x0.x ); // x0.x > x0.y ? 1.0 : 0.0
  //i1.y = 1.0 - i1.x;
  i1 = (x0.x > x0.y) ? vec2(1.0, 0.0) : vec2(0.0, 1.0);
  // x0 = x0 - 0.0 + 0.0 * C.xx ;
  // x1 = x0 - i1 + 1.0 * C.xx ;
  // x2 = x0 - 1.0 + 2.0 * C.xx ;
  vec4 x12 = x0.xyxy + C.xxzz;
  x12.xy -= i1;

// Permutations
  i = mod289(i); // Avoid truncation effects in permutation
  vec3 p = permute( permute( i.y + vec3(0.0, i1.y, 1.0 ))
    + i.x + vec3(0.0, i1.x, 1.0 ));

  vec3 m = max(0.5 - vec3(dot(x0,x0), dot(x12.xy,x12.xy), dot(x12.zw,x12.zw)), 0.0);
  m = m*m ;
  m = m*m ;

// Gradients: 41 points uniformly over a line, mapped onto a diamond.
// The ring size 17*17 = 289 is close to a multiple of 41 (41*7 = 287)

  vec3 x = 2.0 * fract(p * C.www) - 1.0;
  vec3 h = abs(x) - 0.5;
  vec3 ox = floor(x + 0.5);
  vec3 a0 = x - ox;

// Normalise gradients implicitly by scaling m
// Approximation of: m *= inversesqrt( a0*a0 + h*h );
  m *= 1.79284291400159 - 0.85373472095314 * ( a0*a0 + h*h );

// Compute final noise value at P
  vec3 g;
  g.x  = a0.x  * x0.x  + h.x  * x0.y;
  g.yz = a0.yz * x12.xz + h.yz * x12.yw;
  return 130.0 * dot(m, g);
}

float distFromPointToLineSegment(vec2 p, vec2 v0, vec2 v1) {
  vec2 n = v1 - v0;
  n = normalize(n);
  if ((dot(n, p - v0) < 0.0) != (dot(n, v1 - p) < 0.0)) {
    float d0 = length(p - v0);
    float d1 = length(v1 - p);
    if (d0 < d1) {
      return d0;
    }
    return d1;
  }
  vec2 a = v0;
  return length((a - p) - (dot(a - p, n) * n));
}

void main(void) {
  vec2 bolt = vec2(
      dot(pos.xy - bolt_root, vec2(dir.y, -dir.x)),
      dot(pos.xy - bolt_root, dir));
  bolt = bolt / 45.0;
  bolt.x = bolt.x * 2.0;

  float bolt_index = floor(bolt.y);
  float bolt_frac = bolt.y - bolt_index;
  vec2 v[4];
  v[0] = vec2(snoise(vec2(bolt_index - 1.0, rand_offset * 100.0)), bolt_index - 1.0);
  v[1] = vec2(snoise(vec2(bolt_index - 0.0, rand_offset * 100.0)), bolt_index - 0.0);
  v[2] = vec2(snoise(vec2(bolt_index + 1.0, rand_offset * 100.0)), bolt_index + 1.0);
  v[3] = vec2(snoise(vec2(bolt_index + 2.0, rand_offset * 100.0)), bolt_index + 2.0);


  float val = 0.0;
  for (int i = 0; i < 3; i++) {
    float d = distFromPointToLineSegment(bolt, v[i], v[i+1]);
    d = bolt_thickness - sqrt(d);
    if (d < 0.0) {
      d = 0.0;
    }
    d = d / bolt_thickness;
    val = val + d * d;
  }
  val = clamp(val, 0.0, 1.0);
  gl_FragColor = vec4(val, val, val, val) * gl_Color;
}
