// 这里是一些快速操作的api
package game

import (
	"foxel3d_ai/world"
)

// 快速输入物理距离
func PhyPos(x float64, y float64, z float64) world.PhysicalPosition3D {
	return world.PhysicalPosition3D{X: x, Y: y, Z: z}
}

// 把物理位置p所在的、深度为depth的格子设成方块b（不经过mod的On_break/On_place）
func (g *Game) Set_cube(p world.PhysicalPosition3D, depth world.Depth, b world.BlockID) error {
	return g.World.Set_cube(world.Cube_id_at(p, depth), b)
}

// 物理位置p的方块（包含p的最小的那个cube），没加载时返回AIR
func (g *Game) Block_at(p world.PhysicalPosition3D) world.BlockID {
	l, _ := g.World.Leaf_at(p)
	return l.Block
}

// (x, z)这一列从y往下数第一个实心方块上面的空格子的y，找不到时返回false
func (g *Game) Surface_below(x, y, z int64, max_depth int64) (int64, bool) {
	for d := int64(0); d < max_depth; d++ {
		b, partial, ok := g.World.Block_at(x, y-d, z)
		if !ok {
			return 0, false
		}
		if partial || b != world.AIR {
			return y - d + 1, true
		}
	}
	return 0, false
}
