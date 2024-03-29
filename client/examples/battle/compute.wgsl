
@binding(0) @group(0) var<uniform> params : SimParams;
// TODO: is there any performance difference binding these as read only when
// used in the different entry points?
@binding(1) @group(0) var<storage, read_write> gBodies : array<Body>;
@binding(2) @group(0) var<storage, read_write> gParticles : array<Particle>;
@binding(3) @group(0) var<storage, read_write> gShips : array<Ship>;
@binding(4) @group(0) var<storage, read_write> gMissiles : array<Missile>;
@binding(5) @group(0) var<storage, read_write> gAccelerations : array<Acceleration>;
@binding(6) @group(0) var<storage, read_write> gContacts : ContactsContainer;
@binding(7) @group(0) var<storage, read_write> gFreeIDs : FreeIDsContainer;

const pi = 3.14159265359;
const twoPi = 2 * pi;

const proNavGain = 3.0;

const bodyTypeNone = 0u;
const bodyTypeShip = 1u;
const bodyTypeMissile = 2u;

const particleFlagHit = 1u;

fn bodySub(a : Body, b : Body) -> Body {
  return Body(a.pos - b.pos, a.vel - b.vel, angleDiff(a.angle, b.angle), a.angularVel - b.angularVel);
}

fn bodyRotate(a : Body, angle : f32) -> Body {
  let c = cos(angle);
  let s = sin(angle);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  return Body(a.pos * transform, a.vel * transform, a.angle + angle, a.angularVel);
}

fn rotVec(v : vec2f, a : f32) -> vec2f {
  let c = cos(a);
  let s = sin(a);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  return v * transform;
}

fn angleOf(v : vec2f, def : f32) -> f32 {
  return select(def, atan2(-v.x, v.y), length(v) > 0);
}

fn angleDiff(a : f32, b : f32) -> f32 {
  return normalizeAngle(a - b);
}

fn normalizeAngle(a : f32) -> f32 {
  var n = modReplacement(a + pi, twoPi);
  if (n < 0) {
    n += twoPi;
  }
  return n - pi;
}

// modReplacement returns the floating point remainder of x/y.
// TOOD: check whathappens if x is < 0.
fn modReplacement(x : f32, y : f32) -> f32 {
  return x - (y * floor(x/y));
}

