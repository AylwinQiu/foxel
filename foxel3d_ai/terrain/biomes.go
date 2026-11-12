package terrain

import (
	"foxel3d_ai/block"
	"foxel3d_ai/world"
)

// 生物群系：决定地面的高度、起伏和用什么方块
type Biome struct {
	// 唯一的名字，mod请用"mod名:群系名"
	Name string
	// 气候：温度、湿度都在-1到1之间。每一列选气候最接近的生物群系
	Temperature float64
	Humidity    float64
	// 地面的平均高度和起伏
	Base_height float64
	Variation   float64
	// true时用山脊形状的噪声（只往上起伏），适合山脉
	Ridged bool
	// 最上面一层和下面几层的方块，再往下是石头
	Surface          world.BlockID
	Subsurface       world.BlockID
	Subsurface_depth int
	// 地面高于这个高度时地面是石头，0表示没有
	Rock_line int64
	// 地面高于这个高度时盖上雪，0表示没有
	Snow_line int64
}

var biomes []*Biome

// 添加生物群系，名字重复时替换旧的
func Register_biome(b *Biome) {
	for i, old := range biomes {
		if old.Name == b.Name {
			biomes[i] = b
			return
		}
	}
	biomes = append(biomes, b)
}

// 没有时返回nil
func Biome_by_name(name string) *Biome {
	for _, b := range biomes {
		if b.Name == name {
			return b
		}
	}
	return nil
}

// 内置的生物群系
var (
	Plains = &Biome{Name: "foxel:plains", Temperature: 0.2, Humidity: -0.1,
		Base_height: 12, Variation: 5, Surface: block.GRASS, Subsurface: block.DIRT, Subsurface_depth: 4}
	Forest = &Biome{Name: "foxel:forest", Temperature: 0.25, Humidity: 0.5,
		Base_height: 15, Variation: 9, Surface: block.GRASS, Subsurface: block.DIRT, Subsurface_depth: 4}
	Desert = &Biome{Name: "foxel:desert", Temperature: 0.8, Humidity: -0.6,
		Base_height: 10, Variation: 4, Surface: block.SAND, Subsurface: block.SANDSTONE, Subsurface_depth: 6}
	Mountains = &Biome{Name: "foxel:mountains", Temperature: -0.2, Humidity: -0.5,
		Base_height: 24, Variation: 70, Ridged: true, Surface: block.GRASS, Subsurface: block.DIRT, Subsurface_depth: 3,
		Rock_line: 58, Snow_line: 78}
	Taiga = &Biome{Name: "foxel:taiga", Temperature: -0.7, Humidity: 0.3,
		Base_height: 16, Variation: 10, Surface: block.SNOWY_GRASS, Subsurface: block.DIRT, Subsurface_depth: 4}
)

func init() {
	for _, b := range []*Biome{Plains, Forest, Desert, Mountains, Taiga} {
		Register_biome(b)
	}
}
