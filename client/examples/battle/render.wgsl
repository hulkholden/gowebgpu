struct VertexInput {
  @location(0) particlePos : vec2<f32>,
  @location(1) particleAngle: f32,
  @location(2) particleMetadata : u32,
  @location(3) particleCol : u32,
  @location(4) pos : vec2<f32>,  
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

@vertex
fn vertex_main(in : VertexInput) -> VertexOutput {
  let c = cos(in.particleAngle);
  let s = sin(in.particleAngle);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  let pos = (in.particlePos + (in.pos * transform)) / worldScale;

  var output : VertexOutput;
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
