package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ---------------------------------------------------------------------------
// Entity helpers used by the command router
// ---------------------------------------------------------------------------

func (g *Game) apiSpawnEntity(owner string, cmd string, args json.RawMessage) (any, error) {
	var a struct {
		Preset      int32      `json:"preset"`
		PresetName  string     `json:"preset_name"`
		Model       int32      `json:"model"`
		Kind        string     `json:"kind"`
		Position    Vec3Info   `json:"position"`
		Pos         *Vec3Info  `json:"pos"`
		Rotation    *QuatInfo  `json:"rotation"`
		Yaw         float32    `json:"yaw"`
		Pitch       float32    `json:"pitch"`
		Roll        float32    `json:"roll"`
		Color       *ColorInfo `json:"color"`
		Half        *Vec3Info  `json:"half"`
		Radius      float32    `json:"radius"`
		Height      float32    `json:"height"`
		Density     float32    `json:"density"`
		Friction    float32    `json:"friction"`
		Restitution float32    `json:"restitution"`
		Name        string     `json:"name"`
		Static      bool       `json:"static"`
		Health      float32    `json:"health"`
		MaxHealth   float32    `json:"max_health"`
		Owner       string     `json:"owner"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	pos := a.Position
	if a.Pos != nil {
		pos = *a.Pos
	}

	rot := rotFromArgs(a.Rotation, a.Yaw, a.Pitch, a.Roll)

	color := colWhite()
	if a.Color != nil {
		color = a.Color.rl()
	}

	presetIndex := -1
	if a.PresetName != "" {
		for i := range presets {
			if presets[i].Name == a.PresetName {
				presetIndex = i
				break
			}
		}
	} else if a.Preset >= 0 && int(a.Preset) < len(presets) {
		presetIndex = int(a.Preset)
	}

	var e *Entity
	if a.Model != 0 {
		m, ok := g.modModels[a.Model]
		if !ok {
			return nil, fmt.Errorf("model %d not loaded", a.Model)
		}
		bb := rl.GetModelBoundingBox(m)
		half := bv(
			maxf((bb.Max.X-bb.Min.X)*0.5, 0.05),
			maxf((bb.Max.Y-bb.Min.Y)*0.5, 0.05),
			maxf((bb.Max.Z-bb.Min.Z)*0.5, 0.05),
		)
		if a.Half != nil {
			half = a.Half.box()
		}
		p := ShapePreset{
			Name: "ModModel", Category: "MOD", Kind: PropBox,
			Half: half, Density: 150, Friction: 0.6, Restitution: 0.03,
			Color: color, Model: m, HasModel: true,
			Hull: buildModelHull(m), Hulls: buildModelHulls(m), Skeletal: buildBoneHulls(m),
			Animations: g.modModelAnims[a.Model],
		}
		if a.Name != "" {
			p.Name = a.Name
		}
		if a.Density > 0 {
			p.Density = a.Density
		}
		if a.Friction > 0 {
			p.Friction = a.Friction
		}
		if a.Restitution != 0 {
			p.Restitution = a.Restitution
		}
		e = g.spawnPreset(p, pos.pos(), rot, color)
	} else if presetIndex >= 0 {
		e = g.spawnPreset(presets[presetIndex], pos.pos(), rot, color)
	} else {
		kind := PropBox
		if k, ok := propKindFromString(a.Kind); ok {
			kind = k
		}
		p := ShapePreset{
			Name: "ModEntity", Category: "MOD", Kind: kind,
			Density: 100, Friction: 0.6, Restitution: 0.1,
			Color: color,
		}
		if a.Name != "" {
			p.Name = a.Name
		}
		if a.Half != nil {
			p.Half = a.Half.box()
		} else {
			p.Half = bv(0.5, 0.5, 0.5)
		}
		if a.Radius > 0 {
			p.Radius = a.Radius
		} else {
			p.Radius = 0.5
		}
		if a.Height > 0 {
			p.Height = a.Height
		} else {
			p.Height = 1
		}
		if a.Density > 0 {
			p.Density = a.Density
		}
		if a.Friction > 0 {
			p.Friction = a.Friction
		}
		if a.Restitution != 0 {
			p.Restitution = a.Restitution
		}
		if a.Color != nil {
			p.Color = a.Color.rl()
		}
		e = g.spawnPreset(p, pos.pos(), rot, color)
	}

	if e == nil {
		return nil, fmt.Errorf("spawn failed")
	}
	if a.Static {
		e.Body.SetType(box3d.StaticBody)
		e.Static = true
	}
	if a.Name != "" {
		e.Name = a.Name
		e.Body.SetName(a.Name)
	}
	if a.Color != nil {
		e.Color = a.Color.rl()
	}
	if a.Health > 0 {
		e.Health = a.Health
	}
	if a.MaxHealth > 0 {
		e.MaxHealth = a.MaxHealth
	}
	if e.MaxHealth == 0 {
		e.MaxHealth = 100
		e.Health = 100
	}
	ownerName := a.Owner
	if ownerName == "" {
		ownerName = owner
	}
	e.Owner = ownerName
	return g.entityInfo(e), nil
}

func (g *Game) apiEntitySetPos(args json.RawMessage) (any, error) {
	e, err := g.entityFromArgs(args)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return false, nil
	}
	var a struct {
		Position *Vec3Info `json:"position"`
		Pos      *Vec3Info `json:"pos"`
		X        float32   `json:"x"`
		Y        float32   `json:"y"`
		Z        float32   `json:"z"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	p := e.Body.GetPosition()
	if a.Position != nil {
		p = a.Position.pos()
	} else if a.Pos != nil {
		p = a.Pos.pos()
	} else {
		p = bp(a.X, a.Y, a.Z)
	}
	e.Body.SetTransform(p, e.Body.GetRotation())
	e.Body.SetAwake(true)
	return true, nil
}

func (g *Game) apiEntitySetRot(args json.RawMessage) (any, error) {
	e, err := g.entityFromArgs(args)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return false, nil
	}
	var a struct {
		Rotation *QuatInfo `json:"rotation"`
		Yaw      float32   `json:"yaw"`
		Pitch    float32   `json:"pitch"`
		Roll     float32   `json:"roll"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	q := rotFromArgs(a.Rotation, a.Yaw, a.Pitch, a.Roll)
	if a.Rotation == nil && (a.Yaw == 0 && a.Pitch == 0 && a.Roll == 0) {
		q = e.Body.GetRotation()
	}
	e.Body.SetTransform(e.Body.GetPosition(), q)
	e.Body.SetAwake(true)
	return true, nil
}

func (g *Game) apiEntitySetTransform(args json.RawMessage) (any, error) {
	e, err := g.entityFromArgs(args)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return false, nil
	}
	var a struct {
		Position         *Vec3Info `json:"position"`
		Pos              *Vec3Info `json:"pos"`
		X                float32   `json:"x"`
		Y                float32   `json:"y"`
		Z                float32   `json:"z"`
		Rotation         *QuatInfo `json:"rotation"`
		Yaw, Pitch, Roll float32
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	p := e.Body.GetPosition()
	if a.Position != nil {
		p = a.Position.pos()
	} else if a.Pos != nil {
		p = a.Pos.pos()
	}
	q := e.Body.GetRotation()
	if a.Rotation != nil || a.Yaw != 0 || a.Pitch != 0 || a.Roll != 0 {
		q = rotFromArgs(a.Rotation, a.Yaw, a.Pitch, a.Roll)
	}
	e.Body.SetTransform(p, q)
	e.Body.SetAwake(true)
	return true, nil
}

func (g *Game) apiEntityPhysics(cmd string, args json.RawMessage) (any, error) {
	e, err := g.entityFromArgs(args)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return false, nil
	}
	var a struct {
		FX    float32   `json:"fx"`
		FY    float32   `json:"fy"`
		FZ    float32   `json:"fz"`
		IX    float32   `json:"ix"`
		IY    float32   `json:"iy"`
		IZ    float32   `json:"iz"`
		TX    float32   `json:"tx"`
		TY    float32   `json:"ty"`
		TZ    float32   `json:"tz"`
		Point *Vec3Info `json:"point"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	point := e.Body.GetPosition()
	if a.Point != nil {
		point = a.Point.pos()
	}
	switch cmd {
	case "entity.apply_force":
		e.Body.ApplyForceToCenter(bv(a.FX, a.FY, a.FZ), true)
	case "entity.apply_impulse":
		e.Body.ApplyLinearImpulseToCenter(bv(a.IX, a.IY, a.IZ), true)
	case "entity.apply_torque":
		e.Body.ApplyTorque(bv(a.TX, a.TY, a.TZ), true)
	case "entity.apply_angular_impulse":
		e.Body.ApplyAngularImpulse(bv(a.TX, a.TY, a.TZ), true)
	case "entity.apply_force_at_point":
		e.Body.ApplyForce(bv(a.FX, a.FY, a.FZ), point, true)
	case "entity.apply_impulse_at_point":
		e.Body.ApplyLinearImpulse(bv(a.IX, a.IY, a.IZ), point, true)
	}
	e.Body.SetAwake(true)
	return true, nil
}

func (g *Game) entityFromArgs(args json.RawMessage) (*Entity, error) {
	var a struct {
		ID       int32 `json:"id"`
		Index    int32 `json:"index"`
		EntityID int32 `json:"entity_id"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	id := a.ID
	if id == 0 {
		id = a.EntityID
	}
	if id != 0 {
		return g.resolveAnyEntity(id), nil
	}
	return g.resolveEntity(0, a.Index), nil
}

func (g *Game) apiRemoveEntity(args json.RawMessage) (any, error) {
	e, err := g.entityFromArgs(args)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return false, nil
	}
	g.removeEntity(e)
	return true, nil
}

func colWhite() rl.Color {
	return rl.NewColor(255, 255, 255, 255)
}

func applyMaterial(e *Entity, density, friction, restitution *float32) {
	if e == nil {
		return
	}
	if density != nil && *density > 0 {
		e.Density = *density
		e.Body.ApplyMassFromShapes()
		// Update shape density
		shapes := make([]box3d.ShapeID, e.Body.GetShapeCount())
		e.Body.GetShapes(shapes)
		for _, s := range shapes {
			s.SetDensity(*density, true)
		}
	}
	if friction != nil {
		e.Friction = *friction
		shapes := make([]box3d.ShapeID, e.Body.GetShapeCount())
		e.Body.GetShapes(shapes)
		for _, s := range shapes {
			s.SetFriction(*friction)
		}
	}
	if restitution != nil {
		e.Restitution = *restitution
		shapes := make([]box3d.ShapeID, e.Body.GetShapeCount())
		e.Body.GetShapes(shapes)
		for _, s := range shapes {
			s.SetRestitution(*restitution)
		}
	}
}

func (g *Game) apiCreateConstraint(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Type    string    `json:"type"`
		A       int32     `json:"a"`
		B       int32     `json:"b"`
		AID     int32     `json:"a_id"`
		BID     int32     `json:"b_id"`
		AIndex  int32     `json:"a_index"`
		BIndex  int32     `json:"b_index"`
		Anchor  *Vec3Info `json:"anchor"`
		Axis    *Vec3Info `json:"axis"`
		Collide *bool     `json:"collide"`
		// distance
		Length        float32 `json:"length"`
		EnableSpring  bool    `json:"enable_spring"`
		Hertz         float32 `json:"hertz"`
		DampingRatio  float32 `json:"damping_ratio"`
		MinLength     float32 `json:"min_length"`
		MaxLength     float32 `json:"max_length"`
		EnableLimit   bool    `json:"enable_limit"`
		EnableMotor   bool    `json:"enable_motor"`
		MaxMotorForce float32 `json:"max_motor_force"`
		MotorSpeed    float32 `json:"motor_speed"`
		// revolute
		TargetAngle    float32 `json:"target_angle"`
		LowerAngle     float32 `json:"lower_angle"`
		UpperAngle     float32 `json:"upper_angle"`
		MaxMotorTorque float32 `json:"max_motor_torque"`
		// prismatic
		TargetTranslation float32 `json:"target_translation"`
		LowerTranslation  float32 `json:"lower_translation"`
		UpperTranslation  float32 `json:"upper_translation"`
		// spherical
		ConeAngle       float32 `json:"cone_angle"`
		LowerTwistAngle float32 `json:"lower_twist_angle"`
		UpperTwistAngle float32 `json:"upper_twist_angle"`
		// weld
		LinearHertz         float32 `json:"linear_hertz"`
		AngularHertz        float32 `json:"angular_hertz"`
		LinearDampingRatio  float32 `json:"linear_damping_ratio"`
		AngularDampingRatio float32 `json:"angular_damping_ratio"`
		// parallel
		MaxTorque float32 `json:"max_torque"`
		// wheel
		SuspensionHertz        float32 `json:"suspension_hertz"`
		SuspensionDampingRatio float32 `json:"suspension_damping_ratio"`
		LowerSuspensionLimit   float32 `json:"lower_suspension_limit"`
		UpperSuspensionLimit   float32 `json:"upper_suspension_limit"`
		SpinSpeed              float32 `json:"spin_speed"`
		MaxSpinTorque          float32 `json:"max_spin_torque"`
		SteeringHertz          float32 `json:"steering_hertz"`
		SteeringDampingRatio   float32 `json:"steering_damping_ratio"`
		TargetSteeringAngle    float32 `json:"target_steering_angle"`
		EnableSteering         bool    `json:"enable_steering"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.AID != 0 {
		a.A = a.AID
	}
	if a.BID != 0 {
		a.B = a.BID
	}
	eA := g.resolveEntityRef(a.A, a.AIndex)
	eB := g.resolveEntityRef(a.B, a.BIndex)
	if eA == nil || eB == nil {
		return nil, fmt.Errorf("invalid entities")
	}
	bodyA := eA.Body
	bodyB := eB.Body

	anchor := eA.position()
	if a.Anchor != nil {
		anchor = a.Anchor.rl()
	}
	localA := bodyA.GetLocalPoint(toBp(anchor))
	localB := bodyB.GetLocalPoint(toBp(anchor))

	axis := rlv(0, 0, 1)
	if a.Axis != nil {
		axis = normRl(a.Axis.rl())
	}
	localAxisA := bodyA.GetLocalVector(toBv(axis))
	localAxisB := bodyB.GetLocalVector(toBv(axis))

	collide := false
	if a.Collide != nil {
		collide = *a.Collide
	}

	g.modConstraintLast++
	id := g.modConstraintLast
	cType := strings.ToLower(a.Type)
	if cType == "" {
		cType = "weld"
	}

	switch cType {
	case "weld":
		def := box3d.DefaultWeldJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: box3d.QuatIdentity}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: box3d.QuatIdentity}
		def.Base.CollideConnected = collide
		if a.LinearHertz > 0 {
			def.LinearHertz = a.LinearHertz
		}
		if a.AngularHertz > 0 {
			def.AngularHertz = a.AngularHertz
		}
		if a.LinearDampingRatio > 0 {
			def.LinearDampingRatio = a.LinearDampingRatio
		}
		if a.AngularDampingRatio > 0 {
			def.AngularDampingRatio = a.AngularDampingRatio
		}
		joint := g.world.CreateWeldJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "weld", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "distance", "rope", "spring":
		def := box3d.DefaultDistanceJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: box3d.QuatIdentity}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: box3d.QuatIdentity}
		def.Base.CollideConnected = collide
		dx := anchor.X - eB.position().X
		dy := anchor.Y - eB.position().Y
		dz := anchor.Z - eB.position().Z
		curDist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if a.Length > 0 {
			def.Length = a.Length
		} else {
			def.Length = curDist
		}
		if a.EnableSpring {
			def.EnableSpring = true
			def.Hertz = a.Hertz
			def.DampingRatio = a.DampingRatio
			if a.Hertz == 0 {
				def.Hertz = 4
			}
			if a.DampingRatio == 0 {
				def.DampingRatio = 0.5
			}
		}
		if a.EnableLimit {
			def.EnableLimit = true
			def.MinLength = a.MinLength
			def.MaxLength = a.MaxLength
		}
		if a.EnableMotor {
			def.EnableMotor = true
			def.MaxMotorForce = a.MaxMotorForce
			def.MotorSpeed = a.MotorSpeed
		}
		joint := g.world.CreateDistanceJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: cType, EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "revolute", "hinge":
		def := box3d.DefaultRevoluteJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: quatAlignZ(localAxisA)}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: quatAlignZ(localAxisB)}
		def.Base.CollideConnected = collide
		def.TargetAngle = a.TargetAngle
		if a.EnableSpring {
			def.EnableSpring = true
			def.Hertz = a.Hertz
			def.DampingRatio = a.DampingRatio
		}
		if a.EnableLimit {
			def.EnableLimit = true
			def.LowerAngle = a.LowerAngle
			def.UpperAngle = a.UpperAngle
		}
		if a.EnableMotor {
			def.EnableMotor = true
			def.MaxMotorTorque = a.MaxMotorTorque
			def.MotorSpeed = a.MotorSpeed
		}
		joint := g.world.CreateRevoluteJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "revolute", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "prismatic", "slider":
		def := box3d.DefaultPrismaticJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: quatAlignX(localAxisA)}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: quatAlignX(localAxisB)}
		def.Base.CollideConnected = collide
		if a.EnableSpring {
			def.EnableSpring = true
			def.Hertz = a.Hertz
			def.DampingRatio = a.DampingRatio
			def.TargetTranslation = a.TargetTranslation
		}
		if a.EnableLimit {
			def.EnableLimit = true
			def.LowerTranslation = a.LowerTranslation
			def.UpperTranslation = a.UpperTranslation
		}
		if a.EnableMotor {
			def.EnableMotor = true
			def.MaxMotorForce = a.MaxMotorForce
			def.MotorSpeed = a.MotorSpeed
		}
		joint := g.world.CreatePrismaticJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "prismatic", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "spherical", "ball":
		def := box3d.DefaultSphericalJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: box3d.QuatIdentity}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: box3d.QuatIdentity}
		def.Base.CollideConnected = collide
		if a.EnableSpring {
			def.EnableSpring = true
			def.Hertz = a.Hertz
			def.DampingRatio = a.DampingRatio
		}
		if a.ConeAngle > 0 {
			def.EnableConeLimit = true
			def.ConeAngle = a.ConeAngle
		}
		if a.LowerTwistAngle != 0 || a.UpperTwistAngle != 0 {
			def.EnableTwistLimit = true
			def.LowerTwistAngle = a.LowerTwistAngle
			def.UpperTwistAngle = a.UpperTwistAngle
		}
		if a.MaxMotorTorque > 0 {
			def.EnableMotor = true
			def.MaxMotorTorque = a.MaxMotorTorque
		}
		joint := g.world.CreateSphericalJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "spherical", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "motor":
		def := box3d.DefaultMotorJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: box3d.Vec3Zero, Q: box3d.QuatIdentity}
		def.Base.LocalFrameB = box3d.Transform{P: box3d.Vec3Zero, Q: box3d.QuatIdentity}
		def.Base.CollideConnected = collide
		def.LinearVelocity = bv(a.MotorSpeed, 0, 0)
		if a.MaxMotorForce > 0 {
			def.MaxVelocityForce = a.MaxMotorForce
		}
		joint := g.world.CreateMotorJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "motor", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "filter":
		def := box3d.DefaultFilterJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.CollideConnected = false
		joint := g.world.CreateFilterJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "filter", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "parallel":
		def := box3d.DefaultParallelJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: box3d.QuatIdentity}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: box3d.QuatIdentity}
		def.Base.CollideConnected = collide
		if a.Hertz > 0 {
			def.Hertz = a.Hertz
		}
		if a.DampingRatio > 0 {
			def.DampingRatio = a.DampingRatio
		}
		if a.MaxTorque > 0 {
			def.MaxTorque = a.MaxTorque
		}
		joint := g.world.CreateParallelJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "parallel", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	case "wheel":
		def := box3d.DefaultWheelJointDef()
		def.Base.BodyIdA = bodyA
		def.Base.BodyIdB = bodyB
		def.Base.LocalFrameA = box3d.Transform{P: localA, Q: quatAlignX(localAxisA)}
		def.Base.LocalFrameB = box3d.Transform{P: localB, Q: quatAlignX(localAxisB)}
		def.Base.CollideConnected = collide
		if a.SuspensionHertz > 0 {
			def.EnableSuspensionSpring = true
			def.SuspensionHertz = a.SuspensionHertz
			def.SuspensionDampingRatio = a.SuspensionDampingRatio
		}
		if a.LowerSuspensionLimit != 0 || a.UpperSuspensionLimit != 0 {
			def.EnableSuspensionLimit = true
			def.LowerSuspensionLimit = a.LowerSuspensionLimit
			def.UpperSuspensionLimit = a.UpperSuspensionLimit
		}
		if a.SpinSpeed != 0 {
			def.EnableSpinMotor = true
			def.SpinSpeed = a.SpinSpeed
			def.MaxSpinTorque = a.MaxSpinTorque
		}
		if a.EnableSteering {
			def.EnableSteering = true
			def.SteeringHertz = a.SteeringHertz
			def.SteeringDampingRatio = a.SteeringDampingRatio
			def.TargetSteeringAngle = a.TargetSteeringAngle
		}
		joint := g.world.CreateWheelJoint(&def)
		g.modConstraints = append(g.modConstraints, &ModConstraint{ID: id, Type: "wheel", EntityA: eA.ID, EntityB: eB.ID, JointID: joint, Owner: owner})
	default:
		return nil, fmt.Errorf("unknown constraint type %q", a.Type)
	}
	return id, nil
}

