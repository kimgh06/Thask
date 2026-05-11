package service

import (
	"math"
)

type rectBox struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

type routeCell struct {
	X int
	Y int
}

// snapToGrid rounds a value to the nearest grid point.
func snapToGrid(v float64) float64 {
	return math.Round(v/gridSize) * gridSize
}

// siftingCrossMin reorders nodes within each layer to minimize edge crossings.
// For each node, it tries ALL possible positions in its layer and picks the one
// with the fewest crossings. Much more powerful than barycentric/adjacent exchange.
func pointInRect(p Point, box rectBox) bool {
	return p.X >= box.Left && p.X <= box.Right && p.Y >= box.Top && p.Y <= box.Bottom
}

func orientation(a, b, c Point) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func onSegment(a, b, p Point) bool {
	const eps = 1e-6
	return p.X >= math.Min(a.X, b.X)-eps &&
		p.X <= math.Max(a.X, b.X)+eps &&
		p.Y >= math.Min(a.Y, b.Y)-eps &&
		p.Y <= math.Max(a.Y, b.Y)+eps
}

func segmentsIntersect(a1, a2, b1, b2 Point) bool {
	const eps = 1e-6
	o1 := orientation(a1, a2, b1)
	o2 := orientation(a1, a2, b2)
	o3 := orientation(b1, b2, a1)
	o4 := orientation(b1, b2, a2)

	if (o1 > eps && o2 < -eps || o1 < -eps && o2 > eps) &&
		(o3 > eps && o4 < -eps || o3 < -eps && o4 > eps) {
		return true
	}
	if math.Abs(o1) <= eps && onSegment(a1, a2, b1) {
		return true
	}
	if math.Abs(o2) <= eps && onSegment(a1, a2, b2) {
		return true
	}
	if math.Abs(o3) <= eps && onSegment(b1, b2, a1) {
		return true
	}
	if math.Abs(o4) <= eps && onSegment(b1, b2, a2) {
		return true
	}
	return false
}

func segmentIntersectsRect(a, b Point, box rectBox) bool {
	if pointInRect(a, box) || pointInRect(b, box) {
		return true
	}

	topLeft := Point{X: box.Left, Y: box.Top}
	topRight := Point{X: box.Right, Y: box.Top}
	bottomRight := Point{X: box.Right, Y: box.Bottom}
	bottomLeft := Point{X: box.Left, Y: box.Bottom}

	return segmentsIntersect(a, b, topLeft, topRight) ||
		segmentsIntersect(a, b, topRight, bottomRight) ||
		segmentsIntersect(a, b, bottomRight, bottomLeft) ||
		segmentsIntersect(a, b, bottomLeft, topLeft)
}

func polylineIntersectsRect(points []Point, box rectBox) bool {
	for i := 0; i < len(points)-1; i++ {
		if segmentIntersectsRect(points[i], points[i+1], box) {
			return true
		}
	}
	return false
}

func pointsApproxEqual(a, b Point) bool {
	const eps = 1e-6
	return math.Abs(a.X-b.X) <= eps && math.Abs(a.Y-b.Y) <= eps
}

func polylineLength(points []Point) float64 {
	total := 0.0
	for i := 0; i < len(points)-1; i++ {
		total += math.Hypot(points[i+1].X-points[i].X, points[i+1].Y-points[i].Y)
	}
	return total
}

func interpolatePoint(a, b Point, t float64) Point {
	return Point{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
	}
}

func interpolateYAtX(x1, y1, x2, y2, x float64) float64 {
	if x2 == x1 {
		return (y1 + y2) / 2
	}
	t := (x - x1) / (x2 - x1)
	return y1 + (y2-y1)*t
}

func edgeYRangeAcrossRect(x1, y1, x2, y2, left, right float64) (float64, float64, bool) {
	if x1 == x2 {
		if x1 < left || x1 > right {
			return 0, 0, false
		}
		if y1 < y2 {
			return y1, y2, true
		}
		return y2, y1, true
	}
	segLeft := math.Max(left, math.Min(x1, x2))
	segRight := math.Min(right, math.Max(x1, x2))
	if segLeft > segRight {
		return 0, 0, false
	}
	yLeft := interpolateYAtX(x1, y1, x2, y2, segLeft)
	yRight := interpolateYAtX(x1, y1, x2, y2, segRight)
	if yLeft < yRight {
		return yLeft, yRight, true
	}
	return yRight, yLeft, true
}

func clampFloat(v, minV, maxV float64) float64 {
	if minV > maxV {
		return (minV + maxV) / 2
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func translateRectBox(box rectBox, dx, dy float64) rectBox {
	return rectBox{
		Left:   box.Left + dx,
		Right:  box.Right + dx,
		Top:    box.Top + dy,
		Bottom: box.Bottom + dy,
	}
}

func inflateRectBox(box rectBox, padX, padY float64) rectBox {
	return rectBox{
		Left:   box.Left - padX,
		Right:  box.Right + padX,
		Top:    box.Top - padY,
		Bottom: box.Bottom + padY,
	}
}
