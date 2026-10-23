// 引擎将world和canvas组合。
package main

import (
	"foxel3d/canvas"
	"foxel3d/world"
)

const (
	// 3D情况下屏幕长度除以物理长度的值
	CANVAS_DIV_PHY_3D float64 = 30.0
	// 2D情况下屏幕长度除以物理长度的值
	CANVAS_DIV_PHY_2D float64 = 30.0
)

func Phy_pos_to_canvas(x float64)

type EngineInterface interface {
	// todo
	// 拿到世界
	Get_world() *world.WorldInterface[any]
	// 拿到canvas
	Get_canvas() *canvas.Canvas[any]
	// 在canvas上绘制对应id的cube，需要提供相机方向。
	Draw_cubes_by_id(ids []world.CubeID, camera_direction canvas.Canvas3D)
	// 获得可见的cubes的ids
	Get_visable_cube_ids(camera_direction canvas.Canvas3D) []world.CubeID
	// todo
}

type Engine struct {
	canvas canvas.Canvas[any]
	world  world.WorldInterface[any]
}

// 新建一个Engine
func NewEngine(world world.World, canvas canvas.Canvas[any]) {
	//todo
}
