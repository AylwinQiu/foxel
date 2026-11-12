//! 把world画到canvas上：按相机加载chunk、卸载远处的chunk、生成网格、视锥剔除。
//!
//! 浮点原点：world里的坐标可能很大，canvas里的坐标都减去相机位置，相机永远在原点附近，
//! 走到很远的地方画面也不会因为浮点数精度不够而抖动。

mod mesher;
#[cfg(test)]
mod tests;

use std::collections::{HashMap, HashSet};
use std::time::{Duration, Instant};

pub use mesher::build_quads;

use crate::block::Registry;
use crate::canvas::{Canvas, Color, Images, MeshId, Vec3};
use crate::world::{CHUNK_DEPTH, CHUNK_SIZE, CubeId, Position, World};

/// 上一帧的统计，调试用
#[derive(Clone, Copy, Debug, Default)]
pub struct Stats {
    /// 已加载的chunk、有网格的chunk、这一帧画了的chunk
    pub loaded: usize,
    pub meshes: usize,
    pub drawn: usize,
    /// 邻居都加载好了、还在排队等着生成网格的chunk
    pub pending: usize,
}

struct ChunkMesh {
    id: MeshId,
    /// 网格是空的（比如全是空气或者全埋在地下）
    empty: bool,
}

pub struct Engine {
    /// 水平方向加载多少个chunk（半径）
    pub view_distance: i64,
    pub stats: Stats,
    cam_pos: Position,
    /// 单位向量
    cam_dir: Vec3,
    meshes: HashMap<CubeId, ChunkMesh>,
    /// 需要重新生成网格的chunk
    dirty: HashSet<CubeId>,
    next_id: MeshId,
}

impl Default for Engine {
    fn default() -> Self {
        Self::new()
    }
}

/// chunk的外接球半径
const CHUNK_RADIUS: f64 = CHUNK_SIZE as f64 * 0.8660254037844386;

fn chunk_coord(v: f64) -> i64 {
    (v / CHUNK_SIZE as f64).floor() as i64
}

impl Engine {
    /// 相机默认在(10, 10, 10)看向原点
    pub fn new() -> Engine {
        let mut e = Engine {
            view_distance: 8,
            stats: Stats::default(),
            cam_pos: Position::default(),
            cam_dir: Vec3::new(0.0, 0.0, -1.0),
            meshes: HashMap::new(),
            dirty: HashSet::new(),
            next_id: 0,
        };
        e.set_camera(Position::new(10.0, 10.0, 10.0), Position::new(-1.0, -1.0, -1.0));
        e
    }

    // ---------------------------------------------------------------- 相机

    /// 设置相机在世界里的位置和朝向，direction只取方向，长度为0时只改位置
    pub fn set_camera(&mut self, position: Position, direction: Position) {
        self.cam_pos = position;
        let l = direction.length();
        if l > 0.0 {
            self.cam_dir = Vec3::new(direction.x / l, direction.y / l, direction.z / l);
        }
    }

    /// 相机的位置和朝向（单位向量）
    pub fn camera(&self) -> (Position, Position) {
        (self.cam_pos, Position::new(self.cam_dir.x, self.cam_dir.y, self.cam_dir.z))
    }

    /// 物理坐标 -> canvas坐标：减去相机位置
    pub fn to_canvas(&self, p: Position) -> Vec3 {
        Vec3::new(p.x - self.cam_pos.x, p.y - self.cam_pos.y, p.z - self.cam_pos.z)
    }

    // ---------------------------------------------------------------- 加载

    /// 换了一个世界：删掉所有网格
    pub fn reset(&mut self, canvas: &mut dyn Canvas) {
        for m in self.meshes.values() {
            canvas.delete_mesh(m.id);
        }
        self.meshes.clear();
        self.dirty.clear();
    }

