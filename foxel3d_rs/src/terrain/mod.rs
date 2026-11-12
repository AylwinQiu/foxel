//! 地形生成：噪声、生物群系、feature（矿石、树、花……）。
//!
//! 先按生物群系填好地面（表层、下面几层、石头，最底下一层基岩），再按顺序执行每个feature。
//! 地形是确定性的：同一个种子、同一个位置永远生成一样的chunk，所以没改过的chunk不用存档。
//!
//! feature的回调是普通的Rust闭包，Lua的回调由lua模块包装成闭包传进来，terrain不知道Lua的存在。

mod noise;
#[cfg(test)]
mod tests;

use std::cell::RefCell;
use std::collections::HashMap;
use std::rc::Rc;

pub use noise::Perlin;

use crate::rng::{Rng, hash, unit};
use crate::world::{AIR, BlockId, CHUNK_DEPTH, CHUNK_SIZE, ChunkData, CubeId, MIN_DEPTH};

/// 生物群系：决定地面的高度、起伏和用什么方块
#[derive(Clone, Debug)]
pub struct Biome {
    pub name: String,
    /// 气候：温度、湿度都在-1到1之间。每一列选气候最接近的生物群系
    pub temperature: f64,
    pub humidity: f64,
    /// 地面的平均高度和起伏
    pub base_height: f64,
    pub variation: f64,
    /// true时用山脊形状的噪声（只往上起伏），适合山脉
    pub ridged: bool,
    /// 最上面一层和下面几层的方块，再往下是石头
    pub surface: BlockId,
    pub subsurface: BlockId,
    pub subsurface_depth: i64,
    /// 地面高于这个高度时地面是石头，None表示没有
    pub rock_line: Option<i64>,
    /// 地面高于这个高度时盖上雪，None表示没有
    pub snow_line: Option<i64>,
}

/// 整个世界共用的设置
#[derive(Clone, Debug)]
pub struct Settings {
    pub stone: BlockId,
    pub bedrock: BlockId,
    pub snow: BlockId,
    /// 世界在竖直方向上的chunk范围（两端都包括），最下面一层方块是基岩
    pub min_chunk_y: i64,
    pub max_chunk_y: i64,
}

impl Settings {
    /// 基岩所在的y
    pub fn bottom_y(&self) -> i64 {
        self.min_chunk_y * CHUNK_SIZE
    }
}

/// 一列的地形
#[derive(Clone, Copy, Debug)]
pub struct Column {
    /// 地面最上面那个方块的y
    pub height: i64,
    /// 在TerrainCore::biomes里的下标
    pub biome: usize,
}

/// 噪声、生物群系和设置。feature回调里的Context、Lua的terrain.column都通过它查地形
pub struct TerrainCore {
    pub seed: i64,
    pub biomes: Vec<Biome>,
    pub settings: Settings,
    height: Perlin,
    ridge: Perlin,
    detail: Perlin,
    temperature: Perlin,
    humidity: Perlin,
    columns: RefCell<HashMap<(i64, i64), Column>>,
}

/// 气候距离的尺度：越小生物群系的交界越窄
const CLIMATE_SIGMA2: f64 = 0.04;

impl TerrainCore {
    pub fn new(seed: i64, biomes: Vec<Biome>, settings: Settings) -> TerrainCore {
        assert!(!biomes.is_empty(), "至少要有一个生物群系");
        let s = seed as u64;
        TerrainCore {
            seed,
            biomes,
            settings,
            height: Perlin::new(s ^ 0x1001),
            ridge: Perlin::new(s ^ 0x2002),
            detail: Perlin::new(s ^ 0x3003),
            temperature: Perlin::new(s ^ 0x4004),
            humidity: Perlin::new(s ^ 0x5005),
            columns: RefCell::new(HashMap::new()),
        }
    }

    /// (x, z)这一列的地形
    pub fn column(&self, x: i64, z: i64) -> Column {
        if let Some(c) = self.columns.borrow().get(&(x, z)) {
            return *c;
        }
        let c = self.compute_column(x as f64, z as f64);
        let mut cache = self.columns.borrow_mut();
        if cache.len() > 1 << 18 {
            cache.clear();
        }
        cache.insert((x, z), c);
        c
    }

    pub fn biome(&self, c: Column) -> &Biome {
        &self.biomes[c.biome]
    }

