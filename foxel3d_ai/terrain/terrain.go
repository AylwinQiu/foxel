// 程序生成地形：高度图 + 生物群系 + feature（树、矿石……）。
//
// 每一列(x, z)先算出气候（温度、湿度），选气候最接近的生物群系；地面高度按气候距离把各个生物群系的高度混合起来，
// 所以生物群系交界的地方是平滑过渡的。然后在每个chunk里依次调用所有feature放树、矿石之类的东西。
//
// 同一个种子永远生成同一个世界，和chunk生成的顺序无关。
//
// mod可以：
//   - Register_biome 添加新的生物群系
//   - Register_feature 添加新的feature（比如花、建筑）
//   - 修改已有feature的参数，比如 Oak_trees.Density["mymod:jungle"] = 0.1
//
// 这些都要在世界创建（New）之前做，一般写在mod包的init()里。
package terrain

import (
	"math"

	"foxel3d_ai/block"
	"foxel3d_ai/world"
)

const (
	// 世界竖直方向的范围（chunk的Y）：y从-32到127
	MIN_CHUNK_Y = -2
	MAX_CHUNK_Y = 7
	// 最下面一层是基岩
	BOTTOM_Y = MIN_CHUNK_Y * world.CHUNK_SIZE
)

type Generator struct {
	Seed     int64
	Biomes   []*Biome
	Features []Feature

	height, ridge, detail, temperature, humidity *perlin
	columns                                      map[[2]int64]Column
}

// 一列的地形
type Column struct {
	// 地面最上面那个方块的y
	Height int64
	Biome  *Biome
}

var _ world.Generator = (*Generator)(nil)

// 用种子和现在注册了的生物群系、feature创建生成器
func New(seed int64) *Generator {
	s := uint64(seed)
	return &Generator{
		Seed:        seed,
		Biomes:      append([]*Biome(nil), biomes...),
		Features:    append([]Feature(nil), features...),
		height:      new_perlin(s ^ 0x1001),
		ridge:       new_perlin(s ^ 0x2002),
		detail:      new_perlin(s ^ 0x3003),
		temperature: new_perlin(s ^ 0x4004),
		humidity:    new_perlin(s ^ 0x5005),
		columns:     map[[2]int64]Column{},
	}
}

func (g *Generator) Vertical_range() (int64, int64) { return MIN_CHUNK_Y, MAX_CHUNK_Y }

// (x, z)这一列的地形
func (g *Generator) Column(x, z int64) Column {
	key := [2]int64{x, z}
	if c, ok := g.columns[key]; ok {
		return c
	}
	if len(g.columns) > 1<<18 {
		clear(g.columns)
	}
	c := g.compute_column(float64(x), float64(z))
	g.columns[key] = c
	return c
}

func (g *Generator) Height(x, z int64) int64 { return g.Column(x, z).Height }
func (g *Generator) Biome(x, z int64) *Biome { return g.Column(x, z).Biome }

// 气候距离的尺度：越小生物群系的交界越窄
const climate_sigma2 = 0.04

func (g *Generator) compute_column(x, z float64) Column {
	t := clamp(g.temperature.fbm(x/700, z/700, 3)*1.8, -1, 1)
	h := clamp(g.humidity.fbm(x/550, z/550, 3)*1.8, -1, 1)
	plain := g.height.fbm(x/140, z/140, 4)
	r := 1 - math.Abs(g.ridge.fbm(x/220, z/220, 4))
	ridged := r * r

	total, height, best_w := 0.0, 0.0, -1.0
	var best *Biome
	for _, b := range g.Biomes {
		d := (t-b.Temperature)*(t-b.Temperature) + (h-b.Humidity)*(h-b.Humidity)
		w := math.Exp(-d / climate_sigma2)
		shape := plain
		if b.Ridged {
			shape = ridged
		}
		height += w * (b.Base_height + b.Variation*shape)
		total += w
		if w > best_w {
			best_w, best = w, b
		}
	}
	if best == nil {
		return Column{Height: 0, Biome: &Biome{Name: "none", Surface: block.STONE, Subsurface: block.STONE}}
	}
	height = height/math.Max(total, 1e-12) + g.detail.fbm(x/24, z/24, 2)*1.5
	return Column{Height: int64(math.Floor(height)), Biome: best}
}

func clamp(v, lo, hi float64) float64 { return math.Min(math.Max(v, lo), hi) }

// 不考虑feature时(x, y, z)是什么方块
func (g *Generator) Base_block(c Column, y int64) world.BlockID {
	switch {
	case y < BOTTOM_Y || y > c.Height:
		return world.AIR
	case y == BOTTOM_Y:
		return block.BEDROCK
	}
	b := c.Biome
	rock := b.Rock_line != 0 && c.Height >= b.Rock_line
	if y == c.Height {
		switch {
		case b.Snow_line != 0 && y >= b.Snow_line:
			return block.SNOW
		case rock:
			return block.STONE
		}
		return b.Surface
	}
	if c.Height-y < int64(b.Subsurface_depth) && !rock {
		return b.Subsurface
	}
	return block.STONE
}

func (g *Generator) Generate_chunk(pos world.CubeID, out *world.ChunkData) {
	if pos.Y < MIN_CHUNK_Y || pos.Y > MAX_CHUNK_Y {
		return
	}
	x0, y0, z0 := pos.X*world.CHUNK_SIZE, pos.Y*world.CHUNK_SIZE, pos.Z*world.CHUNK_SIZE
	for lz := range world.CHUNK_SIZE {
		for lx := range world.CHUNK_SIZE {
			c := g.Column(x0+int64(lx), z0+int64(lz))
			// 整列都在地面上面
			if y0 > c.Height {
				continue
			}
			for ly := range world.CHUNK_SIZE {
				out.Set(lx, ly, lz, g.Base_block(c, y0+int64(ly)))
			}
		}
	}
	ctx := &Context{Chunk: pos, Data: out, Gen: g}
	for _, f := range g.Features {
		f.Place(ctx)
	}
}