    /// 加载相机附近的chunk、卸载远处的chunk、重新生成改动过的chunk的网格，最多花budget的时间
    pub fn update(&mut self, world: &mut World, canvas: &mut dyn Canvas, blocks: &Registry, budget: Duration) {
        let start = Instant::now();
        for id in world.take_changes() {
            self.mark_dirty(world, id);
        }
        let r = self.view_distance;
        let (cx, cz) = (chunk_coord(self.cam_pos.x), chunk_coord(self.cam_pos.z));

        // 远处的卸载（多留几个chunk，在边界上走来走去时不会反复加载）
        for pos in world.loaded_chunks() {
            let (dx, dz) = (pos.x - cx, pos.z - cz);
            if dx * dx + dz * dz > (r + 3) * (r + 3) {
                if let Some(m) = self.meshes.remove(&pos) {
                    canvas.delete_mesh(m.id);
                }
                self.dirty.remove(&pos);
                world.unload_chunk(pos);
            }
        }

        // 近的先加载。多加载一圈：生成网格需要邻居，这样视距以内的chunk都能有网格
        let (lo, hi) = world.generator().vertical_range();
        let lr = r + 1;
        let mut missing = Vec::new();
        for dz in -lr..=lr {
            for dx in -lr..=lr {
                if dx * dx + dz * dz > lr * lr {
                    continue;
                }
                for y in lo..=hi {
                    let pos = CubeId::new(cx + dx, y, cz + dz, CHUNK_DEPTH);
                    if !world.is_loaded(pos) {
                        missing.push(pos);
                    }
                }
            }
        }
        self.sort_by_distance(&mut missing);
        for pos in missing {
            if start.elapsed() > budget / 2 {
                break;
            }
            world.load_chunk(pos);
            self.dirty.insert(pos);
        }

        // 生成网格：6个邻居都加载了才能知道边上的面要不要画
        let mut todo: Vec<CubeId> =
            self.dirty.iter().copied().filter(|&p| world.is_loaded(p) && neighbors_loaded(world, p)).collect();
        self.sort_by_distance(&mut todo);
        let mut built = 0;
        for &pos in &todo {
            if built > 0 && start.elapsed() > budget {
                break;
            }
            self.build_mesh(world, canvas, blocks, pos);
            self.dirty.remove(&pos);
            built += 1;
        }

        self.stats.loaded = world.loaded_count();
        self.stats.pending = todo.len() - built;
        self.stats.meshes = self.meshes.values().filter(|m| !m.empty).count();
    }

    /// 相机周围radius个chunk以内是否都加载好、网格也生成好了（刚进游戏时等它变成true）
    pub fn ready(&self, world: &World, radius: i64) -> bool {
        let p = self.cam_pos;
        let (cx, cy, cz) = (chunk_coord(p.x), chunk_coord(p.y), chunk_coord(p.z));
        for dy in -radius..=radius {
            for dz in -radius..=radius {
                for dx in -radius..=radius {
                    let pos = CubeId::new(cx + dx, cy + dy, cz + dz, CHUNK_DEPTH);
                    if world.in_range(pos) && (!self.meshes.contains_key(&pos) || self.dirty.contains(&pos)) {
                        return false;
                    }
                }
            }
        }
        true
    }

    fn sort_by_distance(&self, list: &mut [CubeId]) {
        let c = self.cam_pos;
        let d2 = |pos: &CubeId| {
            let p = pos.center() - c;
            p.dot(p)
        };
        list.sort_by(|a, b| d2(a).total_cmp(&d2(b)));
    }

    /// id这个cube改动了：它所在的chunk要重新生成网格，碰到chunk边上时隔壁的chunk也要
    fn mark_dirty(&mut self, world: &World, id: CubeId) {
        let pos = id.chunk();
        self.dirty.insert(pos);
        let (lo, hi) = (id.min(), id.max());
        let (clo, chi) = (pos.min(), pos.max());
        let mut touch = |dx, dy, dz| {
            let n = pos.neighbor(dx, dy, dz);
            if world.is_loaded(n) {
                self.dirty.insert(n);
            }
        };
        if lo.x == clo.x {
            touch(-1, 0, 0);
        }
        if hi.x == chi.x {
            touch(1, 0, 0);
        }
        if lo.y == clo.y {
            touch(0, -1, 0);
        }
        if hi.y == chi.y {
            touch(0, 1, 0);
        }
        if lo.z == clo.z {
            touch(0, 0, -1);
        }
        if hi.z == chi.z {
            touch(0, 0, 1);
        }
    }

