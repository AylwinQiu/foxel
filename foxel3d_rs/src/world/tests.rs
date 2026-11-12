use std::collections::HashMap;

use super::*;

const STONE: BlockId = 1;
const DIRT: BlockId = 2;

/// y < 0是石头，其他是空气
struct Flat;

impl Generator for Flat {
    fn generate_chunk(&mut self, pos: CubeId) -> ChunkData {
        let mut out = ChunkData::new();
        for y in 0..CHUNK_SIZE as usize {
            if pos.y * CHUNK_SIZE + (y as i64) < 0 {
                for z in 0..CHUNK_SIZE as usize {
                    for x in 0..CHUNK_SIZE as usize {
                        out.set(x, y, z, STONE);
                    }
                }
            }
        }
        out
    }

    fn vertical_range(&self) -> (i64, i64) {
        (-2, 1)
    }
}

fn flat_world() -> World {
    let mut w = World::new(Box::new(Flat));
    for y in -2..=1 {
        for z in -1..=0 {
            for x in -1..=0 {
                w.load_chunk(CubeId::new(x, y, z, CHUNK_DEPTH));
            }
        }
    }
    w
}

fn solid(b: BlockId) -> bool {
    b != AIR
}

#[test]
fn cube_id() {
    let p = Position::new(-0.3, 1.7, 5.2);
    let id = CubeId::at(p, 0);
    assert_eq!(id, CubeId::new(-1, 1, 5, 0));
    assert_eq!(id.chunk(), CubeId::new(-1, 0, 0, CHUNK_DEPTH));
    let small = CubeId::at(p, MIN_DEPTH);
    assert!(id.contains(small) && !small.contains(id));
    assert_eq!(small.parent_at(0), id);
    for o in 0..8 {
        let c = id.child(o);
        assert_eq!(c.octant(), o);
        assert_eq!(c.parent_at(0), id);
    }
}

#[test]
fn generate_and_merge() {
    let w = flat_world();
    // 全是石头的chunk合并成了一个叶子
    let (n, at) = w.find(CubeId::block(-3, -5, -3)).unwrap();
    assert_eq!((n.block, at.depth), (STONE, CHUNK_DEPTH));
    assert_eq!(w.block_at(0, 0, 0), Some((AIR, false)));
    // 范围外是空气，没加载的是None
    assert_eq!(w.leaf_at(Position::new(0.0, 1000.0, 0.0)).unwrap().block, AIR);
    assert!(w.leaf_at(Position::new(100.0, 0.0, 0.0)).is_none());
}

#[test]
fn set_cube_and_fields() {
    let mut w = flat_world();
    w.set_cube(CubeId::block(2, -1, 3), AIR).unwrap();
    assert_eq!(w.block_at(2, -1, 3), Some((AIR, false)));
    assert_eq!(w.block_at(3, -1, 3), Some((STONE, false)));
    // 填回去又合并成一个叶子
    w.set_cube(CubeId::block(2, -1, 3), STONE).unwrap();
    assert_eq!(w.find(CubeId::block(2, -1, 3)).unwrap().1.depth, CHUNK_DEPTH);

    // 最小的方块（1/16格）
    let small = CubeId::at(Position::new(0.1, 0.1, 0.1), MIN_DEPTH);
    assert_eq!(small.side(), 1.0 / 16.0);
    w.set_cube(small, DIRT).unwrap();
    assert_eq!(w.block_at(0, 0, 0), Some((AIR, true)));
    assert_eq!(w.leaf_at(small.center()).unwrap(), Leaf { id: small, block: DIRT });
    assert_eq!(w.leaf_at(small.center() + Position::new(0.0625, 0.0, 0.0)).unwrap().block, AIR);

    // 场在cube被覆盖时删掉
    w.field_write("test:x", small, "hello".into());
    assert_eq!(w.field_read("test:x", small), Some("hello"));
    w.set_cube(CubeId::block(0, 0, 0), AIR).unwrap();
    assert_eq!(w.field_read("test:x", small), None);

    assert_eq!(w.take_changes().len(), 4);
    assert_eq!(w.set_cube(CubeId::block(100, 0, 0), STONE), Err(WorldError::NotLoaded));
    assert_eq!(w.set_cube(CubeId::new(0, 0, 0, MIN_DEPTH - 1), STONE), Err(WorldError::BadDepth));
}

