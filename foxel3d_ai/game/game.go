// 游戏：玩家、物品栏、界面、存档和mod，把engine、terrain、block组合起来。
//
// 用法：
//
//	c := &canvas.Raylib{}
//	c.Init(1280, 720)
//	defer c.Close()
//	g, err := game.New_game(game.Config{World_name: "default", Seed: 1, View_distance: 8}, c)
//	g.Run()
package game

import (
	"fmt"
	"math"
	"os"
	"time"

	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/engine"
	"foxel3d_ai/terrain"
	"foxel3d_ai/world"
)

const HOTBAR_SIZE = 9

// 多久自动存档一次（秒）
const AUTOSAVE_INTERVAL = 300

type Config struct {
	// 存档的名字，保存在saves/名字/下面
	World_name string
	// 新世界的种子，读存档时用存档里的
	Seed int64
	// 视距（chunk数）
	View_distance int64
}

type Game struct {
	Config  Config
	Canvas  canvas.Canvas
	World   *world.World
	Engine  *engine.Engine
	Terrain *terrain.Generator
	Player  Player
	// 物品栏：9格，Selected是现在选中的那一格
	Hotbar   [HOTBAR_SIZE]world.BlockID
	Selected int
	// 挖掉、放置的格子大小：0是1格，-1是1/2格，-2是1/4格
	Build_depth world.Depth
	// 现在指着的方块，没有时是nil
	Target *world.Hit
	// mod的数据，跟着存档保存。key请用"mod名:名字"
	Mod_data map[string]string
	// 游戏开始后过了多少秒
	Time float64

	input          canvas.Input
	loading        bool
	inventory_open bool
	show_debug     bool
	show_help      bool
	message        string
	message_until  float64
	cooldown       float64
	// 刚点击回到游戏：松开鼠标左键之前不挖、不放
	wait_release bool
	last_save    float64
}

// 新建游戏：有存档时读存档，否则用cfg.Seed生成新世界。c必须已经Init过
func New_game(cfg Config, c canvas.Canvas) (*Game, error) {
	if cfg.View_distance <= 0 {
		cfg.View_distance = 8
	}
	g := &Game{Config: cfg, Canvas: c, Mod_data: map[string]string{}, loading: true}
	save, err := read_save(g.Save_path())
	if err != nil {
		return nil, err
	}
	if save != nil {
		cfg.Seed = save.Seed
		g.Config.Seed = save.Seed
	}
	g.Terrain = terrain.New(cfg.Seed)
	g.World = world.New_world(g.Terrain)
	g.Engine = engine.New_engine(g.World, c)
	g.Engine.View_distance = cfg.View_distance
	copy(g.Hotbar[:], block.Default_hotbar)
	if save != nil {
		g.apply_save(save)
	} else {
		h := g.Terrain.Height(0, 0)
		g.Player = Player{Pos: world.PhysicalPosition3D{X: 0.5, Y: float64(h) + 1, Z: 0.5}}
	}

	sky := canvas.SkyBlue
	far := float64(cfg.View_distance * world.CHUNK_SIZE)
	c.Set_background(sky)
	c.Set_fog(sky, far*0.55, far-4)
	c.Camera3D_vis_angle(100, 0, 0, true)
	for _, m := range mods {
		if m.On_start != nil {
			m.On_start(g)
		}
	}
	return g, nil
}

// 一直运行到窗口关闭，关闭时存档
func (g *Game) Run() {
	for !g.Canvas.Should_close() {
		g.Frame()
	}
	if err := g.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "foxel: 存档失败：", err)
	}
}

