struct Particles {
  particles : array<Particle>,
}
@binding(0) @group(0) var<uniform> params : SimParams;
@binding(1) @group(0) var<storage, read> particlesA : Particles;
@binding(2) @group(0) var<storage, read_write> particlesB : Particles;

// https://github.com/austinEng/Project6-Vulkan-Flocking/blob/master/data/shaders/computeparticles/particle.comp
@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  var index = GlobalInvocationID.x;

  var vPos = particlesA.particles[index].pos;
  var vVel = particlesA.particles[index].vel;
  var cMass = vec2(0.0);
  var cVel = vec2(0.0);
  var colVel = vec2(0.0);
  var cMassCount = 0u;
  var cVelCount = 0u;

  let myTeam = particlesA.particles[index].team;

  for (var i = 0u; i < arrayLength(&particlesA.particles); i++) {
    if (i == index) {
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
  vVel = normalize(vVel) * clamp(length(vVel), 0.0, 0.1);
  // kinematic update
  vPos = vPos + (vVel * params.deltaT);
  // Wrap around boundary
  if (vPos.x < -1.0) {
    vPos.x += 2.0;
  }
  if (vPos.x > 1.0) {
    vPos.x -= -2.0;
  }
  if (vPos.y < -1.0) {
    vPos.y += 2.0;
  }
  if (vPos.y > 1.0) {
    vPos.y -= -2.0;
  }

  // Write back
  particlesB.particles[index].pos = vPos;
  particlesB.particles[index].vel = vVel;
}
