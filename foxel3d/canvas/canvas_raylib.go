// 该文件完全ai自主控制
package canvas

import (
	"math"
	"reflect"

	rl "github.com/gen2brain/raylib-go/raylib"

	"foxel3d/world"
)

// Raylib 是基于raylib的Canvas实现。
// 采用保留模式：Add_* 只记录绘制指令，Draw 每帧把记录的指令全部画出来，Clean 清空指令。
// 内部状态放在指针里，所以 Raylib 按值传递、链式调用时都共享同一份状态。
type Raylib struct {
	s *raylibState
}

type raylibState struct {
	camera rl.Camera3D
	// 纵向、横向视角（角度制，总是正数），fovH为0表示没设置过，此时横向视角按窗口宽高比由纵向视角算出
	fovV, fovH float64
	// 为true时忽略fovV，纵向视角按窗口宽高比由横向视角算出
	fovAuto bool
	// 倾斜角（角度制），相机顺时针转为正
	tilt    float64
	cubes   []raylibCube
	lines3d []raylibLine3D
	lines2d []raylibLine2D

	// 可比较的image按值缓存贴图，跨Clean保留，Close时释放
	texCache map[Image]rl.Texture2D
	// 不可比较的image没法缓存，生成的贴图在Clean时释放
	texTemp []rl.Texture2D
}

type raylibCube struct {
	side   float32
	center rl.Vector3
	// 转轴（单位向量）和转角（角度制，rlgl需要），angle为0时不转
	axis  rl.Vector3
	angle float32

	shaderType ShaderType
	color      rl.Color
	faces      [6]rl.Texture2D
}

type raylibLine3D struct {
	start rl.Vector3
	end   rl.Vector3
	color rl.Color
}

type raylibLine2D struct {
	start rl.Vector2
	end   rl.Vector2
	color rl.Color
}

// 编译期检查 Raylib 实现了 Canvas[Raylib]
var _ Canvas[Raylib] = Raylib{}

var raylibBackground = rl.RayWhite

// 图片shader采样成贴图时每个面的分辨率
const raylibFaceTexSize = 64

func (r Raylib) Init(width int, height int) Raylib {
	// 窗口可以拖动修改大小，必须在InitWindow之前设置；3D视角的宽高比raylib每帧按当前窗口大小计算
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(int32(width), int32(height), "foxel3d")
	rl.SetTargetFPS(60)
	return Raylib{s: &raylibState{
		camera: rl.Camera3D{
			Position:   rl.NewVector3(10, 10, 10),
			Target:     rl.NewVector3(0, 0, 0),
			Up:         rl.NewVector3(0, 1, 0),
			Fovy:       45,
			Projection: rl.CameraPerspective,
		},
		fovV:     45,
		texCache: map[Image]rl.Texture2D{},
	}}
}

// rot 的长度按弧度解释（pi 即转180度）
// 图片shader的6张图依次对应 +X, -X, +Y, -Y, +Z, -Z 面
func (r Raylib) Add_cube(side float64, center Canvas3D, shader Shader, rot world.Rotation3D) Raylib {
	c := raylibCube{
		side:       float32(side),
		center:     toRlVec3(center),
		shaderType: shader.Type(),
	}

	if l := math.Sqrt(rot.X*rot.X + rot.Y*rot.Y + rot.Z*rot.Z); l > 0 {
		c.axis = rl.NewVector3(float32(rot.X/l), float32(rot.Y/l), float32(rot.Z/l))
		c.angle = float32(l * 180 / math.Pi)
	}

	switch c.shaderType {
	case ShaderTypePureColor:
		c.color = toRlColor(shader.Shader_pure_color())
	case ShaderTypeImages:
		imgs := shader.Shader_Images().Imgs
		for i, img := range imgs {
			c.faces[i] = r.s.faceTexture(img)
		}
	}

	r.s.cubes = append(r.s.cubes, c)
	return r
}

func (r Raylib) Add_line3d(start Canvas3D, end Canvas3D, c Color) Raylib {
	r.s.lines3d = append(r.s.lines3d, raylibLine3D{
		start: toRlVec3(start),
		end:   toRlVec3(end),
		color: toRlColor(c),
	})
	return r
}