@compute @workgroup_size(64)
fn computeAcceleration(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {

  let index = GlobalInvocationID.x;
  var acc = Acceleration();
  switch particleType(index) {
    case bodyTypeNone: {
    }
    case bodyTypeShip: {
      acc = flock(index);
    }
    case bodyTypeMissile: {
      let missile = gMissiles[index];
      if (missile.targetIdx >= 0) {
        acc = updateMissile(index, u32(missile.targetIdx));
      }
    }
    default: {
    }
  }
  gAccelerations[index] = acc;
}

@compute @workgroup_size(64)
fn applyAcceleration(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;
  var body = gBodies[index];
  let acc = gAccelerations[index];

  body.vel += acc.linearAcc * params.deltaT;
  body.pos += body.vel * params.deltaT;

  body.angularVel += acc.angularAcc * params.deltaT;
  body.angle = normalizeAngle(body.angle + body.angularVel * params.deltaT);

  if (particleType(index) == bodyTypeShip) {
    // Bounce off the boundary.
    let under = (body.pos < params.minBound) & (body.vel < vec2());
    let over = (body.pos > params.maxBound) & (body.vel > vec2());
    body.vel = select(body.vel, -body.vel * params.boundaryBounceFactor, under | over);
    body.pos = clamp(body.pos, params.minBound, params.maxBound);

    // clamp velocity for a more pleasing simulation
    body.vel = normalize(body.vel) * clamp(length(body.vel), 0.0, params.maxShipSpeed);
  }

  gBodies[index] = body;
}

@compute @workgroup_size(64)
fn computeCollisions(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;

  if (particleType(index) == bodyTypeMissile) {
    let missile = gMissiles[index];
    let targetIdx = missile.targetIdx;
    if (targetIdx >= 0) {
      let body = gBodies[index];
      let targetB = gBodies[targetIdx];
      let collide = distance(body.pos, targetB.pos) < params.missileCollisionDist;
      if (collide) {
        addContact(index, u32(targetIdx));
      }
    }
  }
}

fn addContact(aIdx : u32, bIdx : u32) {
  let contactIdx = atomicAdd(&gContacts.count, 1);
  if (contactIdx < arrayLength(&gContacts.elements)) {
    gContacts.elements[contactIdx] = Contact(aIdx, bIdx);
  }
}

// This is not parallelised but the number of contacts should be low each frame.
@compute @workgroup_size(1)
fn applyCollisions(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let contactCount = min(atomicLoad(&gContacts.count), u32(arrayLength(&gContacts.elements)));
  for (var contactIdx = 0u; contactIdx < contactCount; contactIdx++) {
    let aIdx = gContacts.elements[contactIdx].aIdx;
    let bIdx = gContacts.elements[contactIdx].bIdx;
    setParticleHit(aIdx);
    setParticleHit(bIdx);
  }
}

@compute @workgroup_size(64)
fn updateMissileLifecycle(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;
  if (particleHit(index)) {
    killParticle(index);
    return;
  }

  switch particleType(index) {
    case bodyTypeNone: {
    }
    case bodyTypeShip: {
    }
    case bodyTypeMissile: {
      gMissiles[index].age += params.deltaT;
      if (gMissiles[index].age > params.maxMissileAge) {
        killParticle(index);
        return;
      }
    }
    default: {
    }
  }
}

@compute @workgroup_size(64)
fn selectTargets(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;

  switch particleType(index) {
    case bodyTypeNone: {
    }
    case bodyTypeShip: {
      let ship = gShips[index];
      if (params.time >= ship.nextShotTime) {
        gShips[index].targetIdx = findTarget(index);
      }
    }
    case bodyTypeMissile: {
    }
    default: {
    }
  }
}

@compute @workgroup_size(64)
fn spawnMissiles(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;

  switch particleType(index) {
    case bodyTypeNone: {
    }
    case bodyTypeShip: {
      let ship = gShips[index];
      if (params.time >= ship.nextShotTime && ship.targetIdx >= 0) {
        let mIdxSigned = getFreeID();
        if (mIdxSigned >= 0) {
          let mIdx = u32(mIdxSigned);
          // TODO: need to somehow synchronise writes so we're not changing particle types
          // as we're iterating over them.
          gBodies[mIdx] = gBodies[index];
          gParticles[mIdx].metadata = makeParticleMetadata(bodyTypeMissile, particleTeam(index));
          gParticles[mIdx].flags = 0;
          gParticles[mIdx].col = 0xffff00ff;
          gMissiles[mIdx].age = 0.0;
          gMissiles[mIdx].targetIdx = ship.targetIdx;

          gShips[index].nextShotTime = params.time + params.shipShotCooldown;
          gShips[index].targetIdx = -1;
        }
      }
    }
    case bodyTypeMissile: {
    }
    default: {
    }
  }
}

fn addFreeID(freeIdx : u32) -> bool {
  let capacity = arrayLength(&gFreeIDs.elements);

  // Loop until we push or the buffer is full.
  while (true) {
    let oldValue = atomicLoad(&gFreeIDs.count);
    if (oldValue >= capacity) {
      break;
    }
    let newValue = oldValue + 1;
    let result = atomicCompareExchangeWeak(&gFreeIDs.count, oldValue, newValue);
    if (result.exchanged) {
      gFreeIDs.elements[oldValue] = freeIdx;
      return true;
    }
  }
  return false;
}

fn getFreeID() -> i32 {
  // Loop until we pop or the buffer is empty.
  while (true) {
    let oldValue = atomicLoad(&gFreeIDs.count);
    if (oldValue <= 0) {
      break;
    }
    let newValue = oldValue - 1;
    let result = atomicCompareExchangeWeak(&gFreeIDs.count, oldValue, newValue);
    if (result.exchanged) {
      return i32(gFreeIDs.elements[newValue]);
    }
  }
  return -1;
}

fn randomizeBody(b : Body) -> Body {
  var newBody = Body();
  newBody.pos = 2.0 * (rand22(b.pos) - 0.5) * 1000.0;
  newBody.vel = 2.0 * (rand22(b.vel) - 0.5) * 0.0;
  newBody.angle = 0.0;
  newBody.angularVel = 0.0;
  return newBody;
}

fn killParticle(index : u32) {
  resetBody(index);
  resetParticle(index);
  resetShip(index);
  resetMissile(index);
  addFreeID(index);
}

fn resetBody(index : u32) {
  gBodies[index] = randomizeBody(gBodies[index]);
}

fn resetMissile(index : u32) {
  var m = Missile();
  m.targetIdx = -1;
  gMissiles[index] = m;
}

fn resetShip(index : u32) {
  var s = Ship();
  gShips[index] = s;
}

fn resetParticle(index : u32) {
  let p = Particle();
  gParticles[index] = p;
}

fn findTarget(selfIdx : u32) -> i32 {
	let selfTeam = particleTeam(selfIdx);
  let selfType = particleType(selfIdx);
	if (selfType != bodyTypeShip) {
		return -1;
	}

  let body = gBodies[selfIdx];
  // TODO: figure out how to represent "anti missile missile".
	let wantType = select(bodyTypeShip, bodyTypeMissile, false);
	var closestIdx = -1;
	var closestDist = 0.0;
  for (var otherIdx = 0u; otherIdx < arrayLength(&gBodies); otherIdx++) {
    let otherB = gBodies[otherIdx];
		if (selfIdx == otherIdx || particleTeam(otherIdx) == selfTeam || particleType(otherIdx) != wantType) {
			continue;
		}
    let dist = distance(body.pos, otherB.pos);
		if (closestIdx < 0 || dist < closestDist) {
			closestDist = dist;
			closestIdx = i32(otherIdx);
		}
	}
	return closestIdx;
}

fn setParticleHit(index : u32) {
  gParticles[index].flags |= particleFlagHit;
}

fn particleHit(index : u32) -> bool {
  let flags = gParticles[index].flags;
  return (flags & particleFlagHit) != 0;
}

fn makeParticleMetadata(bodyType : u32, team : u32) -> u32 {
  return (bodyType << 8) | team;
}

fn particleType(index : u32) -> u32 {
  let metadata = gParticles[index].metadata;
  return (metadata >> 8) & 0xff;
}

fn particleTeam(index : u32) -> u32 {
  let metadata = gParticles[index].metadata;
  return metadata & 0xff;
}

fn flock(selfIdx : u32) -> Acceleration {
  var cMass = vec2(0.0);
  var cVel = vec2(0.0);
  var colVel = vec2(0.0);
  var cMassCount = 0u;
  var cVelCount = 0u;

  let selfTeam = particleTeam(selfIdx);
  let currentBody = gBodies[selfIdx];
  for (var otherIdx = 0u; otherIdx < arrayLength(&gBodies); otherIdx++) {
    if (otherIdx == selfIdx || particleType(otherIdx) != bodyTypeShip) {
      continue;
    }
    let other = gBodies[otherIdx];
    let pos = other.pos.xy;
    let vel = other.vel.xy;
    let dPos = pos - currentBody.pos;
    let dist = length(dPos);
    if (dist < params.avoidDistance) {
      colVel -= dPos;
    }
    if (particleTeam(otherIdx) == selfTeam) {
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
    cMass = (cMass / vec2(f32(cMassCount))) - currentBody.pos;
  }
  if (cVelCount > 0) {
    cVel /= f32(cVelCount);
  }

  var acc = Acceleration();
  let dVel = (colVel * params.avoidScale) + (cMass * params.cMassScale) + (cVel * params.cVelScale);
  acc.linearAcc = dVel / params.deltaT;

  // Set the desired reference frame to the current state, attempting to orient with the velocity vector.
  // TODO: we could ignore linear component of this - maybe the compiler does that for us?
  let desiredBody = Body(currentBody.pos, currentBody.vel, angleOf(currentBody.vel, currentBody.angle), 0.0);
  let rel = bodySub(desiredBody, currentBody);
  acc.angularAcc = computeTurnAcceleration(rel.angle, rel.angularVel);
  return acc;
}

fn updateMissile(selfIdx : u32, targetIdx : u32) -> Acceleration {
  let currentBody = gBodies[selfIdx];
  let targetBody = gBodies[targetIdx];

  let targetVec = currentBody.pos - targetBody.pos;
  let targetDist = length(targetVec);
  let targetDir = normalize(targetVec);  // TODO: handle zero targetDist

  let desiredDir = targetDir;
  let desiredDist = clamp(targetDist, 0.0, 0.0);      // TODO: this would be min/max distance
  let desiredPos = targetBody.pos + desiredDir * desiredDist;
  let desiredAngle = angleOf(currentBody.vel, currentBody.angle); // angleOf(-targetDir);   // TODO: for ships use -targetDir
  let desiredBody = Body(desiredPos, targetBody.vel, desiredAngle, 0.0);

  let rel = bodySub(desiredBody, currentBody);
  // Transform into the missile's coordinate system.
  let localRel = bodyRotate(rel, -currentBody.angle);

  var localLinAcc = vec2f(0, 0);
  // Apply proportional navigation to track towards the target.
  localLinAcc.x = proNav2D(localRel.pos, localRel.vel);
	// Accelerate forward as fast as possible while staying under maxMissileSpeed (with respect to target).
  if (params.maxMissileSpeed == 0) {
    localLinAcc.y = params.maxMissileAcc;
  } else {
    // Relative velocity is negative as we're closing on the target.
    let speed = -localRel.vel.y;
    if (speed < params.maxMissileSpeed) {
      let maxMissileAcc = min((params.maxMissileSpeed - speed) / params.deltaT, params.maxMissileAcc);
      localLinAcc.y = maxMissileAcc;
    }
  }

  // Limit acceleration
  let l = length(localLinAcc);
  if (l > params.maxMissileAcc) {
    localLinAcc *= params.maxMissileAcc / l;
  }

  var acc = Acceleration();
  acc.linearAcc = rotVec(localLinAcc, currentBody.angle);
  acc.angularAcc = computeTurnAcceleration(rel.angle, rel.angularVel);
  return acc;
}

// A version of https://en.wikipedia.org/wiki/Proportional_navigation simplified for 2D.
fn proNav2D(r : vec2f, v : vec2f) -> f32 {
  return -proNavGain * perpDot(r, v) * length(v) / dot(r, r);
}

fn perpDot(a: vec2f, b: vec2f) -> f32 {
	return a.x*b.y - a.y*b.x;
}

fn computeTurnAcceleration(relAng : f32, relAngVel : f32) -> f32 {
  let maxMissileAcc = params.maxMissileAngAcc;
	// Compute the maximum velocity we can turn at and still stop in time.
	// Given v^2 = u^2 + 2as, assuming v=0 then u = sqrt(-2as).
	// The most we can accelerate in this frame is (sqrt(2as)-u)/t.
	// https://physics.stackexchange.com/questions/312692.
  let absAngDiff = abs(relAng);
  let absAngSign = sign(relAng);
	let maxBrakingVel = sqrt(2.0 * maxMissileAcc * absAngDiff) * absAngSign;
	return clamp((maxBrakingVel + relAngVel) / params.deltaT, -maxMissileAcc, maxMissileAcc);
}

// https://gist.github.com/munrocket/236ed5ba7e409b8bdf1ff6eca5dcdc39
fn hash22(p: vec2u) -> vec2u {
    var v = p * 1664525u + 1013904223u;
    v.x += v.y * 1664525u; v.y += v.x * 1664525u;
    v ^= v >> vec2u(16u);
    v.x += v.y * 1664525u; v.y += v.x * 1664525u;
    v ^= v >> vec2u(16u);
    return v;
}

fn rand22(f: vec2f) -> vec2f { return vec2f(hash22(bitcast<vec2u>(f))) / f32(0xffffffff); }