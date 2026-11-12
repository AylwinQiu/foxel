package main

import (
	"foxel3d/canvas"
	"foxel3d/engine"
	"foxel3d/world"
	"foxel3d/game"
)

func main() {
	w := world.New_World()
	c := &canvas.Raylib{}
	c.Init(800, 450)
	defer c.Close()
	e := engine.NewEngine(world.WorldInterface[any](&w), c)

	// 深度1的cube边长是2；(1, 1, 1)正好是格子中心，所以offset是0，cube原地转
	center := world.PhysicalPosition3D{X: 1, Y: 1, Z: 1}
	depth, _ := world.NewDepth(1)
	id := e.Get_world().Add_simple_cube(world.Phy2tree(center, depth))
	e.Set_images(id, engine.ColorImages(canvas.Color{R: 230, G: 120, B: 40, A: 255}))
	e.Look_at(world.PhysicalPosition3D{X: 6, Y: 5, Z: 7}, center)

	// 每帧绕(1,1,0)轴多转一点，60fps下大约每3秒转一圈
	angle := 0.0
	for !e.Should_close() {
		angle += 0.035
		e.Get_world().Set_rotation(id, world.Rotation3D{X: angle * 0.7071, Y: angle * 0.7071}, false)

		e.Begin_frame()
		//e.Draw_world()
		game.Draw_frame(e)
		drawAxes(e)
		e.End_frame()
	}
}

// 坐标轴：x红 y绿 z蓝
func drawAxes(e engine.EngineInterface) {
	origin := world.PhysicalPosition3D{}
	e.Add_line3d(origin, world.PhysicalPosition3D{X: 4}, canvas.Color{R: 255, A: 255})
	e.Add_line3d(origin, world.PhysicalPosition3D{Y: 4}, canvas.Color{G: 200, A: 255})
	e.Add_line3d(origin, world.PhysicalPosition3D{Z: 4}, canvas.Color{B: 255, A: 255})
}