    fn compute_column(&self, x: f64, z: f64) -> Column {
        let t = (self.temperature.fbm(x / 700.0, z / 700.0, 3) * 1.8).clamp(-1.0, 1.0);
        let h = (self.humidity.fbm(x / 550.0, z / 550.0, 3) * 1.8).clamp(-1.0, 1.0);
        let plain = self.height.fbm(x / 140.0, z / 140.0, 4);
        let r = 1.0 - self.ridge.fbm(x / 220.0, z / 220.0, 4).abs();
        let ridged = r * r;

        let (mut total, mut height, mut best_w, mut best) = (0.0, 0.0, -1.0, 0);
        for (i, b) in self.biomes.iter().enumerate() {
            let d = (t - b.temperature).powi(2) + (h - b.humidity).powi(2);
            let w = (-d / CLIMATE_SIGMA2).exp();
            let shape = if b.ridged { ridged } else { plain };
            height += w * (b.base_height + b.variation * shape);
            total += w;
            if w > best_w {
                best_w = w;
                best = i;
            }
        }
        let height = height / f64::max(total, 1e-12) + self.detail.fbm(x / 24.0, z / 24.0, 2) * 1.5;
        Column { height: height.floor() as i64, biome: best }
    }

    /// 不考虑feature时(x, y, z)是什么方块
    pub fn base_block(&self, c: Column, y: i64) -> BlockId {
        let s = &self.settings;
        if y < s.bottom_y() || y > c.height {
            return AIR;
        }
        if y == s.bottom_y() {
            return s.bedrock;
        }
        let b = self.biome(c);
        let rock = b.rock_line.is_some_and(|r| c.height >= r);
        if y == c.height {
            if b.snow_line.is_some_and(|l| y >= l) {
                return s.snow;
            }
            return if rock { s.stone } else { b.surface };
        }
        if c.height - y < b.subsurface_depth && !rock {
            return b.subsurface;
        }
        s.stone
    }
}

/// 正在生成的chunk，feature通过它读写方块
pub struct Context {
    /// chunk的CubeId
    pub chunk: CubeId,
    pub data: ChunkData,
    pub core: Rc<TerrainCore>,
}

impl Context {
    /// chunk最小的角的方块坐标
    pub fn origin(&self) -> (i64, i64, i64) {
        (self.chunk.x * CHUNK_SIZE, self.chunk.y * CHUNK_SIZE, self.chunk.z * CHUNK_SIZE)
    }

    fn local(&self, x: i64, y: i64, z: i64) -> Option<(usize, usize, usize)> {
        let (x0, y0, z0) = self.origin();
        let (lx, ly, lz) = (x - x0, y - y0, z - z0);
        let inside = |v: i64| (0..CHUNK_SIZE).contains(&v);
        (inside(lx) && inside(ly) && inside(lz)).then_some((lx as usize, ly as usize, lz as usize))
    }

    /// 方块坐标(x, y, z)现在是什么。不在这个chunk里时返回None
    pub fn get(&self, x: i64, y: i64, z: i64) -> Option<BlockId> {
        self.local(x, y, z).map(|(x, y, z)| self.data.get(x, y, z))
    }

    /// 把方块坐标(x, y, z)设成b，不在这个chunk里时什么都不做
    pub fn set(&mut self, x: i64, y: i64, z: i64, b: BlockId) {
        if let Some((x, y, z)) = self.local(x, y, z) {
            self.data.set(x, y, z, b);
        }
    }

    /// 只有原来是空气时才设成b（比如树叶不要盖掉别的树干）
    pub fn set_if_air(&mut self, x: i64, y: i64, z: i64, b: BlockId) {
        if self.get(x, y, z) == Some(AIR) {
            self.set(x, y, z, b);
        }
    }

    /// 放一个比1小的cube，不在这个chunk里时什么都不做
    pub fn set_small(&mut self, id: CubeId, b: BlockId) {
        if id.depth >= MIN_DEPTH && id.depth <= CHUNK_DEPTH && self.chunk.contains(id) {
            self.data.small.push((id, b));
        }
    }

    pub fn column(&self, x: i64, z: i64) -> Column {
        self.core.column(x, z)
    }

    /// (x, z)这一列地面最上面的方块（不考虑feature）
    pub fn ground(&self, x: i64, z: i64) -> BlockId {
        let c = self.core.column(x, z);
        self.core.base_block(c, c.height)
    }

    /// 确定性的随机数：同一个世界、同一个位置、同一个salt总是一样的
    pub fn rand(&self, x: i64, y: i64, z: i64, salt: u64) -> Rng {
        Rng::new(hash(&[self.core.seed as u64, x as u64, y as u64, z as u64, salt]))
    }
}

/// 回调出错时返回错误信息，这个feature之后会被停用
pub type FeatureFn = Rc<dyn Fn(&mut Context) -> Result<(), String>>;
/// 在地面(x, y, z)上放东西，y是地面最上面那个方块的高度
pub type PlaceFn = Rc<dyn Fn(&mut Context, i64, i64, i64, &mut Rng) -> Result<(), String>>;

/// 矿石：每个chunk放per_chunk条矿脉，每条size个方块，只替换replace
#[derive(Clone)]
pub struct Ore {
    pub salt: u64,
    pub block: BlockId,
    pub replace: BlockId,
    pub per_chunk: u32,
    pub size: u32,
    pub min_y: i64,
    pub max_y: i64,
}

