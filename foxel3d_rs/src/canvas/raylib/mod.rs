//! 基于raylib的Canvas实现。
//!
//! 采用保留模式：add_*只记录绘制指令，draw每帧把记录的指令全部画出来，clean清空指令。
//! 所有图片都放在一张贴图集（atlas）里，网格和2D图片都从这里取。

mod atlas;
mod input;

use std::collections::HashMap;
use std::ffi::CString;

use ::raylib::ffi as rl;

use super::{Canvas, Color, Image, Images, Input, MeshId, Quad, Vec2, Vec3};
use atlas::Atlas;

/// add_text的字号（像素）
const TEXT_SIZE: f32 = 20.0;
const NEAR: f64 = 0.05;
const FAR: f64 = 4000.0;

// 网格用的shader：贴图颜色乘顶点颜色，alpha小于0.5的像素不画，然后加上雾
const MESH_VS: &str = r#"#version 330
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
"#;

const MESH_FS: &str = r#"#version 330
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
"#;

struct CubeDraw {
    side: f32,
    center: rl::Vector3,
    /// 转轴（单位向量）和转角（角度制，rlgl需要），angle为0时不转
    axis: rl::Vector3,
    angle: f32,
    /// 依次是east, west, up, down, south, north，和CUBE_FACES一致
    tiles: [usize; 6],
}

/// 2D的东西按添加顺序画
enum Overlay {
    Text(CString, Vec2, Color),
    Line(Vec2, Vec2, Color),
    Rect(Vec2, Vec2, Color),
    Image(usize, Vec2, Vec2),
}

struct Locs {
    fog_color: i32,
    fog_start: i32,
    fog_end: i32,
    view_pos: i32,
}

pub struct RaylibCanvas {
    camera: rl::Camera3D,
    /// 纵向、横向视角（角度制）。fov_h为0表示没设置过，这时横向视角按窗口宽高比由纵向视角算出
    fov_v: f64,
    fov_h: f64,
    /// 为true时忽略fov_v，纵向视角按窗口宽高比由横向视角算出
    fov_auto: bool,
    /// 倾斜角（角度制），相机顺时针转为正
    tilt: f64,

    background: Color,
    fog: Color,
    fog_start: f64,
    fog_end: f64,

    shader: rl::Shader,
    material: rl::Material,
    locs: Locs,
    atlas: Atlas,

    meshes: HashMap<MeshId, rl::Mesh>,
    draws: Vec<(MeshId, Vec3)>,
    cubes: Vec<CubeDraw>,
    lines: Vec<(rl::Vector3, rl::Vector3, rl::Color)>,
    overlay: Vec<Overlay>,

    input: Input,
    mouse: input::MouseState,
    /// 这一帧画完以后存截图
    screenshot: Option<CString>,
}

fn v3(v: Vec3) -> rl::Vector3 {
    rl::Vector3 { x: v.x as f32, y: v.y as f32, z: v.z as f32 }
}

fn v2(v: Vec2) -> rl::Vector2 {
    rl::Vector2 { x: v.x as f32, y: v.y as f32 }
}

fn col(c: Color) -> rl::Color {
    rl::Color { r: c.r, g: c.g, b: c.b, a: c.a }
}

fn cstr(s: &str) -> CString {
    CString::new(s.replace('\0', "")).unwrap()
}

