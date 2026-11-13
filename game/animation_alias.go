package game

// Aliases for geometry/hull building primitives used by the game.
// The implementations live in the animation package so they can be reused and
// tested without duplicating code in the game package.

import (
	"sandbox3d/animation"
)

type (
	BoneHull     = animation.BoneHull
	BoneCollider = animation.BoneCollider
)

var (
	BuildModelHulls      = animation.BuildModelHulls
	BuildBoneHulls       = animation.BuildBoneHulls
	SplitIntoLeaves      = animation.SplitIntoLeaves
	BuildHullsFromPoints = animation.BuildHullsFromPoints
	CreateLeafHull       = animation.CreateLeafHull
	HullBounds           = animation.HullBounds
	HullLongestAxis      = animation.HullLongestAxis
	HullAxisValue        = animation.HullAxisValue
	HullPartition        = animation.HullPartition
	BuildModelHull       = animation.BuildModelHull
)