    fn build_mesh(&mut self, world: &World, canvas: &mut dyn Canvas, blocks: &Registry, pos: CubeId) {
        let quads = build_quads(world, blocks, pos).unwrap_or_default();
        let next = &mut self.next_id;
        let m = self.meshes.entry(pos).or_insert_with(|| {
            *next += 1;
            ChunkMesh { id: *next, empty: true }
        });
        m.empty = quads.is_empty();
        canvas.set_mesh(m.id, &quads);
    }

    // ---------------------------------------------------------------- 绘制

    /// 四个侧面加一个相机所在的近平面（只剔除相机背后的东西），远平面不剔除。法线都指向视锥里面
    fn frustum_planes(&self, canvas: &dyn Canvas) -> [Vec3; 5] {
        let [tl, tr, bl, br] = canvas.frustum_corner_rays(self.cam_dir);
        let side = |a: Vec3, b: Vec3| {
            let n = a.cross(b).normalize();
            if n.dot(self.cam_dir) < 0.0 { n.scale(-1.0) } else { n }
        };
        [side(tl, bl), side(tr, br), side(tl, tr), side(bl, br), self.cam_dir]
    }

    /// 把视锥内所有chunk的网格画到canvas上
    pub fn draw_world(&mut self, canvas: &mut dyn Canvas) {
        canvas.camera(Vec3::default(), self.cam_dir);
        let planes = self.frustum_planes(canvas);
        self.stats.drawn = 0;
        for (pos, m) in &self.meshes {
            if m.empty {
                continue;
            }
            let d = self.to_canvas(pos.center());
            if planes.iter().any(|n| n.dot(d) < -CHUNK_RADIUS) {
                continue;
            }
            canvas.add_mesh(m.id, self.to_canvas(pos.min()));
            self.stats.drawn += 1;
        }
    }

    /// 世界里的3D线条，用物理坐标
    pub fn add_line3d(&self, canvas: &mut dyn Canvas, start: Position, end: Position, c: Color) {
        canvas.add_line3d(self.to_canvas(start), self.to_canvas(end), c);
    }

    /// 长方体的12条边，lo和hi是两个对角
    pub fn add_box_lines(&self, canvas: &mut dyn Canvas, lo: Position, hi: Position, c: Color) {
        let p = |x: bool, y: bool, z: bool| {
            Position::new(if x { hi.x } else { lo.x }, if y { hi.y } else { lo.y }, if z { hi.z } else { lo.z })
        };
        for a in [false, true] {
            for b in [false, true] {
                self.add_line3d(canvas, p(false, a, b), p(true, a, b), c);
                self.add_line3d(canvas, p(a, false, b), p(a, true, b), c);
                self.add_line3d(canvas, p(a, b, false), p(a, b, true), c);
            }
        }
    }

    /// 世界里一个会动的cube（不在world里），用物理坐标
    pub fn add_cube(&self, canvas: &mut dyn Canvas, center: Position, side: f64, images: &Images, rotation: Vec3) {
        canvas.add_cube(side, self.to_canvas(center), images, rotation);
    }
}

fn neighbors_loaded(world: &World, pos: CubeId) -> bool {
    [(1, 0, 0), (-1, 0, 0), (0, 1, 0), (0, -1, 0), (0, 0, 1), (0, 0, -1)].iter().all(|&(dx, dy, dz)| {
        let n = pos.neighbor(dx, dy, dz);
        !world.in_range(n) || world.is_loaded(n)
    })
}
