use std::fmt;
use std::ops::{Add, Mul, Sub};

/// 根cube的深度：整个世界是边长2^32的cube
pub const IDENTITY_DEPTH: Depth = 32;
/// 最小的cube：边长1/16
pub const MIN_DEPTH: Depth = -4;
/// 一个chunk是边长16的cube，世界按chunk加载、保存、画
pub const CHUNK_DEPTH: Depth = 4;
pub const CHUNK_SIZE: i64 = 1 << CHUNK_DEPTH;

/// cube的深度，边长是2^depth。0是普通方块
pub type Depth = i32;

/// 深度为d的cube的边长
pub fn side(d: Depth) -> f64 {
    2f64.powi(d)
}

/// 物理坐标。x向东，y向上，z向南（北是-z），1就是1格
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct Position {
    pub x: f64,
    pub y: f64,
    pub z: f64,
}

impl Position {
    pub const fn new(x: f64, y: f64, z: f64) -> Position {
        Position { x, y, z }
    }

    pub fn length(self) -> f64 {
        (self.x * self.x + self.y * self.y + self.z * self.z).sqrt()
    }

    pub fn dot(self, b: Position) -> f64 {
        self.x * b.x + self.y * b.y + self.z * b.z
    }

    /// 第a个分量（0是x，1是y，2是z）
    pub fn axis(self, a: usize) -> f64 {
        [self.x, self.y, self.z][a]
    }

    /// 第a个分量加上d
    pub fn add_axis(mut self, a: usize, d: f64) -> Position {
        match a {
            0 => self.x += d,
            1 => self.y += d,
            _ => self.z += d,
        }
        self
    }
}

impl Add for Position {
    type Output = Position;
    fn add(self, b: Position) -> Position {
        Position::new(self.x + b.x, self.y + b.y, self.z + b.z)
    }
}

impl Sub for Position {
    type Output = Position;
    fn sub(self, b: Position) -> Position {
        Position::new(self.x - b.x, self.y - b.y, self.z - b.z)
    }
}

impl Mul<f64> for Position {
    type Output = Position;
    fn mul(self, k: f64) -> Position {
        Position::new(self.x * k, self.y * k, self.z * k)
    }
}

/// 八叉树里一个格子的编号：深度为depth的格子，x方向覆盖物理坐标[x*边长, (x+1)*边长)，y、z一样。
/// 同一个位置、同一个深度永远是同一个CubeId，所以可以存档，也可以当HashMap的key。
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
pub struct CubeId {
    pub x: i64,
    pub y: i64,
    pub z: i64,
    pub depth: Depth,
}

impl CubeId {
    pub const fn new(x: i64, y: i64, z: i64, depth: Depth) -> CubeId {
        CubeId { x, y, z, depth }
    }

    /// depth 0的格子(x, y, z)，也就是[x, x+1)×[y, y+1)×[z, z+1)的普通方块
    pub const fn block(x: i64, y: i64, z: i64) -> CubeId {
        CubeId::new(x, y, z, 0)
    }

    /// 包含物理坐标p的、深度为d的格子
    pub fn at(p: Position, d: Depth) -> CubeId {
        let s = side(d);
        CubeId::new((p.x / s).floor() as i64, (p.y / s).floor() as i64, (p.z / s).floor() as i64, d)
    }

    pub fn side(self) -> f64 {
        side(self.depth)
    }

    /// 最小的角（x、y、z都最小）
    pub fn min(self) -> Position {
        let s = self.side();
        Position::new(self.x as f64 * s, self.y as f64 * s, self.z as f64 * s)
    }

    /// 最大的角
    pub fn max(self) -> Position {
        let s = self.side();
        Position::new((self.x + 1) as f64 * s, (self.y + 1) as f64 * s, (self.z + 1) as f64 * s)
    }

    pub fn center(self) -> Position {
        (self.min() + self.max()) * 0.5
    }

    /// 包含这个cube的、深度为d的cube，d必须不小于self.depth
    pub fn parent_at(self, d: Depth) -> CubeId {
        let shift = (d - self.depth) as u32;
        // 算术右移就是向下取整的除法，负数也对
        CubeId::new(self.x >> shift, self.y >> shift, self.z >> shift, d)
    }

    /// 第o个子cube，o是卦限：x正为1，y正为2，z正为4
    pub fn child(self, o: usize) -> CubeId {
        let o = o as i64;
        CubeId::new(self.x * 2 + (o & 1), self.y * 2 + (o >> 1 & 1), self.z * 2 + (o >> 2 & 1), self.depth - 1)
    }

    /// 这个cube在父cube里的卦限
    pub fn octant(self) -> usize {
        ((self.x & 1) | (self.y & 1) << 1 | (self.z & 1) << 2) as usize
    }

    /// other是不是在这个cube里面（包括other就是它自己）
    pub fn contains(self, other: CubeId) -> bool {
        other.depth <= self.depth && other.parent_at(self.depth) == self
    }

    /// 同样大小、挨着的cube，dx、dy、dz是格子数
    pub fn neighbor(self, dx: i64, dy: i64, dz: i64) -> CubeId {
        CubeId::new(self.x + dx, self.y + dy, self.z + dz, self.depth)
    }

    /// 这个cube所在的chunk，self.depth必须不大于CHUNK_DEPTH
    pub fn chunk(self) -> CubeId {
        self.parent_at(CHUNK_DEPTH)
    }
}

impl fmt::Display for CubeId {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "({}, {}, {})@{}", self.x, self.y, self.z, self.depth)
    }
}
