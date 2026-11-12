//! 窗口的接口。换前端（比如换成wgpu、网页）只要实现Canvas这个trait，其他代码都不用动。
//!
//! 每一帧：clean() -> add_xxx() -> draw()。add_xxx添加的东西只画这一帧。
//! 网格（set_mesh）是例外：上传一次之后一直留在显卡里，每帧只用add_mesh说明画在哪里。
//! 大量静止的方块（比如地形）应该做成网格，add_cube只适合少量会动的东西。

mod image;
mod input;
mod null;
pub mod raylib;

pub use image::{Color, Image, Images};
pub use input::{Input, Key};
pub use null::NullCanvas;

/// canvas里的3D坐标。engine用浮点原点：相机永远在(0, 0, 0)附近，所以这里的数都不大
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct Vec3 {
    pub x: f64,
    pub y: f64,
    pub z: f64,
}

impl Vec3 {
    pub const fn new(x: f64, y: f64, z: f64) -> Vec3 {
        Vec3 { x, y, z }
    }

    pub fn dot(self, b: Vec3) -> f64 {
        self.x * b.x + self.y * b.y + self.z * b.z
    }

    pub fn cross(self, b: Vec3) -> Vec3 {
        Vec3::new(self.y * b.z - self.z * b.y, self.z * b.x - self.x * b.z, self.x * b.y - self.y * b.x)
    }

    pub fn length(self) -> f64 {
        self.dot(self).sqrt()
    }

    pub fn scale(self, k: f64) -> Vec3 {
        Vec3::new(self.x * k, self.y * k, self.z * k)
    }

    #[allow(clippy::should_implement_trait)]
    pub fn add(self, b: Vec3) -> Vec3 {
        Vec3::new(self.x + b.x, self.y + b.y, self.z + b.z)
    }

    pub fn normalize(self) -> Vec3 {
        self.scale(1.0 / self.length())
    }
}

/// 屏幕上的2D坐标（像素），(0, 0)是左上角
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct Vec2 {
    pub x: f64,
    pub y: f64,
}

impl Vec2 {
    pub const fn new(x: f64, y: f64) -> Vec2 {
        Vec2 { x, y }
    }
}

/// 网格里的一个四边形面。
/// corners从正面看是逆时针排列的，背面不画；uv是对应角在图片上的相对位置（0到1，左上角是(0, 0)）。
/// 图片里alpha小于一半的像素不画，可以用来做树叶、玻璃这种镂空的面。
#[derive(Clone)]
pub struct Quad {
    pub corners: [Vec3; 4],
    pub uv: [Vec2; 4],
    /// None是白色
    pub img: Option<Image>,
    /// 乘在图片颜色上，用来做明暗。白色是原样
    pub tint: Color,
}

/// 网格的编号，由使用者自己分配
pub type MeshId = u64;

pub trait Canvas {
    /// 窗口是否需要关闭
    fn should_close(&self) -> bool;
    /// 窗口大小（像素）
    fn window_size(&self) -> (u32, u32);
    /// 上一帧用了多少秒
    fn frame_time(&self) -> f64;

    /// 清除这一帧添加的所有东西（网格不会被删除）
    fn clean(&mut self);
    /// 绘制一帧，然后读入下一帧的输入
    fn draw(&mut self);

    /// 设置3D相机的位置和朝向，direction只取方向。长度为0时什么都不做
    fn camera(&mut self, location: Vec3, direction: Vec3);
    /// 设置相机的横向视角、纵向视角和倾斜角度，都用角度制。
    /// auto为true时忽略vertical，纵向视角由horizontal按窗口宽高比算出，画面不会被拉伸。
    /// 倾斜是相机顺时针为正。
    fn view_angle(&mut self, horizontal: f64, vertical: f64, tilt: f64, auto: bool);
    /// 相机朝direction看时视锥四条棱的方向（单位向量）：左上、右上、左下、右下
    fn frustum_corner_rays(&self, direction: Vec3) -> [Vec3; 4];
    /// 背景（天空）的颜色
    fn set_background(&mut self, c: Color);
    /// 雾：离相机start以内没有雾，到end完全变成c。只有网格有雾
    fn set_fog(&mut self, c: Color, start: f64, end: f64);

    /// 网格：创建或替换id这个网格。quads是空的时候等于delete_mesh
    fn set_mesh(&mut self, id: MeshId, quads: &[Quad]);
    /// 删除网格，不存在时什么都不做
    fn delete_mesh(&mut self, id: MeshId);
    /// 这一帧把网格画在origin处（网格里的坐标都加上origin）
    fn add_mesh(&mut self, id: MeshId, origin: Vec3);

    /// 一个cube，side是边长，center是中心。rotation的方向是转轴，长度是转角（弧度）
    fn add_cube(&mut self, side: f64, center: Vec3, images: &Images, rotation: Vec3);
    fn add_line3d(&mut self, start: Vec3, end: Vec3, c: Color);

    // 2D的东西按添加的顺序画，后添加的盖在上面，都画在3D的上面

    /// 文字，pos是左上角，字号固定20像素，只支持ASCII，可以用\n换行
    fn add_text(&mut self, text: &str, pos: Vec2, c: Color);
    /// 文字画出来有多大
    fn measure_text(&self, text: &str) -> Vec2;
    fn add_line2d(&mut self, start: Vec2, end: Vec2, c: Color);
    /// 实心矩形，pos是左上角
    fn add_rect(&mut self, pos: Vec2, size: Vec2, c: Color);
    /// 图片，pos是左上角
    fn add_image(&mut self, img: &Image, pos: Vec2, size: Vec2);

    /// 这一帧的输入。窗口没有焦点时什么都没有
    fn input(&self) -> &Input;
    /// 锁定鼠标：光标隐藏，鼠标可以一直往一个方向移动（第一人称视角用）
    fn set_cursor_locked(&mut self, locked: bool);
    fn is_cursor_locked(&self) -> bool;

    /// 下一次draw时把画面存成png。不支持的前端可以什么都不做
    fn screenshot(&mut self, _path: &std::path::Path) {}
}
