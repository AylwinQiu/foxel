//! terrain模块：在世界打开之前登记生物群系和feature（方块用名字），world.open时换成BlockId交给terrain。
//! 同名的生物群系、feature再登记一次时只改给出的字段，mod可以这样调整内置的地形。

use std::collections::HashMap;
use std::rc::Rc;

use mlua::{Function, Lua, Result, Table, UserData, UserDataMethods, Value};

use super::convert::{self, error};
use super::{Shared, add};
use crate::block::Registry;
use crate::rng::Rng;
use crate::terrain::{Biome, Context, Feature, FeatureKind, Ore, Scatter, Settings, TerrainCore};
use crate::world::{BlockId, CubeId};

#[derive(Clone)]
struct BiomeDef {
    name: String,
    temperature: f64,
    humidity: f64,
    base_height: f64,
    variation: f64,
    ridged: bool,
    surface: String,
    subsurface: String,
    subsurface_depth: i64,
    rock_line: Option<i64>,
    snow_line: Option<i64>,
}

#[derive(Clone)]
enum FeatureDef {
    Ore { salt: u64, block: String, replace: String, per_chunk: u32, size: u32, min_y: i64, max_y: i64 },
    Scatter { salt: u64, density: HashMap<String, f64>, soil: Vec<String>, radius: i64, height: i64, place: Function },
    Custom(Function),
}

#[derive(Clone)]
struct SettingsDef {
    stone: String,
    bedrock: String,
    snow: String,
    min_chunk_y: i64,
    max_chunk_y: i64,
}

/// world.open之前登记的地形
#[derive(Clone)]
pub struct TerrainDefs {
    biomes: Vec<BiomeDef>,
    features: Vec<(String, FeatureDef)>,
    settings: SettingsDef,
}

impl Default for TerrainDefs {
    fn default() -> Self {
        TerrainDefs {
            biomes: Vec::new(),
            features: Vec::new(),
            settings: SettingsDef {
                stone: "foxel:stone".into(),
                bedrock: "foxel:bedrock".into(),
                snow: "foxel:snow".into(),
                min_chunk_y: -2,
                max_chunk_y: 7,
            },
        }
    }
}

fn resolve(reg: &Registry, name: &str, what: &str) -> std::result::Result<BlockId, String> {
    reg.by_name(name).ok_or_else(|| format!("{what}用到的方块{name}没有注册"))
}

impl TerrainDefs {
    /// 换成terrain用的数据：方块名字换成BlockId，Lua函数包成Rust闭包
    pub fn build(
        &self,
        lua: &Lua,
        reg: &Registry,
        seed: i64,
    ) -> std::result::Result<(TerrainCore, Vec<Feature>), String> {
        if self.biomes.is_empty() {
            return Err("一个生物群系都没有，先用terrain.biome登记".into());
        }
        let s = &self.settings;
        let settings = Settings {
            stone: resolve(reg, &s.stone, "terrain.settings")?,
            bedrock: resolve(reg, &s.bedrock, "terrain.settings")?,
            snow: resolve(reg, &s.snow, "terrain.settings")?,
            min_chunk_y: s.min_chunk_y,
            max_chunk_y: s.max_chunk_y,
        };
        let mut biomes = Vec::new();
        for b in &self.biomes {
            let what = format!("生物群系{}", b.name);
            biomes.push(Biome {
                name: b.name.clone(),
                temperature: b.temperature,
                humidity: b.humidity,
                base_height: b.base_height,
                variation: b.variation,
                ridged: b.ridged,
                surface: resolve(reg, &b.surface, &what)?,
                subsurface: resolve(reg, &b.subsurface, &what)?,
                subsurface_depth: b.subsurface_depth,
                rock_line: b.rock_line,
                snow_line: b.snow_line,
            });
        }
        let mut features = Vec::new();
        for (name, def) in &self.features {
            let what = format!("feature {name}");
            let kind = match def {
                FeatureDef::Ore { salt, block, replace, per_chunk, size, min_y, max_y } => FeatureKind::Ore(Ore {
                    salt: *salt,
                    block: resolve(reg, block, &what)?,
                    replace: resolve(reg, replace, &what)?,
                    per_chunk: *per_chunk,
                    size: *size,
                    min_y: *min_y,
                    max_y: *max_y,
                }),
                FeatureDef::Scatter { salt, density, soil, radius, height, place } => {
                    let soil = soil.iter().map(|n| resolve(reg, n, &what)).collect::<std::result::Result<_, _>>()?;
                    let density = biomes.iter().map(|b| density.get(&b.name).copied().unwrap_or(0.0)).collect();
                    let (weak, f) = (lua.weak(), place.clone());
                    FeatureKind::Scatter(Scatter {
                        salt: *salt,
                        density,
                        soil,
                        radius: *radius,
                        height: *height,
                        place: Rc::new(move |ctx: &mut Context, x, y, z, r: &mut Rng| {
                            let lua = weak.try_upgrade().ok_or("Lua已经关闭")?;
                            lua.scope(|scope| {
                                let c = scope.create_userdata_ref_mut(ctx)?;
                                let r = scope.create_userdata_ref_mut(r)?;
                                f.call::<()>((c, x, y, z, r))
                            })
                            .map_err(|e| e.to_string())
                        }),
                    })
                }
                FeatureDef::Custom(func) => {
                    let (weak, f) = (lua.weak(), func.clone());
                    FeatureKind::Custom(Rc::new(move |ctx: &mut Context| {
                        let lua = weak.try_upgrade().ok_or("Lua已经关闭")?;
                        lua.scope(|scope| f.call::<()>(scope.create_userdata_ref_mut(ctx)?)).map_err(|e| e.to_string())
                    }))
                }
            };
            features.push(Feature { name: name.clone(), kind });
        }
        Ok((TerrainCore::new(seed, biomes, settings), features))
    }
}

