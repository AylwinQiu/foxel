// 示例mod：草地上长着小花，按G在身边种一圈花。
//
// 演示了mod的三种接口：
//   - 注册新方块（花茎、花瓣）
//   - 注册地形feature（生成世界时在草地上种花）
//   - 注册游戏事件（按键种花，挖到花时显示消息）
//
// 花是用1/4格大小的方块拼起来的：一格茎，上面一格花瓣。玩家可以穿过花（Solid是false）。
package flowers

import (
	"fmt"
	"math"
	"math/rand/v2"

	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/game"
	"foxel3d_ai/terrain"
	"foxel3d_ai/world"
)

func flower(name, display string, c canvas.Color) world.BlockID {
	return block.Register(block.Type{Name: name, Display: display, Images: canvas.Images{}.Color(c), Opaque: true})
}

var (
	STEM   = flower("flowers:stem", "Flower Stem", canvas.Color{R: 70, G: 150, B: 50, A: 255})
	RED    = flower("flowers:red", "Red Petal", canvas.Color{R: 220, G: 40, B: 50, A: 255})
	YELLOW = flower("flowers:yellow", "Yellow Petal", canvas.Color{R: 250, G: 215, B: 40, A: 255})
	BLUE   = flower("flowers:blue", "Blue Petal", canvas.Color{R: 80, G: 120, B: 240, A: 255})

	petals = []world.BlockID{RED, YELLOW, BLUE}
	// 每种生物群系每列长花的概率
	density = map[string]float64{terrain.Plains.Name: 0.05, terrain.Forest.Name: 0.02}
)

// 在方块(x, y, z)这个格子的底部种一朵花，y是地面上面那个空格子。set负责真正放方块
func plant(x, y, z int64, r *rand.Rand, set func(id world.CubeID, b world.BlockID)) {
	const n = 1 << -world.MIN_DEPTH // 一格分成4份
	sx, sz := x*n+1+r.Int64N(2), z*n+1+r.Int64N(2)
	height := 1 + r.Int64N(2)
	for i := range height {
		set(world.CubeID{X: sx, Y: y*n + i, Z: sz, Depth: world.MIN_DEPTH}, STEM)
	}
	set(world.CubeID{X: sx, Y: y*n + height, Z: sz, Depth: world.MIN_DEPTH}, petals[r.IntN(len(petals))])
}

func init() {
	terrain.Register_feature(terrain.Feature_func(func(ctx *terrain.Context) {
		ctx.Scatter(0, 0xf10e, func(x, z int64) float64 {
			return density[ctx.Column(x, z).Biome.Name]
		}, func(x, z int64, r *rand.Rand) {
			if ctx.Ground(x, z) != block.GRASS {
				return
			}
			y := ctx.Column(x, z).Height + 1
			// 上面已经有东西了（比如树干），或者不在这个chunk里
			if b, ok := ctx.Get(x, y, z); !ok || b != world.AIR {
				return
			}
			plant(x, y, z, r, ctx.Set_small)
		})
	}))

	game.Register_mod(game.Mod{
		Name: "flowers",
		On_update: func(g *game.Game, in canvas.Input, dt float64) {
			if !g.Playing() || !in.Is_pressed(canvas.KEY_G) {
				return
			}
			p := g.Player.Pos
			r := rand.New(rand.NewPCG(uint64(g.Time*1000), 7))
			planted := 0
			for range 16 {
				x := int64(math.Floor(p.X)) + r.Int64N(9) - 4
				z := int64(math.Floor(p.Z)) + r.Int64N(9) - 4
				y, ok := g.Surface_below(x, int64(math.Floor(p.Y))+3, z, 8)
				if !ok {
					continue
				}
				below, _, _ := g.World.Block_at(x, y-1, z)
				here, partial, _ := g.World.Block_at(x, y, z)
				if (below != block.GRASS && below != block.SNOWY_GRASS) || here != world.AIR || partial {
					continue
				}
				plant(x, y, z, r, func(id world.CubeID, b world.BlockID) { g.World.Set_cube(id, b) })
				planted++
			}
			g.Show_message(fmt.Sprintf("Planted %d flowers", planted))
		},
		On_break: func(g *game.Game, id world.CubeID, b world.BlockID) bool {
			for _, p := range petals {
				if b == p {
					g.Show_message("You picked a " + block.Get(b).Display)
				}
			}
			return true
		},
	})
}
