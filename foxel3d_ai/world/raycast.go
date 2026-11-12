package world

import "math"

// 射线打到的东西
type Hit struct {
	// 打到的叶子cube
	Leaf Leaf
	// 射线进入这个cube的点
	Point PhysicalPosition3D
	// 打到的那个面朝外的方向，比如{0, 1, 0}是上面。射线从cube里面出发时是{0, 0, 0}
	Normal [3]int64
	// 从起点到Point的距离
	Distance float64
}

// 从origin沿dir找第一个hit(方块)为true的叶子cube，最远max_distance。
// 按MIN_DEPTH的格子一格一格走，碰到没加载的chunk就停下
func (w *World) Raycast(origin PhysicalPosition3D, dir PhysicalPosition3D, max_distance float64, hit func(BlockID) bool) (Hit, bool) {
	l := dir.Length()
	if l == 0 {
		return Hit{}, false
	}
	d := [3]float64{dir.X / l, dir.Y / l, dir.Z / l}
	o := [3]float64{origin.X, origin.Y, origin.Z}
	s := MIN_DEPTH.Side()
	start := Cube_id_at(origin, MIN_DEPTH)
	cell := [3]int64{start.X, start.Y, start.Z}

	var step [3]int64
	var t_max, t_delta [3]float64
	for a := range 3 {
		switch {
		case d[a] > 0:
			step[a] = 1
			t_max[a] = (float64(cell[a]+1)*s - o[a]) / d[a]
		case d[a] < 0:
			step[a] = -1
			t_max[a] = (float64(cell[a])*s - o[a]) / d[a]
		default:
			t_max[a] = math.Inf(1)
		}
		t_delta[a] = s / math.Abs(d[a])
	}

	t := 0.0
	var normal [3]int64
	for t <= max_distance {
		id := CubeID{cell[0], cell[1], cell[2], MIN_DEPTH}
		leaf, loaded := w.Leaf_at(id.Center())
		if !loaded {
			return Hit{}, false
		}
		if hit(leaf.Block) {
			p := PhysicalPosition3D{o[0] + d[0]*t, o[1] + d[1]*t, o[2] + d[2]*t}
			return Hit{leaf, p, normal, t}, true
		}
		a := 0
		if t_max[1] < t_max[a] {
			a = 1
		}
		if t_max[2] < t_max[a] {
			a = 2
		}
		t = t_max[a]
		cell[a] += step[a]
		t_max[a] += t_delta[a]
		normal = [3]int64{}
		normal[a] = -step[a]
	}
	return Hit{}, false
}
