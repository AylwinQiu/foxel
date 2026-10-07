//! Raylib-backed implementation of [`Canvas`](crate::canvas::canvas::Canvas).
//!
//! The frontend consumes a batch of [`DrawIns`] values and maps it to raylib's
//! immediate-mode drawing API.

use std::collections::HashMap;

use raylib::prelude::{
    Color, Font, RaylibDraw, RaylibFont, RaylibHandle, RaylibThread, Rectangle, Vector2,
};

use crate::canvas::canvas::{self, Canvas1d, Canvas2d, CanvasColor, DrawIns};

const DEFAULT_FONT_SIZE: f32 = 20.0;
const DEFAULT_FONT_SPACING: f32 = 1.0;

/// A raylib window and the resources used to render a batch of instructions.
pub struct RaylibCanvas {
    raylib: RaylibHandle,
    thread: RaylibThread,
    instruction_list: Vec<DrawIns>,
    fonts: HashMap<String, Font>,
}

impl RaylibCanvas {
    /// Creates a raylib window with the requested initial size.
    ///
    /// Raylib requires positive dimensions, so zero or negative values are
    /// clamped to one pixel. The window title is fixed because the `Canvas`
    /// interface does not expose a title setting.
    pub fn new(shape: (i32, i32), resizable: bool) -> Self {
        let mut builder = raylib::init();
        builder.size(shape.0.max(1), shape.1.max(1)).title("Foxel");

        if resizable {
            builder.resizable();
        }

        let (raylib, thread) = builder.build();

        Self {
            raylib,
            thread,
            instruction_list: Vec::new(),
            fonts: HashMap::new(),
        }
    }

    fn queue_missing_fonts(&mut self) {
        for font_path in self.instruction_list.iter().filter_map(|instruction| {
            let DrawIns::Text { font, .. } = instruction else {
                return None;
            };

            (!font.is_empty() && font != "default" && !font.as_bytes().contains(&0))
                .then_some(font.clone())
        }) {
            if self.fonts.contains_key(&font_path) {
                continue;
            }

            // The trait cannot report a load error. Keep rendering with raylib's
            // default font when a supplied path cannot be opened.
            if let Ok(font) = self.raylib.load_font(&self.thread, &font_path) {
                self.fonts.insert(font_path, font);
            }
        }
    }
}

impl canvas::Canvas for RaylibCanvas {
    fn draw(mut self, ins: Vec<DrawIns>) -> Self {
        self.instruction_list = ins;

        for instruction in &mut self.instruction_list {
            if let DrawIns::Text { text, .. } = instruction {
                // raylib's Rust bindings reject interior NUL bytes.
                *text = text.replace('\0', "\u{FFFD}");
            }
        }

        self.queue_missing_fonts();

        let default_font = self.raylib.get_font_default();
        let mut drawing = self.raylib.begin_drawing(&self.thread);
        drawing.clear_background(Color::RAYWHITE);

        for instruction in &self.instruction_list {
            match instruction {
                DrawIns::Square {
                    side: Canvas1d(side),
                    center: Canvas2d(center_x, center_y),
                    color: CanvasColor(red, green, blue, alpha),
                    rotation,
                } => {
                    let side = *side as f32;
                    drawing.draw_rectangle_pro(
                        Rectangle::new(
                            *center_x as f32 - side / 2.0,
                            *center_y as f32 - side / 2.0,
                            side,
                            side,
                        ),
                        Vector2::new(side / 2.0, side / 2.0),
                        *rotation,
                        Color::new(*red, *green, *blue, *alpha),
                    );
                }
                DrawIns::Line {
                    start: Canvas2d(start_x, start_y),
                    end: Canvas2d(end_x, end_y),
                    color: CanvasColor(red, green, blue, alpha),
                } => {
                    drawing.draw_line(
                        *start_x,
                        *start_y,
                        *end_x,
                        *end_y,
                        Color::new(*red, *green, *blue, *alpha),
                    );
                }
                DrawIns::Text {
                    text,
                    center: Canvas2d(center_x, center_y),
                    color: CanvasColor(red, green, blue, alpha),
                    rotation,
                    font,
                } => {
                    let color = Color::new(*red, *green, *blue, *alpha);
                    let center = Vector2::new(*center_x as f32, *center_y as f32);

                    if let Some(font) = self.fonts.get(font) {
                        let size = font.measure_text(text, DEFAULT_FONT_SIZE, DEFAULT_FONT_SPACING);
                        drawing.draw_text_pro(
                            font,
                            text,
                            center,
                            Vector2::new(size.x / 2.0, size.y / 2.0),
                            *rotation,
                            DEFAULT_FONT_SIZE,
                            DEFAULT_FONT_SPACING,
                            color,
                        );
                    } else {
                        let size = default_font.measure_text(
                            text,
                            DEFAULT_FONT_SIZE,
                            DEFAULT_FONT_SPACING,
                        );
                        drawing.draw_text_pro(
                            &default_font,
                            text,
                            center,
                            Vector2::new(size.x / 2.0, size.y / 2.0),
                            *rotation,
                            DEFAULT_FONT_SIZE,
                            DEFAULT_FONT_SPACING,
                            color,
                        );
                    }
                }
            }
        }

        drop(drawing);
        self
    }

    fn should_close(&self) -> bool {
        self.raylib.window_should_close()
    }

    fn get_size(&self) -> (usize, usize) {
        // raylib reports the logical screen size (the drawing coordinate space),
        // not the HiDPI framebuffer size. It is never negative in practice.
        (
            self.raylib.get_screen_width().max(0) as usize,
            self.raylib.get_screen_height().max(0) as usize,
        )
    }
}
