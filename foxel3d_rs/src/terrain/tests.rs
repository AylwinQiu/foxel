use std::collections::HashSet;
use std::rc::Rc;

use super::*;
use crate::world::{Generator as _, World};

const STONE: BlockId = 1;
const BEDROCK: BlockId = 2;
const SNOW: BlockId = 3;
const GRASS: BlockId = 4;
const LOG: BlockId = 5;
const LEAVES: BlockId = 6;

fn biome(name: &str, temperature: f64, humidity: f64, base: f64, variation: f64, ridged: bool) -> Biome {
    Biome {
        name: name.into(),
        temperature,
        humidity,
        base_height: base,
        variation,
        ridged,
        surface: GRASS,
        subsurface: STONE,
        subsurface_depth: 3,
        rock_line: None,
        snow_line: None,
    }
}

/// 和src/game/worldgen.lua一样的5个生物群系
fn core(seed: i64) -> Rc<TerrainCore> {
    let biomes = vec![
        biome("plains", 0.2, -0.1, 12.0, 5.0, false),
        biome("forest", 0.25, 0.5, 15.0, 9.0, false),
        biome("desert", 0.8, -0.6, 10.0, 4.0, false),
        biome("mountains", -0.2, -0.5, 24.0, 70.0, true),
        biome("taiga", -0.7, 0.3, 16.0, 10.0, false),
    ];
    let settings = Settings { stone: STONE, bedrock: BEDROCK, snow: SNOW, min_chunk_y: -2, max_chunk_y: 7 };
    Rc::new(TerrainCore::new(seed, biomes, settings))
}

/// 一棵简单的树：3格树干，顶上3x3的叶子，会伸到隔壁chunk
fn trees() -> Feature {
    let place: PlaceFn = Rc::new(|ctx, x, y, z, _| {
        for dz in -1..=1 {
            for dx in -1..=1 {
                ctx.set_if_air(x + dx, y + 4, z + dz, LEAVES);
            }
        }
        for dy in 1..=3 {
            ctx.set(x, y + dy, z, LOG);
        }
        Ok(())
    });
    let density = vec![0.05; 5];
    Feature {
        name: "trees".into(),
        kind: FeatureKind::Scatter(Scatter { salt: 7, density, soil: vec![GRASS], radius: 1, height: 5, place }),
    }
}

#[test]
fn deterministic() {
    let pos = CubeId::new(3, 0, -2, CHUNK_DEPTH);
    let a = Generator::new(core(42), vec![trees()]).generate_chunk(pos);
    let mut g = Generator::new(core(42), vec![trees()]);
    // 先生成一些别的chunk，结果不能变
    g.generate_chunk(CubeId::new(4, 0, -2, CHUNK_DEPTH));
    let b = g.generate_chunk(pos);
    assert!(a.blocks == b.blocks);
    let c = Generator::new(core(43), vec![trees()]).generate_chunk(pos);
    assert!(a.blocks != c.blocks, "seed ignored");
}

#[test]
fn trees_are_whole_across_chunks() {
    let core = core(7);
    let mut w = World::new(Box::new(Generator::new(core.clone(), vec![trees()])));
    for y in -2..=7 {
        for z in -3..=3 {
            for x in -3..=3 {
                w.load_chunk(CubeId::new(x, y, z, CHUNK_DEPTH));
            }
        }
    }
    // 每个树干上面要么是树干，要么是树叶；跨chunk的树两边要对得上
    let (mut logs, mut leaves) = (0, 0);
    for x in -40..40 {
        for z in -40..40 {
            let h = core.column(x, z).height;
            for y in h + 1..h + 6 {
                match w.block_at(x, y, z).unwrap().0 {
                    LOG => {
                        logs += 1;
                        let above = w.block_at(x, y + 1, z).unwrap().0;
                        assert!(above == LOG || above == LEAVES, "log at {x} {y} {z} has {above} above");
                    }
                    LEAVES => leaves += 1,
                    _ => {}
                }
            }
        }
    }
    assert!(logs > 0 && leaves > logs, "logs {logs} leaves {leaves}");
}

#[test]
fn biomes_and_heights() {
    let core = core(1);
    let mut seen = HashSet::new();
    let (mut lo, mut hi) = (i64::MAX, i64::MIN);
    for x in (-4000..=4000).step_by(40) {
        for z in (-4000..=4000).step_by(40) {
            let c = core.column(x, z);
            seen.insert(c.biome);
            lo = lo.min(c.height);
            hi = hi.max(c.height);
        }
    }
    assert_eq!(seen.len(), 5);
    let s = &core.settings;
    assert!(lo > s.bottom_y() + 5 && hi < (s.max_chunk_y + 1) * CHUNK_SIZE - 25, "height {lo}..{hi}");
}

#[test]
fn ore_replaces_only_stone() {
    let ore = Feature {
        name: "ore".into(),
        kind: FeatureKind::Ore(Ore {
            salt: 1,
            block: SNOW,
            replace: STONE,
            per_chunk: 20,
            size: 10,
            min_y: -32,
            max_y: 0,
        }),
    };
    let mut g = Generator::new(core(3), vec![ore]);
    let d = g.generate_chunk(CubeId::new(0, -1, 0, CHUNK_DEPTH));
    assert!(d.blocks.iter().filter(|&&b| b == SNOW).count() > 20);
    // 基岩那一层不会被换掉
    let d = g.generate_chunk(CubeId::new(0, -2, 0, CHUNK_DEPTH));
    for z in 0..16 {
        for x in 0..16 {
            assert_eq!(d.get(x, 0, z), BEDROCK);
        }
    }
}
