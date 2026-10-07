/// This file define the interface of communicartion between game and frontend implementation
/// like raylib and html.
///

#[derive(Clone, Copy)]
pub struct Canvas1d(pub i32);

#[derive(Clone, Copy)]
pub struct Canvas2d(pub i32, pub i32);

#[derive(Clone, Copy)]
pub struct CanvasColor(pub u8, pub u8, pub u8, pub u8); // rgba

#[derive(Clone)]
pub enum DrawIns {
    Square {
        // add a square.
        side: Canvas1d,
        center: Canvas2d,
        color: CanvasColor,
        rotation: f32,
    },
    Line {
        // add a line.
        start: Canvas2d,
        end: Canvas2d,
        color: CanvasColor,
    },
    Text {
        // add a text
        text: String,
        center: Canvas2d,
        color: CanvasColor,
        rotation: f32,
        font: String,
    },
}



pub trait Canvas {
    fn draw(self: Self, ins: Vec<DrawIns>) -> Self;
    fn should_close(self: &Self) -> bool;
    // 返回窗口大小。
    fn get_size(self:&Self) -> (usize, usize);
}
