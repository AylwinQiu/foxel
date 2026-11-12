// 该文件完全ai自主控制
package canvas

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"foxel3d_ai/world"
)

// Raylib 是基于raylib的Canvas实现。
// 采用保留模式：Add_* 只记录绘制指令，Draw 每帧把记录的指令全部画出来，Clean 清空指令。
// 所有图片都放在一张贴图集（atlas）里，网格和2D图片都从这里取。
//
// 用法：r := &canvas.Raylib{}; r.Init(1280, 720); defer r.Close()
type Raylib struct {
	camera rl.Camera3D
	// 纵向、横向视角（角度制，总是正数），fovH为0表示没设置过，此时横向视角按窗口宽高比由纵向视角算出
	fovV, fovH float64
	// 为true时忽略fovV，纵向视角按窗口宽高比由横向视角算出
	fovAuto bool
	// 倾斜角（角度制），相机顺时针转为正
	tilt float64

	background Color
	fog        Color
	fog_start  float64
	fog_end    float64

	shader   rl.Shader
	material rl.Material
	locs     struct{ fog_color, fog_start, fog_end, view_pos int32 }
	atlas    *atlas

	meshes  map[MeshID]rl.Mesh
	draws   []mesh_draw
	cubes   []raylib_cube
	lines3d []raylib_line3d
	// 2D的东西按添加顺序画
	overlay []func()

	input Input
	mouse mouse_state
}

type mesh_draw struct {
	id     MeshID
	origin Canvas3D
}

type raylib_cube struct {
	side   float32
	center rl.Vector3
	// 转轴（单位向量）和转角（角度制，rlgl需要），angle为0时不转
	axis  rl.Vector3
	angle float32
	// 依次是East, West, Up, Down, South, North，和cube_faces一致
	tiles [6]int
}

type raylib_line3d struct {
	start rl.Vector3
	end   rl.Vector3
	color rl.Color
}

// 编译期检查 Raylib 实现了 Canvas
var _ Canvas = (*Raylib)(nil)

// Add_2d_text的字号（像素）
const raylib_text_size = 20

