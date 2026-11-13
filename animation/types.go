package animation

import box3d "github.com/gataengine/box3d-go"

// BoneHull is one convex-decomposition leaf belonging to a single bone. Points
// are stored in raw (un-centered) model space so they can be posed with the
// bone matrix and then centered by the model transform at runtime.
type BoneHull struct {
	Bone   int32
	Points []box3d.Vec3
}

// BoneCollider is a cheap per-bone collision proxy. The geometry is stored in
// raw model space and only its transform changes at runtime, so the expensive
// hull cook happens once when the model loads instead of every animation frame.
//
// A collider is either a sphere (Sphere == true) or a capsule between A and B.
// When UseHull is true, the primitive fit was too loose (e.g. a boxy torso
// fit as a capsule would include empty rounded corners) and the exact convex
// hull is used instead. Hull bones are re-cooked per frame but only for the
// few bones where primitives fit poorly.
type BoneCollider struct {
	Bone    int32
	Sphere  bool
	Center  box3d.Vec3 // sphere centre, raw model space
	A       box3d.Vec3 // capsule start, raw model space
	B       box3d.Vec3 // capsule end, raw model space
	Radius  float32
	UseHull bool // true when the exact hull should be used instead of this primitive

	// Runtime cache used to skip shape updates when the pose barely changed.
	LastA   box3d.Vec3
	LastB   box3d.Vec3
	LastC   box3d.Vec3
	LastR   float32
	HasLast bool
}
