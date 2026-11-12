use std::collections::HashMap;

use ::raylib::ffi as rl;

use crate::canvas::{Color, Image};

// 贴图集：所有图片都采样成TILE见方的小块，放进一张大贴图里。
// 每块四周多放1像素（复制边上的像素），这样贴图边缘不会混进隔壁图片的颜色。
const SIZE: i32 = 2048;
pub const TILE: i32 = 32;
const STRIDE: i32 = TILE + 2;
const PER_ROW: i32 = SIZE / STRIDE;

const TILE_WHITE: usize = 0;
const TILE_MAGENTA: usize = 1;

pub struct Atlas {
    pub tex: rl::Texture2D,
    /// 图片id -> 第几块
    tiles: HashMap<u64, usize>,
    next: usize,
    full: bool,
}

impl Atlas {
    /// 必须在窗口创建之后调用
    pub fn new() -> Atlas {
        let tex = unsafe {
            let img = rl::GenImageColor(SIZE, SIZE, rl::Color { r: 0, g: 0, b: 0, a: 0 });
            let tex = rl::LoadTextureFromImage(img);
            rl::UnloadImage(img);
            // 不插值，像素风的小图片放大后不会糊
            rl::SetTextureFilter(tex, rl::TextureFilter::TEXTURE_FILTER_POINT as i32);
            tex
        };
        let mut a = Atlas { tex, tiles: HashMap::new(), next: 0, full: false };
        a.add(&Image::solid(Color::WHITE));
        a.add(&Image::solid(Color::MAGENTA));
        a
    }

    /// 图片在贴图集里是第几块，第一次见到时采样放进去。None是白色
    pub fn tile_of(&mut self, img: Option<&Image>) -> usize {
        match img {
            None => TILE_WHITE,
            Some(img) => match self.tiles.get(&img.id()) {
                Some(&t) => t,
                None => self.add(img),
            },
        }
    }

    fn add(&mut self, img: &Image) -> usize {
        if self.next >= (PER_ROW * PER_ROW) as usize {
            if !self.full {
                eprintln!("foxel: 贴图集满了，之后的图片都显示成品红色");
                self.full = true;
            }
            self.tiles.insert(img.id(), TILE_MAGENTA);
            return TILE_MAGENTA;
        }
        let t = self.next;
        self.next += 1;
        let mut data = Vec::with_capacity((STRIDE * STRIDE * 4) as usize);
        for py in 0..STRIDE {
            for px in 0..STRIDE {
                // 外面一圈复制最边上的像素
                let sx = (px - 1).clamp(0, TILE - 1);
                let sy = (py - 1).clamp(0, TILE - 1);
                let c = img.get_pixel((sx as f64 + 0.5) / TILE as f64, (sy as f64 + 0.5) / TILE as f64);
                data.extend_from_slice(&[c.r, c.g, c.b, c.a]);
            }
        }
        let (x, y) = self.tile_origin(t);
        let rec = rl::Rectangle { x: (x - 1) as f32, y: (y - 1) as f32, width: STRIDE as f32, height: STRIDE as f32 };
        unsafe { rl::UpdateTextureRec(self.tex, rec, data.as_ptr().cast()) };
        self.tiles.insert(img.id(), t);
        t
    }

    /// 第t块图片（不含外面一圈）左上角的像素坐标
    pub fn tile_origin(&self, t: usize) -> (i32, i32) {
        let t = t as i32;
        ((t % PER_ROW) * STRIDE + 1, (t / PER_ROW) * STRIDE + 1)
    }

    /// 第t块图片上的相对位置(u, v)在贴图集里的坐标
    pub fn uv(&self, t: usize, u: f64, v: f64) -> (f32, f32) {
        let (x, y) = self.tile_origin(t);
        (((x as f64 + u * TILE as f64) / SIZE as f64) as f32, ((y as f64 + v * TILE as f64) / SIZE as f64) as f32)
    }
}
