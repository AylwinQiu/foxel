// 引擎将world和canvas组合。
//
// 相机位置保存在engine里（物理坐标，float64精度）。传给canvas时所有坐标都先减去相机位置再乘CANVAS_DIV_PHY_3D，
// 所以canvas的相机永远在原点，离原点很远时也不会因为raylib的float32精度而抖动。
//
// 一帧的典型写法：
//
//	e.Begin_frame()
//	e.Draw_world()
//	e.Add_line3d(a, b, color) // 可选：额外画的东西
//	e.End_frame()
package engine

import (
	"math"

	"foxel3d/canvas"
	"foxel3d/world"
)

const (
	// 3D情况下屏幕长度除以物理长度的值
	CANVAS_DIV_PHY_3D float64 = 30.0
	// 2D情况下屏幕长度除以物理长度的值
	CANVAS_DIV_PHY_2D float64 = 30.0
)

// 物理长度换算成canvas里的3D长度
func Phy_pos_to_canvas(x float64) float64 {
	return x * CANVAS_DIV_PHY_3D
}

type EngineInterface interface {
	// 拿到世界，里面装的是*world.World，修改会直接作用在engine的世界上
	Get_world() world.WorldInterface[any]
	// 拿到canvas
	Get_canvas() canvas.Canvas

	// 设置相机在世界里的位置和朝向，direction只取方向，长度为0时只改位置
	Set_camera(position world.PhysicalPosition3D, direction canvas.Canvas3D)
	// 让相机在position看向target
	Look_at(position world.PhysicalPosition3D, target world.PhysicalPosition3D)
	// 相机的位置和朝向（单位向量）
	Get_camera() (world.PhysicalPosition3D, canvas.Canvas3D)
	// 设置视角，参数含义同canvas.Camera3D_vis_angle
	Set_vis_angle(horizontal float64, vertical float64, tilt float64, auto bool)

	// 设置cube 6个面的图片，images是canvas.Images{}（6个面都是nil）时恢复默认图片
	Set_images(id world.CubeID, images canvas.Images)
	// 没有单独设置图片的cube用这个
	Set_default_images(images canvas.Images)

	// 在canvas上绘制对应id的cube，需要提供相机方向。不做可见性判断，找不到的id跳过
	// camera_direction会成为新的相机朝向，长度为0时用当前朝向
	Draw_cubes_by_id(ids []world.CubeID, camera_direction canvas.Canvas3D)
	// 获得可见的cubes的ids（只有叶子cube，不含根），camera_direction同上
	Get_visable_cube_ids(camera_direction canvas.Canvas3D) []world.CubeID
	// 把视锥内所有可见的cube画到canvas上
	Draw_world()
	// 世界里的3D线条，用物理坐标
	Add_line3d(start world.PhysicalPosition3D, end world.PhysicalPosition3D, c canvas.Color)
	// 屏幕上的2D线条，单位是像素
	Add_2d_line(start canvas.Canvas2D, end canvas.Canvas2D, c canvas.Color)
	// 屏幕上的文字，pos是左上角，单位是像素
	Add_2d_text(text string, pos canvas.Canvas2D, c canvas.Color)

	// 清空canvas，开始新的一帧
	Begin_frame()
	// 把这一帧画出来
	End_frame()
	// 窗口是否需要关闭
	Should_close() bool
	// todo
}

type Engine struct {
	canvas canvas.Canvas
	world  world.WorldInterface[any]

	cameraPos world.PhysicalPosition3D
	// 单位向量
	cameraDir canvas.Canvas3D

	images        map[world.CubeID]canvas.Images
	defaultImages canvas.Images
}

// 编译期检查 Engine 实现了 EngineInterface
var _ EngineInterface = (*Engine)(nil)