impl RaylibCanvas {
    /// 创建一个新的窗口
    pub fn new(width: i32, height: i32, title: &str) -> RaylibCanvas {
        unsafe {
            // 窗口可以拖动修改大小，必须在InitWindow之前设置；3D视角的宽高比每帧按当前窗口大小计算
            rl::SetConfigFlags(
                rl::ConfigFlags::FLAG_WINDOW_RESIZABLE as u32
                    | rl::ConfigFlags::FLAG_MSAA_4X_HINT as u32
                    | rl::ConfigFlags::FLAG_VSYNC_HINT as u32,
            );
            rl::SetTraceLogLevel(rl::TraceLogLevel::LOG_WARNING as i32);
            let t = cstr(title);
            rl::InitWindow(width, height, t.as_ptr());
            rl::SetTargetFPS(60);
            // Esc留给游戏用（解锁鼠标），不要让它关窗口
            rl::SetExitKey(0);

            let vs = cstr(MESH_VS);
            let fs = cstr(MESH_FS);
            let shader = rl::LoadShaderFromMemory(vs.as_ptr(), fs.as_ptr());
            let loc = |name: &str| {
                let n = cstr(name);
                rl::GetShaderLocation(shader, n.as_ptr())
            };
            let locs = Locs {
                fog_color: loc("fogColor"),
                fog_start: loc("fogStart"),
                fog_end: loc("fogEnd"),
                view_pos: loc("viewPos"),
            };
            let atlas = Atlas::new();
            let mut material = rl::LoadMaterialDefault();
            material.shader = shader;
            rl::SetMaterialTexture(&mut material, rl::MaterialMapIndex::MATERIAL_MAP_ALBEDO as i32, atlas.tex);

            RaylibCanvas {
                camera: rl::Camera3D {
                    position: rl::Vector3 { x: 10.0, y: 10.0, z: 10.0 },
                    target: rl::Vector3 { x: 0.0, y: 0.0, z: 0.0 },
                    up: rl::Vector3 { x: 0.0, y: 1.0, z: 0.0 },
                    fovy: 45.0,
                    projection: rl::CameraProjection::CAMERA_PERSPECTIVE as i32,
                },
                fov_v: 45.0,
                fov_h: 0.0,
                fov_auto: false,
                tilt: 0.0,
                background: Color::SKY_BLUE,
                fog: Color::SKY_BLUE,
                fog_start: 1e6,
                fog_end: 2e6,
                shader,
                material,
                locs,
                atlas,
                meshes: HashMap::new(),
                draws: Vec::new(),
                cubes: Vec::new(),
                lines: Vec::new(),
                overlay: Vec::new(),
                input: Input::default(),
                mouse: Default::default(),
                screenshot: None,
            }
        }
    }

    /// 横向、纵向半视角的正切，也就是视线方向距离为1处，画面右边缘和上边缘离中心的距离
    fn half_fov_tan(&self) -> (f64, f64) {
        let (w, h) = self.window_size();
        let aspect = if w > 0 && h > 0 { w as f64 / h as f64 } else { 1.0 };
        let tan = |deg: f64| (deg * std::f64::consts::PI / 360.0).tan();
        if self.fov_h == 0.0 {
            let v = tan(self.fov_v);
            (v * aspect, v)
        } else if self.fov_auto {
            let h = tan(self.fov_h);
            (h, h / aspect)
        } else {
            (tan(self.fov_h), tan(self.fov_v))
        }
    }

    fn projection(&self) -> rl::Matrix {
        let (th, tv) = self.half_fov_tan();
        let (r, t) = (NEAR * th, NEAR * tv);
        unsafe { rl::MatrixFrustum(-r, r, -t, t, NEAR, FAR) }
    }

    /// 相机顺时针转等于画面逆时针转，所以在视图空间里绕+Z（指向屏幕外）转+tilt
    fn view(&self) -> rl::Matrix {
        unsafe {
            let v = rl::MatrixLookAt(self.camera.position, self.camera.target, self.camera.up);
            if self.tilt != 0.0 { rl::MatrixMultiply(v, rl::MatrixRotateZ(self.tilt.to_radians() as f32)) } else { v }
        }
    }

    fn draw_cube(&self, c: &CubeDraw) {
        unsafe {
            rl::rlPushMatrix();
            rl::rlTranslatef(c.center.x, c.center.y, c.center.z);
            if c.angle != 0.0 {
                rl::rlRotatef(c.angle, c.axis.x, c.axis.y, c.axis.z);
            }
            rl::rlScalef(c.side, c.side, c.side);
            rl::rlSetTexture(self.atlas.tex.id);
            rl::rlBegin(rl::RL_QUADS as i32);
            for (i, f) in CUBE_FACES.iter().enumerate() {
                rl::rlColor4ub(f.shade, f.shade, f.shade, 255);
                rl::rlNormal3f(f.normal[0], f.normal[1], f.normal[2]);
                for (k, p) in f.corners.iter().enumerate() {
                    let (u, v) = self.atlas.uv(c.tiles[i], FACE_UV[k][0], FACE_UV[k][1]);
                    rl::rlTexCoord2f(u, v);
                    rl::rlVertex3f(p[0], p[1], p[2]);
                }
            }
            rl::rlEnd();
            rl::rlSetTexture(0);
            rl::rlPopMatrix();
        }
    }
}

impl Drop for RaylibCanvas {
    fn drop(&mut self) {
        let ids: Vec<MeshId> = self.meshes.keys().copied().collect();
        for id in ids {
            self.delete_mesh(id);
        }
        unsafe {
            rl::UnloadTexture(self.atlas.tex);
            rl::UnloadShader(self.shader);
            rl::CloseWindow();
        }
    }
}

