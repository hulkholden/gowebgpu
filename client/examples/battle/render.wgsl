struct VertexInput {
  @location(0) particlePos : vec2<f32>,
  @location(1) particleAngle: f32,
  @location(2) particleMetadata : u32,
  @location(3) particleCol : u32,
  @builtin(vertex_index) vertexIndex : u32,
}

struct VertexOutput {
  @builtin(position) position : vec4<f32>,
  @location(0) color : vec4<f32>,
  @location(1) @interpolate(flat) metadata : u32,
}

// TODO: provide this as a matrix.
const worldScale = 1000.0;

// TODO: dedupe.
const bodyTypeNone = 0u;
const bodyTypeShip = 1u;
const bodyTypeMissile = 2u;

// Ship: 1 triangle (3 vertices).
const shipVerts = array<vec2<f32>, 3>(
  vec2<f32>(-5.0, -10.0), vec2<f32>(5.0, -10.0), vec2<f32>(0.0, 10.0),
);

// Missile: pointed tip + rectangular body (3 triangles, 9 vertices).
const missileVerts = array<vec2<f32>, 9>(
  vec2<f32>(0.0, 12.0),  vec2<f32>(-1.5, 4.0),  vec2<f32>(1.5, 4.0),   // nose
  vec2<f32>(-1.5, 4.0),  vec2<f32>(-1.5, -12.0), vec2<f32>(1.5, 4.0),  // body left
  vec2<f32>(1.5, 4.0),   vec2<f32>(-1.5, -12.0), vec2<f32>(1.5, -12.0) // body right
);

fn renderParticle(in : VertexInput, expectedType : u32, localPos : vec2<f32>) -> VertexOutput {
  var output : VertexOutput;
  let bodyType = particleType(in.particleMetadata);
  if (bodyType != expectedType) {
    // Wrong type for this draw call — degenerate position.
    output.position = vec4(0.0, 0.0, 0.0, 1.0);
    output.color = vec4(0.0);
    output.metadata = in.particleMetadata;
    return output;
  }

  let c = cos(in.particleAngle);
  let s = sin(in.particleAngle);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  let pos = (in.particlePos + (localPos * transform)) / worldScale;

  output.position = vec4(pos, 0.0, 1.0);
  // TODO: why doesn't unpack4xU8 work?
  output.color = vec4(
    f32((in.particleCol >> 16) & 0xff) / 255.0,
    f32((in.particleCol >> 8) & 0xff) / 255.0,
    f32((in.particleCol >> 0) & 0xff) / 255.0,
     1.0);
  output.metadata = in.particleMetadata;
  return output;
}

@vertex
fn vertex_main_ship(in : VertexInput) -> VertexOutput {
  return renderParticle(in, bodyTypeShip, shipVerts[in.vertexIndex]);
}

@vertex
fn vertex_main_missile(in : VertexInput) -> VertexOutput {
  var output = renderParticle(in, bodyTypeMissile, missileVerts[in.vertexIndex]);
  output.color = vec4(1.0, 1.0, 1.0, 1.0);
  return output;
}

@fragment
fn fragment_main(attrs : VertexOutput) -> @location(0) vec4<f32> {
  if (particleType(attrs.metadata) == bodyTypeNone) {
    //return vec4(245/255.0, 141/255.0, 66/255.0, 1);
    discard;
  }
  return attrs.color;
}

fn particleType(metadata : u32) -> u32 {
  return (metadata >> 8) & 0xff;
}
