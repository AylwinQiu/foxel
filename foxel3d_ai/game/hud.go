// 界面：准星、物品栏、调试信息、帮助、暂停、背包
package game

import (
	"fmt"
	"math"
	"strings"

	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

var (
	panel     = canvas.Color{R: 0, G: 0, B: 0, A: 130}
	highlight = canvas.Color{R: 255, G: 255, B: 255, A: 230}
)

const help_text = `WASD     move            Space   jump / fly up
Mouse    look            Shift   sneak / fly down
Left     break block     Ctrl    sprint
Right    place block     F       toggle flying
Middle   pick block      R       block size 1, 1/2, 1/4
1-9      select slot     Tab     all blocks
Wheel    select slot     F5      save
F3       debug info      Esc     pause
H        hide this help`

func (g *Game) screen() canvas.Canvas2D {
	w, h := g.Canvas.Get_window_size()
	return canvas.Canvas2D{X: float64(w), Y: float64(h)}
}

// 文字加一个半透明的底
func (g *Game) text_box(text string, pos canvas.Canvas2D) {
	size := g.Canvas.Measure_2d_text(text)
	g.Canvas.Add_2d_rect(canvas.Canvas2D{X: pos.X - 6, Y: pos.Y - 4}, canvas.Canvas2D{X: size.X + 12, Y: size.Y + 8}, panel)
	g.Canvas.Add_2d_text(text, pos, canvas.White)
}

// 屏幕中间横着居中的文字
func (g *Game) centered_text(text string, y float64) {
	size := g.Canvas.Measure_2d_text(text)
	g.text_box(text, canvas.Canvas2D{X: (g.screen().X - size.X) / 2, Y: y})
}

// 指着的格子画一个框
func (g *Game) draw_target() {
	if !g.Playing() {
		return
	}
	id, ok := g.Target_cell()
	if !ok {
		return
	}
	eps := world.PhysicalPosition3D{X: 0.003, Y: 0.003, Z: 0.003}
	g.Engine.Add_box_lines(id.Min().Sub(eps), id.Max().Add(eps), canvas.Color{R: 20, G: 20, B: 20, A: 255})
}

func (g *Game) draw_hud() {
	c := g.Canvas
	s := g.screen()
	if g.loading {
		c.Add_2d_rect(canvas.Canvas2D{}, s, canvas.Color{R: 40, G: 60, B: 90, A: 255})
		g.centered_text("Generating world...", s.Y/2-10)
		return
	}

	// 准星
	cx, cy := s.X/2, s.Y/2
	c.Add_2d_rect(canvas.Canvas2D{X: cx - 9, Y: cy - 1}, canvas.Canvas2D{X: 18, Y: 2}, highlight)
	c.Add_2d_rect(canvas.Canvas2D{X: cx - 1, Y: cy - 9}, canvas.Canvas2D{X: 2, Y: 18}, highlight)

	g.draw_hotbar()

	if g.show_debug {
		g.text_box(g.debug_text(), canvas.Canvas2D{X: 12, Y: 12})
	} else if g.show_help {
		g.text_box(help_text, canvas.Canvas2D{X: 12, Y: 12})
	} else {
		g.text_box("H: help", canvas.Canvas2D{X: 12, Y: 12})
	}
	if g.Time < g.message_until {
		g.centered_text(g.message, 60)
	}

	switch {
	case g.inventory_open:
		g.draw_inventory()
	case !c.Is_cursor_locked():
		c.Add_2d_rect(canvas.Canvas2D{}, s, canvas.Color{R: 0, G: 0, B: 0, A: 90})
		g.centered_text("Paused - click to play", s.Y/2-40)
	}
}

const slot = 52.0

func (g *Game) draw_hotbar() {
	c := g.Canvas
	s := g.screen()
	x0 := (s.X - slot*HOTBAR_SIZE) / 2
	y0 := s.Y - slot - 14
	c.Add_2d_rect(canvas.Canvas2D{X: x0 - 4, Y: y0 - 4}, canvas.Canvas2D{X: slot*HOTBAR_SIZE + 8, Y: slot + 8}, panel)
	for i, b := range g.Hotbar {
		pos := canvas.Canvas2D{X: x0 + float64(i)*slot, Y: y0}
		if i == g.Selected {
			frame(c, pos, canvas.Canvas2D{X: slot, Y: slot}, 3, highlight)
		}
		if b != world.AIR {
			c.Add_2d_image(block.Get(b).Images.South, canvas.Canvas2D{X: pos.X + 8, Y: pos.Y + 8}, canvas.Canvas2D{X: slot - 16, Y: slot - 16})
		}
	}
	name := block.Get(g.Hotbar[g.Selected]).Display
	g.centered_text(name, y0-40)
	size := "Size " + size_name(g.Build_depth)
	g.text_box(size, canvas.Canvas2D{X: x0 + slot*HOTBAR_SIZE + 18, Y: y0 + slot/2 - 10})
}

// 空心的框
func frame(c canvas.Canvas, pos, size canvas.Canvas2D, w float64, color canvas.Color) {
	c.Add_2d_rect(pos, canvas.Canvas2D{X: size.X, Y: w}, color)
	c.Add_2d_rect(canvas.Canvas2D{X: pos.X, Y: pos.Y + size.Y - w}, canvas.Canvas2D{X: size.X, Y: w}, color)
	c.Add_2d_rect(pos, canvas.Canvas2D{X: w, Y: size.Y}, color)
	c.Add_2d_rect(canvas.Canvas2D{X: pos.X + size.X - w, Y: pos.Y}, canvas.Canvas2D{X: w, Y: size.Y}, color)
}

func (g *Game) debug_text() string {
	p := g.Player.Pos
	chunk := world.Cube_id_at(p, world.CHUNK_DEPTH)
	col := g.Terrain.Column(int64(math.Floor(p.X)), int64(math.Floor(p.Z)))
	st := g.Engine.Stats
	lines := []string{
		fmt.Sprintf("FPS %.0f", 1/max(g.Canvas.Get_frame_time(), 1e-6)),
		fmt.Sprintf("XYZ %.2f %.2f %.2f", p.X, p.Y, p.Z),
		fmt.Sprintf("Chunk %d %d %d", chunk.X, chunk.Y, chunk.Z),
		fmt.Sprintf("Biome %s", col.Biome.Name),
		fmt.Sprintf("Chunks loaded %d, meshes %d, drawn %d, pending %d", st.Loaded, st.Meshes, st.Drawn, st.Pending),
		fmt.Sprintf("Seed %d", g.Config.Seed),
	}
	if g.Target != nil {
		l := g.Target.Leaf
		lines = append(lines, fmt.Sprintf("Looking at %s %v", block.Get(l.Block).Name, l.Id))
	}
	return strings.Join(lines, "\n")
}

// 背包：所有能放的方块
func (g *Game) inventory_blocks() []world.BlockID {
	var out []world.BlockID
	for _, b := range block.All() {
		if !block.Get(b).Hidden {
			out = append(out, b)
		}
	}
	return out
}

const inv_cols = 9

func (g *Game) inventory_origin(n int) canvas.Canvas2D {
	s := g.screen()
	rows := (n + inv_cols - 1) / inv_cols
	return canvas.Canvas2D{X: (s.X - slot*inv_cols) / 2, Y: (s.Y-slot*float64(rows))/2 - 40}
}

// 鼠标指着背包里的第几个方块，没有时是-1
func (g *Game) inventory_hover(in canvas.Input, n int) int {
	o := g.inventory_origin(n)
	col := int((in.Mouse_x - o.X) / slot)
	row := int((in.Mouse_y - o.Y) / slot)
	if in.Mouse_x < o.X || in.Mouse_y < o.Y || col >= inv_cols {
		return -1
	}
	if i := row*inv_cols + col; i < n {
		return i
	}
	return -1
}

func (g *Game) draw_inventory() {
	c := g.Canvas
	s := g.screen()
	c.Add_2d_rect(canvas.Canvas2D{}, s, canvas.Color{R: 0, G: 0, B: 0, A: 120})
	blocks := g.inventory_blocks()
	o := g.inventory_origin(len(blocks))
	rows := (len(blocks) + inv_cols - 1) / inv_cols
	c.Add_2d_rect(canvas.Canvas2D{X: o.X - 8, Y: o.Y - 8}, canvas.Canvas2D{X: slot*inv_cols + 16, Y: slot*float64(rows) + 16}, panel)
	g.centered_text("Click a block to put it in the selected slot (Tab to close)", o.Y-50)
	hover := g.inventory_hover(g.input, len(blocks))
	for i, b := range blocks {
		pos := canvas.Canvas2D{X: o.X + float64(i%inv_cols)*slot, Y: o.Y + float64(i/inv_cols)*slot}
		if i == hover {
			frame(c, pos, canvas.Canvas2D{X: slot, Y: slot}, 2, highlight)
		}
		c.Add_2d_image(block.Get(b).Images.South, canvas.Canvas2D{X: pos.X + 8, Y: pos.Y + 8}, canvas.Canvas2D{X: slot - 16, Y: slot - 16})
	}
	if hover >= 0 {
		g.text_box(block.Get(blocks[hover]).Display, canvas.Canvas2D{X: g.input.Mouse_x + 16, Y: g.input.Mouse_y + 16})
	}
}

func (g *Game) inventory_click(in canvas.Input) {
	if !in.Is_pressed(canvas.MOUSE_LEFT) {
		return
	}
	blocks := g.inventory_blocks()
	if i := g.inventory_hover(in, len(blocks)); i >= 0 {
		g.Hotbar[g.Selected] = blocks[i]
	}
}