/// 相机的上方向：默认是+Y；视线和Y轴平行（正上/正下看）时改用-Z，避免相机退化
fn camera_up(dir: Vec3) -> Vec3 {
    let up = Vec3::new(0.0, 1.0, 0.0);
    if dir.cross(up).length() < 1e-6 { Vec3::new(0.0, 0.0, -1.0) } else { up }
}

/// 每个四边形拆成两个三角形
const QUAD_ORDER: [usize; 6] = [0, 1, 2, 0, 2, 3];

impl Canvas for RaylibCanvas {
    fn should_close(&self) -> bool {
        unsafe { rl::WindowShouldClose() }
    }

    fn window_size(&self) -> (u32, u32) {
        unsafe { (rl::GetScreenWidth().max(0) as u32, rl::GetScreenHeight().max(0) as u32) }
    }

    fn frame_time(&self) -> f64 {
        unsafe { rl::GetFrameTime() as f64 }
    }

    fn clean(&mut self) {
        self.draws.clear();
        self.cubes.clear();
        self.lines.clear();
        self.overlay.clear();
    }

    fn draw(&mut self) {
        unsafe {
            rl::BeginDrawing();
            rl::ClearBackground(col(self.background));

            // BeginMode3D只支持一个fovy，这里用自己算的矩阵覆盖它设置的投影和视图矩阵，EndMode3D照常恢复2D的矩阵
            rl::BeginMode3D(self.camera);
            rl::rlSetMatrixProjection(self.projection());
            rl::rlSetMatrixModelview(self.view());

            let p = self.camera.position;
            let set = |loc: i32, v: &[f32], ty: rl::ShaderUniformDataType| {
                rl::SetShaderValue(self.shader, loc, v.as_ptr().cast(), ty as i32)
            };
            use rl::ShaderUniformDataType::*;
            set(self.locs.view_pos, &[p.x, p.y, p.z], SHADER_UNIFORM_VEC3);
            let f = self.fog;
            set(
                self.locs.fog_color,
                &[f.r as f32 / 255.0, f.g as f32 / 255.0, f.b as f32 / 255.0, 1.0],
                SHADER_UNIFORM_VEC4,
            );
            set(self.locs.fog_start, &[self.fog_start as f32], SHADER_UNIFORM_FLOAT);
            set(self.locs.fog_end, &[self.fog_end as f32], SHADER_UNIFORM_FLOAT);
            for (id, o) in &self.draws {
                if let Some(m) = self.meshes.get(id) {
                    rl::DrawMesh(*m, self.material, rl::MatrixTranslate(o.x as f32, o.y as f32, o.z as f32));
                }
            }
            for c in &self.cubes {
                self.draw_cube(c);
            }
            for (a, b, c) in &self.lines {
                rl::DrawLine3D(*a, *b, *c);
            }
            rl::EndMode3D();

            let font = rl::GetFontDefault();
            for o in &self.overlay {
                match o {
                    Overlay::Text(t, p, c) => {
                        rl::DrawTextEx(font, t.as_ptr(), v2(*p), TEXT_SIZE, TEXT_SIZE / 10.0, col(*c))
                    }
                    Overlay::Line(a, b, c) => rl::DrawLineV(v2(*a), v2(*b), col(*c)),
                    Overlay::Rect(p, s, c) => rl::DrawRectangleV(v2(*p), v2(*s), col(*c)),
                    Overlay::Image(t, p, s) => {
                        let (x, y) = self.atlas.tile_origin(*t);
                        let tile = atlas::TILE as f32;
                        let src = rl::Rectangle { x: x as f32, y: y as f32, width: tile, height: tile };
                        let dst = rl::Rectangle { x: p.x as f32, y: p.y as f32, width: s.x as f32, height: s.y as f32 };
                        rl::DrawTexturePro(
                            self.atlas.tex,
                            src,
                            dst,
                            rl::Vector2 { x: 0.0, y: 0.0 },
                            0.0,
                            col(Color::WHITE),
                        );
                    }
                }
            }
            // 要在EndDrawing之前读，之后缓冲区就交换了
            if let Some(path) = self.screenshot.take() {
                // 2D的东西还攒在rlgl的批里，先画出来
                rl::rlDrawRenderBatchActive();
                let img = rl::LoadImageFromScreen();
                if !rl::ExportImage(img, path.as_ptr()) {
                    eprintln!("foxel: 截图保存失败：{}", path.to_string_lossy());
                }
                rl::UnloadImage(img);
            }
            rl::EndDrawing();
        }
        self.input = input::read_input(&mut self.mouse);
    }

