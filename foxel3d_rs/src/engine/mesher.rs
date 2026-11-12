//! 把一个chunk变成canvas的网格：只生成露在外面的面（旁边是不透明方块的面被省掉），
//! 大的cube按1x1的格子切开，所以贴图在大cube上也是一格一张。

use crate::block::{BlockType, Registry};
use crate::canvas::{Color, Image, Images, Quad, Vec2, Vec3};
use crate::world::{AIR, BlockId, CHUNK_SIZE, Cube, CubeId, MIN_DEPTH, Position, World, side};

/// cube的6个面
struct Face {
    normal: [i64; 3],
    /// 4个角（从外面看逆时针：左上、左下、右下、右上），每个坐标0表示取最小值，1表示取最大值
    corners: [[usize; 3]; 4],
    /// 明暗：上面最亮，下面最暗
    tint: u8,
    /// 一个角在它所在的1x1格子里的相对位置(lx, ly, lz) -> 图片上的(u, v)
    uv: fn(f64, f64, f64) -> (f64, f64),
    /// 这个面用Images里的哪张图片
    image: fn(&Images) -> Option<&Image>,
}

const FACES: [Face; 6] = [
    Face {
        normal: [1, 0, 0],
        corners: [[1, 1, 1], [1, 0, 1], [1, 0, 0], [1, 1, 0]],
        tint: 170,
        uv: |_, ly, lz| (1.0 - lz, 1.0 - ly),
        image: |i| i.east.as_ref(),
    },
    Face {
        normal: [-1, 0, 0],
        corners: [[0, 1, 0], [0, 0, 0], [0, 0, 1], [0, 1, 1]],
        tint: 170,
        uv: |_, ly, lz| (lz, 1.0 - ly),
        image: |i| i.west.as_ref(),
    },
    Face {
        normal: [0, 1, 0],
        corners: [[0, 1, 0], [0, 1, 1], [1, 1, 1], [1, 1, 0]],
        tint: 255,
        uv: |lx, _, lz| (lx, lz),
        image: |i| i.up.as_ref(),
    },
    Face {
        normal: [0, -1, 0],
        corners: [[0, 0, 1], [0, 0, 0], [1, 0, 0], [1, 0, 1]],
        tint: 130,
        uv: |lx, _, lz| (lx, 1.0 - lz),
        image: |i| i.down.as_ref(),
    },
    Face {
        normal: [0, 0, 1],
        corners: [[0, 1, 1], [0, 0, 1], [1, 0, 1], [1, 1, 1]],
        tint: 210,
        uv: |lx, ly, _| (lx, 1.0 - ly),
        image: |i| i.south.as_ref(),
    },
    Face {
        normal: [0, 0, -1],
        corners: [[1, 1, 0], [1, 0, 0], [0, 0, 0], [0, 1, 0]],
        tint: 210,
        uv: |lx, ly, _| (1.0 - lx, 1.0 - ly),
        image: |i| i.north.as_ref(),
    },
];

/// b旁边是nb时，b朝nb的那个面是不是被挡住了
fn hidden(blocks: &Registry, b: BlockId, nb: BlockId) -> bool {
    if nb == AIR {
        return false;
    }
    if blocks.is_opaque(nb) {
        return true;
    }
    nb == b && blocks.get(b).cull_same
}

/// 一个面的四边形。lo、hi是cube相对chunk原点的两个对角，cell是它所在的1x1格子的最小角
fn face_quad(f: &Face, t: &BlockType, lo: [f64; 3], hi: [f64; 3], cell: [f64; 3]) -> Quad {
    let mut corners = [Vec3::default(); 4];
    let mut uv = [Vec2::default(); 4];
    for (k, c) in f.corners.iter().enumerate() {
        let p: [f64; 3] = std::array::from_fn(|a| if c[a] == 1 { hi[a] } else { lo[a] });
        corners[k] = Vec3::new(p[0], p[1], p[2]);
        let (u, v) = (f.uv)(p[0] - cell[0], p[1] - cell[1], p[2] - cell[2]);
        uv[k] = Vec2::new(u, v);
    }
    Quad { corners, uv, img: (f.image)(&t.images).cloned(), tint: Color::rgb(f.tint, f.tint, f.tint) }
}

/// 1x1格子（最小角是cell）的f面外面紧挨着的是被分成小cube的格子时，看那一层小cube是不是把这个面整个挡住了
fn face_covered(w: &World, blocks: &Registry, b: BlockId, cell: Position, f: &Face) -> bool {
    let n = (1i64 << -MIN_DEPTH) as f64;
    // 法线所在的轴，和面上的另外两个轴
    let an = (0..3).find(|&a| f.normal[a] != 0).unwrap();
    let (t1, t2) = ((an + 1) % 3, (an + 2) % 3);
    for i in 0..n as i64 {
        for j in 0..n as i64 {
            // 面上第(i, j)个最小格子的中心，往外挪半个最小格子
            let mut p = [0.0; 3];
            p[t1] = (i as f64 + 0.5) / n;
            p[t2] = (j as f64 + 0.5) / n;
            p[an] = if f.normal[an] > 0 { 1.0 + 0.5 / n } else { -0.5 / n };
            match w.leaf_at(cell + Position::new(p[0], p[1], p[2])) {
                Some(nb) if hidden(blocks, b, nb.block) => {}
                _ => return false,
            }
        }
    }
    true
}

