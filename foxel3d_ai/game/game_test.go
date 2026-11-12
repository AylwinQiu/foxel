package game

import (
	"math"
	"os"
	"testing"

	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

// 什么都不画的canvas，输入由测试设置
type fake_canvas struct {
	input  canvas.Input
	locked bool
}

func (f *fake_canvas) Init(int, int)                                                      {}
func (f *fake_canvas) Close()                                                             {}
func (f *fake_canvas) Add_cube(float64, canvas.Canvas3D, canvas.Images, world.Rotation3D) {}
func (f *fake_canvas) Add_line3d(canvas.Canvas3D, canvas.Canvas3D, canvas.Color)          {}
func (f *fake_canvas) Clean()                                                             {}
func (f *fake_canvas) Draw()                                                              {}
func (f *fake_canvas) Should_close() bool                                                 { return false }
func (f *fake_canvas) Get_window_size() (uint64, uint64)                                  { return 800, 600 }
func (f *fake_canvas) Get_frame_time() float64                                            { return 1.0 / 60 }
func (f *fake_canvas) Camera3D_loc_and_dir(canvas.Canvas3D, canvas.Canvas3D)              {}
func (f *fake_canvas) Camera3D_vis_angle(float64, float64, float64, bool)                 {}
func (f *fake_canvas) Set_background(canvas.Color)                                        {}
func (f *fake_canvas) Set_fog(canvas.Color, float64, float64)                             {}
func (f *fake_canvas) Set_mesh(canvas.MeshID, []canvas.Quad)                              {}
func (f *fake_canvas) Delete_mesh(canvas.MeshID)                                          {}
func (f *fake_canvas) Add_mesh(canvas.MeshID, canvas.Canvas3D)                            {}
func (f *fake_canvas) Add_2d_text(string, canvas.Canvas2D, canvas.Color)                  {}
func (f *fake_canvas) Measure_2d_text(string) canvas.Canvas2D                             { return canvas.Canvas2D{} }
func (f *fake_canvas) Add_2d_line(canvas.Canvas2D, canvas.Canvas2D, canvas.Color)         {}
func (f *fake_canvas) Add_2d_rect(canvas.Canvas2D, canvas.Canvas2D, canvas.Color)         {}
func (f *fake_canvas) Add_2d_image(canvas.Image, canvas.Canvas2D, canvas.Canvas2D)        {}
func (f *fake_canvas) Get_input() canvas.Input                                            { return f.input }
func (f *fake_canvas) Set_cursor_locked(l bool)                                           { f.locked = l }
func (f *fake_canvas) Is_cursor_locked() bool                                             { return f.locked }
func (f *fake_canvas) Get_frustum_corner_rays(d canvas.Canvas3D) (canvas.Canvas3D, canvas.Canvas3D, canvas.Canvas3D, canvas.Canvas3D) {
	return d, d, d, d
}

func hold(keys ...canvas.Key) canvas.Input {
	in := canvas.Input{Held: map[canvas.Key]bool{}, Pressed: map[canvas.Key]bool{}}
	for _, k := range keys {
		in.Held[k] = true
	}
	return in
}

// 在临时文件夹里开一个新游戏，等世界加载好，玩家站在一块平地上
func new_test_game(t *testing.T) (*Game, *fake_canvas) {
	t.Helper()
	dir, _ := os.Getwd()
	os.Chdir(t.TempDir())
	t.Cleanup(func() { os.Chdir(dir) })
	c := &fake_canvas{locked: true}
	g, err := New_game(Config{World_name: "test", Seed: 5, View_distance: 3}, c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; g.loading; i++ {
		g.Frame()
		if i > 1000 {
			t.Fatal("loading forever")
		}
	}
	// 在高处铺一块10x10的石头平台，上面清空
	for x := int64(-5); x < 5; x++ {
		for z := int64(-5); z < 5; z++ {
			g.World.Set_cube(world.Block_id(x, 60, z), block.STONE)
			for y := int64(61); y < 66; y++ {
				g.World.Set_cube(world.Block_id(x, y, z), world.AIR)
			}
		}
	}
	g.Player = Player{Pos: PhyPos(0.5, 61, 0.5)}
	return g, c
}

func run_frames(g *Game, c *fake_canvas, in canvas.Input, n int) {
	for range n {
		c.input = in
		g.Frame()
	}
}

func TestStandAndWalk(t *testing.T) {
	g, c := new_test_game(t)
	run_frames(g, c, hold(), 30)
	if !g.Player.On_ground || math.Abs(g.Player.Pos.Y-61) > 0.01 {
		t.Fatal("not standing", g.Player)
	}
	// 往北走1秒
	run_frames(g, c, hold(canvas.KEY_W), 60)
	if d := 0.5 - g.Player.Pos.Z; math.Abs(d-WALK_SPEED) > 0.3 {
		t.Fatal("walk distance", d)
	}
	// 跳起来再落下
	run_frames(g, c, hold(canvas.KEY_SPACE), 5)
	if g.Player.Pos.Y < 61.2 {
		t.Fatal("no jump", g.Player.Pos.Y)
	}
	run_frames(g, c, hold(), 60)
	if math.Abs(g.Player.Pos.Y-61) > 0.01 {
		t.Fatal("not landed", g.Player.Pos.Y)
	}
}

func TestWallsAndSteps(t *testing.T) {
	g, c := new_test_game(t)
	run_frames(g, c, hold(), 10)
	// 北边z=-2有一堵两格高的墙
	for x := int64(-5); x < 5; x++ {
		g.World.Set_cube(world.Block_id(x, 61, -2), block.STONE)
		g.World.Set_cube(world.Block_id(x, 62, -2), block.STONE)
	}
	run_frames(g, c, hold(canvas.KEY_W), 60)
	if z := g.Player.Pos.Z; math.Abs(z-(-1+PLAYER_WIDTH/2)) > 0.01 {
		t.Fatal("wall", z)
	}
	// 东边铺一层半格高的台阶：直接走上去
	g.Player.Pos = PhyPos(0.5, 61, 0.5)
	for z := int64(-2); z < 10; z++ {
		for x := int64(2); x < 10; x++ {
			g.World.Set_cube(world.CubeID{X: x, Y: 122, Z: z, Depth: -1}, block.STONE)
		}
	}
	g.Player.Yaw = math.Pi / 2
	run_frames(g, c, hold(canvas.KEY_W), 30)
	if p := g.Player.Pos; p.X < 1.5 || math.Abs(p.Y-61.5) > 0.01 {
		t.Fatal("step", p)
	}
}

// 暂停时点击回到游戏：这次点击不挖方块，按住也不挖，松开再按才挖
func TestResumeClickDoesNotBreak(t *testing.T) {
	g, c := new_test_game(t)
	g.Player.Pitch = -1.2
	run_frames(g, c, hold(), 10)
	cell, ok := g.Target_cell()
	if !ok {
		t.Fatal("no target")
	}
	c.locked = false
	click := hold(canvas.MOUSE_LEFT)
	click.Pressed[canvas.MOUSE_LEFT] = true
	run_frames(g, c, click, 1)
	if !c.locked {
		t.Fatal("not resumed")
	}
	run_frames(g, c, hold(canvas.MOUSE_LEFT), 60)
	if b, _, _ := g.World.Block_at(cell.X, cell.Y, cell.Z); b == world.AIR {
		t.Fatal("resume click broke a block")
	}
	run_frames(g, c, hold(), 1)
	run_frames(g, c, click, 1)
	if b, _, _ := g.World.Block_at(cell.X, cell.Y, cell.Z); b != world.AIR {
		t.Fatal("second click did not break")
	}
}

func TestBreakPlaceAndSave(t *testing.T) {
	g, c := new_test_game(t)
	run_frames(g, c, hold(), 10)
	// 低头看脚下前面的地面
	g.Player.Pitch = -1.2
	run_frames(g, c, hold(), 1)
	cell, ok := g.Target_cell()
	if !ok || cell.Y != 60 {
		t.Fatal("target", cell, ok)
	}
	if !g.Break(cell) {
		t.Fatal("break")
	}
	// 不能放在自己身上
	if g.Place(world.Block_id(0, 61, 0), block.STONE) {
		t.Fatal("placed inside player")
	}
	// 1/4大小的方块
	g.Build_depth = world.MIN_DEPTH
	small := world.CubeID{X: 12, Y: 244, Z: 12, Depth: world.MIN_DEPTH}
	if !g.Place(small, block.GLASS) || g.Place(small, block.STONE) {
		t.Fatal("small place")
	}
	g.World.Field_write("test:note", small, "hi")
	g.Mod_data["test:x"] = "1"
	g.Hotbar[3] = block.BRICKS
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	// 用另一个种子开同一个存档：种子、方块、场、物品栏都是存档里的
	g2, err := New_game(Config{World_name: "test", Seed: 99, View_distance: 3}, c)
	if err != nil {
		t.Fatal(err)
	}
	for g2.loading {
		g2.Frame()
	}
	if g2.Config.Seed != 5 || g2.Hotbar[3] != block.BRICKS || g2.Mod_data["test:x"] != "1" {
		t.Fatal("save meta")
	}
	if b, _, _ := g2.World.Block_at(cell.X, cell.Y, cell.Z); b != world.AIR {
		t.Fatal("broken block came back")
	}
	if l, _ := g2.World.Leaf_at(small.Center()); l.Block != block.GLASS || l.Id != small {
		t.Fatal("small block", l)
	}
	if v, _ := g2.World.Field_read("test:note", small); v != "hi" {
		t.Fatal("field")
	}
	if math.Abs(g2.Player.Pos.Y-61) > 0.01 {
		t.Fatal("player", g2.Player.Pos)
	}
}