    fn camera(&mut self, location: Vec3, direction: Vec3) {
        if direction.length() == 0.0 {
            return;
        }
        let dir = direction.normalize();
        self.camera.position = v3(location);
        self.camera.target = v3(location.add(dir));
        self.camera.up = v3(camera_up(dir));
    }

    /// 视角取绝对值后必须在(0, 180)内（auto时不检查vertical），否则整个调用不生效
    fn view_angle(&mut self, horizontal: f64, vertical: f64, tilt: f64, auto: bool) {
        let (h, v) = (horizontal.abs(), vertical.abs());
        let valid = |d: f64| d > 0.0 && d < 180.0;
        if !valid(h) || (!auto && !valid(v)) || !tilt.is_finite() {
            return;
        }
        self.fov_h = h;
        if !auto {
            self.fov_v = v;
        }
        self.fov_auto = auto;
        self.tilt = tilt;
    }

    fn frustum_corner_rays(&self, direction: Vec3) -> [Vec3; 4] {
        let mut f = direction;
        if f.length() == 0.0 {
            let (p, t) = (self.camera.position, self.camera.target);
            f = Vec3::new((t.x - p.x) as f64, (t.y - p.y) as f64, (t.z - p.z) as f64);
        }
        let f = f.normalize();
        let right = f.cross(camera_up(f)).normalize();
        let up = right.cross(f);
        let (th, tv) = self.half_fov_tan();
        let (sin, cos) = self.tilt.to_radians().sin_cos();
        // 画面上(x, y)处的射线：view()里画面相对相机转了+tilt，这里转回-tilt得到相机自身坐标系里的方向
        let ray = |x: f64, y: f64| {
            let rx = x * cos + y * sin;
            let ry = -x * sin + y * cos;
            right.scale(rx).add(up.scale(ry)).add(f).normalize()
        };
        [ray(-th, tv), ray(th, tv), ray(-th, -tv), ray(th, -tv)]
    }

    fn set_background(&mut self, c: Color) {
        self.background = c;
    }

    fn set_fog(&mut self, c: Color, start: f64, end: f64) {
        self.fog = c;
        self.fog_start = start;
        self.fog_end = end;
    }

    fn set_mesh(&mut self, id: MeshId, quads: &[Quad]) {
        self.delete_mesh(id);
        if quads.is_empty() {
            return;
        }
        let n = quads.len() * 6;
        let mut verts: Vec<f32> = Vec::with_capacity(n * 3);
        let mut uvs: Vec<f32> = Vec::with_capacity(n * 2);
        let mut cols: Vec<u8> = Vec::with_capacity(n * 4);
        for q in quads {
            let t = self.atlas.tile_of(q.img.as_ref());
            for k in QUAD_ORDER {
                let c = q.corners[k];
                let (u, v) = self.atlas.uv(t, q.uv[k].x, q.uv[k].y);
                verts.extend_from_slice(&[c.x as f32, c.y as f32, c.z as f32]);
                uvs.extend_from_slice(&[u, v]);
                cols.extend_from_slice(&[q.tint.r, q.tint.g, q.tint.b, q.tint.a]);
            }
        }
        unsafe {
            let mut mesh: rl::Mesh = std::mem::zeroed();
            mesh.vertexCount = n as i32;
            mesh.triangleCount = (n / 3) as i32;
            mesh.vertices = verts.as_mut_ptr();
            mesh.texcoords = uvs.as_mut_ptr();
            mesh.colors = cols.as_mut_ptr();
            rl::UploadMesh(&mut mesh, false);
            // 数据已经在显卡里了。去掉指向Rust内存的指针，UnloadMesh就不会去释放它们
            mesh.vertices = std::ptr::null_mut();
            mesh.texcoords = std::ptr::null_mut();
            mesh.colors = std::ptr::null_mut();
            self.meshes.insert(id, mesh);
        }
    }

    fn delete_mesh(&mut self, id: MeshId) {
        if let Some(m) = self.meshes.remove(&id) {
            unsafe { rl::UnloadMesh(m) };
        }
    }

    fn add_mesh(&mut self, id: MeshId, origin: Vec3) {
        self.draws.push((id, origin));
    }