func (g *Game) resolveEntityRef(id, index int32) *Entity {
	if id != 0 {
		if e := g.resolveAnyEntity(id); e != nil {
			return e
		}
	}
	if index >= 0 {
		if index < int32(len(g.props)) {
			return g.props[index]
		}
	}
	return nil
}

func quatAlignZ(v box3d.Vec3) box3d.Quat {
	v = normalizeVec3(v)
	if vlen(v.X, v.Y, v.Z) < 1e-6 {
		return box3d.QuatIdentity
	}
	// rotate (0,0,1) to v
	z := box3d.Vec3{Z: 1}
	cross := box3d.Cross(z, v)
	dot := box3d.Dot(z, v)
	if dot < -0.9999 {
		return box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, float32(math.Pi))
	}
	s := float32(math.Sqrt(float64((1 + dot) * 2)))
	inv := 1 / s
	return normalizeQuat(box3d.Quat{V: box3d.Vec3{X: cross.X * inv, Y: cross.Y * inv, Z: cross.Z * inv}, S: s * 0.5})
}

func quatAlignX(v box3d.Vec3) box3d.Quat {
	v = normalizeVec3(v)
	if vlen(v.X, v.Y, v.Z) < 1e-6 {
		return box3d.QuatIdentity
	}
	x := box3d.Vec3{X: 1}
	cross := box3d.Cross(x, v)
	dot := box3d.Dot(x, v)
	if dot < -0.9999 {
		return box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, float32(math.Pi))
	}
	s := float32(math.Sqrt(float64((1 + dot) * 2)))
	inv := 1 / s
	return normalizeQuat(box3d.Quat{V: box3d.Vec3{X: cross.X * inv, Y: cross.Y * inv, Z: cross.Z * inv}, S: s * 0.5})
}

