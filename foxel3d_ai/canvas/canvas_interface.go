// 该文件定义屏幕的接口，屏幕同时有输入功能。类似terminal。
//
// 坐标约定：x正方向是东，y正方向是上，z正方向是南（北是-z），右手系。
// 2D坐标单位是像素，左上角是原点，x向右，y向下。
package canvas

import "foxel3d_ai/world"

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
	// 获得图片相对位置的颜色，(0, 0)是左上角
	// @ensure 0.0<=x_frac<=1.0 && 0.0<=y_frac<=1.0
	Get_pixel(x_frac float64, y_frac float64) Color
}

// 知道自己像素大小的图片。canvas按这个大小采样，像素风贴图不会糊
type SizedImage interface {
	Image
	Size() (width int, height int)
}

// cube 6个面的图片，nil的面是白色。
// 每张图从cube外面看是正的：侧面图片的上方朝上，Up面图片的上方朝北，Down面图片的上方朝南。
//
// 同一张图片会被canvas缓存成贴图，所以Image必须是可以比较的（指针或者不含slice/map的值），
// 而且画了之后内容不要再改。
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
	return images.All(Load_picture(url))
}

// 6个面都用img
func (images Images) All(img Image) Images {
	return Images{Up: img, Down: img, North: img, South: img, West: img, East: img}
}

// 6个面都是纯色c
func (images Images) Color(c Color) Images {
	return images.All(SolidColor(c))
}

// 上面是top，下面是bottom，4个侧面是side，比如草方块、木头
func (images Images) Column(top Image, side Image, bottom Image) Images {
	return Images{Up: top, Down: bottom, North: side, South: side, West: side, East: side}
}

// 网格里的一个四边形面。
// Corners从正面看是逆时针排列的，背面不画；UV是对应角在图片上的相对位置（0到1，左上角是(0, 0)）。
// 图片里alpha小于一半的像素不画，可以用来做树叶、玻璃这种镂空的面。
type Quad struct {
	Corners [4]Canvas3D
	UV      [4]Canvas2D
	Img     Image
	// 乘在图片颜色上，用来做明暗。{255, 255, 255, 255}是原样
	Tint Color
}

// 网格的编号，由使用者自己分配
type MeshID uint64

// 一个新的窗口需要符合这个接口。
//
// 每一帧：Clean() -> Add_xxx() -> Draw()。Add_xxx添加的东西只画这一帧。
// 网格（Set_mesh）是例外：上传一次之后一直留在显卡里，每帧只用Add_mesh说明画在哪里。
// 大量静止的方块（比如地形）应该做成网格，Add_cube只适合少量会动的东西。
type Canvas interface {
	// 创建一个新的窗口。
	Init(width int, height int)
	// 关闭窗口，释放资源
	Close()
	// 添加一个cube绘制，side是边长，center是中心
	Add_cube(side float64, center Canvas3D, img Images, rot world.Rotation3D)
	// 添加一个线的绘制
	Add_line3d(start Canvas3D, end Canvas3D, c Color)
	// 清除这一帧添加的所有东西（网格不会被删除）
	Clean()
	// 绘制一帧，然后读入下一帧的输入
	Draw()
	// 屏幕是否需要关闭
	Should_close() bool
	// 获得窗口大小
	Get_window_size() (uint64, uint64)
	// 上一帧用了多少秒
	Get_frame_time() float64
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
	// 背景（天空）的颜色
	Set_background(c Color)
	// 雾：离相机start以内没有雾，到end完全变成c。网格（Set_mesh）才有雾
	Set_fog(c Color, start float64, end float64)

	// 网格：创建或替换id这个网格。quads是空的时候等于Delete_mesh
	Set_mesh(id MeshID, quads []Quad)
	// 删除网格，不存在时什么都不做
	Delete_mesh(id MeshID)
	// 这一帧把网格画在origin处（网格里的坐标都加上origin）
	Add_mesh(id MeshID, origin Canvas3D)

	//接下来是一些2d接口，因为游戏可能需要绘制一些2d信息，包括文字等。按添加的顺序画，后添加的盖在上面
	//
	// 添加文字在特定位置，pos是左上角，字号固定20像素，只支持ASCII，可以用\n换行
	Add_2d_text(text string, pos Canvas2D, color Color)
	// 文字画出来有多大
	Measure_2d_text(text string) Canvas2D
	// 添加2d线条
	Add_2d_line(start Canvas2D, end Canvas2D, color Color)
	// 添加实心矩形，pos是左上角
	Add_2d_rect(pos Canvas2D, size Canvas2D, color Color)
	// 添加图片，pos是左上角
	Add_2d_image(img Image, pos Canvas2D, size Canvas2D)

	// 控制反馈
	//
	// 获得这一帧的输入，同一帧里调用几次结果都一样。窗口没有焦点时什么都没有
	Get_input() Input
	// 锁定鼠标：光标隐藏，鼠标可以一直往一个方向移动（第一人称视角用）
	Set_cursor_locked(locked bool)
	Is_cursor_locked() bool
}

// 一帧的输入
type Input struct {
	// 这一帧按着的键和鼠标键，包括这一帧内按下又松开的
	Held map[Key]bool
	// 这一帧刚按下的键和鼠标键
	Pressed map[Key]bool
	// 这一帧鼠标移动了多少像素，x向右、y向下为正。鼠标刚进入窗口、刚锁定的那一帧是0
	Mouse_dx float64
	Mouse_dy float64
	// 鼠标在窗口里的位置（像素）
	Mouse_x float64
	Mouse_y float64
	// 滚轮，向上滚为正
	Wheel float64
}

func (in Input) Is_held(k Key) bool    { return in.Held[k] }
func (in Input) Is_pressed(k Key) bool { return in.Pressed[k] }

type Key int

const (
	KEY_NONE Key = iota
	KEY_A
	KEY_B
	KEY_C
	KEY_D
	KEY_E
	KEY_F
	KEY_G
	KEY_H
	KEY_I
	KEY_J
	KEY_K
	KEY_L
	KEY_M
	KEY_N
	KEY_O
	KEY_P
	KEY_Q
	KEY_R
	KEY_S
	KEY_T
	KEY_U
	KEY_V
	KEY_W
	KEY_X
	KEY_Y
	KEY_Z
	KEY_0
	KEY_1
	KEY_2
	KEY_3
	KEY_4
	KEY_5
	KEY_6
	KEY_7
	KEY_8
	KEY_9
	KEY_SPACE
	KEY_ENTER
	KEY_ESCAPE
	KEY_TAB
	KEY_BACKSPACE
	KEY_LEFT_SHIFT
	KEY_RIGHT_SHIFT
	KEY_LEFT_CTRL
	KEY_RIGHT_CTRL
	KEY_LEFT_ALT
	KEY_RIGHT_ALT
	KEY_UP
	KEY_DOWN
	KEY_LEFT
	KEY_RIGHT
	KEY_F1
	KEY_F2
	KEY_F3
	KEY_F4
	KEY_F5
	KEY_F6
	KEY_F7
	KEY_F8
	KEY_F9
	KEY_F10
	KEY_F11
	KEY_F12
	MOUSE_LEFT
	MOUSE_RIGHT
	MOUSE_MIDDLE
)
