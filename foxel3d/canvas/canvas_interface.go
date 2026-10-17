// 该文件定义屏幕的接口，屏幕同时有输入功能。类似terminal。

package canvas

type Canvas3D struct {
	X float64
	Y float64
	Z float64
}

type Canvas2D struct {
	X float64
	Y float64
}
type ShaderType int8

const (
	PureColor ShaderType = iota
	Images
	//todo：可能未来会有更多shader种类加入
)

// 定义了图片读取接口
type image interface {
	//输入width和height的相对分数，输出图片的颜色在这里的。两个输入都在（0，1）之间
	get(float64, float64) Color
}
type images struct {
	//六个面的填充图片在这里
	imgs [6]image
}

type Shader interface {
	//Shader的填充类型。纯色，图片
	Type() ShaderType
	//该cell全部用纯色填充
	Shader_pure_color() Color
	//该cell用6张图片填充
	Shader_Images() images
	//todo:可能未来会有更多类型的shader加入
}

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// 三维方块的转动量，方向是轴心，长度是转动弧度（弧度制）。pi就是转180度这样
// 用右手螺旋定则
type Rotation3D struct {
	X float64
	Y float64
	Z float64
}

type Canvas[Self any] interface {
	// 创建一个新的窗口。
	Init(width int, height int) Self
	// 添加一个cube绘制
	Add_cube(side float64, center Canvas3D, shader Shader, rot Rotation3D) Self
	// 添加一个线的绘制
	Add_line3d(start Canvas3D, end Canvas3D, c Color) Self
	// 添加一个2d线条在屏幕上
	Add_line2d(start Canvas2D, end Canvas2D, c Color) Self
	//清除所有
	clean() Self
	// 绘制一帧
	Draw() Self
	// 屏幕是否需要关闭
	Should_close() bool
	// 获得窗口大小
	Get_window_size() (uint64, uint64)
	//todo:我们会继续增加新的功能如果需要的话
}
