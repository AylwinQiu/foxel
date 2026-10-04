/// This file define the interface of communicartion between game and frontend implementation
/// like raylib and html.
///

pub struct Canvas1d(pub i32);

pub struct Canvas2d(pub i32, pub i32);

pub struct CanvasColor(pub u8, pub u8, pub u8, pub u8); // rgba

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

pub trait CanvasOld {
    fn add_square(
        self: Self,
        side: Canvas1d,
        center: Canvas2d,
        color: CanvasColor,
        rotation: i32,
    ) -> Self;
    fn add_line(self: Self, start: Canvas2d, end: Canvas2d, color: CanvasColor) -> Self;
    fn add_text(
        self: Self,
        text: &str,
        center: Canvas2d,
        color: CanvasColor,
        rotation: i32,
        font: &str,
    ) -> Self;
    fn draw(self: Self) -> Self;
    fn clean(self: Self) -> Self;
}

pub trait Canvas {
    fn draw(self: Self, ins: Vec<DrawIns>) -> Self;
    fn should_close(self: &Self) -> bool;
}