// 运行一帧：处理输入、移动玩家、加载世界、画出来
func (g *Game) Frame() {
	dt := math.Min(g.Canvas.Get_frame_time(), 0.05)
	g.Time += dt
	g.input = g.Canvas.Get_input()

	if g.loading {
		// 刚开始时多花点时间加载，加载好了再让玩家动
		g.Engine.Budget = 50 * time.Millisecond
		g.Engine.Set_camera(g.Player.Eye(), g.Player.Look_dir())
		g.Engine.Update()
		if g.Engine.Ready(2) {
			g.loading = false
			g.Engine.Budget = 8 * time.Millisecond
			g.unstuck_player()
		}
	} else {
		g.update(dt)
		g.Engine.Set_camera(g.Player.Eye(), g.Player.Look_dir())
		g.Engine.Update()
	}

	g.Engine.Begin_frame()
	g.Engine.Draw_world()
	g.draw_target()
	g.draw_hud()
	for _, m := range mods {
		if m.On_hud != nil {
			m.On_hud(g)
		}
	}
	g.Engine.End_frame()
}

func (g *Game) update(dt float64) {
	in := g.input
	locked := g.Canvas.Is_cursor_locked()

	switch {
	case in.Is_pressed(canvas.KEY_TAB):
		g.inventory_open = !g.inventory_open
		g.Canvas.Set_cursor_locked(!g.inventory_open)
	case in.Is_pressed(canvas.KEY_ESCAPE):
		g.inventory_open = false
		g.Canvas.Set_cursor_locked(false)
	case !locked && !g.inventory_open && in.Is_pressed(canvas.MOUSE_LEFT):
		// 点一下回到游戏，这次点击不算挖方块
		g.Canvas.Set_cursor_locked(true)
		g.wait_release = true
	}
	if in.Is_pressed(canvas.KEY_F3) {
		g.show_debug = !g.show_debug
	}
	if in.Is_pressed(canvas.KEY_H) {
		g.show_help = !g.show_help
	}
	if in.Is_pressed(canvas.KEY_F5) {
		if err := g.Save(); err != nil {
			g.Show_message("Save failed: " + err.Error())
		} else {
			g.Show_message("World saved")
		}
	}
	if in.Is_pressed(canvas.KEY_F) {
		g.Player.Flying = !g.Player.Flying
		g.Player.Velocity = world.PhysicalPosition3D{}
		g.Show_message(map[bool]string{true: "Flying", false: "Walking"}[g.Player.Flying])
	}
	if in.Is_pressed(canvas.KEY_R) {
		g.Build_depth--
		if g.Build_depth < world.MIN_DEPTH {
			g.Build_depth = 0
		}
		g.Show_message("Block size " + size_name(g.Build_depth))
	}
	for i := range HOTBAR_SIZE {
		if in.Is_pressed(canvas.KEY_1 + canvas.Key(i)) {
			g.Selected = i
		}
	}
	if in.Wheel != 0 && !g.inventory_open {
		step := -1
		if in.Wheel < 0 {
			step = 1
		}
		g.Selected = (g.Selected + step + HOTBAR_SIZE) % HOTBAR_SIZE
	}

	g.update_player(in, dt, g.Playing())
	g.Target = nil
	if hit, ok := g.World.Raycast(g.Player.Eye(), g.Player.Look_dir(), REACH, func(b world.BlockID) bool { return b != world.AIR }); ok {
		g.Target = &hit
	}
	if g.Playing() {
		g.interact(in, dt)
	}
	if g.inventory_open {
		g.inventory_click(in)
	}
	for _, m := range mods {
		if m.On_update != nil {
			m.On_update(g, in, dt)
		}
	}
	if g.Time-g.last_save > AUTOSAVE_INTERVAL {
		if err := g.Save(); err != nil {
			g.Show_message("Autosave failed: " + err.Error())
		}
	}
}

// 玩家正在玩（鼠标锁定、没有打开物品栏）
func (g *Game) Playing() bool {
	return !g.loading && !g.inventory_open && g.Canvas.Is_cursor_locked()
}

// 这一帧的输入
func (g *Game) Input() canvas.Input { return g.input }

// 在屏幕上方显示一条消息，几秒后消失
func (g *Game) Show_message(text string) {
	g.message = text
	g.message_until = g.Time + 2.5
}

func size_name(d world.Depth) string {
	switch d {
	case 0:
		return "1"
	case -1:
		return "1/2"
	default:
		return "1/4"
	}
}