/// def[k]，没有时用old
fn get_or<T: mlua::FromLua>(def: &Table, k: &str, old: Option<T>) -> Result<T> {
    match def.get::<Option<T>>(k)? {
        Some(v) => Ok(v),
        None => old.ok_or_else(|| mlua::Error::runtime(format!("缺少{k}"))),
    }
}

/// 方块参数：名字或者BlockId都行，存成名字
fn block_name(s: &Shared, def: &Table, k: &str, old: Option<String>) -> Result<String> {
    match def.get::<Value>(k)? {
        Value::Nil => old.ok_or_else(|| mlua::Error::runtime(format!("缺少{k}"))),
        v => {
            let reg = s.blocks()?;
            let id = convert::block_id(&reg, &v)?;
            Ok(reg.get(id).name.clone())
        }
    }
}

fn check_open(s: &Shared) -> Result<()> {
    if s.world.try_borrow().map(|w| w.is_some()).unwrap_or(true) {
        return error("世界已经打开，地形不能再改了");
    }
    Ok(())
}

/// 同名的feature替换掉（保持原来的位置），没有时加到最后
fn put_feature(s: &Shared, name: String, def: FeatureDef) {
    let mut t = s.terrain.borrow_mut();
    match t.features.iter_mut().find(|(n, _)| *n == name) {
        Some(slot) => slot.1 = def,
        None => t.features.push((name, def)),
    }
}

fn old_feature(s: &Shared, name: &str) -> Option<FeatureDef> {
    s.terrain.borrow().features.iter().find(|(n, _)| n == name).map(|(_, d)| d.clone())
}

impl UserData for Context {
    fn add_methods<M: UserDataMethods<Self>>(m: &mut M) {
        let block = |lua: &Lua, v: &Value| -> Result<BlockId> {
            if let Value::Integer(i) = v {
                return Ok(*i as BlockId);
            }
            let s = lua.app_data_ref::<Rc<Shared>>().unwrap();
            convert::block_id(&*s.blocks()?, v)
        };
        // chunk最小的角的方块坐标
        m.add_method("origin", |_, c, ()| Ok(c.origin()));
        // chunk的cube
        m.add_method("chunk", |lua, c, ()| convert::cube_lua(lua, c.chunk));
        // 世界的种子
        m.add_method("seed", |_, c, ()| Ok(c.core.seed));
        // 方块坐标(x, y, z)是什么，不在这个chunk里时是nil
        m.add_method("get", |_, c, (x, y, z): (i64, i64, i64)| Ok(c.get(x, y, z)));
        // 设成方块b，不在这个chunk里时什么都不做
        m.add_method_mut("set", move |lua, c, (x, y, z, b): (i64, i64, i64, Value)| {
            c.set(x, y, z, block(lua, &b)?);
            Ok(())
        });
        // 只有原来是空气时才设
        m.add_method_mut("set_if_air", move |lua, c, (x, y, z, b): (i64, i64, i64, Value)| {
            c.set_if_air(x, y, z, block(lua, &b)?);
            Ok(())
        });
        // 放一个比1小的cube：深度为depth的格子(x, y, z)
        m.add_method_mut("set_small", move |lua, c, (x, y, z, depth, b): (i64, i64, i64, i32, Value)| {
            c.set_small(CubeId::new(x, y, z, depth), block(lua, &b)?);
            Ok(())
        });
        // 这一列地面的高度和生物群系的名字
        m.add_method("column", |_, c, (x, z): (i64, i64)| {
            let col = c.column(x, z);
            Ok((col.height, c.core.biome(col).name.clone()))
        });
        // 这一列地面最上面的方块（不考虑feature）
        m.add_method("ground", |_, c, (x, z): (i64, i64)| Ok(c.ground(x, z)));
        // 确定性的随机数：同一个世界、同一个位置、同一个salt总是一样的
        m.add_method("rand", |_, c, (x, y, z, salt): (i64, i64, i64, i64)| Ok(c.rand(x, y, z, salt as u64)));
    }
}

