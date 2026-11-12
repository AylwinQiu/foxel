use std::collections::{HashMap, VecDeque};

use super::{Canvas, Color, Image, Images, Input, MeshId, Quad, Vec2, Vec3};

/// 什么都不画的canvas，测试和没有窗口的环境用。
/// 输入从inputs里一帧一帧取，取完以后should_close变成true
#[derive(Default)]
pub struct NullCanvas {
    pub inputs: VecDeque<Input>,
    /// 现在的网格和每个网格有几个面
    pub meshes: HashMap<MeshId, usize>,
    /// 这一帧add_mesh了哪些网格
    pub drawn: Vec<MeshId>,
    /// 一共画了几帧
    pub frames: u64,
    pub locked: bool,
    input: Input,
    closing: bool,
}

impl NullCanvas {
    pub fn new() -> NullCanvas {
        NullCanvas::default()
    }

    /// 之后每一帧依次用这些输入，用完就关闭窗口
    pub fn with_inputs(inputs: impl IntoIterator<Item = Input>) -> NullCanvas {
        let mut c = NullCanvas { inputs: inputs.into_iter().collect(), ..Default::default() };
        c.next_input();
        c
    }

    fn next_input(&mut self) {
        match self.inputs.pop_front() {
            Some(i) => self.input = i,
            None => {
                self.input = Input::default();
                self.closing = true;
            }
        }
    }
}

impl Canvas for NullCanvas {
    fn should_close(&self) -> bool {
        self.closing
    }
    fn window_size(&self) -> (u32, u32) {
        (800, 600)
    }
    fn frame_time(&self) -> f64 {
        1.0 / 60.0
    }
    fn clean(&mut self) {
        self.drawn.clear();
    }
    fn draw(&mut self) {
        self.frames += 1;
        self.next_input();
    }
    fn camera(&mut self, _: Vec3, _: Vec3) {}
    fn view_angle(&mut self, _: f64, _: f64, _: f64, _: bool) {}
    /// 固定是90度视角
    fn frustum_corner_rays(&self, d: Vec3) -> [Vec3; 4] {
        let f = d.normalize();
        let mut up = Vec3::new(0.0, 1.0, 0.0);
        if f.cross(up).length() < 1e-6 {
            up = Vec3::new(0.0, 0.0, -1.0);
        }
        let right = f.cross(up).normalize();
        let up = right.cross(f);
        let ray = |x: f64, y: f64| right.scale(x).add(up.scale(y)).add(f).normalize();
        [ray(-1.0, 1.0), ray(1.0, 1.0), ray(-1.0, -1.0), ray(1.0, -1.0)]
    }
    fn set_background(&mut self, _: Color) {}
    fn set_fog(&mut self, _: Color, _: f64, _: f64) {}
    fn set_mesh(&mut self, id: MeshId, quads: &[Quad]) {
        if quads.is_empty() {
            self.meshes.remove(&id);
        } else {
            self.meshes.insert(id, quads.len());
        }
    }
    fn delete_mesh(&mut self, id: MeshId) {
        self.meshes.remove(&id);
    }
    fn add_mesh(&mut self, id: MeshId, _: Vec3) {
        self.drawn.push(id);
    }
    fn add_cube(&mut self, _: f64, _: Vec3, _: &Images, _: Vec3) {}
    fn add_line3d(&mut self, _: Vec3, _: Vec3, _: Color) {}
    fn add_text(&mut self, _: &str, _: Vec2, _: Color) {}
    fn measure_text(&self, text: &str) -> Vec2 {
        let lines = text.split('\n');
        let w = lines.clone().map(|l| l.len()).max().unwrap_or(0);
        Vec2::new(w as f64 * 10.0, lines.count() as f64 * 20.0)
    }
    fn add_line2d(&mut self, _: Vec2, _: Vec2, _: Color) {}
    fn add_rect(&mut self, _: Vec2, _: Vec2, _: Color) {}
    fn add_image(&mut self, _: &Image, _: Vec2, _: Vec2) {}
    fn input(&self) -> &Input {
        &self.input
    }
    fn set_cursor_locked(&mut self, locked: bool) {
        self.locked = locked;
    }
    fn is_cursor_locked(&self) -> bool {
        self.locked
    }
}
