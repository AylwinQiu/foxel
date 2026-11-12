package terrain

import (
	"testing"

	"foxel3d_ai/block"
	"foxel3d_ai/world"
)

func TestDeterministic(t *testing.T) {
	pos := world.CubeID{X: 3, Y: 0, Z: -2, Depth: world.CHUNK_DEPTH}
	var a, b world.ChunkData
	New(42).Generate_chunk(pos, &a)
	g := New(42)
	// 先生成一些别的chunk，结果不能变
	var tmp world.ChunkData
	g.Generate_chunk(world.CubeID{X: 4, Y: 0, Z: -2, Depth: world.CHUNK_DEPTH}, &tmp)
	g.Generate_chunk(pos, &b)
	if a.Blocks != b.Blocks {
		t.Fatal("not deterministic")
	}
	var c world.ChunkData
	New(43).Generate_chunk(pos, &c)
	if a.Blocks == c.Blocks {
		t.Fatal("seed ignored")
	}
}

// 把一片区域生成到一个world里
func generate_area(g *Generator, r int64) *world.World {
	w := world.New_world(g)
	for y := int64(MIN_CHUNK_Y); y <= MAX_CHUNK_Y; y++ {
		for z := -r; z <= r; z++ {
			for x := -r; x <= r; x++ {
				w.Load_chunk(world.CubeID{X: x, Y: y, Z: z, Depth: world.CHUNK_DEPTH})
			}
		}
	}
	return w
}

func TestTreesAreWhole(t *testing.T) {
	g := New(7)
	w := generate_area(g, 4)
	// 每个树干方块上面要么是树干，要么是树叶；跨chunk的树两边要对得上
	logs, leaves := 0, 0
	for x := int64(-60); x < 60; x++ {
		for z := int64(-60); z < 60; z++ {
			h := g.Height(x, z)
			for y := h + 1; y < h+15; y++ {
				b, _, _ := w.Block_at(x, y, z)
				switch b {
				case block.OAK_LOG, block.SPRUCE_LOG:
					logs++
					above, _, _ := w.Block_at(x, y+1, z)
					if above != b && above != block.OAK_LEAVES && above != block.SPRUCE_LEAVES {
						t.Fatalf("log at %d %d %d has %d above", x, y, z, above)
					}
				case block.OAK_LEAVES, block.SPRUCE_LEAVES:
					leaves++
				}
			}
		}
	}
	if logs == 0 || leaves < logs {
		t.Fatalf("logs %d leaves %d", logs, leaves)
	}
}

func TestBiomesAndHeights(t *testing.T) {
	g := New(1)
	seen := map[string]bool{}
	lo, hi := int64(1<<40), int64(-1<<40)
	for x := int64(-4000); x <= 4000; x += 40 {
		for z := int64(-4000); z <= 4000; z += 40 {
			c := g.Column(x, z)
			seen[c.Biome.Name] = true
			lo, hi = min(lo, c.Height), max(hi, c.Height)
		}
	}
	if len(seen) != 5 {
		t.Fatalf("biomes %v", seen)
	}
	if lo <= BOTTOM_Y+5 || hi >= (MAX_CHUNK_Y+1)*world.CHUNK_SIZE-25 {
		t.Fatalf("height range %d..%d", lo, hi)
	}
	t.Logf("height %d..%d", lo, hi)
}