func normalizeVec3(v box3d.Vec3) box3d.Vec3 {
	l := vlen(v.X, v.Y, v.Z)
	if l < 1e-6 {
		return v
	}
	return box3d.Vec3{X: v.X / l, Y: v.Y / l, Z: v.Z / l}
}

// ---------------------------------------------------------------------------
// Audio
// ---------------------------------------------------------------------------

func (g *Game) apiLoadSound(args json.RawMessage) (any, error) {
	var a struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Name == "" || a.Path == "" {
		return nil, fmt.Errorf("name and path are required")
	}
	if !rl.IsAudioDeviceReady() {
		rl.InitAudioDevice()
	}
	s := rl.LoadSound(a.Path)
	g.modSounds[a.Name] = s
	return true, nil
}

func (g *Game) apiLoadMusic(args json.RawMessage) (any, error) {
	var a struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Name == "" || a.Path == "" {
		return nil, fmt.Errorf("name and path are required")
	}
	if !rl.IsAudioDeviceReady() {
		rl.InitAudioDevice()
	}
	m := rl.LoadMusicStream(a.Path)
	g.modMusic[a.Name] = m
	g.modMusicVol[a.Name] = 1
	return true, nil
}

// ---------------------------------------------------------------------------
// Weapon
// ---------------------------------------------------------------------------