    fn add_cube(&mut self, side: f64, center: Vec3, images: &Images, rotation: Vec3) {
        let l = rotation.length();
        let (axis, angle) =
            if l > 0.0 { (v3(rotation.scale(1.0 / l)), l.to_degrees() as f32) } else { (v3(Vec3::default()), 0.0) };
        let faces = [&images.east, &images.west, &images.up, &images.down, &images.south, &images.north];
        let tiles = faces.map(|f| self.atlas.tile_of(f.as_ref()));
        self.cubes.push(CubeDraw { side: side as f32, center: v3(center), axis, angle, tiles });
    }

    fn add_line3d(&mut self, start: Vec3, end: Vec3, c: Color) {
        self.lines.push((v3(start), v3(end), col(c)));
    }

    fn add_text(&mut self, text: &str, pos: Vec2, c: Color) {
        self.overlay.push(Overlay::Text(cstr(text), pos, c));
    }

    fn measure_text(&self, text: &str) -> Vec2 {
        let t = cstr(text);
        let v = unsafe { rl::MeasureTextEx(rl::GetFontDefault(), t.as_ptr(), TEXT_SIZE, TEXT_SIZE / 10.0) };
        Vec2::new(v.x as f64, v.y as f64)
    }

    fn add_line2d(&mut self, start: Vec2, end: Vec2, c: Color) {
        self.overlay.push(Overlay::Line(start, end, c));
    }

    fn add_rect(&mut self, pos: Vec2, size: Vec2, c: Color) {
        self.overlay.push(Overlay::Rect(pos, size, c));
    }

    fn add_image(&mut self, img: &Image, pos: Vec2, size: Vec2) {
        let t = self.atlas.tile_of(Some(img));
        self.overlay.push(Overlay::Image(t, pos, size));
    }

    fn input(&self) -> &Input {
        &self.input
    }

    fn set_cursor_locked(&mut self, locked: bool) {
        unsafe {
            if locked && !rl::IsCursorHidden() {
                rl::DisableCursor();
            } else if !locked && rl::IsCursorHidden() {
                rl::EnableCursor();
            }
        }
    }

    fn is_cursor_locked(&self) -> bool {
        unsafe { rl::IsCursorHidden() }
    }

    fn screenshot(&mut self, path: &std::path::Path) {
        if let Some(dir) = path.parent() {
            let _ = std::fs::create_dir_all(dir);
        }
        self.screenshot = Some(cstr(&path.to_string_lossy()));
    }
}

struct CubeFace {
    normal: [f32; 3],
    corners: [[f32; 3]; 4],
    shade: u8,
}

/// 单位cube（中心在原点）每个面的4个角，从外面看逆时针：左上、左下、右下、右上，
/// 对应图片的(0,0)、(0,1)、(1,1)、(1,0)。顺序是east, west, up, down, south, north
const CUBE_FACES: [CubeFace; 6] = [
    CubeFace {
        normal: [1., 0., 0.],
        corners: [[0.5, 0.5, 0.5], [0.5, -0.5, 0.5], [0.5, -0.5, -0.5], [0.5, 0.5, -0.5]],
        shade: 170,
    },
    CubeFace {
        normal: [-1., 0., 0.],
        corners: [[-0.5, 0.5, -0.5], [-0.5, -0.5, -0.5], [-0.5, -0.5, 0.5], [-0.5, 0.5, 0.5]],
        shade: 170,
    },
    CubeFace {
        normal: [0., 1., 0.],
        corners: [[-0.5, 0.5, -0.5], [-0.5, 0.5, 0.5], [0.5, 0.5, 0.5], [0.5, 0.5, -0.5]],
        shade: 255,
    },
    CubeFace {
        normal: [0., -1., 0.],
        corners: [[-0.5, -0.5, 0.5], [-0.5, -0.5, -0.5], [0.5, -0.5, -0.5], [0.5, -0.5, 0.5]],
        shade: 130,
    },
    CubeFace {
        normal: [0., 0., 1.],
        corners: [[-0.5, 0.5, 0.5], [-0.5, -0.5, 0.5], [0.5, -0.5, 0.5], [0.5, 0.5, 0.5]],
        shade: 210,
    },
    CubeFace {
        normal: [0., 0., -1.],
        corners: [[0.5, 0.5, -0.5], [0.5, -0.5, -0.5], [-0.5, -0.5, -0.5], [-0.5, 0.5, -0.5]],
        shade: 210,
    },
];

const FACE_UV: [[f64; 2]; 4] = [[0.0, 0.0], [0.0, 1.0], [1.0, 1.0], [1.0, 0.0]];