// 新建一个Engine，c必须已经Init过。相机默认在(10, 10, 10)看向原点，cube默认是灰色
func NewEngine(w world.WorldInterface[any], c canvas.Canvas) *Engine {
	e := &Engine{
		canvas:        c,
		world:         w,
		images:        map[world.CubeID]canvas.Images{},
		defaultImages: canvas.Images{}.Color(canvas.Color{R: 160, G: 160, B: 160, A: 255}),
	}
	e.Look_at(world.PhysicalPosition3D{X: 10, Y: 10, Z: 10}, world.PhysicalPosition3D{})
	return e
}

// ---------------------------------------------------------------- world / canvas

func (e *Engine) Get_world() world.WorldInterface[any] {
	return e.world
}

func (e *Engine) Get_canvas() canvas.Canvas {
	return e.canvas
}

// ---------------------------------------------------------------- 相机

func (e *Engine) Set_camera(position world.PhysicalPosition3D, direction canvas.Canvas3D) {
	e.cameraPos = position
	l := math.Sqrt(direction.X*direction.X + direction.Y*direction.Y + direction.Z*direction.Z)
	if l == 0 {
		return
	}
	e.cameraDir = canvas.Canvas3D{X: direction.X / l, Y: direction.Y / l, Z: direction.Z / l}
	e.canvas.Camera3D_loc_and_dir(canvas.Canvas3D{}, e.cameraDir)
}

func (e *Engine) Look_at(position world.PhysicalPosition3D, target world.PhysicalPosition3D) {
	e.Set_camera(position, canvas.Canvas3D{
		X: target.X - position.X,
		Y: target.Y - position.Y,
		Z: target.Z - position.Z,
	})
}

func (e *Engine) Get_camera() (world.PhysicalPosition3D, canvas.Canvas3D) {
	return e.cameraPos, e.cameraDir
}

func (e *Engine) Set_vis_angle(horizontal float64, vertical float64, tilt float64, auto bool) {
	e.canvas.Camera3D_vis_angle(horizontal, vertical, tilt, auto)
}

// camera_direction长度不为0时改成新的相机朝向
func (e *Engine) useDirection(camera_direction canvas.Canvas3D) {
	e.Set_camera(e.cameraPos, camera_direction)
}

// ---------------------------------------------------------------- 图片

func (e *Engine) Set_images(id world.CubeID, images canvas.Images) {
	if images == (canvas.Images{}) {
		delete(e.images, id)
		return
	}
	e.images[id] = images
}

func (e *Engine) Set_default_images(images canvas.Images) {
	e.defaultImages = images
}

func (e *Engine) imagesOf(id world.CubeID) canvas.Images {
	if img, ok := e.images[id]; ok {
		return img
	}
	return e.defaultImages
}

// ---------------------------------------------------------------- 可见性

// cube的外接球半径（以边长为单位）
const cubeRadius = 0.8661

// 子树包围球半径（以边长为单位）：每层offset不超过0.5个边长时，
// 子树里所有东西离这个cube中心不超过 sqrt(3) 个边长
const subtreeRadius = 1.7321

// 平面过相机位置，法线指向视锥里面
type frustumPlane struct {
	normal canvas.Canvas3D
}

// 四个侧面加一个相机所在的近平面（只剔除相机背后的东西），远平面不剔除
func (e *Engine) frustumPlanes() [5]frustumPlane {
	tl, tr, bl, br := e.canvas.Get_frustum_corner_rays(e.cameraDir)
	side := func(a, b canvas.Canvas3D) frustumPlane {
		n := cross(a, b)
		l := math.Sqrt(dot(n, n))
		n = canvas.Canvas3D{X: n.X / l, Y: n.Y / l, Z: n.Z / l}
		// 法线朝向视线这一侧
		if dot(n, e.cameraDir) < 0 {
			n = canvas.Canvas3D{X: -n.X, Y: -n.Y, Z: -n.Z}
		}
		return frustumPlane{n}
	}
	return [5]frustumPlane{side(tl, bl), side(tr, br), side(tl, tr), side(bl, br), {e.cameraDir}}
}

