package engine

import (
	"testing"

	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

// 什么都不画的canvas，只记下网格
type fake_canvas struct {
	meshes map[canvas.MeshID][]canvas.Quad
	drawn  []canvas.MeshID
}

func (f *fake_canvas) Init(int, int)                                                      {}
func (f *fake_canvas) Close()                                                             {}
func (f *fake_canvas) Add_cube(float64, canvas.Canvas3D, canvas.Images, world.Rotation3D) {}
func (f *fake_canvas) Add_line3d(canvas.Canvas3D, canvas.Canvas3D, canvas.Color)          {}
func (f *fake_canvas) Clean()                                                             { f.drawn = nil }
func (f *fake_canvas) Draw()                                                              {}
func (f *fake_canvas) Should_close() bool                                                 { return false }
func (f *fake_canvas) Get_window_size() (uint64, uint64)                                  { return 800, 600 }
func (f *fake_canvas) Get_frame_time() float64                                            { return 1.0 / 60 }
func (f *fake_canvas) Camera3D_loc_and_dir(canvas.Canvas3D, canvas.Canvas3D)              {}
func (f *fake_canvas) Camera3D_vis_angle(float64, float64, float64, bool)                 {}
func (f *fake_canvas) Set_background(canvas.Color)                                        {}
func (f *fake_canvas) Set_fog(canvas.Color, float64, float64)                             {}
func (f *fake_canvas) Delete_mesh(id canvas.MeshID)                                       { delete(f.meshes, id) }
func (f *fake_canvas) Add_mesh(id canvas.MeshID, _ canvas.Canvas3D)                       { f.drawn = append(f.drawn, id) }
func (f *fake_canvas) Add_2d_text(string, canvas.Canvas2D, canvas.Color)                  {}
func (f *fake_canvas) Measure_2d_text(string) canvas.Canvas2D                             { return canvas.Canvas2D{} }
func (f *fake_canvas) Add_2d_line(canvas.Canvas2D, canvas.Canvas2D, canvas.Color)         {}
func (f *fake_canvas) Add_2d_rect(canvas.Canvas2D, canvas.Canvas2D, canvas.Color)         {}
func (f *fake_canvas) Add_2d_image(canvas.Image, canvas.Canvas2D, canvas.Canvas2D)        {}
func (f *fake_canvas) Get_input() canvas.Input                                            { return canvas.Input{} }
func (f *fake_canvas) Set_cursor_locked(bool)                                             {}
func (f *fake_canvas) Is_cursor_locked() bool                                             { return false }
func (f *fake_canvas) Set_mesh(id canvas.MeshID, quads []canvas.Quad) {
	if len(quads) == 0 {
		delete(f.meshes, id)
		return
	}
	f.meshes[id] = quads
}

// 横着看过去60度视角的视锥
func (f *fake_canvas) Get_frustum_corner_rays(d canvas.Canvas3D) (canvas.Canvas3D, canvas.Canvas3D, canvas.Canvas3D, canvas.Canvas3D) {
	// 只在测试里朝-z看
	return canvas.Canvas3D{X: -0.5, Y: 0.5, Z: -1}, canvas.Canvas3D{X: 0.5, Y: 0.5, Z: -1},
		canvas.Canvas3D{X: -0.5, Y: -0.5, Z: -1}, canvas.Canvas3D{X: 0.5, Y: -0.5, Z: -1}
}

// 全是空气，竖直方向只有一层chunk
type empty struct{}

func (empty) Vertical_range() (int64, int64)                { return 0, 0 }
func (empty) Generate_chunk(world.CubeID, *world.ChunkData) {}

func setup(t *testing.T) (*world.World, *Engine, *fake_canvas) {
	t.Helper()
	w := world.New_world(empty{})
	c := &fake_canvas{meshes: map[canvas.MeshID][]canvas.Quad{}}
	e := New_engine(w, c)
	e.View_distance = 3
	e.Budget = 1 << 40
	e.Set_camera(world.PhysicalPosition3D{X: 8, Y: 8, Z: 8}, world.PhysicalPosition3D{Z: -1})
	e.Update()
	return w, e, c
}

func quads_of(t *testing.T, w *world.World, x, y, z int64) int {
	t.Helper()
	return len(Build_quads(w, world.Block_id(x, y, z).Chunk()))
}

func TestFaceCulling(t *testing.T) {
	w, _, _ := setup(t)
	w.Set_cube(world.Block_id(3, 3, 3), block.STONE)
	if n := quads_of(t, w, 3, 3, 3); n != 6 {
		t.Fatal("one block", n)
	}
	// 两块挨着：中间的两个面省掉
	w.Set_cube(world.Block_id(4, 3, 3), block.STONE)
	if n := quads_of(t, w, 3, 3, 3); n != 10 {
		t.Fatal("two blocks", n)
	}
	// 玻璃挨着玻璃省掉，树叶挨着树叶不省
	w.Set_cube(world.Block_id(3, 3, 3), block.GLASS)
	w.Set_cube(world.Block_id(4, 3, 3), block.GLASS)
	if n := quads_of(t, w, 3, 3, 3); n != 10 {
		t.Fatal("glass", n)
	}
	w.Set_cube(world.Block_id(3, 3, 3), block.OAK_LEAVES)
	w.Set_cube(world.Block_id(4, 3, 3), block.OAK_LEAVES)
	if n := quads_of(t, w, 3, 3, 3); n != 12 {
		t.Fatal("leaves", n)
	}
	// 不透明的石头挨着树叶：石头的面要画，树叶朝石头的面省掉
	w.Set_cube(world.Block_id(4, 3, 3), block.STONE)
	if n := quads_of(t, w, 3, 3, 3); n != 11 {
		t.Fatal("leaves+stone", n)
	}
}

func TestSmallCubes(t *testing.T) {
	w, _, _ := setup(t)
	w.Set_cube(world.Block_id(3, 3, 3), block.STONE)
	// 石头上面放一个1/4的小方块：小方块底面被挡住，石头顶面不能省（小方块盖不满）
	small := world.Cube_id_at(world.PhysicalPosition3D{X: 3.1, Y: 4.1, Z: 3.1}, world.MIN_DEPTH)
	w.Set_cube(small, block.DIRT)
	if n := quads_of(t, w, 3, 3, 3); n != 6+5 {
		t.Fatal("small on stone", n)
	}
	// 小方块的UV只占贴图的一部分
	for _, q := range Build_quads(w, small.Chunk()) {
		if q.Corners[0].Y > 4 && q.Corners[0].Y <= 4.25 && q.Corners[0].X >= 3 && q.Corners[0].X <= 3.25 {
			for _, uv := range q.UV {
				if uv.X > 0.25+1e-9 && uv.Y > 0.25+1e-9 && uv.X < 0.75 {
					t.Fatal("uv", q.UV)
				}
			}
		}
	}
	// 4个小方块把下半格铺满：石头顶面被挡住
	for _, p := range [][2]float64{{0, 0}, {0.5, 0}, {0, 0.5}, {0.5, 0.5}} {
		w.Set_cube(world.Cube_id_at(world.PhysicalPosition3D{X: 3 + p[0], Y: 4, Z: 3 + p[1]}, -1), block.DIRT)
	}
	// 每个1/2的小cube露出顶面和朝外的两个侧面；石头顶面被盖满了
	if n := quads_of(t, w, 3, 3, 3); n != 5+4*3 {
		t.Fatal("half layer", n)
	}
}

func TestChunkBorderAndDirty(t *testing.T) {
	w, e, c := setup(t)
	// 两个chunk交界处的两块：两边网格都要省掉中间的面
	w.Set_cube(world.Block_id(15, 3, 3), block.STONE)
	w.Set_cube(world.Block_id(16, 3, 3), block.STONE)
	e.Update()
	total := 0
	for _, q := range c.meshes {
		total += len(q)
	}
	if total != 10 {
		t.Fatal("border", total)
	}
	// 能看到相机前面的chunk，背后的不画
	e.Set_camera(world.PhysicalPosition3D{X: 16, Y: 8, Z: 40}, world.PhysicalPosition3D{Z: -1})
	e.Begin_frame()
	e.Draw_world()
	if e.Stats.Drawn != 2 {
		t.Fatal("drawn", e.Stats.Drawn)
	}
	e.Set_camera(world.PhysicalPosition3D{X: 8, Y: 8, Z: -40}, world.PhysicalPosition3D{Z: -1})
	e.Begin_frame()
	e.Draw_world()
	if e.Stats.Drawn != 0 {
		t.Fatal("behind", e.Stats.Drawn)
	}
}

func TestLoadUnload(t *testing.T) {
	w, e, _ := setup(t)
	if !e.Ready(1) {
		t.Fatal("not ready")
	}
	w.Set_cube(world.Block_id(1, 1, 1), block.STONE)
	// 走远：原来的chunk被卸载，改动保存在内存里
	e.Set_camera(world.PhysicalPosition3D{X: 500, Y: 8, Z: 8}, world.PhysicalPosition3D{Z: -1})
	e.Update()
	if w.Is_loaded(world.CubeID{Depth: world.CHUNK_DEPTH}) {
		t.Fatal("not unloaded")
	}
	e.Set_camera(world.PhysicalPosition3D{X: 8, Y: 8, Z: 8}, world.PhysicalPosition3D{Z: -1})
	e.Update()
	if b, _, _ := w.Block_at(1, 1, 1); b != block.STONE {
		t.Fatal("lost edit")
	}
}
