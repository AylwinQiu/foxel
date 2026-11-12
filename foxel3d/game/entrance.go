package game

import (
	"foxel3d/engine"
	"foxel3d/world"
)

// 游戏帧绘制入口，同时获得控制反馈
func Draw_frame(e engine.EngineInterface) {
	blue := engine.ColorImages(engine.Blue)
	// world.Depth{}是深度0，边长1
	engine.Add_cube(e, world.PhysicalPosition3D{X: 0.0, Y: 0.0, Z: 0.0}, blue, engine.NoRotation, world.Depth{})
}