func (e *Engine) sphereVisible(planes [5]frustumPlane, center world.PhysicalPosition3D, radius float64) bool {
	d := canvas.Canvas3D{X: center.X - e.cameraPos.X, Y: center.Y - e.cameraPos.Y, Z: center.Z - e.cameraPos.Z}
	for _, p := range planes {
		if dot(p.normal, d) < -radius {
			return false
		}
	}
	return true
}

// 视锥内可见的叶子cube（不含根）
func (e *Engine) visibleCubes() []world.CubeInfo {
	planes := e.frustumPlanes()
	// 子树里的cube可能因为offset和rotation伸出格子，用宽松的包围球剪枝
	all := e.world.Collect(func(info world.CubeInfo) bool {
		return e.sphereVisible(planes, info.Center, info.Side*subtreeRadius)
	})
	visible := all[:0]
	for _, info := range all {
		if info.Is_leaf && !info.Is_root && e.sphereVisible(planes, info.Center, info.Side*cubeRadius) {
			visible = append(visible, info)
		}
	}
	return visible
}

func (e *Engine) Get_visable_cube_ids(camera_direction canvas.Canvas3D) []world.CubeID {
	e.useDirection(camera_direction)
	cubes := e.visibleCubes()
	ids := make([]world.CubeID, len(cubes))
	for i, info := range cubes {
		ids[i] = info.Id
	}
	return ids
}

// ---------------------------------------------------------------- 绘制

func (e *Engine) Begin_frame() {
	e.canvas.Clean()
}

func (e *Engine) Draw_world() {
	for _, info := range e.visibleCubes() {
		e.drawCube(info.Id, info.Center, info.Rotation, info.Side)
	}
}

// 每个id单独从world查询位置和旋转（目前是O(cube数)的遍历），大量cube时用Draw_world更快
func (e *Engine) Draw_cubes_by_id(ids []world.CubeID, camera_direction canvas.Canvas3D) {
	e.useDirection(camera_direction)
	for _, id := range ids {
		center, err := e.world.Get_cube_phy_pos_by_id(id)
		if err != nil {
			continue
		}
		rot, err := e.world.Get_cube_rotation_by_id(id)
		if err != nil {
			continue
		}
		side, err := e.world.Get_cube_side_by_id(id)
		if err != nil {
			continue
		}
		e.drawCube(id, center, rot, side)
	}
}

func (e *Engine) drawCube(id world.CubeID, center world.PhysicalPosition3D, rot world.Rotation3D, side float64) {
	e.canvas.Add_cube(Phy_pos_to_canvas(side), e.toCanvas(center), e.imagesOf(id), rot)
}

func (e *Engine) Add_line3d(start world.PhysicalPosition3D, end world.PhysicalPosition3D, c canvas.Color) {
	e.canvas.Add_line3d(e.toCanvas(start), e.toCanvas(end), c)
}

func (e *Engine) Add_2d_line(start canvas.Canvas2D, end canvas.Canvas2D, c canvas.Color) {
	e.canvas.Add_2d_line(start, end, c)
}

func (e *Engine) Add_2d_text(text string, pos canvas.Canvas2D, c canvas.Color) {
	e.canvas.Add_2d_text(text, pos, c)
}

func (e *Engine) End_frame() {
	e.canvas.Draw()
}

func (e *Engine) Should_close() bool {
	return e.canvas.Should_close()
}

// 物理坐标 -> canvas坐标：先减去相机位置，再缩放
func (e *Engine) toCanvas(p world.PhysicalPosition3D) canvas.Canvas3D {
	return canvas.Canvas3D{
		X: Phy_pos_to_canvas(p.X - e.cameraPos.X),
		Y: Phy_pos_to_canvas(p.Y - e.cameraPos.Y),
		Z: Phy_pos_to_canvas(p.Z - e.cameraPos.Z),
	}
}

func dot(a, b canvas.Canvas3D) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func cross(a, b canvas.Canvas3D) canvas.Canvas3D {
	return canvas.Canvas3D{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
