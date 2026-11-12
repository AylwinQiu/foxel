package terrain

import (
	"math/rand/v2"

	"foxel3d_ai/block"
	"foxel3d_ai/world"
)

// feature：在地形上放东西，比如树、矿石、花。
//
// Place对每个chunk调用一次。feature可以比一个chunk大（比如树冠伸到隔壁chunk）：
// 用ctx.Scatter找出附近所有可能放东西的位置，然后只把落在这个chunk里的部分写进去（ctx.Set会自动忽略外面的）。
// 因为随机数只由种子和位置决定，隔壁chunk生成时会算出同一棵树，两边的部分正好对上。
type Feature interface {
	Place(ctx *Context)
}

// 用一个函数当feature
type Feature_func func(ctx *Context)

func (f Feature_func) Place(ctx *Context) { f(ctx) }

var features []Feature

// 添加feature，按添加顺序执行
func Register_feature(f Feature) {
	features = append(features, f)
}

// 正在生成的chunk
type Context struct {
	// chunk的CubeID
	Chunk world.CubeID
	Data  *world.ChunkData
	Gen   *Generator
}

// chunk最小的角的方块坐标
func (c *Context) Origin() (x, y, z int64) {
	return c.Chunk.X * world.CHUNK_SIZE, c.Chunk.Y * world.CHUNK_SIZE, c.Chunk.Z * world.CHUNK_SIZE
}

func (c *Context) local(x, y, z int64) (int, int, int, bool) {
	x0, y0, z0 := c.Origin()
	lx, ly, lz := x-x0, y-y0, z-z0
	ok := lx >= 0 && ly >= 0 && lz >= 0 && lx < world.CHUNK_SIZE && ly < world.CHUNK_SIZE && lz < world.CHUNK_SIZE
	return int(lx), int(ly), int(lz), ok
}

// 方块坐标(x, y, z)现在是什么。不在这个chunk里时返回false
func (c *Context) Get(x, y, z int64) (world.BlockID, bool) {
	lx, ly, lz, ok := c.local(x, y, z)
	if !ok {
		return world.AIR, false
	}
	return c.Data.Get(lx, ly, lz), true
}

// 把方块坐标(x, y, z)设成b，不在这个chunk里时什么都不做
func (c *Context) Set(x, y, z int64, b world.BlockID) {
	if lx, ly, lz, ok := c.local(x, y, z); ok {
		c.Data.Set(lx, ly, lz, b)
	}
}

// 只有原来是空气时才设成b（比如树叶不要盖掉别的树干）
func (c *Context) Set_if_air(x, y, z int64, b world.BlockID) {
	if old, ok := c.Get(x, y, z); ok && old == world.AIR {
		c.Set(x, y, z, b)
	}
}

// 放一个比1小的cube，不在这个chunk里时什么都不做
func (c *Context) Set_small(id world.CubeID, b world.BlockID) {
	if c.Chunk.Contains(id) && id.Depth >= world.MIN_DEPTH {
		c.Data.Small = append(c.Data.Small, world.Placement{Id: id, Block: b})
	}
}

func (c *Context) Column(x, z int64) Column { return c.Gen.Column(x, z) }

// (x, z)这一列地面最上面的方块（不考虑feature）
func (c *Context) Ground(x, z int64) world.BlockID {
	col := c.Gen.Column(x, z)
	return c.Gen.Base_block(col, col.Height)
}

// 确定性的随机数：同一个世界、同一个位置、同一个salt总是一样的
func (c *Context) Rand(x, y, z int64, salt uint64) *rand.Rand {
	h := hash(uint64(c.Gen.Seed), uint64(x), uint64(y), uint64(z), salt)
	return rand.New(rand.NewPCG(h, h^0xa5a5a5a5))
}

// 遍历这个chunk和它周围radius格以内的每一列(x, z)，每列以chance(x, z)的概率调用fn。
// salt每个feature用不同的值，否则不同的feature会挑中同样的位置
func (c *Context) Scatter(radius int64, salt uint64, chance func(x, z int64) float64, fn func(x, z int64, r *rand.Rand)) {
	x0, _, z0 := c.Origin()
	for z := z0 - radius; z < z0+world.CHUNK_SIZE+radius; z++ {
		for x := x0 - radius; x < x0+world.CHUNK_SIZE+radius; x++ {
			h := hash(uint64(c.Gen.Seed), uint64(x), 0, uint64(z), salt)
			if float64(h>>11)/(1<<53) >= chance(x, z) {
				continue
			}
			fn(x, z, rand.New(rand.NewPCG(h, salt)))
		}
	}
}

func hash(vs ...uint64) uint64 {
	h := uint64(0x243f6a8885a308d3)
	for _, v := range vs {
		h ^= v + 0x9e3779b97f4a7c15 + h<<6 + h>>2
		h = (h ^ (h >> 31)) * 0x7fb5d329728ea185
		h = (h ^ (h >> 27)) * 0x81dadef4bc2dd44d
		h ^= h >> 33
	}
	return h
}

// 树：每个生物群系每列长一棵树的概率是Density[群系名]，只长在Soil上
type Tree struct {
	Salt    uint64
	Density map[string]float64
	Soil    []world.BlockID
	// 树干和树冠最多离树根多少格（水平方向）
	Radius int64
	// 在地面(x, y, z)上面长一棵树，y是地面方块的高度
	Grow func(ctx *Context, x, y, z int64, r *rand.Rand)
}