/// 散布：树、花这种"在地面上某些位置放一个东西"的feature。
/// 每一列以density[生物群系]的概率放一个，只放在soil上（soil是空的时候哪里都能放）。
/// 东西可以比一个chunk大（比如树冠伸到隔壁chunk），radius是它最多离中心多少格（水平方向），
/// height是它最多比地面高多少格。因为随机数只由种子和位置决定，隔壁chunk会算出同样的东西，两边正好对上。
#[derive(Clone)]
pub struct Scatter {
    pub salt: u64,
    /// 按生物群系的下标
    pub density: Vec<f64>,
    pub soil: Vec<BlockId>,
    pub radius: i64,
    pub height: i64,
    pub place: PlaceFn,
}

#[derive(Clone)]
pub enum FeatureKind {
    Ore(Ore),
    Scatter(Scatter),
    /// 每个chunk调用一次
    Custom(FeatureFn),
}

#[derive(Clone)]
pub struct Feature {
    pub name: String,
    pub kind: FeatureKind,
}

impl Ore {
    fn place(&self, ctx: &mut Context) {
        let (x0, y0, z0) = ctx.origin();
        if y0 > self.max_y || y0 + CHUNK_SIZE <= self.min_y {
            return;
        }
        let mut r = ctx.rand(ctx.chunk.x, ctx.chunk.y, ctx.chunk.z, self.salt);
        for _ in 0..self.per_chunk {
            let mut p = [0; 3].map(|_| r.range(0, CHUNK_SIZE - 1));
            p[0] += x0;
            p[1] += y0;
            p[2] += z0;
            for _ in 0..self.size {
                if p[1] >= self.min_y && p[1] <= self.max_y && ctx.get(p[0], p[1], p[2]) == Some(self.replace) {
                    ctx.set(p[0], p[1], p[2], self.block);
                }
                p[r.range(0, 2) as usize] += r.range(-1, 1);
            }
        }
    }
}

impl Scatter {
    fn place(&self, ctx: &mut Context) -> Result<(), String> {
        let (x0, y0, z0) = ctx.origin();
        let seed = ctx.core.seed as u64;
        for z in z0 - self.radius..z0 + CHUNK_SIZE + self.radius {
            for x in x0 - self.radius..x0 + CHUNK_SIZE + self.radius {
                let h = hash(&[seed, x as u64, 0, z as u64, self.salt]);
                let col = ctx.column(x, z);
                if unit(h) >= self.density.get(col.biome).copied().unwrap_or(0.0) {
                    continue;
                }
                // 整个东西都不在这个chunk的高度范围里
                let y = col.height;
                if y + self.height < y0 || y >= y0 + CHUNK_SIZE {
                    continue;
                }
                if !self.soil.is_empty() && !self.soil.contains(&ctx.ground(x, z)) {
                    continue;
                }
                let mut r = Rng::new(h ^ self.salt);
                (self.place)(ctx, x, y, z, &mut r)?;
            }
        }
        Ok(())
    }
}

/// 实现world::Generator
pub struct Generator {
    pub core: Rc<TerrainCore>,
    pub features: Vec<Feature>,
    /// 出过错的feature，之后不再执行
    failed: Vec<bool>,
}

impl Generator {
    pub fn new(core: Rc<TerrainCore>, features: Vec<Feature>) -> Generator {
        let failed = vec![false; features.len()];
        Generator { core, features, failed }
    }
}

impl crate::world::Generator for Generator {
    fn vertical_range(&self) -> (i64, i64) {
        (self.core.settings.min_chunk_y, self.core.settings.max_chunk_y)
    }

    fn generate_chunk(&mut self, pos: CubeId) -> ChunkData {
        let mut ctx = Context { chunk: pos, data: ChunkData::new(), core: self.core.clone() };
        let (lo, hi) = self.vertical_range();
        if pos.y < lo || pos.y > hi {
            return ctx.data;
        }
        let (x0, y0, z0) = ctx.origin();
        for lz in 0..CHUNK_SIZE {
            for lx in 0..CHUNK_SIZE {
                let c = self.core.column(x0 + lx, z0 + lz);
                // 整列都在地面上面
                if y0 > c.height {
                    continue;
                }
                for ly in 0..CHUNK_SIZE {
                    let b = self.core.base_block(c, y0 + ly);
                    ctx.data.set(lx as usize, ly as usize, lz as usize, b);
                }
            }
        }
        for (f, failed) in self.features.iter().zip(self.failed.iter_mut()) {
            if *failed {
                continue;
            }
            let result = match &f.kind {
                FeatureKind::Ore(o) => {
                    o.place(&mut ctx);
                    Ok(())
                }
                FeatureKind::Scatter(s) => s.place(&mut ctx),
                FeatureKind::Custom(func) => func(&mut ctx),
            };
            if let Err(e) = result {
                eprintln!("foxel: feature {}出错了，之后不再生成它：{e}", f.name);
                *failed = true;
            }
        }
        ctx.data
    }
}
