// 引擎将world和canvas组合：管理相机，按相机位置加载/卸载chunk，把chunk变成网格交给canvas画。
//
// 相机位置保存在engine里（物理坐标，float64精度）。传给canvas时所有坐标都先减去相机位置，
// 所以canvas的相机永远在原点，离原点很远时也不会因为显卡的float32精度而抖动。
//
// 一帧的典型写法：
//
//	e.Set_camera(pos, dir)
//	e.Update()                // 加载chunk、更新网格，最多花e.Budget的时间
//	e.Begin_frame()
//	e.Draw_world()
//	e.Add_line3d(a, b, color) // 可选：额外画的东西
//	e.End_frame()
package engine

import (
	"math"
	"slices"
	"time"

	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

type Engine struct {
	World  *world.World
	Canvas canvas.Canvas
	// 水平方向加载多少个chunk（半径）
	View_distance int64
	// Update每帧最多花多少时间加载chunk和生成网格
	Budget time.Duration
	// 上一帧的统计，调试用
	Stats Stats

	cam_pos world.PhysicalPosition3D
	// 单位向量
	cam_dir canvas.Canvas3D
	meshes  map[world.CubeID]*chunk_mesh
	// 需要重新生成网格的chunk
	dirty   map[world.CubeID]bool
	next_id canvas.MeshID
}

type Stats struct {
	// 已加载的chunk、有网格的chunk、这一帧画了的chunk
	Loaded, Meshes, Drawn int
	// 邻居都加载好了、还在排队等着生成网格的chunk
	Pending int
}

type chunk_mesh struct {
	id canvas.MeshID
	// 网格是空的（比如全是空气或者全埋在地下）
	empty bool
}

// 新建一个Engine，c必须已经Init过。相机默认在(10, 10, 10)看向原点
func New_engine(w *world.World, c canvas.Canvas) *Engine {
	e := &Engine{
		World:         w,
		Canvas:        c,
		View_distance: 8,
		Budget:        8 * time.Millisecond,
		meshes:        map[world.CubeID]*chunk_mesh{},
		dirty:         map[world.CubeID]bool{},
	}
	w.On_change(e.mark_dirty)
	e.Look_at(world.PhysicalPosition3D{X: 10, Y: 10, Z: 10}, world.PhysicalPosition3D{})
	return e
}

// ---------------------------------------------------------------- 相机

// 设置相机在世界里的位置和朝向，direction只取方向，长度为0时只改位置
func (e *Engine) Set_camera(position world.PhysicalPosition3D, direction world.PhysicalPosition3D) {
	e.cam_pos = position
	l := direction.Length()
	if l == 0 {
		return
	}
	e.cam_dir = canvas.Canvas3D{X: direction.X / l, Y: direction.Y / l, Z: direction.Z / l}
	e.Canvas.Camera3D_loc_and_dir(canvas.Canvas3D{}, e.cam_dir)
}

// 让相机在position看向target
func (e *Engine) Look_at(position world.PhysicalPosition3D, target world.PhysicalPosition3D) {
	e.Set_camera(position, target.Sub(position))
}

// 相机的位置和朝向（单位向量）
func (e *Engine) Get_camera() (world.PhysicalPosition3D, world.PhysicalPosition3D) {
	return e.cam_pos, world.PhysicalPosition3D{X: e.cam_dir.X, Y: e.cam_dir.Y, Z: e.cam_dir.Z}
}

// 物理坐标 -> canvas坐标：减去相机位置
func (e *Engine) To_canvas(p world.PhysicalPosition3D) canvas.Canvas3D {
	return canvas.Canvas3D{X: p.X - e.cam_pos.X, Y: p.Y - e.cam_pos.Y, Z: p.Z - e.cam_pos.Z}
}

// ---------------------------------------------------------------- 加载

// 加载相机附近的chunk、卸载远处的chunk、重新生成改动过的chunk的网格，最多花e.Budget的时间
func (e *Engine) Update() {
	start := time.Now()
	center := e.cam_pos
	r := e.View_distance
	cx, cz := chunk_coord(center.X), chunk_coord(center.Z)

	// 远处的卸载（多留几个chunk，在边界上走来走去时不会反复加载）
	for _, pos := range e.World.Loaded_chunks() {
		if dx, dz := pos.X-cx, pos.Z-cz; dx*dx+dz*dz > (r+3)*(r+3) {
			e.unload(pos)
		}
	}

	// 近的先加载。多加载一圈：生成网格需要邻居，这样视距以内的chunk都能有网格
	lo, hi := e.World.Generator().Vertical_range()
	lr := r + 1
	var missing []world.CubeID
	for dz := -lr; dz <= lr; dz++ {
		for dx := -lr; dx <= lr; dx++ {
			if dx*dx+dz*dz > lr*lr {
				continue
			}
			for y := lo; y <= hi; y++ {
				pos := world.CubeID{X: cx + dx, Y: y, Z: cz + dz, Depth: world.CHUNK_DEPTH}
				if !e.World.Is_loaded(pos) {
					missing = append(missing, pos)
				}
			}
		}
	}
	e.sort_by_distance(missing)
	for _, pos := range missing {
		if time.Since(start) > e.Budget/2 {
			break
		}
		e.World.Load_chunk(pos)
		e.dirty[pos] = true
	}

	// 生成网格：6个邻居都加载了才能知道边上的面要不要画
	var todo []world.CubeID
	for pos := range e.dirty {
		if e.World.Is_loaded(pos) && e.neighbors_loaded(pos) {
			todo = append(todo, pos)
		}
	}
	e.sort_by_distance(todo)
	built := 0
	for _, pos := range todo {
		if built > 0 && time.Since(start) > e.Budget {
			break
		}
		e.build_mesh(pos)
		delete(e.dirty, pos)
		built++
	}

	e.Stats.Loaded = len(e.World.Loaded_chunks())
	e.Stats.Pending = len(todo) - built
	e.Stats.Meshes = 0
	for _, m := range e.meshes {
		if !m.empty {
			e.Stats.Meshes++
		}
	}
}

// 相机周围radius个chunk以内是否都加载好、网格也生成好了（刚进游戏时等它变成true）
func (e *Engine) Ready(radius int64) bool {
	cx, cy, cz := chunk_coord(e.cam_pos.X), chunk_coord(e.cam_pos.Y), chunk_coord(e.cam_pos.Z)
	for dy := -radius; dy <= radius; dy++ {
		for dz := -radius; dz <= radius; dz++ {
			for dx := -radius; dx <= radius; dx++ {
				pos := world.CubeID{X: cx + dx, Y: cy + dy, Z: cz + dz, Depth: world.CHUNK_DEPTH}
				if !e.World.In_range(pos) {
					continue
				}
				if _, ok := e.meshes[pos]; !ok || e.dirty[pos] {
					return false
				}
			}
		}
	}
	return true
}

func chunk_coord(v float64) int64 {
	return int64(math.Floor(v / world.CHUNK_SIZE))
}

func (e *Engine) sort_by_distance(list []world.CubeID) {
	c := e.cam_pos
	d2 := func(pos world.CubeID) float64 {
		p := pos.Center().Sub(c)
		return p.X*p.X + p.Y*p.Y + p.Z*p.Z
	}
	slices.SortFunc(list, func(a, b world.CubeID) int {
		da, db := d2(a), d2(b)
		switch {
		case da < db:
			return -1
		case da > db:
			return 1
		}
		return 0
	})
}

func (e *Engine) neighbors_loaded(pos world.CubeID) bool {
	for _, d := range [6][3]int64{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
		n := pos.Neighbor(d[0], d[1], d[2])
		if e.World.In_range(n) && !e.World.Is_loaded(n) {
			return false
		}
	}
	return true
}

func (e *Engine) unload(pos world.CubeID) {
	if m, ok := e.meshes[pos]; ok {
		e.Canvas.Delete_mesh(m.id)
		delete(e.meshes, pos)
	}
	delete(e.dirty, pos)
	e.World.Unload_chunk(pos)
}

// world改动了id这个cube：它所在的chunk要重新生成网格，碰到chunk边上时隔壁的chunk也要
func (e *Engine) mark_dirty(id world.CubeID) {
	pos := id.Chunk()
	e.dirty[pos] = true
	lo, hi := id.Min(), id.Max()
	clo, chi := pos.Min(), pos.Max()
	touch := func(d [3]int64) {
		if n := pos.Neighbor(d[0], d[1], d[2]); e.World.Is_loaded(n) {
			e.dirty[n] = true
		}
	}
	if lo.X == clo.X {
		touch([3]int64{-1, 0, 0})
	}
	if hi.X == chi.X {
		touch([3]int64{1, 0, 0})
	}
	if lo.Y == clo.Y {
		touch([3]int64{0, -1, 0})
	}
	if hi.Y == chi.Y {
		touch([3]int64{0, 1, 0})
	}
	if lo.Z == clo.Z {
		touch([3]int64{0, 0, -1})
	}
	if hi.Z == chi.Z {
		touch([3]int64{0, 0, 1})
	}
}

func (e *Engine) build_mesh(pos world.CubeID) {
	m, ok := e.meshes[pos]
	if !ok {
		e.next_id++
		m = &chunk_mesh{id: e.next_id}
		e.meshes[pos] = m
	}
	quads := Build_quads(e.World, pos)
	m.empty = len(quads) == 0
	e.Canvas.Set_mesh(m.id, quads)
}

// ---------------------------------------------------------------- 可见性

// 平面过相机位置，法线指向视锥里面
type frustum_plane struct {
	normal canvas.Canvas3D
}

// 四个侧面加一个相机所在的近平面（只剔除相机背后的东西），远平面不剔除
func (e *Engine) frustum_planes() [5]frustum_plane {
	tl, tr, bl, br := e.Canvas.Get_frustum_corner_rays(e.cam_dir)
	side := func(a, b canvas.Canvas3D) frustum_plane {
		n := cross(a, b)
		l := math.Sqrt(dot(n, n))
		n = canvas.Canvas3D{X: n.X / l, Y: n.Y / l, Z: n.Z / l}
		// 法线朝向视线这一侧
		if dot(n, e.cam_dir) < 0 {
			n = canvas.Canvas3D{X: -n.X, Y: -n.Y, Z: -n.Z}
		}
		return frustum_plane{n}
	}
	return [5]frustum_plane{side(tl, bl), side(tr, br), side(tl, tr), side(bl, br), {e.cam_dir}}
}

// 以center为中心、radius为半径的球是否有一部分在视锥里
func (e *Engine) sphere_visible(planes [5]frustum_plane, center world.PhysicalPosition3D, radius float64) bool {
	d := e.To_canvas(center)
	for _, p := range planes {
		if dot(p.normal, d) < -radius {
			return false
		}
	}
	return true
}

// chunk的外接球半径
var chunk_radius = world.CHUNK_SIZE * math.Sqrt(3) / 2

// ---------------------------------------------------------------- 绘制

// 清空canvas，开始新的一帧
func (e *Engine) Begin_frame() {
	e.Canvas.Clean()
}

// 把视锥内所有chunk的网格画到canvas上
func (e *Engine) Draw_world() {
	planes := e.frustum_planes()
	e.Stats.Drawn = 0
	for pos, m := range e.meshes {
		if m.empty || !e.sphere_visible(planes, pos.Center(), chunk_radius) {
			continue
		}
		e.Canvas.Add_mesh(m.id, e.To_canvas(pos.Min()))
		e.Stats.Drawn++
	}
}

// 世界里的3D线条，用物理坐标
func (e *Engine) Add_line3d(start world.PhysicalPosition3D, end world.PhysicalPosition3D, c canvas.Color) {
	e.Canvas.Add_line3d(e.To_canvas(start), e.To_canvas(end), c)
}

// 长方体的12条边，lo和hi是两个对角
func (e *Engine) Add_box_lines(lo world.PhysicalPosition3D, hi world.PhysicalPosition3D, c canvas.Color) {
	p := func(x, y, z bool) world.PhysicalPosition3D {
		q := lo
		if x {
			q.X = hi.X
		}
		if y {
			q.Y = hi.Y
		}
		if z {
			q.Z = hi.Z
		}
		return q
	}
	for _, a := range []bool{false, true} {
		for _, b := range []bool{false, true} {
			e.Add_line3d(p(false, a, b), p(true, a, b), c)
			e.Add_line3d(p(a, false, b), p(a, true, b), c)
			e.Add_line3d(p(a, b, false), p(a, b, true), c)
		}
	}
}

// 世界里一个会动的cube（不在world里），用物理坐标
func (e *Engine) Add_cube(center world.PhysicalPosition3D, side float64, images canvas.Images, rot world.Rotation3D) {
	e.Canvas.Add_cube(side, e.To_canvas(center), images, rot)
}

// 把这一帧画出来
func (e *Engine) End_frame() {
	e.Canvas.Draw()
}

func dot(a, b canvas.Canvas3D) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func cross(a, b canvas.Canvas3D) canvas.Canvas3D {
	return canvas.Canvas3D{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