func (t *Tree) Place(ctx *Context) {
	_, y0, _ := ctx.Origin()
	ctx.Scatter(t.Radius, t.Salt, func(x, z int64) float64 {
		return t.Density[ctx.Column(x, z).Biome.Name]
	}, func(x, z int64, r *rand.Rand) {
		y := ctx.Column(x, z).Height
		// 树最高20格，整棵树都不在这个chunk的高度范围里就不用算了
		if y+20 < y0 || y >= y0+world.CHUNK_SIZE {
			return
		}
		ground := ctx.Ground(x, z)
		for _, s := range t.Soil {
			if s == ground {
				t.Grow(ctx, x, y, z, r)
				return
			}
		}
	})
}

// 矿石：每个chunk放Per_chunk条矿脉，每条Size个方块，只替换石头
type Ore struct {
	Salt      uint64
	Block     world.BlockID
	Per_chunk int
	Size      int
	Min_y     int64
	Max_y     int64
}

func (o *Ore) Place(ctx *Context) {
	x0, y0, z0 := ctx.Origin()
	if y0 > o.Max_y || y0+world.CHUNK_SIZE <= o.Min_y {
		return
	}
	r := ctx.Rand(ctx.Chunk.X, ctx.Chunk.Y, ctx.Chunk.Z, o.Salt)
	for range o.Per_chunk {
		x, y, z := x0+r.Int64N(world.CHUNK_SIZE), y0+r.Int64N(world.CHUNK_SIZE), z0+r.Int64N(world.CHUNK_SIZE)
		for range o.Size {
			if y >= o.Min_y && y <= o.Max_y {
				if b, ok := ctx.Get(x, y, z); ok && b == block.STONE {
					ctx.Set(x, y, z, o.Block)
				}
			}
			switch r.IntN(3) {
			case 0:
				x += r.Int64N(3) - 1
			case 1:
				y += r.Int64N(3) - 1
			default:
				z += r.Int64N(3) - 1
			}
		}
	}
}

// 内置的feature
var (
	Coal = &Ore{Salt: 101, Block: block.COAL_ORE, Per_chunk: 8, Size: 10, Min_y: BOTTOM_Y, Max_y: 90}
	Iron = &Ore{Salt: 102, Block: block.IRON_ORE, Per_chunk: 4, Size: 6, Min_y: BOTTOM_Y, Max_y: 30}

	Oak_trees = &Tree{Salt: 201, Radius: 2, Soil: []world.BlockID{block.GRASS},
		Density: map[string]float64{Plains.Name: 0.004, Forest.Name: 0.045, Mountains.Name: 0.004},
		Grow:    grow_oak}
	Spruce_trees = &Tree{Salt: 202, Radius: 3, Soil: []world.BlockID{block.GRASS, block.SNOWY_GRASS},
		Density: map[string]float64{Taiga.Name: 0.03, Mountains.Name: 0.012},
		Grow:    grow_spruce}
)

func init() {
	Register_feature(Coal)
	Register_feature(Iron)
	Register_feature(Oak_trees)
	Register_feature(Spruce_trees)
}

// 橡树：4到6格高的树干，顶上一团圆圆的树叶
func grow_oak(ctx *Context, x, y, z int64, r *rand.Rand) {
	h := 4 + r.Int64N(3)
	top := y + h
	for dy := int64(-2); dy <= 1; dy++ {
		radius := int64(2)
		if dy >= 0 {
			radius = 1
		}
		for dz := -radius; dz <= radius; dz++ {
			for dx := -radius; dx <= radius; dx++ {
				corner := (dx == -radius || dx == radius) && (dz == -radius || dz == radius)
				// 角上的叶子随机缺一些，最上面一层只留十字
				if corner && (dy == 1 || r.IntN(2) == 0) {
					continue
				}
				ctx.Set_if_air(x+dx, top+dy, z+dz, block.OAK_LEAVES)
			}
		}
	}
	for dy := int64(1); dy <= h; dy++ {
		ctx.Set(x, y+dy, z, block.OAK_LOG)
	}
	ctx.Set(x, y, z, block.DIRT)
}

// 云杉：细高的树干，树叶一层大一层小地往下变宽，像个圆锥
func grow_spruce(ctx *Context, x, y, z int64, r *rand.Rand) {
	h := 7 + r.Int64N(4)
	top := y + h
	ctx.Set_if_air(x, top+1, z, block.SPRUCE_LEAVES)
	for ly := top; ly > y+2; ly-- {
		// 离树顶越远越宽，每隔一层缩回去一点
		d := top - ly
		radius := min(1+d/3, 3)
		if d%2 == 0 {
			radius = max(radius-1, 1)
		}
		for dz := -radius; dz <= radius; dz++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx*dx+dz*dz <= radius*radius+1 {
					ctx.Set_if_air(x+dx, ly, z+dz, block.SPRUCE_LEAVES)
				}
			}
		}
	}
	for dy := int64(1); dy <= h; dy++ {
		ctx.Set(x, y+dy, z, block.SPRUCE_LOG)
	}
	ctx.Set(x, y, z, block.DIRT)
}
