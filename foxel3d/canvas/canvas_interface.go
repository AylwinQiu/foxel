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

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

type Image interface {
	//获得图片相对位置的颜色，
	//@ensure 0.0<=x_frac<=1.0 && 0.0<=y_frac<=1.0
	Get_pixel(x_frac float64, y_frac float64) Color
}

type Images struct {
	Up    Image
	Down  Image
	North Image
	South Image
	West  Image
	East  Image
}

// 可以从网络上通过url或者本地路径来读取资源,读取一张图片并且粘贴到6个面上。
// http(s)开头的从网络下载，其他的当作本地路径，支持png、jpeg、gif。用法：canvas.Images{}.Url("assets/a.png")
// 同一个url只读取一次。读取失败时6个面都是品红色，并在stderr打印原因。
func (images Images) Url(url string) Images {
	return images.All(loadPicture(url))
}

// 6个面都用img
func (images Images) All(img Image) Images {
	return Images{Up: img, Down: img, North: img, South: img, West: img, East: img}
}

// 6个面都是纯色c
func (images Images) Color(c Color) Images {
	return images.All(SolidColor(c))
}

// 一个新的窗口需要符合这个接口。
type Canvas interface {
	// 创建一个新的窗口。
	Init(width int, height int)
	// 添加一个cube绘制
	Add_cube(side float64, center Canvas3D, img Images, rot world.Rotation3D)
	// 添加一个线的绘制
	Add_line3d(start Canvas3D, end Canvas3D, c Color)
	//清除所有
	Clean()
	// 绘制一帧
	Draw()
	// 屏幕是否需要关闭
	Should_close() bool
	// 获得窗口大小
	Get_window_size() (uint64, uint64)
	// 设置3d相机的位置和朝向，direction是看向的方向，向量长度没有意义。只要direction的方向。
	Camera3D_loc_and_dir(location Canvas3D, direction Canvas3D)
	// 设置3d相机的横向视角、纵向视角和倾斜角度，都用角度制。
	// horizontal和vertical是负数时取绝对值。
	// auto为true时忽略vertical，纵向视角由horizontal按窗口宽高比算出，画面不会被拉伸。
	// auto为false时横向、纵向视角都固定，和窗口宽高比不一致时画面会被拉伸。
	// 倾斜是相机顺时针为正，逆时针为负。
	Camera3D_vis_angle(horizontal float64, vertical float64, tilt float64, auto bool)
	// 获得视锥体四角射线，返回时左上TL, 右上TR, 左下BL, 右下BR, 射线长度是1.0，方向是从相机往外指。
	Get_frustum_corner_rays(camera_direction Canvas3D) (Canvas3D, Canvas3D, Canvas3D, Canvas3D)
	//todo:我们会继续增加新的功能如果需要的话
	//
	//接下来是一些2d接口，因为游戏可能需要绘制一些2d信息，包括文字deng
	//
	//添加文字在特定位置。
	Add_2d_text(text string, pos Canvas2D, color Color)
	//添加2d线条
	Add_2d_line(start Canvas2D, end Canvas2D, color Color)

	//控制反馈
	//获得当前帧按下的键有什么。
	Get_keys()
}