#[test]
fn save_round_trip() {
    let mut w = flat_world();
    w.set_cube(CubeId::block(1, 0, 1), DIRT).unwrap();
    w.set_cube(CubeId::at(Position::new(5.6, 0.1, 1.0), -1), STONE).unwrap();
    let saved = w.saved_chunks();
    assert_eq!(saved.len(), 1);

    // 卸载再加载
    let pos = CubeId::new(0, 0, 0, CHUNK_DEPTH);
    w.unload_chunk(pos);
    assert!(!w.is_loaded(pos));
    w.load_chunk(pos);
    assert_eq!(w.block_at(1, 0, 1), Some((DIRT, false)));

    // 新世界读存档，DIRT和STONE换了编号
    let mut w2 = World::new(Box::new(Flat));
    let swap = |b: BlockId| HashMap::from([(STONE, DIRT), (DIRT, STONE), (AIR, AIR)])[&b];
    assert_eq!(w2.set_saved_chunks(saved, &swap), 0);
    w2.load_chunk(pos);
    assert_eq!(w2.block_at(1, 0, 1), Some((STONE, false)));
    let l = w2.leaf_at(Position::new(5.6, 0.1, 1.0)).unwrap();
    assert_eq!((l.block, l.id.depth), (DIRT, -1));

    assert!(encode::decode_chunk(&[0, 1]).is_err());
}

#[test]
fn raycast() {
    let mut w = flat_world();
    let hit = w.raycast(Position::new(0.5, 3.3, 0.5), Position::new(0.0, -1.0, 0.0), 10.0, &solid).unwrap();
    assert_eq!(hit.normal, [0, 1, 0]);
    assert!(hit.point.y.abs() < 1e-9 && (hit.distance - 3.3).abs() < 1e-9);

    let small = CubeId::at(Position::new(3.1, 0.1, 0.6), MIN_DEPTH);
    w.set_cube(small, DIRT).unwrap();
    let hit = w.raycast(Position::new(1.0, 0.1, 0.6), Position::new(1.0, 0.0, 0.0), 10.0, &solid).unwrap();
    assert_eq!((hit.leaf.id, hit.normal), (small, [-1, 0, 0]));
    assert!((hit.point.x - small.min().x).abs() < 1e-9);

    assert!(w.raycast(Position::new(0.5, 3.3, 0.5), Position::new(0.0, -1.0, 0.0), 2.0, &solid).is_none());
}

#[test]
fn move_box_walls_and_steps() {
    let mut w = flat_world();
    let start = Position::new(0.5, 0.0, 0.5);
    // 往下掉：停在地面上
    let (p, hit) = w.move_box(start, 0.6, 1.8, Position::new(0.0, -0.5, 0.0), 0.5, &solid);
    assert!(hit[1] && p.y.abs() < 1e-3);
    // 往东走碰到一堵两格高的墙
    for y in 0..2 {
        for z in -2..3 {
            w.set_cube(CubeId::block(2, y, z), STONE).unwrap();
        }
    }
    let (p, hit) = w.move_box(start, 0.6, 1.8, Position::new(3.0, -0.01, 0.0), 0.5, &solid);
    assert!(hit[0] && (p.x - 1.7).abs() < 1e-3, "{p:?}");
    // 往西有个半格高的台阶：直接走上去
    for z in -2..3 {
        w.set_cube(CubeId::new(-2, 0, z * 2, -1), STONE).unwrap();
        w.set_cube(CubeId::new(-2, 0, z * 2 + 1, -1), STONE).unwrap();
        w.set_cube(CubeId::new(-3, 0, z * 2, -1), STONE).unwrap();
        w.set_cube(CubeId::new(-3, 0, z * 2 + 1, -1), STONE).unwrap();
    }
    let (p, _) = w.move_box(start, 0.6, 1.8, Position::new(-1.5, -0.01, 0.0), 0.5, &solid);
    assert!(p.x < -0.5 && (p.y - 0.5).abs() < 1e-3, "{p:?}");
    // 1/16格的小方块也挡得住
    let tiny = CubeId::at(Position::new(0.53, 0.03, -0.53), MIN_DEPTH);
    w.set_cube(tiny, STONE).unwrap();
    assert_eq!(tiny.side(), 1.0 / 16.0);
    let (p, hit) = w.move_box(start, 0.6, 1.8, Position::new(0.0, -0.01, -1.0), 0.0, &solid);
    // 小方块在z=-0.5到-0.5625，玩家前面停在-0.5，中心是-0.5+0.3
    assert!(hit[2] && (p.z - (-0.5 + 0.3)).abs() < 1e-3, "{p:?}");
    w.set_cube(tiny, AIR).unwrap();
    // 不让迈台阶时被挡住
    let (p, hit) = w.move_box(start, 0.6, 1.8, Position::new(-1.5, -0.01, 0.0), 0.0, &solid);
    assert!(hit[0] && p.y.abs() < 1e-3, "{p:?}");
}