func (g *Game) apiRegisterWeapon(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Name            string    `json:"name"`
		Type            string    `json:"type"`
		Damage          float32   `json:"damage"`
		Range           float32   `json:"range"`
		Ammo            int32     `json:"ammo"`
		MaxAmmo         int32     `json:"max_ammo"`
		FireRate        float32   `json:"fire_rate"`
		Recoil          float32   `json:"recoil"`
		Spread          float32   `json:"spread"`
		ProjectileSpeed float32   `json:"projectile_speed"`
		ProjectileLife  float32   `json:"projectile_life"`
		ProjectileSize  float32   `json:"projectile_size"`
		Color           ColorInfo `json:"color"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Name == "" {
		a.Name = "ModWeapon"
	}
	if a.Damage <= 0 {
		a.Damage = 10
	}
	if a.Range <= 0 {
		a.Range = 60
	}
	if a.FireRate <= 0 {
		a.FireRate = 4
	}
	if a.MaxAmmo <= 0 {
		a.MaxAmmo = 60
	}
	if a.Ammo == 0 {
		a.Ammo = a.MaxAmmo
	}
	if a.ProjectileSpeed <= 0 {
		a.ProjectileSpeed = 40
	}
	if a.ProjectileLife <= 0 {
		a.ProjectileLife = 3
	}
	if a.ProjectileSize <= 0 {
		a.ProjectileSize = 0.1
	}
	g.modWeaponLast++
	w := &ModWeapon{
		ID: g.modWeaponLast, Name: a.Name, Type: weaponTypeFromString(a.Type),
		Damage: a.Damage, Range: a.Range, Ammo: a.Ammo, MaxAmmo: a.MaxAmmo,
		FireRate: a.FireRate, Recoil: a.Recoil, Spread: a.Spread,
		ProjectileSpeed: a.ProjectileSpeed, ProjectileLife: a.ProjectileLife,
		ProjectileSize: a.ProjectileSize, Owner: owner,
	}
	if a.Color.provided() {
		w.Color = a.Color.rl()
	} else {
		w.Color = colEnergy
	}
	g.modWeapons = append(g.modWeapons, w)
	return w.ID, nil
}

func (g *Game) findWeapon(id int32) *ModWeapon {
	for _, w := range g.modWeapons {
		if w.ID == id {
			return w
		}
	}
	return nil
}

func (g *Game) unregisterWeapon(id int32, owner string) bool {
	for i, w := range g.modWeapons {
		if w.ID == id && (owner == "" || w.Owner == owner) {
			g.modWeapons = append(g.modWeapons[:i], g.modWeapons[i+1:]...)
			return true
		}
	}
	return false
}

func (g *Game) fireWeapon(id int32, owner string) (any, error) {
	w := g.findWeapon(id)
	if w == nil {
		return nil, fmt.Errorf("weapon %d does not exist", id)
	}
	now := g.time
	if now-w.LastShot < 1/float64(w.FireRate) {
		return map[string]any{"fired": false, "reason": "cooldown"}, nil
	}
	if w.Ammo <= 0 {
		return map[string]any{"fired": false, "reason": "no_ammo"}, nil
	}
	w.LastShot = now
	w.Ammo--
	fwd := g.cameraForward()
	eye := g.eyePosition()
	switch w.Type {
	case WeaponHitscan:
		res := g.castRay(w.Range, catStatic|catProp)
		if res.Hit {
			if res.ShapeId.GetBody().IsValid() {
				if target := g.byBody[res.ShapeId.GetBody()]; target != nil {
					g.damageEntity(target, w.Damage, toRlP(res.Point))
				}
			}
			if g.mods != nil {
				g.mods.FireEvent("weapon.hit", map[string]any{
					"weapon": w.ID, "owner": owner,
					"point":  map[string]float32{"x": toRlP(res.Point).X, "y": toRlP(res.Point).Y, "z": toRlP(res.Point).Z},
					"damage": w.Damage,
				})
			}
		}
		g.burst(eye, w.Color, 4, 2, 0.05)
	case WeaponProjectile:
		speed := w.ProjectileSpeed
		if speed <= 0 {
			speed = 40
		}
		g.spawnProjectile(owner, eye, rlv(fwd.X*speed, fwd.Y*speed, fwd.Z*speed), w.ProjectileSize, w.Damage, w.ProjectileLife, w.Color)
	case WeaponMelee:
		res := g.castRay(w.Range, catStatic|catProp)
		if res.Hit {
			if res.ShapeId.GetBody().IsValid() {
				if target := g.byBody[res.ShapeId.GetBody()]; target != nil {
					g.damageEntity(target, w.Damage, toRlP(res.Point))
				}
			}
		}
	}
	g.toolKick = 1
	g.shake = 0.06
	if g.mods != nil {
		g.mods.FireEvent("weapon.fired", map[string]any{"weapon": w.ID, "owner": owner, "ammo": w.Ammo})
	}
	return map[string]any{"fired": true, "ammo": w.Ammo}, nil
}

// ---------------------------------------------------------------------------
// Tool
// ---------------------------------------------------------------------------

func (g *Game) apiRegisterTool(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Name          string    `json:"name"`
		PrimaryDesc   string    `json:"primary_desc"`
		SecondaryDesc string    `json:"secondary_desc"`
		ReloadDesc    string    `json:"reload_desc"`
		Color         ColorInfo `json:"color"`
		PrimaryHook   string    `json:"primary_hook"`
		SecondaryHook string    `json:"secondary_hook"`
		ReloadHook    string    `json:"reload_hook"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Name == "" {
		a.Name = "ModTool"
	}
	g.modToolLast++
	t := &ModTool{
		ID: g.modToolLast, Name: a.Name, PrimaryDesc: a.PrimaryDesc,
		SecondaryDesc: a.SecondaryDesc, ReloadDesc: a.ReloadDesc,
		PrimaryHook: a.PrimaryHook, SecondaryHook: a.SecondaryHook,
		ReloadHook: a.ReloadHook, Owner: owner,
	}
	if a.Color.provided() {
		t.Color = a.Color.rl()
	} else {
		t.Color = colAccent
	}
	g.modTools = append(g.modTools, t)
	return t.ID, nil
}