pub fn install(lua: &Lua, s: &Rc<Shared>) -> Result<()> {
    let t = lua.create_table()?;

    // 世界共用的设置：stone、bedrock、snow（方块），min_chunk_y、max_chunk_y
    let sh = s.clone();
    add(lua, &t, "settings", move |_, def: Table| {
        check_open(&sh)?;
        let old = sh.terrain.borrow().settings.clone();
        let new = SettingsDef {
            stone: block_name(&sh, &def, "stone", Some(old.stone))?,
            bedrock: block_name(&sh, &def, "bedrock", Some(old.bedrock))?,
            snow: block_name(&sh, &def, "snow", Some(old.snow))?,
            min_chunk_y: get_or(&def, "min_chunk_y", Some(old.min_chunk_y))?,
            max_chunk_y: get_or(&def, "max_chunk_y", Some(old.max_chunk_y))?,
        };
        if new.min_chunk_y > new.max_chunk_y {
            return error("min_chunk_y不能比max_chunk_y大");
        }
        sh.terrain.borrow_mut().settings = new;
        Ok(())
    })?;

    // 登记生物群系；同名的已经有了时只改给出的字段
    let sh = s.clone();
    add(lua, &t, "biome", move |_, def: Table| {
        check_open(&sh)?;
        let name: String = get_or(&def, "name", None)?;
        let old = sh.terrain.borrow().biomes.iter().find(|b| b.name == name).cloned();
        let o = old.as_ref();
        let b = BiomeDef {
            temperature: get_or(&def, "temperature", o.map(|b| b.temperature).or(Some(0.0)))?,
            humidity: get_or(&def, "humidity", o.map(|b| b.humidity).or(Some(0.0)))?,
            base_height: get_or(&def, "base_height", o.map(|b| b.base_height).or(Some(12.0)))?,
            variation: get_or(&def, "variation", o.map(|b| b.variation).or(Some(5.0)))?,
            ridged: get_or(&def, "ridged", o.map(|b| b.ridged).or(Some(false)))?,
            surface: block_name(&sh, &def, "surface", o.map(|b| b.surface.clone()))?,
            subsurface: block_name(&sh, &def, "subsurface", o.map(|b| b.subsurface.clone()))?,
            subsurface_depth: get_or(&def, "subsurface_depth", o.map(|b| b.subsurface_depth).or(Some(4)))?,
            rock_line: def.get::<Option<i64>>("rock_line")?.or(o.and_then(|b| b.rock_line)),
            snow_line: def.get::<Option<i64>>("snow_line")?.or(o.and_then(|b| b.snow_line)),
            name: name.clone(),
        };
        let mut terrain = sh.terrain.borrow_mut();
        match terrain.biomes.iter_mut().find(|x| x.name == name) {
            Some(slot) => *slot = b,
            None => terrain.biomes.push(b),
        }
        Ok(())
    })?;

    // 矿石：每个chunk放per_chunk条矿脉，每条size个方块，只替换replace（默认石头）
    let sh = s.clone();
    add(lua, &t, "ore", move |_, def: Table| {
        check_open(&sh)?;
        let name: String = get_or(&def, "name", None)?;
        let old = match old_feature(&sh, &name) {
            Some(FeatureDef::Ore { salt, block, replace, per_chunk, size, min_y, max_y }) => {
                Some((salt, block, replace, per_chunk, size, min_y, max_y))
            }
            _ => None,
        };
        let o = old.as_ref();
        let stone = sh.terrain.borrow().settings.stone.clone();
        let f = FeatureDef::Ore {
            salt: get_or(&def, "salt", o.map(|o| o.0))?,
            block: block_name(&sh, &def, "block", o.map(|o| o.1.clone()))?,
            replace: block_name(&sh, &def, "replace", o.map(|o| o.2.clone()).or(Some(stone)))?,
            per_chunk: get_or(&def, "per_chunk", o.map(|o| o.3).or(Some(4)))?,
            size: get_or(&def, "size", o.map(|o| o.4).or(Some(6)))?,
            min_y: get_or(&def, "min_y", o.map(|o| o.5).or(Some(i64::MIN)))?,
            max_y: get_or(&def, "max_y", o.map(|o| o.6).or(Some(i64::MAX)))?,
        };
        put_feature(&sh, name, f);
        Ok(())
    })?;

    // 散布：树、花这种在地面上放东西的feature。
    // density = {[生物群系名] = 每列的概率}，soil = {能放在哪些方块上}（不写就是哪里都行），
    // radius、height是东西最多伸出多远、多高，place(ctx, x, y, z, rng)负责放，y是地面方块的高度。
    // 再登记一次同名的时，density按生物群系合并
    let sh = s.clone();
    add(lua, &t, "scatter", move |_, def: Table| {
        check_open(&sh)?;
        let name: String = get_or(&def, "name", None)?;
        let old = match old_feature(&sh, &name) {
            Some(FeatureDef::Scatter { salt, density, soil, radius, height, place }) => {
                Some((salt, density, soil, radius, height, place))
            }
            _ => None,
        };
        let o = old.as_ref();
        let mut density = o.map(|o| o.1.clone()).unwrap_or_default();
        if let Some(d) = def.get::<Option<HashMap<String, f64>>>("density")? {
            density.extend(d);
        }
        let soil = match def.get::<Option<Vec<Value>>>("soil")? {
            Some(list) => {
                let reg = sh.blocks()?;
                list.iter().map(|v| Ok(reg.get(convert::block_id(&reg, v)?).name.clone())).collect::<Result<_>>()?
            }
            None => o.map(|o| o.2.clone()).unwrap_or_default(),
        };
        let f = FeatureDef::Scatter {
            salt: get_or(&def, "salt", o.map(|o| o.0))?,
            density,
            soil,
            radius: get_or(&def, "radius", o.map(|o| o.3).or(Some(0)))?,
            height: get_or(&def, "height", o.map(|o| o.4).or(Some(16)))?,
            place: get_or(&def, "place", o.map(|o| o.5.clone()))?,
        };
        put_feature(&sh, name, f);
        Ok(())
    })?;

    // 自定义feature：generate(ctx)每个chunk调用一次。比scatter慢，能用scatter时尽量用scatter
    let sh = s.clone();
    add(lua, &t, "feature", move |_, def: Table| {
        check_open(&sh)?;
        let name: String = get_or(&def, "name", None)?;
        put_feature(&sh, name, FeatureDef::Custom(get_or(&def, "generate", None)?));
        Ok(())
    })?;

    // 删掉一个生物群系或feature
    let sh = s.clone();
    add(lua, &t, "remove", move |_, name: String| {
        check_open(&sh)?;
        let mut t = sh.terrain.borrow_mut();
        t.biomes.retain(|b| b.name != name);
        t.features.retain(|(n, _)| *n != name);
        Ok(())
    })?;

    // 所有生物群系和feature的名字
    let sh = s.clone();
    add(lua, &t, "biomes", move |lua, ()| {
        lua.create_sequence_from(sh.terrain.borrow().biomes.iter().map(|b| b.name.clone()))
    })?;
    let sh = s.clone();
    add(lua, &t, "features", move |lua, ()| {
        lua.create_sequence_from(sh.terrain.borrow().features.iter().map(|(n, _)| n.clone()))
    })?;

    // 世界打开以后：(x, z)这一列地面的高度和生物群系的名字
    let sh = s.clone();
    add(lua, &t, "column", move |_, (x, z): (i64, i64)| {
        let w = sh.world_ref()?;
        let c = w.core.column(x, z);
        Ok((c.height, w.core.biome(c).name.clone()))
    })?;

    lua.globals().set("terrain", t)
}