// 坐标单位是屏幕像素，左上角为原点
func (r Raylib) Add_line2d(start Canvas2D, end Canvas2D, c Color) Raylib {
	r.s.lines2d = append(r.s.lines2d, raylibLine2D{
		start: rl.NewVector2(float32(start.X), float32(start.Y)),
		end:   rl.NewVector2(float32(end.X), float32(end.Y)),
		color: toRlColor(c),
	})
	return r
}

func (r Raylib) Clean() Raylib {
	r.s.cubes = r.s.cubes[:0]
	r.s.lines3d = r.s.lines3d[:0]
	r.s.lines2d = r.s.lines2d[:0]
	for _, t := range r.s.texTemp {
		rl.UnloadTexture(t)
	}
	r.s.texTemp = r.s.texTemp[:0]
	return r
}

func (r Raylib) Draw() Raylib {
	rl.BeginDrawing()
	rl.ClearBackground(raylibBackground)

	// BeginMode3D只支持一个fovy，这里用自己算的矩阵覆盖它设置的投影和视图矩阵，EndMode3D照常恢复2D的矩阵
	rl.BeginMode3D(r.s.camera)
	rl.SetMatrixProjection(r.s.projection())
	rl.SetMatrixModelview(r.s.view())
	for _, c := range r.s.cubes {
		drawCube(c)
	}
	for _, l := range r.s.lines3d {
		rl.DrawLine3D(l.start, l.end, l.color)
	}
	rl.EndMode3D()

	for _, l := range r.s.lines2d {
		rl.DrawLineV(l.start, l.end, l.color)
	}

	rl.EndDrawing()
	return r
}

func (r Raylib) Should_close() bool {
	return rl.WindowShouldClose()
}

// direction只取方向，长度为0时不改动相机
// 上方向默认是+Y；视线和Y轴平行（正上/正下看）时改用-Z作为上方向，避免相机退化
func (r Raylib) Camera3D_loc_and_dir(location Canvas3D, direction Canvas3D) {
	dir := toRlVec3(direction)
	if rl.Vector3Length(dir) == 0 {
		return
	}
	dir = rl.Vector3Normalize(dir)

	pos := toRlVec3(location)
	r.s.camera.Position = pos
	r.s.camera.Target = rl.Vector3Add(pos, dir)
	r.s.camera.Up = toRlVec3(cameraUp(Canvas3D{X: float64(dir.X), Y: float64(dir.Y), Z: float64(dir.Z)}))
}

// 相机的上方向，dir必须是单位向量
func cameraUp(dir Canvas3D) Canvas3D {
	up := Canvas3D{Y: 1}
	if vec3Len(vec3Cross(dir, up)) < 1e-6 {
		up = Canvas3D{Z: -1}
	}
	return up
}

// 视角取绝对值后必须在(0, 180)内（auto时不检查vertical），否则整个调用不生效
func (r Raylib) Camera3D_vis_angle(horizontal float64, vertical float64, tilt float64, auto bool) {
	horizontal, vertical = math.Abs(horizontal), math.Abs(vertical)
	if !validFov(horizontal) || (!auto && !validFov(vertical)) || math.IsNaN(tilt) || math.IsInf(tilt, 0) {
		return
	}
	r.s.fovH = horizontal
	if !auto {
		r.s.fovV = vertical
	}
	r.s.fovAuto = auto
	r.s.tilt = tilt
}

func validFov(deg float64) bool {
	return deg > 0 && deg < 180
}

// 和raylib默认的近、远裁剪面一致
const (
	raylibNear = 0.05
	raylibFar  = 4000.0
)

func (s *raylibState) projection() rl.Matrix {
	tanH, tanV := s.halfFovTan()
	right, top := raylibNear*tanH, raylibNear*tanV
	return rl.MatrixFrustum(float32(-right), float32(right), float32(-top), float32(top), raylibNear, raylibFar)
}