func (g *Game) findTool(id int32) *ModTool {
	for _, t := range g.modTools {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (g *Game) unregisterTool(id int32, owner string) bool {
	for i, t := range g.modTools {
		if t.ID == id && (owner == "" || t.Owner == owner) {
			g.modTools = append(g.modTools[:i], g.modTools[i+1:]...)
			return true
		}
	}
	return false
}

func (g *Game) fireToolAction(cmd string, args json.RawMessage, owner string) (any, error) {
	var a struct {
		ID int32 `json:"id"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	id := a.ID
	if id == 0 {
		id = g.activeModTool
	}
	t := g.findTool(id)
	if t == nil {
		return nil, fmt.Errorf("tool %d does not exist", id)
	}
	var hook string
	switch cmd {
	case "tool.primary":
		hook = t.PrimaryHook
	case "tool.secondary":
		hook = t.SecondaryHook
	case "tool.reload":
		hook = t.ReloadHook
	}
	if hook != "" && g.mods != nil {
		g.mods.FireEvent(hook, map[string]any{"tool": t.ID, "owner": owner})
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Content presets (runtime)
// ---------------------------------------------------------------------------

func (g *Game) apiAddPreset(args json.RawMessage) (any, error) {
	var a struct {
		Name        string    `json:"name"`
		Category    string    `json:"category"`
		Kind        string    `json:"kind"`
		Half        *Vec3Info `json:"half"`
		Radius      float32   `json:"radius"`
		Height      float32   `json:"height"`
		Density     float32   `json:"density"`
		Friction    float32   `json:"friction"`
		Restitution float32   `json:"restitution"`
		Color       ColorInfo `json:"color"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Name == "" {
		a.Name = "ModPreset"
	}
	if a.Category == "" {
		a.Category = "MOD"
	}
	kind := PropBox
	if k, ok := propKindFromString(a.Kind); ok {
		kind = k
	}
	p := ShapePreset{
		Name: a.Name, Category: a.Category, Kind: kind,
		Density: 100, Friction: 0.6, Restitution: 0.1,
		Color: colWhite(),
	}
	if a.Half != nil {
		p.Half = a.Half.box()
	} else {
		p.Half = bv(0.5, 0.5, 0.5)
	}
	if a.Radius > 0 {
		p.Radius = a.Radius
	} else {
		p.Radius = 0.5
	}
	if a.Height > 0 {
		p.Height = a.Height
	} else {
		p.Height = 1
	}
	if a.Density > 0 {
		p.Density = a.Density
	}
	if a.Friction > 0 {
		p.Friction = a.Friction
	}
	if a.Restitution != 0 {
		p.Restitution = a.Restitution
	}
	if a.Color.provided() {
		p.Color = a.Color.rl()
	}
	presets = append(presets, p)
	rebuildSpawnCategories()
	return int32(len(presets) - 1), nil
}

func (g *Game) removePreset(index int32, name string) bool {
	if name != "" {
		for i := range presets {
			if presets[i].Name == name {
				presets = append(presets[:i], presets[i+1:]...)
				rebuildSpawnCategories()
				return true
			}
		}
		return false
	}
	if index < 0 || int(index) >= len(presets) {
		return false
	}
	presets = append(presets[:index], presets[index+1:]...)
	rebuildSpawnCategories()
	return true
}

// ---------------------------------------------------------------------------
// Resources (sandboxed mod data files)
// ---------------------------------------------------------------------------

func (g *Game) apiResourceRead(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeModPath(owner, a.Path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (g *Game) apiResourceWrite(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Append  bool   `json:"append"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeModPath(owner, a.Path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return nil, err
	}
	if a.Append {
		f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		_, err = f.WriteString(a.Content)
		return err == nil, err
	}
	err = os.WriteFile(full, []byte(a.Content), 0644)
	return err == nil, err
}

func (g *Game) apiResourceExists(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeModPath(owner, a.Path)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(full)
	return err == nil, nil
}

func (g *Game) apiResourceDelete(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeModPath(owner, a.Path)
	if err != nil {
		return nil, err
	}
	err = os.Remove(full)
	return err == nil, err
}

func (g *Game) apiResourceList(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeModPath(owner, a.Path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Particle
// ---------------------------------------------------------------------------

func (g *Game) apiSpawnParticle(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Position Vec3Info  `json:"position"`
		Pos      *Vec3Info `json:"pos"`
		Velocity *Vec3Info `json:"velocity"`
		Color    ColorInfo `json:"color"`
		Size     float32   `json:"size"`
		Life     float32   `json:"life"`
		Gravity  *float32  `json:"gravity"`
		Count    int32     `json:"count"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	pos := a.Position
	if a.Pos != nil {
		pos = *a.Pos
	}
	count := a.Count
	if count <= 0 {
		count = 1
	}
	size := a.Size
	if size <= 0 {
		size = 0.08
	}
	life := a.Life
	if life <= 0 {
		life = 0.5
	}
	gravity := float32(9)
	if a.Gravity != nil {
		gravity = *a.Gravity
	}
	color := colAccent
	if a.Color.provided() {
		color = a.Color.rl()
	}
	for i := int32(0); i < count; i++ {
		vel := rlv(0, 0, 0)
		if a.Velocity != nil {
			vel = a.Velocity.rl()
		}
		g.spawnModParticle(pos.rl(), vel, color, size, life, gravity, owner)
	}
	return count, nil
}
