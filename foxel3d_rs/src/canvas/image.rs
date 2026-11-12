use std::cell::RefCell;
use std::collections::HashMap;
use std::path::Path;
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
pub struct Color {
    pub r: u8,
    pub g: u8,
    pub b: u8,
    pub a: u8,
}

impl Color {
    pub const WHITE: Color = Color::rgb(255, 255, 255);
    pub const BLACK: Color = Color::rgb(0, 0, 0);
    pub const MAGENTA: Color = Color::rgb(255, 0, 255);
    pub const SKY_BLUE: Color = Color::rgb(140, 190, 255);
    pub const TRANSPARENT: Color = Color::rgba(0, 0, 0, 0);

    pub const fn rgb(r: u8, g: u8, b: u8) -> Color {
        Color { r, g, b, a: 255 }
    }

    pub const fn rgba(r: u8, g: u8, b: u8, a: u8) -> Color {
        Color { r, g, b, a }
    }
}

struct ImageData {
    /// 每张图片唯一的编号，canvas按它缓存贴图
    id: u64,
    width: u32,
    height: u32,
    /// 按行存，pixels[y * width + x]是(x, y)的颜色
    pixels: Vec<Color>,
}

/// 一张图片，克隆只是多一个引用。创建之后内容不会再变，canvas按id把它缓存成贴图
#[derive(Clone)]
pub struct Image(Arc<ImageData>);

static NEXT_ID: AtomicU64 = AtomicU64::new(1);

thread_local! {
    static SOLID: RefCell<HashMap<Color, Image>> = RefCell::new(HashMap::new());
    static FILES: RefCell<HashMap<String, Image>> = RefCell::new(HashMap::new());
}

impl Image {
    /// pixels按行存，长度必须是width * height
    pub fn new(width: u32, height: u32, pixels: Vec<Color>) -> Image {
        assert!(width > 0 && height > 0 && pixels.len() == (width * height) as usize, "图片大小不对");
        Image(Arc::new(ImageData { id: NEXT_ID.fetch_add(1, Ordering::Relaxed), width, height, pixels }))
    }

    /// 用f(x, y)生成每个像素
    pub fn from_fn(width: u32, height: u32, mut f: impl FnMut(u32, u32) -> Color) -> Image {
        let mut pixels = Vec::with_capacity((width * height) as usize);
        for y in 0..height {
            for x in 0..width {
                pixels.push(f(x, y));
            }
        }
        Image::new(width, height, pixels)
    }

    /// 纯色图片。同一个颜色总是返回同一张图片
    pub fn solid(c: Color) -> Image {
        SOLID.with(|m| m.borrow_mut().entry(c).or_insert_with(|| Image::new(1, 1, vec![c])).clone())
    }

    /// 读一张png。同一个路径只读一次；读不了时返回品红色并打印原因
    pub fn load(path: &Path) -> Image {
        let key = path.to_string_lossy().to_string();
        if let Some(img) = FILES.with(|m| m.borrow().get(&key).cloned()) {
            return img;
        }
        let img = read_png(path).unwrap_or_else(|e| {
            eprintln!("foxel: 读取图片失败：{key}：{e}");
            Image::solid(Color::MAGENTA)
        });
        FILES.with(|m| m.borrow_mut().insert(key, img.clone()));
        img
    }

    pub fn id(&self) -> u64 {
        self.0.id
    }

    pub fn size(&self) -> (u32, u32) {
        (self.0.width, self.0.height)
    }

    /// 像素(x, y)，超出范围时取最近的边
    pub fn pixel(&self, x: u32, y: u32) -> Color {
        let d = &self.0;
        d.pixels[(y.min(d.height - 1) * d.width + x.min(d.width - 1)) as usize]
    }

    /// 相对位置的颜色，(0, 0)是左上角，(1, 1)是右下角。不做插值
    pub fn get_pixel(&self, x_frac: f64, y_frac: f64) -> Color {
        let (w, h) = self.size();
        self.pixel((x_frac * w as f64).max(0.0) as u32, (y_frac * h as f64).max(0.0) as u32)
    }
}

fn read_png(path: &Path) -> Result<Image, String> {
    let file = std::fs::File::open(path).map_err(|e| e.to_string())?;
    let mut decoder = png::Decoder::new(std::io::BufReader::new(file));
    decoder.set_transformations(png::Transformations::normalize_to_color8() | png::Transformations::ALPHA);
    let mut reader = decoder.read_info().map_err(|e| e.to_string())?;
    let mut buf = vec![0; reader.output_buffer_size().ok_or("图片太大")?];
    let info = reader.next_frame(&mut buf).map_err(|e| e.to_string())?;
    let (w, h) = (info.width, info.height);
    if w == 0 || h == 0 {
        return Err("图片是空的".into());
    }
    let channels = info.color_type.samples();
    let mut pixels = Vec::with_capacity((w * h) as usize);
    for y in 0..h as usize {
        let row = &buf[y * info.line_size..];
        for x in 0..w as usize {
            let p = &row[x * channels..];
            pixels.push(match channels {
                4 => Color::rgba(p[0], p[1], p[2], p[3]),
                3 => Color::rgb(p[0], p[1], p[2]),
                2 => Color::rgba(p[0], p[0], p[0], p[1]),
                _ => Color::rgb(p[0], p[0], p[0]),
            });
        }
    }
    Ok(Image::new(w, h, pixels))
}

/// cube 6个面的图片，None的面是白色。
/// 每张图从cube外面看是正的：侧面图片的上方朝上，up面图片的上方朝北，down面图片的上方朝南。
#[derive(Clone, Default)]
pub struct Images {
    pub up: Option<Image>,
    pub down: Option<Image>,
    pub north: Option<Image>,
    pub south: Option<Image>,
    pub west: Option<Image>,
    pub east: Option<Image>,
}

impl Images {
    /// 6个面都用img
    pub fn all(img: Image) -> Images {
        Images::column(img.clone(), img.clone(), img)
    }

    /// 上面是top，下面是bottom，4个侧面是side，比如草方块、木头
    pub fn column(top: Image, side: Image, bottom: Image) -> Images {
        Images {
            up: Some(top),
            down: Some(bottom),
            north: Some(side.clone()),
            south: Some(side.clone()),
            west: Some(side.clone()),
            east: Some(side),
        }
    }
}
