// 该文件定义屏幕的接口，屏幕同时有输入功能。类似terminal。

package canvas

import "foxel3d/world"

type Canvas3D struct {
	X float64
	Y float64
	Z float64
}

type Canvas2D struct {
	X float64
	Y float64
}

// 方块贴图的样式
type ShaderType uint8

// 定义了不同种类的方块贴图
const (
	ShaderTypePureColor ShaderType = iota
	ShaderTypeImages
	//todo：可能未来会有更多shader种类加入
)

// 定义了图片读取接口
type Image interface {
	//输入width和height的相对分数，输出图片的颜色在这里的。两个输入都在（0，1）之间
	Get(float64, float64) Color
}
type Images struct {
	//六个面的填充图片在这里
	Imgs [6]Image
}

type Shader interface {
	//Shader的填充类型。纯色，图片
	Type() ShaderType
	//该cube全部用纯色填充
	Shader_pure_color() Color
	//该cube用6张图片填充
	Shader_Images() Images
	//todo:可能未来会有更多类型的shader加入
}

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// 一个新的窗口需要符合这个接口。
type Canvas[Self any] interface {
	// 创建一个新的窗口。
	Init(width int, height int) Self
	// 添加一个cube绘制
	Add_cube(side float64, center Canvas3D, shader Shader, rot world.Rotation3D) Self
	// 添加一个线的绘制
	Add_line3d(start Canvas3D, end Canvas3D, c Color) Self
	// 添加一个2d线条在屏幕上
	Add_line2d(start Canvas2D, end Canvas2D, c Color) Self
	//清除所有
	Clean() Self
	// 绘制一帧
	Draw() Self
	// 屏幕是否需要关闭
	Should_close() bool
	// 获得窗口大小
	Get_window_size() (uint64, uint64)
	// 设置3d相机的位置和朝向，direction是看向的方向，向量长度没有意义。只要direction的方向。
	Camera3D_loc_and_dir(location Canvas3D, direction Canvas3D)
	//todo:我们会继续增加新的功能如果需要的话
}
