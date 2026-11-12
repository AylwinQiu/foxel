// 定义了一些封装让engine更容易写
package engine

import (
	"foxel3d/canvas"
	"foxel3d/world"
)

// 常用颜色
var (
	Red   = canvas.Color{R: 255, G: 0, B: 0, A: 255}
	Green = canvas.Color{R: 0, G: 255, B: 0, A: 255}
	Blue  = canvas.Color{R: 0, G: 0, B: 255, A: 255}

	White     = canvas.Color{R: 255, G: 255, B: 255, A: 255}
	Black     = canvas.Color{R: 0, G: 0, B: 0, A: 255}
	Gray      = canvas.Color{R: 128, G: 128, B: 128, A: 255}
	LightGray = canvas.Color{R: 200, G: 200, B: 200, A: 255}
	DarkGray  = canvas.Color{R: 80, G: 80, B: 80, A: 255}

	Yellow  = canvas.Color{R: 255, G: 255, B: 0, A: 255}
	Cyan    = canvas.Color{R: 0, G: 255, B: 255, A: 255}
	Magenta = canvas.Color{R: 255, G: 0, B: 255, A: 255}
	Orange  = canvas.Color{R: 255, G: 165, B: 0, A: 255}
	Purple  = canvas.Color{R: 128, G: 0, B: 128, A: 255}
	Pink    = canvas.Color{R: 255, G: 192, B: 203, A: 255}
	Brown   = canvas.Color{R: 139, G: 69, B: 19, A: 255}

	DarkRed   = canvas.Color{R: 139, G: 0, B: 0, A: 255}
	DarkGreen = canvas.Color{R: 0, G: 100, B: 0, A: 255}
	DarkBlue  = canvas.Color{R: 0, G: 0, B: 139, A: 255}
	SkyBlue   = canvas.Color{R: 135, G: 206, B: 235, A: 255}
	Gold      = canvas.Color{R: 255, G: 215, B: 0, A: 255}

	// 完全透明
	Transparent = canvas.Color{R: 0, G: 0, B: 0, A: 0}
)

// 整个cube都是这个颜色，等于canvas.Images{}.Color(color_name)
func ColorImages(color_name canvas.Color) canvas.Images {
	return canvas.Images{}.Color(color_name)
}

// 6个面都贴url这张图片，等于canvas.Images{}.Url(url)
func UrlImages(url string) canvas.Images {
	return canvas.Images{}.Url(url)
}

// 不旋转
var NoRotation = world.Rotation3D{
	X: 0.0,
	Y: 0.0,
	Z: 0.0,
}

// 在phypos所在、深度为depth的格子（边长2^depth）里添加一个cube，6个面的图片是images，返回它的id
// 纯色cube用ColorImages(color)，贴图片用UrlImages(url)
// 忽略offset：cube的中心是phypos所在格子的中心，不一定正好在phypos
// abs_rotation是cube在世界里的朝向（已经叠加了父cube的旋转），不转就传NoRotation
// 这个格子里已经有cube时不新建，直接改那个cube的图片和朝向（它原来的offset不变）
// phypos在根cube外面时panic（同Phy2tree）
func Add_cube(engine EngineInterface, phypos world.PhysicalPosition3D, images canvas.Images, abs_rotation world.Rotation3D, depth world.Depth) world.CubeID {
	w := engine.Get_world()
	id := w.Add_simple_cube(world.Phy2tree(phypos, depth).Without_offset())
	w.Set_rotation(id, abs_rotation, true)
	engine.Set_images(id, images)
	return id
}
