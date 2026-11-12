use std::time::Duration;

use super::*;
use crate::block::{BlockType, Registry};
use crate::canvas::NullCanvas;
use crate::world::{BlockId, ChunkData, Generator};

/// 全是空气，竖直方向只有一层chunk
struct Empty;

impl Generator for Empty {
    fn generate_chunk(&mut self, _: CubeId) -> ChunkData {
        ChunkData::new()
    }
    fn vertical_range(&self) -> (i64, i64) {
        (0, 0)
    }
}

struct Ids {
    stone: BlockId,
    dirt: BlockId,
    glass: BlockId,
    leaves: BlockId,
}

fn registry() -> (Registry, Ids) {
    let mut r = Registry::new();
    let mut add = |name: &str, opaque: bool, cull_same: bool| {
        r.register(BlockType { name: name.into(), solid: true, opaque, cull_same, ..Default::default() }).unwrap()
    };
    let ids = Ids {
        stone: add("stone", true, false),
        dirt: add("dirt", true, false),
        glass: add("glass", false, true),
        leaves: add("leaves", false, false),
    };
    (r, ids)
}

fn setup() -> (World, Engine, NullCanvas, Registry, Ids) {
    let mut w = World::new(Box::new(Empty));
    let mut e = Engine::new();
    let mut c = NullCanvas::new();
    let (r, ids) = registry();
    e.view_distance = 3;
    e.set_camera(Position::new(8.0, 8.0, 8.0), Position::new(0.0, 0.0, -1.0));
    e.update(&mut w, &mut c, &r, Duration::from_secs(100));
    (w, e, c, r, ids)
}

fn quads_at(w: &World, r: &Registry, x: i64, y: i64, z: i64) -> usize {
    build_quads(w, r, CubeId::block(x, y, z).chunk()).unwrap().len()
}

#[test]
fn face_culling() {
    let (mut w, _, _, r, b) = setup();
    w.set_cube(CubeId::block(3, 3, 3), b.stone).unwrap();
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 6);
    // 两块挨着：中间的两个面省掉
    w.set_cube(CubeId::block(4, 3, 3), b.stone).unwrap();
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 10);
    // 玻璃挨着玻璃省掉，树叶挨着树叶不省
    w.set_cube(CubeId::block(3, 3, 3), b.glass).unwrap();
    w.set_cube(CubeId::block(4, 3, 3), b.glass).unwrap();
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 10);
    w.set_cube(CubeId::block(3, 3, 3), b.leaves).unwrap();
    w.set_cube(CubeId::block(4, 3, 3), b.leaves).unwrap();
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 12);
    // 不透明的石头挨着树叶：石头的面要画，树叶朝石头的面省掉
    w.set_cube(CubeId::block(4, 3, 3), b.stone).unwrap();
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 11);
}

#[test]
fn small_cubes() {
    let (mut w, _, _, r, b) = setup();
    w.set_cube(CubeId::block(3, 3, 3), b.stone).unwrap();
    // 石头上面放一个1/4的小方块：小方块底面被挡住，石头顶面不能省（小方块盖不满）
    let small = CubeId::at(Position::new(3.1, 4.1, 3.1), -2);
    w.set_cube(small, b.dirt).unwrap();
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 6 + 5);
    // 小方块的uv只占贴图的一部分
    for q in build_quads(&w, &r, small.chunk()).unwrap() {
        let c = q.corners[0];
        if c.y > 4.0 && c.y <= 4.25 && c.x >= 3.0 && c.x <= 3.25 {
            for uv in q.uv {
                assert!(!(uv.x > 0.25 + 1e-9 && uv.y > 0.25 + 1e-9 && uv.x < 0.75), "{:?}", q.uv);
            }
        }
    }
    // 4个1/2的小cube把下半格铺满：石头顶面被挡住，每个小cube露出顶面和朝外的两个侧面
    for (dx, dz) in [(0.0, 0.0), (0.5, 0.0), (0.0, 0.5), (0.5, 0.5)] {
        w.set_cube(CubeId::at(Position::new(3.0 + dx, 4.0, 3.0 + dz), -1), b.dirt).unwrap();
    }
    assert_eq!(quads_at(&w, &r, 3, 3, 3), 5 + 4 * 3);
}

#[test]
fn chunk_border_and_culling() {
    let (mut w, mut e, mut c, r, b) = setup();
    // 两个chunk交界处的两块：两边网格都要省掉中间的面
    w.set_cube(CubeId::block(15, 3, 3), b.stone).unwrap();
    w.set_cube(CubeId::block(16, 3, 3), b.stone).unwrap();
    e.update(&mut w, &mut c, &r, Duration::from_secs(100));
    assert_eq!(c.meshes.values().sum::<usize>(), 10);
    // 能看到相机前面的chunk，背后的不画
    e.set_camera(Position::new(16.0, 8.0, 40.0), Position::new(0.0, 0.0, -1.0));
    e.draw_world(&mut c);
    assert_eq!(e.stats.drawn, 2);
    c.drawn.clear();
    e.set_camera(Position::new(8.0, 8.0, -40.0), Position::new(0.0, 0.0, -1.0));
    e.draw_world(&mut c);
    assert_eq!(e.stats.drawn, 0);
}

#[test]
fn unload_keeps_edits() {
    let (mut w, mut e, mut c, r, b) = setup();
    assert!(e.ready(&w, 1));
    w.set_cube(CubeId::block(1, 1, 1), b.stone).unwrap();
    // 走远：原来的chunk被卸载，改动保存在内存里，网格也删掉了
    e.set_camera(Position::new(500.0, 8.0, 8.0), Position::new(0.0, 0.0, -1.0));
    e.update(&mut w, &mut c, &r, Duration::from_secs(100));
    assert!(!w.is_loaded(CubeId::new(0, 0, 0, CHUNK_DEPTH)));
    assert!(w.loaded_count() < 60, "loaded {}", w.loaded_count());
    e.set_camera(Position::new(8.0, 8.0, 8.0), Position::new(0.0, 0.0, -1.0));
    e.update(&mut w, &mut c, &r, Duration::from_secs(100));
    assert_eq!(w.block_at(1, 1, 1), Some((b.stone, false)));
}