// 横向、纵向半视角的正切，也就是视线方向距离为1处，画面右边缘和上边缘离中心的距离
func (s *raylibState) halfFovTan() (tanH, tanV float64) {
	aspect := 1.0
	if w, h := rl.GetScreenWidth(), rl.GetScreenHeight(); w > 0 && h > 0 {
		aspect = float64(w) / float64(h)
	}
	switch {
	case s.fovH == 0:
		tanV = math.Tan(s.fovV * math.Pi / 360)
		tanH = tanV * aspect
	case s.fovAuto:
		tanH = math.Tan(s.fovH * math.Pi / 360)
		tanV = tanH / aspect
	default:
		tanH = math.Tan(s.fovH * math.Pi / 360)
		tanV = math.Tan(s.fovV * math.Pi / 360)
	}
	return tanH, tanV
}

// 用当前的视角、倾斜和窗口宽高比，算出相机朝camera_direction看时视锥四条棱的方向
// 上方向的规则和Camera3D_loc_and_dir一致；camera_direction长度为0时用相机当前的朝向
func (r Raylib) Get_frustum_corner_rays(camera_direction Canvas3D) (Canvas3D, Canvas3D, Canvas3D, Canvas3D) {
	f := camera_direction
	if vec3Len(f) == 0 {
		d := rl.Vector3Subtract(r.s.camera.Target, r.s.camera.Position)
		f = Canvas3D{X: float64(d.X), Y: float64(d.Y), Z: float64(d.Z)}
	}
	f = vec3Normalize(f)
	right := vec3Normalize(vec3Cross(f, cameraUp(f)))
	up := vec3Cross(right, f)

	tanH, tanV := r.s.halfFovTan()
	sin, cos := math.Sincos(r.s.tilt * math.Pi / 180)
	// 画面上(x, y)处的射线：view()里画面相对相机转了+tilt，这里转回-tilt得到相机自身坐标系里的方向
	ray := func(x, y float64) Canvas3D {
		rx := x*cos + y*sin
		ry := -x*sin + y*cos
		d := vec3Add(vec3Add(vec3Scale(right, rx), vec3Scale(up, ry)), f)
		return vec3Normalize(d)
	}
	return ray(-tanH, tanV), ray(tanH, tanV), ray(-tanH, -tanV), ray(tanH, -tanV)
}