const PAD: usize = CHUNK_SIZE as usize + 2;

fn grid_index(x: i64, y: i64, z: i64) -> usize {
    (x + 1) as usize + (z + 1) as usize * PAD + (y + 1) as usize * PAD * PAD
}

/// 生成pos这个chunk的所有面，坐标相对chunk最小的角。chunk没有加载时返回None
pub fn build_quads(w: &World, blocks: &Registry, pos: CubeId) -> Option<Vec<Quad>> {
    // chunk和它外面一圈的1x1格子
    let mut grid = vec![AIR; PAD * PAD * PAD];
    let mut partial = vec![false; PAD * PAD * PAD];
    let mut subdivided: Vec<(CubeId, &Cube)> = Vec::new();
    let origin = pos.min();
    let (x0, y0, z0) = (pos.x * CHUNK_SIZE, pos.y * CHUNK_SIZE, pos.z * CHUNK_SIZE);

    // chunk里面：边长>=1的叶子按格子填满，分成更小cube的格子记下来单独处理。
    // visit_chunk的回调拿不到带生命周期的引用，所以先记下id，之后再用find取节点
    let mut sub_ids = Vec::new();
    let loaded = w.visit_chunk(pos, &mut |id, n| {
        if id.depth < 0 {
            return false;
        }
        let s = 1i64 << id.depth;
        let (lx, ly, lz) = (id.x * s - x0, id.y * s - y0, id.z * s - z0);
        if n.is_leaf() {
            for y in ly..ly + s {
                for z in lz..lz + s {
                    for x in lx..lx + s {
                        grid[grid_index(x, y, z)] = n.block;
                    }
                }
            }
            return false;
        }
        if id.depth == 0 {
            partial[grid_index(lx, ly, lz)] = true;
            sub_ids.push(id);
            return false;
        }
        true
    });
    if !loaded {
        return None;
    }
    for id in sub_ids {
        if let Some((n, _)) = w.find(id) {
            subdivided.push((id, n));
        }
    }

    // 外面一圈：只需要6个面上的格子
    let mut border = |lx: i64, ly: i64, lz: i64| {
        let (b, p) = w.block_at(x0 + lx, y0 + ly, z0 + lz).unwrap_or((AIR, false));
        let i = grid_index(lx, ly, lz);
        grid[i] = b;
        partial[i] = p;
    };
    for i in 0..CHUNK_SIZE {
        for j in 0..CHUNK_SIZE {
            border(-1, i, j);
            border(CHUNK_SIZE, i, j);
            border(i, -1, j);
            border(i, CHUNK_SIZE, j);
            border(i, j, -1);
            border(i, j, CHUNK_SIZE);
        }
    }

    let mut quads = Vec::new();
    for y in 0..CHUNK_SIZE {
        for z in 0..CHUNK_SIZE {
            for x in 0..CHUNK_SIZE {
                let i = grid_index(x, y, z);
                let b = grid[i];
                if b == AIR || partial[i] {
                    continue;
                }
                let t = blocks.get(b);
                let lo = [x as f64, y as f64, z as f64];
                let hi = [lo[0] + 1.0, lo[1] + 1.0, lo[2] + 1.0];
                for f in &FACES {
                    let j = grid_index(x + f.normal[0], y + f.normal[1], z + f.normal[2]);
                    let covered = if partial[j] {
                        face_covered(w, blocks, b, origin + Position::new(lo[0], lo[1], lo[2]), f)
                    } else {
                        hidden(blocks, b, grid[j])
                    };
                    if !covered {
                        quads.push(face_quad(f, t, lo, hi, lo));
                    }
                }
            }
        }
    }

    // 分成小cube的格子：每个小cube查一下面外面紧挨着的叶子，
    // 那个叶子不比它小（能把这个面整个挡住）而且挡得住时才省掉这个面
    fn small(
        w: &World,
        blocks: &Registry,
        origin: Position,
        id: CubeId,
        n: &Cube,
        cell: [f64; 3],
        quads: &mut Vec<Quad>,
    ) {
        if let Some(children) = &n.children {
            for (o, c) in children.iter().enumerate() {
                small(w, blocks, origin, id.child(o), c, cell, quads);
            }
            return;
        }
        if n.block == AIR {
            return;
        }
        let t = blocks.get(n.block);
        let half = id.side() / 2.0;
        let center = id.center();
        let (lo, hi) = (id.min() - origin, id.max() - origin);
        let lo = [lo.x, lo.y, lo.z];
        let hi = [hi.x, hi.y, hi.z];
        for f in &FACES {
            let d = half + side(MIN_DEPTH) / 2.0;
            let p = center + Position::new(f.normal[0] as f64, f.normal[1] as f64, f.normal[2] as f64) * d;
            if let Some(nb) = w.leaf_at(p)
                && nb.id.depth >= id.depth
                && hidden(blocks, n.block, nb.block)
            {
                continue;
            }
            quads.push(face_quad(f, t, lo, hi, cell));
        }
    }
    for (id, n) in subdivided {
        let c = id.min() - origin;
        small(w, blocks, origin, id, n, [c.x, c.y, c.z], &mut quads);
    }
    Some(quads)
}
