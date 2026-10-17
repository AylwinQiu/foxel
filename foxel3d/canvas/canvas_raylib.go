// 该文件完全ai自主控制
package canvas

import (
	"math"
	"reflect"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Raylib 是基于raylib的Canvas实现。
// 采用保留模式：Add_* 只记录绘制指令，Draw 每帧把记录的指令全部画出来，clean 清空指令。
// 内部状态放在指针里，所以 Raylib 按值传递、链式调用时都共享同一份状态。
type Raylib struct {
	s *raylibState
}

type raylibState struct {
	camera  rl.Camera3D
	cubes   []raylibCube
	lines3d []raylibLine3D
	lines2d []raylibLine2D

	// 可比较的image按值缓存贴图，跨clean保留，Close时释放
	texCache map[image]rl.Texture2D
	// 不可比较的image没法缓存，生成的贴图在clean时释放
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
		texCache: map[image]rl.Texture2D{},
	}}
}

// rot 的长度按弧度解释（pi 即转180度）
// 图片shader的6张图依次对应 +X, -X, +Y, -Y, +Z, -Z 面
func (r Raylib) Add_cube(side float64, center Canvas3D, shader Shader, rot Rotation3D) Raylib {
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
	case PureColor:
		c.color = toRlColor(shader.Shader_pure_color())
	case Images:
		imgs := shader.Shader_Images().imgs
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

func (r Raylib) clean() Raylib {
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

	rl.BeginMode3D(r.s.camera)
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

func (r Raylib) Get_window_size() (uint64, uint64) {
	return uint64(rl.GetScreenWidth()), uint64(rl.GetScreenHeight())
}

// 不在Canvas接口里，用完后调用以释放贴图并关闭窗口
func (r Raylib) Close() {
	r.clean()
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
	case PureColor:
		rl.DrawCube(rl.NewVector3(0, 0, 0), c.side, c.side, c.side, c.color)
	case Images:
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

func (s *raylibState) faceTexture(img image) rl.Texture2D {
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
func sampleTexture(img image) rl.Texture2D {
	const n = raylibFaceTexSize
	data := make([]byte, n*n*4)
	for y := range n {
		for x := range n {
			c := Color{255, 255, 255, 255}
			if img != nil {
				c = img.get((float64(x)+0.5)/n, (float64(y)+0.5)/n)
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