func vec3Add(a, b Canvas3D) Canvas3D { return Canvas3D{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }

func vec3Scale(a Canvas3D, k float64) Canvas3D { return Canvas3D{X: a.X * k, Y: a.Y * k, Z: a.Z * k} }

func vec3Cross(a, b Canvas3D) Canvas3D {
	return Canvas3D{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}

func vec3Len(a Canvas3D) float64 { return math.Sqrt(a.X*a.X + a.Y*a.Y + a.Z*a.Z) }

func vec3Normalize(a Canvas3D) Canvas3D { return vec3Scale(a, 1/vec3Len(a)) }

// 相机顺时针转等于画面逆时针转，所以在视图空间里绕+Z（指向屏幕外）转+tilt
func (s *raylibState) view() rl.Matrix {
	v := rl.MatrixLookAt(s.camera.Position, s.camera.Target, s.camera.Up)
	if s.tilt != 0 {
		v = rl.MatrixMultiply(v, rl.MatrixRotateZ(float32(s.tilt*math.Pi/180)))
	}
	return v
}

func (r Raylib) Get_window_size() (uint64, uint64) {
	return uint64(rl.GetScreenWidth()), uint64(rl.GetScreenHeight())
}

// 不在Canvas接口里，用完后调用以释放贴图并关闭窗口
func (r Raylib) Close() {
	r.Clean()
	for _, t := range r.s.texCache {
		rl.UnloadTexture(t)
	}
	r.s.texCache = nil
	rl.CloseWindow()
}

// 每个面：法线，以及面上 u=0,v=0 / u=1,v=0 / u=1,v=1 / u=0,v=1 四个角（单位cube，中心在原点）
// u沿图片宽度，v沿图片高度（v=0为图片顶部），从cube外面看过去图片是正的
var cubeFaces = [6]struct {
	normal  rl.Vector3
	corners [4]rl.Vector3
}{
	{rl.NewVector3(1, 0, 0), [4]rl.Vector3{v3(0.5, 0.5, 0.5), v3(0.5, 0.5, -0.5), v3(0.5, -0.5, -0.5), v3(0.5, -0.5, 0.5)}},
	{rl.NewVector3(-1, 0, 0), [4]rl.Vector3{v3(-0.5, 0.5, -0.5), v3(-0.5, 0.5, 0.5), v3(-0.5, -0.5, 0.5), v3(-0.5, -0.5, -0.5)}},
	{rl.NewVector3(0, 1, 0), [4]rl.Vector3{v3(-0.5, 0.5, -0.5), v3(0.5, 0.5, -0.5), v3(0.5, 0.5, 0.5), v3(-0.5, 0.5, 0.5)}},
	{rl.NewVector3(0, -1, 0), [4]rl.Vector3{v3(-0.5, -0.5, 0.5), v3(0.5, -0.5, 0.5), v3(0.5, -0.5, -0.5), v3(-0.5, -0.5, -0.5)}},
	{rl.NewVector3(0, 0, 1), [4]rl.Vector3{v3(-0.5, 0.5, 0.5), v3(0.5, 0.5, 0.5), v3(0.5, -0.5, 0.5), v3(-0.5, -0.5, 0.5)}},
	{rl.NewVector3(0, 0, -1), [4]rl.Vector3{v3(0.5, 0.5, -0.5), v3(-0.5, 0.5, -0.5), v3(-0.5, -0.5, -0.5), v3(0.5, -0.5, -0.5)}},
}

func v3(x, y, z float32) rl.Vector3 { return rl.NewVector3(x, y, z) }

var faceUV = [4][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}}

func drawCube(c raylibCube) {
	rl.PushMatrix()
	rl.Translatef(c.center.X, c.center.Y, c.center.Z)
	if c.angle != 0 {
		rl.Rotatef(c.angle, c.axis.X, c.axis.Y, c.axis.Z)
	}

	switch c.shaderType {
	case ShaderTypePureColor:
		rl.DrawCube(rl.NewVector3(0, 0, 0), c.side, c.side, c.side, c.color)
	case ShaderTypeImages:
		rl.Scalef(c.side, c.side, c.side)
		for i, f := range cubeFaces {
			// 每个面一张贴图，所以每个面单独一个batch
			rl.SetTexture(c.faces[i].ID)
			rl.Begin(rl.Quads)
			rl.Color4ub(255, 255, 255, 255)
			rl.Normal3f(f.normal.X, f.normal.Y, f.normal.Z)
			// raylib的Quads是逆时针为正面，corners按顺时针排列，这里倒序提交
			for k := 3; k >= 0; k-- {
				rl.TexCoord2f(faceUV[k][0], faceUV[k][1])
				rl.Vertex3f(f.corners[k].X, f.corners[k].Y, f.corners[k].Z)
			}
			rl.End()
		}
		rl.SetTexture(0)
	}

	rl.PopMatrix()
}

func (s *raylibState) faceTexture(img Image) rl.Texture2D {
	cacheable := img != nil && reflect.ValueOf(img).Comparable()
	if cacheable {
		if t, ok := s.texCache[img]; ok {
			return t
		}
	}
	t := sampleTexture(img)
	if cacheable {
		s.texCache[img] = t
	} else {
		s.texTemp = append(s.texTemp, t)
	}
	return t
}

// 在像素中心采样，保证输入严格落在(0,1)内；img为nil时用纯白
func sampleTexture(img Image) rl.Texture2D {
	const n = raylibFaceTexSize
	data := make([]byte, n*n*4)
	for y := range n {
		for x := range n {
			c := Color{255, 255, 255, 255}
			if img != nil {
				c = img.Get((float64(x)+0.5)/n, (float64(y)+0.5)/n)
			}
			p := (y*n + x) * 4
			data[p], data[p+1], data[p+2], data[p+3] = c.R, c.G, c.B, c.A
		}
	}
	return rl.LoadTextureFromImage(rl.NewImage(data, n, n, 1, rl.UncompressedR8g8b8a8))
}

func toRlVec3(v Canvas3D) rl.Vector3 {
	return rl.NewVector3(float32(v.X), float32(v.Y), float32(v.Z))
}

func toRlColor(c Color) rl.Color {
	return rl.NewColor(c.R, c.G, c.B, c.A)
}
