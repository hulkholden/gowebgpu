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

  var pos = particle.pos;
  var vel = particle.vel;
  var angle = particle.angle;
  var angularVel = particle.angularVel;

  vel = flock(particle, index);
  angle = select(0, -atan2(vel.x, vel.y), length(vel) > 0);

  // kinematic update
  pos += (vel * params.deltaT);
  angle += (angularVel * params.deltaT);

  // Bounce off the boundary.
  let under = (pos < minBound) & (vel < vec2());
  let over = (pos > maxBound) & (vel > vec2());
  vel = select(vel, -vel * params.boundaryBounceFactor, under | over);
  pos = clamp(pos, minBound, maxBound);

  // Write back
  particlesB.particles[index].pos = pos;
  particlesB.particles[index].vel = vel;
  particlesB.particles[index].angle = angle;
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