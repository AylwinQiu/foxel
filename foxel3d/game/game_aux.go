// 这里是一些快速操作的api
package game

import (
	"foxel3d/engine"
	"foxel3d/world"
)

// 方块配置，同一个配置只用反复传入就能创建多个不同的fields
type CubeFields interface {
	//添加新的field
	Add_field(name string, filed_value any)
}

// #region 各种方块配置
// 草坪方块场
func New_grass() CubeFields {
	//todo
}

// #endregion

// 添加一个新的cube在对应位置，如果已经有了直接覆盖，忽略offset
func Set_cube(engine *engine.Engine, fields CubeFields, phypos world.PhysicalPosition3D, depth world.Depth)

// 快速输入物理距离
func PhyPos(x float64, y float64, z float64) world.PhysicalPosition3D {
	return world.PhysicalPosition3D{
		X: x,
		Y: y,
		Z: z,
	}
}
