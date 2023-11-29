struct Particles {
  particles : array<Particle>,
}
@binding(0) @group(0) var<uniform> params : SimParams;
@binding(1) @group(0) var<storage, read> particlesA : Particles;
@binding(2) @group(0) var<storage, read_write> particlesB : Particles;

const minBound = vec2f(-1.0);
const maxBound = vec2f(1.0);

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;
  let particle = particlesA.particles[index];

  var vPos = particle.pos;
  var vVel = particle.vel;

  vVel = flock(particle, index);

  // kinematic update
  vPos = vPos + (vVel * params.deltaT);

  // Bounce off the boundary.
  let under = (vPos < minBound) & (vVel < vec2());
  let over = (vPos > maxBound) & (vVel > vec2());
  vVel = select(vVel, -vVel * params.boundaryBounceFactor, under | over);
  vPos = clamp(vPos, minBound, maxBound);

  // Write back
  particlesB.particles[index].pos = vPos;
  particlesB.particles[index].vel = vVel;
}

fn flock(particle : Particle, selfIdx : u32) -> vec2f {
  var vPos = particle.pos;
  var vVel = particle.vel;
  var cMass = vec2(0.0);
  var cVel = vec2(0.0);
  var colVel = vec2(0.0);
  var cMassCount = 0u;
  var cVelCount = 0u;

  let myTeam = particle.team;

  for (var i = 0u; i < arrayLength(&particlesA.particles); i++) {
    if (i == selfIdx) {
      continue;
    }
    let same = particlesA.particles[i].team == myTeam;

    let pos = particlesA.particles[i].pos.xy;
    let vel = particlesA.particles[i].vel.xy;
    let dPos = pos - vPos;
    let dist = length(dPos);
    if (dist < params.avoidDistance) {
      colVel -= dPos;
    }
    if (same) {
      if (dist < params.cMassDistance) {
        cMass += pos;
        cMassCount++;
      }
      if (dist < params.cVelDistance) {
        cVel += vel;
        cVelCount++;
      }
    }
  }
  if (cMassCount > 0) {
    cMass = (cMass / vec2(f32(cMassCount))) - vPos;
  }
  if (cVelCount > 0) {
    cVel /= f32(cVelCount);
  }
  vVel += (colVel * params.avoidScale) + (cMass * params.cMassScale) + (cVel * params.cVelScale);

  // clamp velocity for a more pleasing simulation
  return normalize(vVel) * clamp(length(vVel), 0.0, 0.1);
}