package main

import (
	"foxel3d/canvas"
	"foxel3d/world"
)

// 纯色shader
type pureShader struct {
	c canvas.Color
}

func (s pureShader) Type() canvas.ShaderType         { return canvas.ShaderTypePureColor }
func (s pureShader) Shader_pure_color() canvas.Color { return s.c }
func (s pureShader) Shader_Images() canvas.Images    { return canvas.Images{} }

func main() {
	var c canvas.Raylib
	c = c.Init(800, 450)
	defer c.Close()

	orange := pureShader{canvas.Color{R: 230, G: 120, B: 40, A: 255}}
	origin := canvas.Canvas3D{}

	// 每帧绕(1,1,0)轴多转一点，60fps下大约每3秒转一圈
	angle := 0.0
	for !c.Should_close() {
		angle += 0.035
		c = c.Clean()

		c = c.Add_cube(2, origin, orange, world.Rotation3D{X: angle * 0.7071, Y: angle * 0.7071})

		// 坐标轴：x红 y绿 z蓝
		c = c.Add_line3d(origin, canvas.Canvas3D{X: 4}, canvas.Color{R: 255, A: 255})
		c = c.Add_line3d(origin, canvas.Canvas3D{Y: 4}, canvas.Color{G: 200, A: 255})
		c = c.Add_line3d(origin, canvas.Canvas3D{Z: 4}, canvas.Color{B: 255, A: 255})

		c = c.Draw()
	}
}
