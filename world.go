package main

import (
	"math"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Collision categories. Every shape gets exactly one category and a mask that
// describes what it is allowed to touch. Queries use their own category so the
// player can never be hit by the placement/grab rays.
const (
	catStatic uint64 = 1 << 0
	catProp   uint64 = 1 << 1
	catPlayer uint64 = 1 << 2
	catQuery  uint64 = 1 << 3

	maxProps = 220
)

func staticFilter() box3d.Filter {
	return box3d.Filter{CategoryBits: catStatic, MaskBits: catStatic | catProp | catPlayer | catQuery}
}

func propFilter() box3d.Filter {
	return box3d.Filter{CategoryBits: catProp, MaskBits: catStatic | catProp | catPlayer | catQuery}
}

func playerFilter() box3d.Filter {
	return box3d.Filter{CategoryBits: catPlayer, MaskBits: catStatic | catProp}
}

func queryFilter(mask uint64) box3d.QueryFilter {
	return box3d.QueryFilter{CategoryBits: catQuery, MaskBits: mask}
}

// ---------------------------------------------------------------------------
// Small math / colour helpers
// ---------------------------------------------------------------------------

func rgb(r, g, b uint8) rl.Color {
	return rl.Color{R: r, G: g, B: b, A: 255}
}

func rgba(r, g, b, a uint8) rl.Color {
	return rl.Color{R: r, G: g, B: b, A: a}
}

func withAlpha(c rl.Color, a uint8) rl.Color {
	c.A = a
	return c
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func lerpf(a, b, t float32) float32 {
	return a + (b-a)*t
}

// smoothing returns an exponential blend factor that is stable at any dt.
func smoothing(rate, dt float32) float32 {
	return 1 - float32(math.Exp(float64(-rate*dt)))
}

func vlen(x, y, z float32) float32 {
	return float32(math.Sqrt(float64(x*x + y*y + z*z)))
}

func lenRl(v rl.Vector3) float32 {
	return vlen(v.X, v.Y, v.Z)
}

func normRl(v rl.Vector3) rl.Vector3 {
	l := lenRl(v)
	if l < 1e-6 {
		return rlv(0, 1, 0)
	}
	return rlv(v.X/l, v.Y/l, v.Z/l)
}

func rlv(x, y, z float32) rl.Vector3 {
	return rl.Vector3{X: x, Y: y, Z: z}
}

func bp(x, y, z float32) box3d.Pos {
	return box3d.Pos{X: x, Y: y, Z: z}
}

func bv(x, y, z float32) box3d.Vec3 {
	return box3d.Vec3{X: x, Y: y, Z: z}
}

func toRlV(v box3d.Vec3) rl.Vector3 {
	return rlv(v.X, v.Y, v.Z)
}

func toRlP(p box3d.Pos) rl.Vector3 {
	return rlv(p.X, p.Y, p.Z)
}

func toBv(v rl.Vector3) box3d.Vec3 {
	return bv(v.X, v.Y, v.Z)
}

func toBp(v rl.Vector3) box3d.Pos {
	return bp(v.X, v.Y, v.Z)
}

func rlQuat(q box3d.Quat) rl.Quaternion {
	return rl.Quaternion{X: q.V.X, Y: q.V.Y, Z: q.V.Z, W: q.S}
}

// quatAxisAngle converts a Box3D quaternion into the axis/degrees form used by
// raylib's DrawModelEx.
func quatAxisAngle(q box3d.Quat) (rl.Vector3, float32) {
	s := clampf(q.S, -1, 1)
	angle := 2 * float32(math.Acos(float64(s))) * box3d.RadToDeg
	l := vlen(q.V.X, q.V.Y, q.V.Z)
	if l < 1e-6 || angle < 1e-4 {
		return rlv(0, 1, 0), 0
	}
	return rlv(q.V.X/l, q.V.Y/l, q.V.Z/l), angle
}

func quatYaw(radians float32) box3d.Quat {
	return box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, radians)
}

// ---------------------------------------------------------------------------
// Palette
// ---------------------------------------------------------------------------

// Palette: near-black panels, thin neutral borders and a single amber accent.
// The world itself stays desaturated so props and UI signals carry the colour.
var (
	colSkyTop    = rgb(20, 24, 30)
	colSkyBottom = rgb(121, 131, 139)
	colGround    = rgb(44, 47, 50)
	colWall      = rgb(58, 64, 70)
	colPlatform  = rgb(72, 79, 85)

	colText        = rgb(226, 226, 222)
	colMuted       = rgb(138, 144, 150)
	colAccent      = rgb(230, 166, 66)
	colAccentHi    = rgb(246, 193, 108)
	colAccentDim   = rgba(230, 166, 66, 165)
	colEnergy      = rgb(118, 164, 196)
	colWarn        = rgb(224, 166, 84)
	colDanger      = rgb(210, 88, 74)
	colFreeze      = rgb(126, 194, 218)
	colSpawn       = rgb(118, 164, 196)
	colDuplicate   = rgb(132, 178, 132)
	colGravityTool = rgb(166, 138, 206)

	colPanel       = rgba(13, 14, 16, 232)
	colPanelDeep   = rgba(9, 10, 11, 246)
	colPanelSoft   = rgba(23, 25, 28, 238)
	colPanelHover  = rgba(36, 39, 43, 244)
	colBorder      = rgba(190, 195, 200, 44)
	colBorderHi    = rgba(214, 218, 222, 96)
	colEdge        = rgba(5, 6, 8, 186)
	colDark        = rgb(10, 11, 12)
	colRowSelected = rgba(230, 166, 66, 26)

	colCrate   = rgb(168, 118, 64)
	colCrate2  = rgb(190, 154, 80)
	colCrate3  = rgb(70, 124, 118)
	colBall    = rgb(184, 82, 68)
	colBall2   = rgb(74, 122, 166)
	colBarrel  = rgb(150, 80, 56)
	colSlab    = rgb(160, 136, 98)
	colAnvil   = rgb(108, 116, 124)
	colBoulder = rgb(96, 108, 116)
)

// ---------------------------------------------------------------------------
// Entities and presets
// ---------------------------------------------------------------------------

type PropKind uint8

const (
	PropBox PropKind = iota
	PropSphere
	PropCylinder
)

func (k PropKind) String() string {
	switch k {
	case PropBox:
		return "box"
	case PropSphere:
		return "sphere"
	case PropCylinder:
		return "cylinder"
	default:
		return "unknown"
	}
}

func propKindFromString(s string) (PropKind, bool) {
	switch s {
	case "box":
		return PropBox, true
	case "sphere":
		return PropSphere, true
	case "cylinder":
		return PropCylinder, true
	default:
		return PropBox, false
	}
}

// Entity is one renderable rigid body. Static arena geometry and dynamic props
// share this type; only dynamic entities are saved and targetable.
type Entity struct {
	Body        box3d.BodyID
	Kind        PropKind
	Half        box3d.Vec3
	Radius      float32
	Height      float32
	Sides       int
	Color       rl.Color
	Name        string
	Static      bool
	Frozen      bool
	Density     float32
	Friction    float32
	Restitution float32
	SpawnedAt   float64
	Model       rl.Model
	HasModel    bool
	Hull        *box3d.HullData
	Hulls       []*box3d.HullData
	Mesh        *box3d.MeshData
}

func (e *Entity) position() rl.Vector3 {
	return toRlP(e.Body.GetPosition())
}

func (e *Entity) radiusXZ() float32 {
	switch e.Kind {
	case PropBox:
		return maxf(e.Half.X, e.Half.Z)
	case PropSphere:
		return e.Radius
	case PropCylinder:
		return e.Radius
	}
	return 0.5
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

type ShapePreset struct {
	Name        string
	Category    string
	Kind        PropKind
	Half        box3d.Vec3
	Radius      float32
	Height      float32
	Density     float32
	Friction    float32
	Restitution float32
	Color       rl.Color
	Model       rl.Model
	HasModel    bool
	Hull        *box3d.HullData
	Hulls       []*box3d.HullData
	Mesh        *box3d.MeshData
}

// support returns how far the centre of an unrotated preset extends along a
// world normal. Used to keep freshly spawned objects just above the surface
// they are aimed at.
func (p ShapePreset) support(n box3d.Vec3) float32 {
	switch p.Kind {
	case PropBox:
		return absf(n.X)*p.Half.X + absf(n.Y)*p.Half.Y + absf(n.Z)*p.Half.Z
	case PropSphere:
		return p.Radius
	case PropCylinder:
		return float32(math.Sqrt(float64(n.X*n.X+n.Z*n.Z)))*p.Radius + absf(n.Y)*p.Height*0.5
	}
	return 0.5
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

var presets = []ShapePreset{
	{
		Name: "Crate", Kind: PropBox, Half: bv(0.5, 0.5, 0.5),
		Density: 150, Friction: 0.62, Restitution: 0.03, Color: colCrate,
	},
	{
		Name: "Ball", Kind: PropSphere, Radius: 0.5,
		Density: 240, Friction: 0.38, Restitution: 0.34, Color: colBall,
	},
	{
		Name: "Barrel", Kind: PropCylinder, Radius: 0.46, Height: 1.2,
		Density: 130, Friction: 0.55, Restitution: 0.05, Color: colBarrel,
	},
	{
		Name: "Slab", Kind: PropBox, Half: bv(1.15, 0.18, 0.62),
		Density: 170, Friction: 0.60, Restitution: 0.04, Color: colSlab,
	},
	{
		Name: "Anvil", Kind: PropBox, Half: bv(0.46, 0.44, 0.64),
		Density: 950, Friction: 0.70, Restitution: 0.02, Color: colAnvil,
	},
	{
		Name: "Block", Kind: PropBox, Half: bv(1.0, 1.0, 1.0),
		Density: 140, Friction: 0.64, Restitution: 0.03, Color: colCrate3,
	},
	{
		Name: "Pebble", Kind: PropSphere, Radius: 0.27,
		Density: 520, Friction: 0.42, Restitution: 0.28, Color: colCrate2,
	},
	{
		Name: "Boulder", Kind: PropSphere, Radius: 0.95,
		Density: 210, Friction: 0.50, Restitution: 0.18, Color: colBoulder,
	},
	{
		Name: "Plank", Kind: PropBox, Half: bv(0.14, 0.14, 1.7),
		Density: 120, Friction: 0.55, Restitution: 0.04, Color: colSlab,
	},
	{
		Name: "Column", Kind: PropCylinder, Radius: 0.32, Height: 2.6,
		Density: 220, Friction: 0.62, Restitution: 0.02, Color: colPlatform,
	},
}

var defaultSpawn = bp(0, 2.2, 12)

// ---------------------------------------------------------------------------
// World construction
// ---------------------------------------------------------------------------

func newPhysicsWorld() box3d.WorldID {
	wd := box3d.DefaultWorldDef()
	wd.Gravity = box3d.Vec3{Y: -20}
	wd.EnableSleep = true
	wd.EnableContinuous = true
	return box3d.CreateWorld(&wd)
}

func (g *Game) registerEntity(e *Entity) {
	if e == nil {
		return
	}
	g.entities = append(g.entities, e)
	g.byBody[e.Body] = e
	if !e.Static {
		g.props = append(g.props, e)
	}
}

func (g *Game) addBoxEntity(static bool, pos box3d.Pos, rot box3d.Quat, half box3d.Vec3, c rl.Color, name string, density, friction, restitution float32) *Entity {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = pos
	bd.Rotation = rot
	bd.Name = name
	if static {
		bd.Type = box3d.StaticBody
	} else {
		bd.LinearDamping = 0.04
		bd.AngularDamping = 0.14
	}

	body := g.world.CreateBody(&bd)
	if body.IsNull() {
		return nil
	}

	sd := box3d.DefaultShapeDef()
	sd.Density = density
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = friction
	sd.BaseMaterial.Restitution = restitution
	if static {
		sd.Filter = staticFilter()
	} else {
		sd.Filter = propFilter()
	}

	hull := box3d.MakeBoxHull(half.X, half.Y, half.Z)
	body.CreateHullShape(&sd, &hull.Base)

	e := &Entity{
		Body: body, Kind: PropBox, Half: half, Sides: 4,
		Color: c, Name: name, Static: static,
		Density: density, Friction: friction, Restitution: restitution,
		SpawnedAt: g.time,
	}
	g.registerEntity(e)
	return e
}

// addHullEntity creates a rigid body whose collision shape is an arbitrary
// convex hull (typically generated from a content model's mesh). It keeps
// Kind=PropBox so the existing box-based rendering/UI/physics switch logic
// keeps working; the Hull field carries the original hull for diagnostics and
// the Model field (when set) drives rendering.
func (g *Game) addHullEntity(static bool, pos box3d.Pos, rot box3d.Quat, half box3d.Vec3, c rl.Color, name string, density, friction, restitution float32, hull *box3d.HullData, model rl.Model, hasModel bool) *Entity {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = pos
	bd.Rotation = rot
	bd.Name = name
	if static {
		bd.Type = box3d.StaticBody
	} else {
		bd.LinearDamping = 0.04
		bd.AngularDamping = 0.14
	}

	body := g.world.CreateBody(&bd)
	if body.IsNull() {
		return nil
	}

	sd := box3d.DefaultShapeDef()
	sd.Density = density
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = friction
	sd.BaseMaterial.Restitution = restitution
	if static {
		sd.Filter = staticFilter()
	} else {
		sd.Filter = propFilter()
	}

	if hull == nil {
		h := box3d.MakeBoxHull(half.X, half.Y, half.Z)
		hull = &h.Base
	}
	body.CreateHullShape(&sd, hull)

	e := &Entity{
		Body: body, Kind: PropBox, Half: half, Sides: 4,
		Color: c, Name: name, Static: static,
		Density: density, Friction: friction, Restitution: restitution,
		SpawnedAt: g.time,
		Model:     model, HasModel: hasModel, Hull: hull,
	}
	g.registerEntity(e)
	return e
}

// addMeshEntity creates a rigid body whose collision shape is the exact
// triangle mesh of a content model. Mesh data must outlive the shape, so the
// pointer is stored on the entity.
func (g *Game) addMeshEntity(static bool, pos box3d.Pos, rot box3d.Quat, half box3d.Vec3, c rl.Color, name string, density, friction, restitution float32, mesh *box3d.MeshData, model rl.Model, hasModel bool) *Entity {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = pos
	bd.Rotation = rot
	bd.Name = name
	if static {
		bd.Type = box3d.StaticBody
	} else {
		bd.LinearDamping = 0.04
		bd.AngularDamping = 0.14
	}

	body := g.world.CreateBody(&bd)
	if body.IsNull() {
		return nil
	}

	sd := box3d.DefaultShapeDef()
	sd.Density = density
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = friction
	sd.BaseMaterial.Restitution = restitution
	if static {
		sd.Filter = staticFilter()
	} else {
		sd.Filter = propFilter()
	}

	if mesh == nil {
		h := box3d.MakeBoxHull(half.X, half.Y, half.Z)
		body.CreateHullShape(&sd, &h.Base)
	} else {
		body.CreateMeshShape(&sd, mesh, box3d.Vec3One)
		// Keep the centre of mass on the body origin so the model falls
		// straight under gravity instead of tumbling around an off-centre
		// mesh centroid.
		md := body.GetMassData()
		md.Center = box3d.Vec3Zero
		body.SetMassData(md)
	}

	e := &Entity{
		Body: body, Kind: PropBox, Half: half, Sides: 4,
		Color: c, Name: name, Static: static,
		Density: density, Friction: friction, Restitution: restitution,
		SpawnedAt: g.time,
		Model:     model, HasModel: hasModel, Mesh: mesh,
	}
	g.registerEntity(e)
	return e
}

// addHullsEntity attaches a set of convex hulls to one body, giving a concave
// approximation of a model while staying compatible with dynamic collisions.
func (g *Game) addHullsEntity(static bool, pos box3d.Pos, rot box3d.Quat, half box3d.Vec3, c rl.Color, name string, density, friction, restitution float32, hulls []*box3d.HullData, model rl.Model, hasModel bool) *Entity {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = pos
	bd.Rotation = rot
	bd.Name = name
	if static {
		bd.Type = box3d.StaticBody
	} else {
		bd.LinearDamping = 0.04
		bd.AngularDamping = 0.14
	}

	body := g.world.CreateBody(&bd)
	if body.IsNull() {
		return nil
	}

	sd := box3d.DefaultShapeDef()
	sd.Density = density
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = friction
	sd.BaseMaterial.Restitution = restitution
	if static {
		sd.Filter = staticFilter()
	} else {
		sd.Filter = propFilter()
	}

	attached := 0
	for _, h := range hulls {
		if h == nil {
			continue
		}
		body.CreateHullShape(&sd, h)
		attached++
	}
	if attached == 0 {
		h := box3d.MakeBoxHull(half.X, half.Y, half.Z)
		body.CreateHullShape(&sd, &h.Base)
		attached = 1
	}

	e := &Entity{
		Body: body, Kind: PropBox, Half: half, Sides: 4,
		Color: c, Name: name, Static: static,
		Density: density, Friction: friction, Restitution: restitution,
		SpawnedAt: g.time,
		Model:     model, HasModel: hasModel, Hulls: hulls,
	}
	if len(hulls) > 0 {
		e.Hull = hulls[0]
	}
	g.registerEntity(e)
	return e
}

func (g *Game) addSphereEntity(pos box3d.Pos, rot box3d.Quat, radius float32, c rl.Color, name string, density, friction, restitution float32) *Entity {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = pos
	bd.Rotation = rot
	bd.Name = name
	bd.LinearDamping = 0.02
	bd.AngularDamping = 0.06

	body := g.world.CreateBody(&bd)
	if body.IsNull() {
		return nil
	}

	sd := box3d.DefaultShapeDef()
	sd.Density = density
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = friction
	sd.BaseMaterial.Restitution = restitution
	sd.BaseMaterial.RollingResistance = 0.04
	sd.Filter = propFilter()

	sphere := box3d.Sphere{Center: box3d.Vec3Zero, Radius: radius}
	body.CreateSphereShape(&sd, &sphere)

	e := &Entity{
		Body: body, Kind: PropSphere, Radius: radius, Sides: 16,
		Color: c, Name: name,
		Density: density, Friction: friction, Restitution: restitution,
		SpawnedAt: g.time,
	}
	g.registerEntity(e)
	return e
}

func (g *Game) addCylinderEntity(pos box3d.Pos, rot box3d.Quat, radius, height float32, c rl.Color, name string, density, friction, restitution float32) *Entity {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = pos
	bd.Rotation = rot
	bd.Name = name
	bd.LinearDamping = 0.04
	bd.AngularDamping = 0.18

	body := g.world.CreateBody(&bd)
	if body.IsNull() {
		return nil
	}

	sd := box3d.DefaultShapeDef()
	sd.Density = density
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = friction
	sd.BaseMaterial.Restitution = restitution
	sd.Filter = propFilter()

	const sides = 16
	hull := box3d.CreateCylinder(height, radius, -height*0.5, sides)
	body.CreateHullShape(&sd, hull)

	e := &Entity{
		Body: body, Kind: PropCylinder, Radius: radius, Height: height, Sides: sides,
		Color: c, Name: name,
		Density: density, Friction: friction, Restitution: restitution,
		SpawnedAt: g.time,
	}
	g.registerEntity(e)
	return e
}

func (g *Game) spawnPreset(p ShapePreset, pos box3d.Pos, rot box3d.Quat, c rl.Color) *Entity {
	if c.A == 0 {
		c = p.Color
	}
	switch p.Kind {
	case PropBox:
		if len(p.Hulls) > 0 {
			return g.addHullsEntity(false, pos, rot, p.Half, c, p.Name, p.Density, p.Friction, p.Restitution, p.Hulls, p.Model, p.HasModel)
		}
		if p.Hull != nil {
			return g.addHullEntity(false, pos, rot, p.Half, c, p.Name, p.Density, p.Friction, p.Restitution, p.Hull, p.Model, p.HasModel)
		}
		e := g.addBoxEntity(false, pos, rot, p.Half, c, p.Name, p.Density, p.Friction, p.Restitution)
		if p.HasModel {
			e.Model = p.Model
			e.HasModel = true
		}
		e.Hull = p.Hull
		return e
	case PropSphere:
		return g.addSphereEntity(pos, rot, p.Radius, c, p.Name, p.Density, p.Friction, p.Restitution)
	case PropCylinder:
		return g.addCylinderEntity(pos, rot, p.Radius, p.Height, c, p.Name, p.Density, p.Friction, p.Restitution)
	}
	return nil
}

func (g *Game) buildArena() {
	// Floor: top surface is exactly y = 0.
	g.addBoxEntity(true, bp(0, -0.5, 0), box3d.QuatIdentity, bv(30, 0.5, 30), colGround, "Ground", 1000, 0.9, 0.0)

	// Perimeter walls (4 m tall, so props stay in the yard).
	g.addBoxEntity(true, bp(0, 2, 30.5), box3d.QuatIdentity, bv(30.5, 2, 0.5), colWall, "Wall", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(0, 2, -30.5), box3d.QuatIdentity, bv(30.5, 2, 0.5), colWall, "Wall", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(30.5, 2, 0), box3d.QuatIdentity, bv(0.5, 2, 30.5), colWall, "Wall", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(-30.5, 2, 0), box3d.QuatIdentity, bv(0.5, 2, 30.5), colWall, "Wall", 1000, 0.7, 0.0)

	// Playground pieces.
	g.addBoxEntity(true, bp(-11, 0.75, -6), box3d.QuatIdentity, bv(3.2, 0.3, 3.2), colPlatform, "Platform", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(11, 1.6, -9), box3d.QuatIdentity, bv(4.0, 0.3, 2.2), colPlatform, "Platform", 1000, 0.7, 0.0)

	rampRot := box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, -0.18)
	g.addBoxEntity(true, bp(0, 0.62, -8.5), rampRot, bv(2.6, 0.18, 4.5), colPlatform, "Ramp", 1000, 0.7, 0.0)

	// A short stepped stack on the left.
	g.addBoxEntity(true, bp(-18, 0.2, 7), box3d.QuatIdentity, bv(1.2, 0.2, 1.2), colPlatform, "Step", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(-17.2, 0.6, 7), box3d.QuatIdentity, bv(1.2, 0.2, 1.2), colPlatform, "Step", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(-16.4, 1.0, 7), box3d.QuatIdentity, bv(1.2, 0.2, 1.2), colPlatform, "Step", 1000, 0.7, 0.0)

	// Open-topped storage bay.
	g.addBoxEntity(true, bp(13, 0.8, 9.0), box3d.QuatIdentity, bv(2.6, 0.8, 0.25), colWall, "Bay", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(10.4, 0.8, 6.5), box3d.QuatIdentity, bv(0.25, 0.8, 2.75), colWall, "Bay", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(15.6, 0.8, 6.5), box3d.QuatIdentity, bv(0.25, 0.8, 2.75), colWall, "Bay", 1000, 0.7, 0.0)

	// A couple of pillars to hide behind.
	g.addBoxEntity(true, bp(-21, 1.5, -13), box3d.QuatIdentity, bv(0.55, 1.5, 0.55), colWall, "Pillar", 1000, 0.7, 0.0)
	g.addBoxEntity(true, bp(20, 1.5, -16), box3d.QuatIdentity, bv(0.55, 1.5, 0.55), colWall, "Pillar", 1000, 0.7, 0.0)
}

func (g *Game) buildStarterProps() {
	crate := presets[0]
	ball := presets[1]
	barrel := presets[2]
	slab := presets[3]
	anvil := presets[4]

	// Crate ziggurat in front of the player.
	crateCols := []rl.Color{colCrate, colCrate2, colCrate3}
	for layer := 0; layer < 3; layer++ {
		n := 3 - layer
		for ix := 0; ix < n; ix++ {
			for iz := 0; iz < n; iz++ {
				x := -float32(n-1)*0.5 + float32(ix)*1.02
				z := -4 + (-float32(n-1)*0.5+float32(iz))*1.02
				y := 0.51 + float32(layer)*1.02
				c := crateCols[(ix+iz+layer)%len(crateCols)]
				g.spawnPreset(crate, bp(x, y, z), box3d.QuatIdentity, c)
			}
		}
	}

	// Balls and barrels to knock around.
	g.spawnPreset(ball, bp(-6.8, 0.52, 1.2), box3d.QuatIdentity, colBall)
	g.spawnPreset(ball, bp(-7.7, 0.52, 2.0), box3d.QuatIdentity, colBall2)
	g.spawnPreset(ball, bp(-7.1, 1.55, 1.6), box3d.QuatIdentity, colCrate2)

	g.spawnPreset(barrel, bp(5.6, 0.62, 1.0), quatYaw(0.4), colBarrel)
	g.spawnPreset(barrel, bp(6.5, 0.62, 1.9), quatYaw(-0.7), colBarrel)
	g.spawnPreset(barrel, bp(4.9, 0.62, 2.2), quatYaw(1.1), colCrate3)

	g.spawnPreset(slab, bp(2.2, 0.2, 2.8), quatYaw(0.35), colSlab)
	g.spawnPreset(slab, bp(3.9, 0.75, 2.9), quatYaw(-0.5), colCrate2)
	g.spawnPreset(anvil, bp(2.3, 0.46, 4.5), box3d.QuatIdentity, colAnvil)

	// Props that start on the raised platforms and inside the bay.
	g.spawnPreset(crate, bp(-11.0, 1.58, -6.0), quatYaw(0.2), colCrate2)
	g.spawnPreset(crate, bp(-11.8, 1.58, -5.2), quatYaw(-0.3), colCrate)
	g.spawnPreset(crate, bp(11.0, 2.43, -9.0), quatYaw(0.5), colCrate3)
	g.spawnPreset(ball, bp(12.0, 2.45, -9.3), box3d.QuatIdentity, colBall2)
	g.spawnPreset(barrel, bp(13.0, 0.62, 6.2), quatYaw(0.8), colBarrel)
	g.spawnPreset(crate, bp(14.2, 0.52, 7.2), quatYaw(-0.4), colCrate)

	// Scattered singles.
	scatter := []struct {
		x, y, z float32
		c       rl.Color
	}{
		{-4.2, 0.52, 6.1, colCrate2},
		{4.1, 0.52, 6.3, colCrate3},
		{8.0, 0.52, -3.2, colCrate},
		{-3.0, 0.52, -10.0, colCrate3},
		{17.0, 0.52, 1.0, colCrate2},
		{-16.0, 0.52, -2.0, colCrate},
	}
	for _, s := range scatter {
		g.spawnPreset(crate, bp(s.x, s.y, s.z), quatYaw(g.rng.Float32()*6.28), s.c)
	}
}

// resetWorld destroys the whole physics world and recreates the arena. This is
// also used by load so the hull database and body IDs start clean.
func (g *Game) resetWorld(withStarter bool, spawn box3d.Pos, yaw, pitch float32, fly bool) {
	if !g.world.IsNull() {
		g.world.Destroy()
	}

	g.world = newPhysicsWorld()
	g.applyGravity()
	g.entities = nil
	g.props = nil
	g.byBody = make(map[box3d.BodyID]*Entity)
	g.held = nil
	g.aim = nil
	g.holdDist = 0
	g.particles = nil
	g.waves = nil
	g.toasts = nil
	g.shake = 0
	g.acc = 0
	g.time = 0
	g.lastSpawn = nil
	g.toolKick = 0
	g.player = newPlayer(g.world, spawn, yaw, pitch, fly)
	g.buildArena()
	if withStarter {
		g.buildStarterProps()
	}
	g.updateCamera()
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

func (g *Game) eyePosition() rl.Vector3 {
	p := g.player.body.GetPosition()
	return rlv(p.X, p.Y+0.68, p.Z)
}

func (g *Game) forwardFlat() rl.Vector3 {
	return rlv(float32(math.Sin(float64(g.player.yaw))), 0, -float32(math.Cos(float64(g.player.yaw))))
}

func (g *Game) rightFlat() rl.Vector3 {
	return rlv(float32(math.Cos(float64(g.player.yaw))), 0, float32(math.Sin(float64(g.player.yaw))))
}

func (g *Game) cameraForward() rl.Vector3 {
	cp := float32(math.Cos(float64(g.player.pitch)))
	return rlv(
		float32(math.Sin(float64(g.player.yaw)))*cp,
		float32(math.Sin(float64(g.player.pitch))),
		-float32(math.Cos(float64(g.player.yaw)))*cp,
	)
}

func (g *Game) castRay(maxDist float32, mask uint64) box3d.RayResult {
	origin := g.eyePosition()
	dir := g.cameraForward()
	return g.world.CastRayClosest(toBp(origin), bv(dir.X*maxDist, dir.Y*maxDist, dir.Z*maxDist), queryFilter(mask))
}

func (g *Game) castPropRay(maxDist float32) (*Entity, box3d.RayResult) {
	res := g.castRay(maxDist, catProp)
	if !res.Hit || res.ShapeId.IsNull() {
		return nil, res
	}
	e := g.byBody[res.ShapeId.GetBody()]
	if e == nil || e.Static {
		return nil, res
	}
	return e, res
}

// defaultSpawnPoint returns a safe drop point for the spawn menu, preferring
// the current placement ghost and falling back to a spot a few metres ahead.
func (g *Game) defaultSpawnPoint() rl.Vector3 {
	if g.ghostOK {
		return g.ghostPos
	}
	eye := g.eyePosition()
	f := g.cameraForward()
	d := g.placementDistance()
	p := rlv(eye.X+f.X*d, eye.Y+f.Y*d, eye.Z+f.Z*d)
	pc := g.player.body.GetPosition()
	if vlen(p.X-pc.X, p.Y-pc.Y, p.Z-pc.Z) < 2.2 {
		p.X += f.X * 2.5
		p.Z += f.Z * 2.5
	}
	p.X = clampf(p.X, -28, 28)
	p.Y = clampf(p.Y, 0.4, 24)
	p.Z = clampf(p.Z, -28, 28)
	return p
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

func (g *Game) updateAim() {
	g.aim = nil
	if g.state != statePlaying || g.held != nil {
		return
	}
	e, res := g.castPropRay(28)
	g.aim = e
	if res.Hit {
		g.aimPoint = toRlP(res.Point)
		g.aimNormal = toRlV(res.Normal)
	} else {
		g.aimPoint = g.eyePosition()
		g.aimNormal = rlv(0, 1, 0)
	}
}

func (g *Game) updateGhost() {
	g.ghostOK = false
	// No placement preview while carrying something: the ghost would land on
	// the held prop and paint a translucent shape right in front of the camera.
	if g.state != statePlaying || g.held != nil {
		return
	}

	p := presets[g.selected]
	origin := g.eyePosition()
	dir := g.cameraForward()
	d := g.placementDistance()
	res := g.world.CastRayClosest(toBp(origin), bv(dir.X*40, dir.Y*40, dir.Z*40), queryFilter(catStatic|catProp))

	var pos rl.Vector3
	surface := false
	if res.Hit {
		hit := toRlP(res.Point)
		hitDist := lenRl(rlv(hit.X-origin.X, hit.Y-origin.Y, hit.Z-origin.Z))
		// Only snap to a surface when it is inside the wheel-controlled range;
		// otherwise the ghost floats at the selected distance, which makes the
		// wheel visibly useful in open air.
		if hitDist <= d {
			surface = true
			n := normRl(toRlV(res.Normal))
			support := p.support(toBv(n))
			pos = hit
			pos.X += n.X * (support + 0.03)
			pos.Y += n.Y * (support + 0.03)
			pos.Z += n.Z * (support + 0.03)
			g.ghostNormal = n
		}
	}
	if !surface {
		pos = rlv(origin.X+dir.X*d, origin.Y+dir.Y*d, origin.Z+dir.Z*d)
		g.ghostNormal = rlv(0, 1, 0)
	}

	pos.X = clampf(pos.X, -29, 29)
	pos.Z = clampf(pos.Z, -29, 29)
	pos.Y = clampf(pos.Y, 0.15, 26)
	g.ghostPos = pos

	eye := g.eyePosition()
	dist := lenRl(rlv(pos.X-eye.X, pos.Y-eye.Y, pos.Z-eye.Z))
	pc := g.player.body.GetPosition()
	playerDist := vlen(pos.X-pc.X, pos.Y-pc.Y, pos.Z-pc.Z)
	g.ghostOK = dist > 1.0 && dist < 41.0 && playerDist > 1.05
}

func (g *Game) spawnSelected() {
	if g.state != statePlaying || !g.ghostOK {
		return
	}
	if len(g.props) >= maxProps {
		g.toast("Prop limit reached", colWarn)
		return
	}
	p := presets[g.selected]
	e := g.spawnPreset(p, toBp(g.ghostPos), box3d.QuatIdentity, p.Color)
	if e != nil {
		g.propSerial++
		g.lastSpawn = e
		g.burst(g.ghostPos, e.Color, 10, 3.5, 0.09)
		g.wakeAround(g.ghostPos, 1.2)
	}
}

func (g *Game) spawnSelectedAt(pos rl.Vector3) {
	if g.state != statePlaying {
		return
	}
	if len(g.props) >= maxProps {
		g.toast("Prop limit reached", colWarn)
		return
	}
	p := presets[g.selected]
	support := p.support(bv(0, 1, 0))
	if pos.Y < support+0.06 {
		pos.Y = support + 0.06
	}
	pos.X = clampf(pos.X, -29, 29)
	pos.Z = clampf(pos.Z, -29, 29)
	e := g.spawnPreset(p, toBp(pos), box3d.QuatIdentity, p.Color)
	if e == nil {
		g.toast("Spawn failed", colDanger)
		return
	}
	g.lastSpawn = e
	g.burst(pos, e.Color, 10, 3.5, 0.09)
	g.wakeAround(pos, 1.4)
}

func (g *Game) grab() {
	if g.state != statePlaying {
		return
	}
	e, res := g.castPropRay(28)
	if e == nil {
		g.toast("No target in range", colMuted)
		return
	}
	if e.Frozen {
		g.toast("Target is frozen", colFreeze)
		return
	}
	g.held = e
	// Remember the grabbed point in body-local space so the prop pivots
	// around the grab point instead of its centre.
	hit := toRlP(res.Point)
	bp := e.position()
	q := e.Body.GetRotation()
	// inverse rotate hit-bp by q
	g.holdOffset = toBv(rl.Vector3RotateByQuaternion(rlv(hit.X-bp.X, hit.Y-bp.Y, hit.Z-bp.Z), rl.Quaternion{X: -q.V.X, Y: -q.V.Y, Z: -q.V.Z, W: q.S}))
	// The held prop is pulled to the wheel-controlled range. Very large props
	// keep a minimum clearance so they do not fill the camera.
	size := e.radiusXZ()
	g.holdDist = clampf(maxf(g.placementDistance(), 2.0+size*1.4), minPlaceDist, maxPlaceDist)
	g.spawnDist = g.holdDist
}

func (g *Game) throwHeld() {
	if g.held == nil {
		return
	}
	e := g.held
	dir := g.cameraForward()
	pv := g.player.body.GetLinearVelocity()
	v := box3d.Vec3{
		X: dir.X*22 + pv.X*0.35,
		Y: dir.Y*22 + pv.Y*0.25 + 1.5,
		Z: dir.Z*22 + pv.Z*0.35,
	}
	e.Body.SetLinearVelocity(v)
	e.Body.SetAngularVelocity(bv(
		(g.rng.Float32()-0.5)*7,
		(g.rng.Float32()-0.5)*7,
		(g.rng.Float32()-0.5)*7,
	))
	g.held = nil
	if g.shake < 0.10 {
		g.shake = 0.10
	}
	g.burst(e.position(), e.Color, 12, 4.5, 0.08)
}

func (g *Game) updateHeld(dt float32) {
	if g.held == nil {
		return
	}
	if g.byBody[g.held.Body] == nil {
		g.held = nil
		return
	}

	e := g.held
	// Push/pull while holding E (GMod style).
	if rl.IsKeyDown(rl.KeyE) {
		if rl.IsKeyDown(rl.KeyW) {
			g.holdDist = clampf(g.holdDist+0.15, minPlaceDist, maxPlaceDist)
		}
		if rl.IsKeyDown(rl.KeyS) {
			g.holdDist = clampf(g.holdDist-0.15, minPlaceDist, maxPlaceDist)
		}
	}
	// When rotating, apply rotation directly around the grab point instead of
	// spinning around the centre of mass.
	if rl.IsKeyDown(rl.KeyE) || rl.IsKeyDown(rl.KeyR) {
		md := rl.GetMouseDelta()
		right := toBv(normRl(rl.Vector3CrossProduct(g.cameraForward(), rlv(0, 1, 0))))
		scale := float32(0.003)
		q := e.Body.GetRotation()
		yawQ := box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, -md.X*scale)
		pitchQ := box3d.MakeQuatFromAxisAngle(right, -md.Y*scale)
		q = box3d.MulQuat(pitchQ, box3d.MulQuat(yawQ, q))
		e.Body.SetTransform(e.Body.GetPosition(), q)
		e.Body.SetAngularVelocity(box3d.Vec3Zero)
	}
	target := g.eyePosition()
	dir := g.cameraForward()
	target.X += dir.X * g.holdDist
	target.Y += dir.Y * g.holdDist
	target.Z += dir.Z * g.holdDist
	// Float the prop slightly below the crosshair so the player can still see
	// ahead while carrying wide objects such as slabs and anvils.
	target.Y -= 0.18 + e.radiusXZ()*0.20

	// Pivot around the grabbed point: offset the target by the (rotated)
	// grab offset so the grabbed point, not the centre, tracks the crosshair.
	q := e.Body.GetRotation()
	rotOff := rl.Vector3RotateByQuaternion(toRlV(g.holdOffset), rlQuat(q))
	target.X -= rotOff.X
	target.Y -= rotOff.Y
	target.Z -= rotOff.Z

	p := e.Body.GetPosition()
	dx := target.X - p.X
	dy := target.Y - p.Y
	dz := target.Z - p.Z
	dist := vlen(dx, dy, dz)
	if dist > 14 {
		g.held = nil
		return
	}

	tx, ty, tz := float32(0), float32(0), float32(0)
	if dist > 1e-4 {
		speed := clampf(dist*18, 0, 40)
		tx = dx / dist * speed
		ty = dy / dist * speed
		tz = dz / dist * speed
	}
	v := e.Body.GetLinearVelocity()
	blend := smoothing(30, dt)
	e.Body.SetLinearVelocity(box3d.Vec3{
		X: lerpf(v.X, tx, blend),
		Y: lerpf(v.Y, ty, blend),
		Z: lerpf(v.Z, tz, blend),
	})
	av := e.Body.GetAngularVelocity()
	if !rl.IsKeyDown(rl.KeyR) {
		e.Body.SetAngularVelocity(bv(av.X*0.9, av.Y*0.9, av.Z*0.9))
	}
}

func (g *Game) deleteAimed() {
	if g.state != statePlaying {
		return
	}
	e, _ := g.castPropRay(28)
	if e == nil {
		g.toast("Aim at a prop to delete it", colMuted)
		return
	}
	c := e.Color
	pos := e.position()
	g.removeEntity(e)
	g.burst(pos, c, 16, 5.5, 0.11)
}

func (g *Game) removeEntity(e *Entity) {
	if e == nil {
		return
	}
	if g.held == e {
		g.held = nil
	}
	if g.aim == e {
		g.aim = nil
	}

	e.Body.Destroy()
	delete(g.byBody, e.Body)

	for i, x := range g.entities {
		if x == e {
			g.entities = append(g.entities[:i], g.entities[i+1:]...)
			break
		}
	}
	if !e.Static {
		for i, x := range g.props {
			if x == e {
				g.props = append(g.props[:i], g.props[i+1:]...)
				break
			}
		}
	}
}

func (g *Game) removeAllProps() {
	props := append([]*Entity(nil), g.props...)
	for _, e := range props {
		g.removeEntity(e)
	}
	g.held = nil
}

func (g *Game) blast() {
	if g.state != statePlaying {
		return
	}

	center := g.eyePosition()
	res := g.castRay(40, catStatic|catProp)
	if res.Hit {
		center = toRlP(res.Point)
	} else {
		f := g.cameraForward()
		center.X += f.X * 6
		center.Y += f.Y * 6
		center.Z += f.Z * 6
	}

	const radius = 7.5
	const strength = 18.0
	g.held = nil
	for _, e := range g.props {
		if e.Frozen {
			continue
		}
		p := e.position()
		dx := p.X - center.X
		dy := p.Y - center.Y
		dz := p.Z - center.Z
		d := vlen(dx, dy, dz)
		if d < 1e-4 || d > radius {
			continue
		}
		falloff := 1 - d/radius
		nx, ny, nz := dx/d, dy/d, dz/d
		ny += 0.35
		nl := vlen(nx, ny, nz)
		if nl < 1e-4 {
			continue
		}
		speed := strength * falloff
		mass := e.Body.GetMass()
		if mass <= 0 {
			mass = 1
		}
		e.Body.ApplyLinearImpulseToCenter(bv(
			nx/nl*mass*speed,
			ny/nl*mass*speed,
			nz/nl*mass*speed,
		), true)
		e.Body.ApplyAngularImpulse(bv(
			(g.rng.Float32()-0.5)*mass*falloff*2,
			(g.rng.Float32()-0.5)*mass*falloff*2,
			(g.rng.Float32()-0.5)*mass*falloff*2,
		), true)
	}

	g.waves = append(g.waves, &Shockwave{
		Center: center, Radius: 0.25, MaxRadius: radius,
		Life: 0.55, MaxLife: 0.55, Color: colEnergy,
	})
	g.burst(center, colEnergy, 34, 13, 0.13)
	g.shake = 0.50
}

func (g *Game) wakeAround(center rl.Vector3, radius float32) {
	for _, e := range g.props {
		if e.Frozen {
			continue
		}
		p := e.position()
		if vlen(p.X-center.X, p.Y-center.Y, p.Z-center.Z) < radius {
			e.Body.SetAwake(true)
		}
	}
}

func (g *Game) enforceBounds() {
	props := append([]*Entity(nil), g.props...)
	for _, e := range props {
		p := e.Body.GetPosition()
		if p.Y < -12 || absf(p.X) > 70 || absf(p.Z) > 70 {
			g.removeEntity(e)
		}
	}
}

func (g *Game) respawnPlayer() {
	g.player.respawn(g)
	g.shake = 0.25
	g.toast("Recovered - watch the drop!", colWarn)
}

// ---------------------------------------------------------------------------
// Tool actions that operate on the entity list
// ---------------------------------------------------------------------------

// spawnCopy creates a byte-for-byte functional copy of an existing entity.
func (g *Game) spawnCopy(e *Entity, pos box3d.Pos, rot box3d.Quat) *Entity {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case PropBox:
		if len(e.Hulls) > 0 {
			return g.addHullsEntity(false, pos, rot, e.Half, e.Color, e.Name, e.Density, e.Friction, e.Restitution, e.Hulls, e.Model, e.HasModel)
		}
		if e.Hull != nil {
			return g.addHullEntity(false, pos, rot, e.Half, e.Color, e.Name, e.Density, e.Friction, e.Restitution, e.Hull, e.Model, e.HasModel)
		}
		ne := g.addBoxEntity(false, pos, rot, e.Half, e.Color, e.Name, e.Density, e.Friction, e.Restitution)
		ne.Model = e.Model
		ne.HasModel = e.HasModel
		ne.Hull = e.Hull
		ne.Hulls = e.Hulls
		ne.Mesh = e.Mesh
		return ne
	case PropSphere:
		return g.addSphereEntity(pos, rot, e.Radius, e.Color, e.Name, e.Density, e.Friction, e.Restitution)
	case PropCylinder:
		return g.addCylinderEntity(pos, rot, e.Radius, e.Height, e.Color, e.Name, e.Density, e.Friction, e.Restitution)
	}
	return nil
}

func (g *Game) setFrozen(e *Entity, frozen bool) {
	if e == nil || e.Static {
		return
	}
	if e.Frozen == frozen {
		return
	}
	e.Frozen = frozen
	if frozen {
		if g.held == e {
			g.held = nil
		}
		e.Body.SetLinearVelocity(box3d.Vec3Zero)
		e.Body.SetAngularVelocity(box3d.Vec3Zero)
		e.Body.SetType(box3d.StaticBody)
	} else {
		e.Body.SetType(box3d.DynamicBody)
		e.Body.SetGravityScale(1)
		e.Body.SetAwake(true)
	}
}

func (g *Game) toggleFrozen(e *Entity) {
	if e == nil || e.Static {
		g.toast("No target to freeze", colMuted)
		return
	}
	g.setFrozen(e, !e.Frozen)
	if e.Frozen {
		g.toast("Frozen: "+e.Name, colFreeze)
	} else {
		g.toast("Thawed: "+e.Name, colFreeze)
	}
}

func (g *Game) setAllFrozen(frozen bool) int {
	n := 0
	for _, e := range g.props {
		if e.Frozen != frozen {
			g.setFrozen(e, frozen)
			n++
		}
	}
	return n
}

func (g *Game) frozenCount() int {
	n := 0
	for _, e := range g.props {
		if e.Frozen {
			n++
		}
	}
	return n
}

func (g *Game) removeInRadius(center rl.Vector3, radius float32) int {
	if radius <= 0 {
		return 0
	}
	props := append([]*Entity(nil), g.props...)
	n := 0
	for _, e := range props {
		p := e.position()
		if vlen(p.X-center.X, p.Y-center.Y, p.Z-center.Z) <= radius {
			g.removeEntity(e)
			n++
		}
	}
	if n > 0 {
		g.burst(center, colDanger, minInt(10+n, 36), 7, 0.10)
		g.shake = 0.22
	}
	return n
}

func (g *Game) duplicateAimed(atAimPoint bool) {
	if g.state != statePlaying {
		return
	}
	e, res := g.castPropRay(28)
	if e == nil {
		g.toast("No target to duplicate", colMuted)
		return
	}
	if len(g.props) >= maxProps {
		g.toast("Prop limit reached", colWarn)
		return
	}

	pos := e.Body.GetPosition()
	rot := e.Body.GetRotation()
	if atAimPoint && res.Hit {
		p := toRlP(res.Point)
		n := normRl(toRlV(res.Normal))
		off := e.radiusXZ()*2 + 0.08
		p.X += n.X * off
		p.Y += n.Y * off
		p.Z += n.Z * off
		pos = toBp(p)
	} else {
		// Offset the copy along the view direction so it does not spawn inside
		// the original body and get violently ejected.
		f := g.cameraForward()
		off := e.radiusXZ()*2 + 0.12
		pos = bp(pos.X+f.X*off, pos.Y+f.Y*off+0.10, pos.Z+f.Z*off)
	}

	copy := g.spawnCopy(e, pos, rot)
	if copy == nil {
		g.toast("Duplicate failed", colDanger)
		return
	}
	g.lastSpawn = copy
	g.burst(copy.position(), copy.Color, 8, 3.0, 0.08)
}

func (g *Game) launchAimed() {
	if g.state != statePlaying {
		return
	}
	e, _ := g.castPropRay(32)
	if e == nil {
		g.toast("No target to launch", colMuted)
		return
	}
	if e.Frozen {
		g.toast("Target is frozen", colFreeze)
		return
	}
	dir := g.cameraForward()
	mass := e.Body.GetMass()
	if mass <= 0 {
		mass = 1
	}
	e.Body.ApplyLinearImpulseToCenter(bv(
		dir.X*mass*24,
		(dir.Y+0.22)*mass*24,
		dir.Z*mass*24,
	), true)
	e.Body.ApplyAngularImpulse(bv(
		(g.rng.Float32()-0.5)*mass*4,
		(g.rng.Float32()-0.5)*mass*4,
		(g.rng.Float32()-0.5)*mass*4,
	), true)
	g.burst(e.position(), colEnergy, 12, 5, 0.09)
	g.shake = 0.12
}