// 网格用的shader：贴图颜色乘顶点颜色，alpha小于0.5的像素不画，然后加上雾
const mesh_vs = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec4 vertexColor;
uniform mat4 mvp;
uniform mat4 matModel;
uniform vec3 viewPos;
out vec2 fragTexCoord;
out vec4 fragColor;
out float fragDist;
void main() {
    fragTexCoord = vertexTexCoord;
    fragColor = vertexColor;
    fragDist = length((matModel * vec4(vertexPosition, 1.0)).xyz - viewPos);
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const mesh_fs = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
in float fragDist;
uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec4 fogColor;
uniform float fogStart;
uniform float fogEnd;
out vec4 finalColor;
void main() {
    vec4 c = texture(texture0, fragTexCoord) * fragColor * colDiffuse;
    if (c.a < 0.5) discard;
    float f = clamp((fragDist - fogStart) / max(fogEnd - fogStart, 0.001), 0.0, 1.0);
    finalColor = vec4(mix(c.rgb, fogColor.rgb, f), 1.0);
}
`

func (r *Raylib) Init(width int, height int) {
	// 窗口可以拖动修改大小，必须在InitWindow之前设置；3D视角的宽高比每帧按当前窗口大小计算
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint | rl.FlagVsyncHint)
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(int32(width), int32(height), "foxel3d")
	rl.SetTargetFPS(60)
	// Esc留给游戏用（解锁鼠标），不要让它关窗口
	rl.SetExitKey(0)

	r.camera = rl.Camera3D{
		Position:   rl.NewVector3(10, 10, 10),
		Target:     rl.NewVector3(0, 0, 0),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}
	r.fovV = 45
	r.background = SkyBlue
	r.fog = SkyBlue
	r.fog_start, r.fog_end = 1e6, 2e6
	r.meshes = map[MeshID]rl.Mesh{}
	r.atlas = new_atlas()

	r.shader = rl.LoadShaderFromMemory(mesh_vs, mesh_fs)
	r.locs.fog_color = rl.GetShaderLocation(r.shader, "fogColor")
	r.locs.fog_start = rl.GetShaderLocation(r.shader, "fogStart")
	r.locs.fog_end = rl.GetShaderLocation(r.shader, "fogEnd")
	r.locs.view_pos = rl.GetShaderLocation(r.shader, "viewPos")
	// 用局部变量：传给C的指针所在的Go对象里不能有Go指针，r里面有
	m := rl.LoadMaterialDefault()
	m.Shader = r.shader
	rl.SetMaterialTexture(&m, rl.MapDiffuse, r.atlas.tex)
	r.material = m
}

func (r *Raylib) Close() {
	for id := range r.meshes {
		r.Delete_mesh(id)
	}
	rl.UnloadTexture(r.atlas.tex)
	rl.UnloadShader(r.shader)
	rl.CloseWindow()
}

// rot 的长度按弧度解释（pi 即转180度）
func (r *Raylib) Add_cube(side float64, center Canvas3D, img Images, rot world.Rotation3D) {
	c := raylib_cube{
		side:   float32(side),
		center: to_rl_vec3(center),
	}
	if l := math.Sqrt(rot.X*rot.X + rot.Y*rot.Y + rot.Z*rot.Z); l > 0 {
		c.axis = rl.NewVector3(float32(rot.X/l), float32(rot.Y/l), float32(rot.Z/l))
		c.angle = float32(l * 180 / math.Pi)
	}
	for i, face := range [6]Image{img.East, img.West, img.Up, img.Down, img.South, img.North} {
		c.tiles[i] = r.atlas.tile_of(face)
	}
	r.cubes = append(r.cubes, c)
}

func (r *Raylib) Add_line3d(start Canvas3D, end Canvas3D, c Color) {
	r.lines3d = append(r.lines3d, raylib_line3d{to_rl_vec3(start), to_rl_vec3(end), to_rl_color(c)})
}

func (r *Raylib) Add_2d_text(text string, pos Canvas2D, color Color) {
	r.overlay = append(r.overlay, func() {
		rl.DrawTextEx(rl.GetFontDefault(), text, to_rl_vec2(pos), raylib_text_size, raylib_text_size/10, to_rl_color(color))
	})
}

func (r *Raylib) Measure_2d_text(text string) Canvas2D {
	v := rl.MeasureTextEx(rl.GetFontDefault(), text, raylib_text_size, raylib_text_size/10)
	return Canvas2D{float64(v.X), float64(v.Y)}
}

func (r *Raylib) Add_2d_line(start Canvas2D, end Canvas2D, color Color) {
	r.overlay = append(r.overlay, func() {
		rl.DrawLineV(to_rl_vec2(start), to_rl_vec2(end), to_rl_color(color))
	})
}

func (r *Raylib) Add_2d_rect(pos Canvas2D, size Canvas2D, color Color) {
	r.overlay = append(r.overlay, func() {
		rl.DrawRectangleV(to_rl_vec2(pos), to_rl_vec2(size), to_rl_color(color))
	})
}

func (r *Raylib) Add_2d_image(img Image, pos Canvas2D, size Canvas2D) {
	t := r.atlas.tile_of(img)
	r.overlay = append(r.overlay, func() {
		x, y := r.atlas.tile_origin(t)
		src := rl.NewRectangle(float32(x), float32(y), atlas_tile, atlas_tile)
		dst := rl.NewRectangle(float32(pos.X), float32(pos.Y), float32(size.X), float32(size.Y))
		rl.DrawTexturePro(r.atlas.tex, src, dst, rl.Vector2{}, 0, rl.White)
	})
}

func (r *Raylib) Set_background(c Color) { r.background = c }

func (r *Raylib) Set_fog(c Color, start float64, end float64) {
	r.fog, r.fog_start, r.fog_end = c, start, end
}

func (r *Raylib) Clean() {
	r.draws = r.draws[:0]
	r.cubes = r.cubes[:0]
	r.lines3d = r.lines3d[:0]
	r.overlay = r.overlay[:0]
}

// 每个四边形拆成两个三角形
var quad_order = [6]int{0, 1, 2, 0, 2, 3}

func (r *Raylib) Set_mesh(id MeshID, quads []Quad) {
	if len(quads) == 0 {
		r.Delete_mesh(id)
		return
	}
	n := len(quads) * 6
	verts := make([]float32, 0, n*3)
	uvs := make([]float32, 0, n*2)
	cols := make([]uint8, 0, n*4)
	for i := range quads {
		q := &quads[i]
		t := r.atlas.tile_of(q.Img)
		for _, k := range quad_order {
			c := q.Corners[k]
			u, v := r.atlas.uv(t, q.UV[k].X, q.UV[k].Y)
			verts = append(verts, float32(c.X), float32(c.Y), float32(c.Z))
			uvs = append(uvs, u, v)
			cols = append(cols, q.Tint.R, q.Tint.G, q.Tint.B, q.Tint.A)
		}
	}
	mesh := rl.Mesh{
		VertexCount:   int32(n),
		TriangleCount: int32(n / 3),
		Vertices:      &verts[0],
		Texcoords:     &uvs[0],
		Colors:        &cols[0],
	}
	rl.UploadMesh(&mesh, false)
	// 数据已经在显卡里了。去掉指向Go内存的指针，之后把mesh传给C时才安全
	mesh.Vertices, mesh.Texcoords, mesh.Colors = nil, nil, nil
	r.Delete_mesh(id)
	r.meshes[id] = mesh
}

func (r *Raylib) Delete_mesh(id MeshID) {
	if m, ok := r.meshes[id]; ok {
		rl.UnloadMesh(&m)
		delete(r.meshes, id)
	}
}

func (r *Raylib) Add_mesh(id MeshID, origin Canvas3D) {
	r.draws = append(r.draws, mesh_draw{id, origin})
}

func (r *Raylib) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(to_rl_color(r.background))

	// BeginMode3D只支持一个fovy，这里用自己算的矩阵覆盖它设置的投影和视图矩阵，EndMode3D照常恢复2D的矩阵
	rl.BeginMode3D(r.camera)
	rl.SetMatrixProjection(r.projection())
	rl.SetMatrixModelview(r.view())

	p := r.camera.Position
	rl.SetShaderValue(r.shader, r.locs.view_pos, []float32{p.X, p.Y, p.Z}, rl.ShaderUniformVec3)
	f := r.fog
	rl.SetShaderValue(r.shader, r.locs.fog_color, []float32{float32(f.R) / 255, float32(f.G) / 255, float32(f.B) / 255, 1}, rl.ShaderUniformVec4)
	rl.SetShaderValue(r.shader, r.locs.fog_start, []float32{float32(r.fog_start)}, rl.ShaderUniformFloat)
	rl.SetShaderValue(r.shader, r.locs.fog_end, []float32{float32(r.fog_end)}, rl.ShaderUniformFloat)
	for _, d := range r.draws {
		if m, ok := r.meshes[d.id]; ok {
			rl.DrawMesh(m, r.material, rl.MatrixTranslate(float32(d.origin.X), float32(d.origin.Y), float32(d.origin.Z)))
		}
	}
	for _, c := range r.cubes {
		r.draw_cube(c)
	}
	for _, l := range r.lines3d {
		rl.DrawLine3D(l.start, l.end, l.color)
	}
	rl.EndMode3D()

	for _, f := range r.overlay {
		f()
	}
	rl.EndDrawing()

	r.input = r.read_input()
}

func (r *Raylib) Should_close() bool {
	return rl.WindowShouldClose()
}

func (r *Raylib) Get_window_size() (uint64, uint64) {
	return uint64(rl.GetScreenWidth()), uint64(rl.GetScreenHeight())
}

func (r *Raylib) Get_frame_time() float64 {
	return float64(rl.GetFrameTime())
}

// direction只取方向，长度为0时不改动相机
// 上方向默认是+Y；视线和Y轴平行（正上/正下看）时改用-Z作为上方向，避免相机退化
func (r *Raylib) Camera3D_loc_and_dir(location Canvas3D, direction Canvas3D) {
	if vec3_len(direction) == 0 {
		return
	}
	dir := vec3_normalize(direction)
	pos := to_rl_vec3(location)
	r.camera.Position = pos
	r.camera.Target = rl.Vector3Add(pos, to_rl_vec3(dir))
	r.camera.Up = to_rl_vec3(camera_up(dir))
}

// 相机的上方向，dir必须是单位向量
func camera_up(dir Canvas3D) Canvas3D {
	up := Canvas3D{Y: 1}
	if vec3_len(vec3_cross(dir, up)) < 1e-6 {
		up = Canvas3D{Z: -1}
	}
	return up
}

// 视角取绝对值后必须在(0, 180)内（auto时不检查vertical），否则整个调用不生效
func (r *Raylib) Camera3D_vis_angle(horizontal float64, vertical float64, tilt float64, auto bool) {
	horizontal, vertical = math.Abs(horizontal), math.Abs(vertical)
	if !valid_fov(horizontal) || (!auto && !valid_fov(vertical)) || math.IsNaN(tilt) || math.IsInf(tilt, 0) {
		return
	}
	r.fovH = horizontal
	if !auto {
		r.fovV = vertical
	}
	r.fovAuto = auto
	r.tilt = tilt
}

func valid_fov(deg float64) bool {
	return deg > 0 && deg < 180
}

const (
	raylib_near = 0.05
	raylib_far  = 4000.0
)

func (r *Raylib) projection() rl.Matrix {
	tanH, tanV := r.half_fov_tan()
	right, top := raylib_near*tanH, raylib_near*tanV
	return rl.MatrixFrustum(float32(-right), float32(right), float32(-top), float32(top), raylib_near, raylib_far)
}

// 横向、纵向半视角的正切，也就是视线方向距离为1处，画面右边缘和上边缘离中心的距离
func (r *Raylib) half_fov_tan() (tanH, tanV float64) {
	aspect := 1.0
	if w, h := rl.GetScreenWidth(), rl.GetScreenHeight(); w > 0 && h > 0 {
		aspect = float64(w) / float64(h)
	}
	switch {
	case r.fovH == 0:
		tanV = math.Tan(r.fovV * math.Pi / 360)
		tanH = tanV * aspect
	case r.fovAuto:
		tanH = math.Tan(r.fovH * math.Pi / 360)
		tanV = tanH / aspect
	default:
		tanH = math.Tan(r.fovH * math.Pi / 360)
		tanV = math.Tan(r.fovV * math.Pi / 360)
	}
	return tanH, tanV
}

// 用当前的视角、倾斜和窗口宽高比，算出相机朝camera_direction看时视锥四条棱的方向
// 上方向的规则和Camera3D_loc_and_dir一致；camera_direction长度为0时用相机当前的朝向
func (r *Raylib) Get_frustum_corner_rays(camera_direction Canvas3D) (Canvas3D, Canvas3D, Canvas3D, Canvas3D) {
	f := camera_direction
	if vec3_len(f) == 0 {
		d := rl.Vector3Subtract(r.camera.Target, r.camera.Position)
		f = Canvas3D{X: float64(d.X), Y: float64(d.Y), Z: float64(d.Z)}
	}
	f = vec3_normalize(f)
	right := vec3_normalize(vec3_cross(f, camera_up(f)))
	up := vec3_cross(right, f)

	tanH, tanV := r.half_fov_tan()
	sin, cos := math.Sincos(r.tilt * math.Pi / 180)
	// 画面上(x, y)处的射线：view()里画面相对相机转了+tilt，这里转回-tilt得到相机自身坐标系里的方向
	ray := func(x, y float64) Canvas3D {
		rx := x*cos + y*sin
		ry := -x*sin + y*cos
		return vec3_normalize(vec3_add(vec3_add(vec3_scale(right, rx), vec3_scale(up, ry)), f))
	}
	return ray(-tanH, tanV), ray(tanH, tanV), ray(-tanH, -tanV), ray(tanH, -tanV)
}

// 相机顺时针转等于画面逆时针转，所以在视图空间里绕+Z（指向屏幕外）转+tilt
func (r *Raylib) view() rl.Matrix {
	v := rl.MatrixLookAt(r.camera.Position, r.camera.Target, r.camera.Up)
	if r.tilt != 0 {
		v = rl.MatrixMultiply(v, rl.MatrixRotateZ(float32(r.tilt*math.Pi/180)))
	}
	return v
}

// 单位cube（中心在原点）每个面的4个角，从外面看逆时针：左上、左下、右下、右上，
// 对应图片的(0,0)、(0,1)、(1,1)、(1,0)。顺序是East, West, Up, Down, South, North
var cube_faces = [6]struct {
	normal  rl.Vector3
	corners [4]rl.Vector3
	shade   uint8
}{
	{v3(1, 0, 0), [4]rl.Vector3{v3(.5, .5, .5), v3(.5, -.5, .5), v3(.5, -.5, -.5), v3(.5, .5, -.5)}, 170},
	{v3(-1, 0, 0), [4]rl.Vector3{v3(-.5, .5, -.5), v3(-.5, -.5, -.5), v3(-.5, -.5, .5), v3(-.5, .5, .5)}, 170},
	{v3(0, 1, 0), [4]rl.Vector3{v3(-.5, .5, -.5), v3(-.5, .5, .5), v3(.5, .5, .5), v3(.5, .5, -.5)}, 255},
	{v3(0, -1, 0), [4]rl.Vector3{v3(-.5, -.5, .5), v3(-.5, -.5, -.5), v3(.5, -.5, -.5), v3(.5, -.5, .5)}, 130},
	{v3(0, 0, 1), [4]rl.Vector3{v3(-.5, .5, .5), v3(-.5, -.5, .5), v3(.5, -.5, .5), v3(.5, .5, .5)}, 210},
	{v3(0, 0, -1), [4]rl.Vector3{v3(.5, .5, -.5), v3(.5, -.5, -.5), v3(-.5, -.5, -.5), v3(-.5, .5, -.5)}, 210},
}

var face_uv = [4][2]float64{{0, 0}, {0, 1}, {1, 1}, {1, 0}}

func v3(x, y, z float32) rl.Vector3 { return rl.NewVector3(x, y, z) }

func (r *Raylib) draw_cube(c raylib_cube) {
	rl.PushMatrix()
	rl.Translatef(c.center.X, c.center.Y, c.center.Z)
	if c.angle != 0 {
		rl.Rotatef(c.angle, c.axis.X, c.axis.Y, c.axis.Z)
	}
	rl.Scalef(c.side, c.side, c.side)
	rl.SetTexture(r.atlas.tex.ID)
	rl.Begin(rl.Quads)
	for i, f := range cube_faces {
		rl.Color4ub(f.shade, f.shade, f.shade, 255)
		rl.Normal3f(f.normal.X, f.normal.Y, f.normal.Z)
		for k := range 4 {
			u, v := r.atlas.uv(c.tiles[i], face_uv[k][0], face_uv[k][1])
			rl.TexCoord2f(u, v)
			rl.Vertex3f(f.corners[k].X, f.corners[k].Y, f.corners[k].Z)
		}
	}
	rl.End()
	rl.SetTexture(0)
	rl.PopMatrix()
}

func vec3_add(a, b Canvas3D) Canvas3D { return Canvas3D{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }

func vec3_scale(a Canvas3D, k float64) Canvas3D { return Canvas3D{X: a.X * k, Y: a.Y * k, Z: a.Z * k} }

func vec3_cross(a, b Canvas3D) Canvas3D {
	return Canvas3D{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}

func vec3_len(a Canvas3D) float64 { return math.Sqrt(a.X*a.X + a.Y*a.Y + a.Z*a.Z) }

func vec3_normalize(a Canvas3D) Canvas3D { return vec3_scale(a, 1/vec3_len(a)) }

func to_rl_vec3(v Canvas3D) rl.Vector3 {
	return rl.NewVector3(float32(v.X), float32(v.Y), float32(v.Z))
}

func to_rl_vec2(v Canvas2D) rl.Vector2 {
	return rl.NewVector2(float32(v.X), float32(v.Y))
}

func to_rl_color(c Color) rl.Color {
	return rl.NewColor(c.R, c.G, c.B, c.A)
}
