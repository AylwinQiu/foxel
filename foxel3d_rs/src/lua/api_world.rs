//! world、cube模块：打开和保存世界、读写方块、射线、碰撞、场

use std::rc::Rc;

use mlua::{Function, Lua, LuaSerdeExt, Result, Table, Value};

use super::convert::{self, cube, cube_lua, error, pos, pos_lua};
use super::{OpenWorld, Shared, add};
use crate::save::{self, SaveData};
use crate::terrain::Generator;
use crate::world::{AIR, BlockId, CHUNK_DEPTH, CHUNK_SIZE, CubeId, MIN_DEPTH, World, side};

/// 打开一个世界：有存档时读存档，否则用seed生成新世界。返回存档里Lua的meta，新世界时是nil
fn open(lua: &Lua, s: &Shared, opts: Table) -> Result<Value> {
    if s.world.borrow().is_some() {
        return error("世界已经打开了");
    }
    let name: String = opts.get::<Option<String>>("save")?.unwrap_or_else(|| "default".into());
    if name.is_empty() || name.contains(['/', '\\']) || name.starts_with('.') {
        return error(format!("存档名字不对：{name}"));
    }
    let dir = s.saves.join(&name);
    let data = save::read(&dir).map_err(|e| mlua::Error::runtime(format!("读存档{}失败：{e}", dir.display())))?;
    let seed = match &data {
        Some(d) => d.seed,
        None => opts.get::<Option<i64>>("seed")?.unwrap_or_else(|| {
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_nanos() as i64 % 1_000_000)
                .unwrap_or(0)
        }),
    };

    s.blocks.borrow_mut().freeze();
    let reg = s.blocks()?;
    let (core, features) = s.terrain.borrow().build(lua, &reg, seed).map_err(mlua::Error::runtime)?;
    let core = Rc::new(core);
    let mut world = World::new(Box::new(Generator::new(core.clone(), features)));

    let mut meta = Value::Nil;
    if let Some(d) = data {
        let ids: Vec<BlockId> = d
            .palette
            .iter()
            .map(|n| {
                reg.by_name(n).unwrap_or_else(|| {
                    eprintln!("foxel: 存档里的方块{n}没有注册（mod被删掉了？），变成空气");
                    AIR
                })
            })
            .collect();
        let dropped = world.set_saved_chunks(d.chunks, &|b| ids.get(b as usize).copied().unwrap_or(AIR));
        if dropped > 0 {
            eprintln!("foxel: 存档里有{dropped}个chunk坏了，重新生成");
        }
        world.set_fields(d.fields);
        meta = lua.to_value(&d.meta)?;
    }
    drop(reg);
    s.engine()?.reset(s.canvas()?.as_mut());
    *s.world.borrow_mut() = Some(OpenWorld { world, core, save_dir: dir, seed });
    Ok(meta)
}

fn write_save(lua: &Lua, s: &Shared, meta: Value) -> Result<()> {
    let w = s.world_ref()?;
    let reg = s.blocks()?;
    let data = SaveData {
        seed: w.seed,
        palette: (0..reg.len()).map(|i| reg.get(i as BlockId).name.clone()).collect(),
        chunks: w.world.saved_chunks(),
        fields: w.world.fields().clone(),
        meta: lua.from_value(meta)?,
    };
    save::write(&w.save_dir, &data).map_err(|e| mlua::Error::runtime(format!("存档失败：{e}")))
}

/// 每种方块是不是"打得到"：nil时是不是空气，否则调用filter(BlockId)。
/// 先对所有方块算好，射线检测时world被借用着，不能再回调Lua
fn hit_table(s: &Shared, filter: Option<Function>) -> Result<Vec<bool>> {
    let n = s.blocks()?.len();
    (0..n)
        .map(|b| match &filter {
            None => Ok(b != AIR as usize),
            Some(f) => f.call::<bool>(b),
        })
        .collect()
}

fn solid_table(s: &Shared) -> Result<Vec<bool>> {
    let reg = s.blocks()?;
    Ok((0..reg.len()).map(|b| reg.is_solid(b as BlockId)).collect())
}

pub fn install(lua: &Lua, s: &Rc<Shared>) -> Result<()> {
    let w = lua.create_table()?;

    let sh = s.clone();
    add(lua, &w, "open", move |lua, opts: Option<Table>| {
        let opts = match opts {
            Some(t) => t,
            None => lua.create_table()?,
        };
        open(lua, &sh, opts)
    })?;

    // 存档。meta是Lua自己的数据（玩家、物品栏……），能转成JSON的值都行
    let sh = s.clone();
    add(lua, &w, "save", move |lua, meta: Value| write_save(lua, &sh, meta))?;

    let sh = s.clone();
    add(lua, &w, "is_open", move |_, ()| Ok(sh.world.try_borrow().map(|w| w.is_some()).unwrap_or(true)))?;

    let sh = s.clone();
    add(lua, &w, "seed", move |_, ()| Ok(sh.world_ref()?.seed))?;

    // 方块(x, y, z)：返回BlockId和它是不是被分成了更小的cube（这时BlockId没有意义）。没加载时是nil
    let sh = s.clone();
    add(lua, &w, "get", move |_, (x, y, z): (i64, i64, i64)| {
        Ok(match sh.world_ref()?.world.block_at(x, y, z) {
            Some((b, split)) => (Some(b), Some(split)),
            None => (None, None),
        })
    })?;

    // 把方块(x, y, z)设成b。成功返回true，失败返回nil和原因
    let sh = s.clone();
    add(lua, &w, "set", move |_, (x, y, z, b): (i64, i64, i64, Value)| {
        let b = convert::block_id(&*sh.blocks()?, &b)?;
        Ok(match sh.world()?.world.set_cube(CubeId::block(x, y, z), b) {
            Ok(()) => (true, None),
            Err(e) => (false, Some(e.to_string())),
        })
    })?;

    // 把cube整个设成b（AIR就是挖掉），里面原来的东西和场都删掉
    let sh = s.clone();
    add(lua, &w, "set_cube", move |_, (c, b): (Value, Value)| {
        let b = convert::block_id(&*sh.blocks()?, &b)?;
        Ok(match sh.world()?.world.set_cube(cube(&c)?, b) {
            Ok(()) => (true, None),
            Err(e) => (false, Some(e.to_string())),
        })
    })?;

    // 找到cube：返回方块、真正的叶子cube（cube在更大的叶子里时是那个叶子）、是不是被分开了。没加载时是nil
    let sh = s.clone();
    add(lua, &w, "find", move |lua, c: Value| {
        let id = cube(&c)?;
        if id.depth < MIN_DEPTH || id.depth > CHUNK_DEPTH {
            return error(format!("cube的深度要在{MIN_DEPTH}到{CHUNK_DEPTH}之间"));
        }
        let w = sh.world_ref()?;
        Ok(match w.world.find(id) {
            Some((n, at)) => (Some(n.block), Some(cube_lua(lua, at)?), Some(!n.is_leaf())),
            None => (None, None, None),
        })
    })?;

    // 包含位置p的叶子cube：返回方块和cube。没加载时是nil
    let sh = s.clone();
    add(lua, &w, "leaf_at", move |lua, p: Value| {
        let w = sh.world_ref()?;
        Ok(match w.world.leaf_at(pos(&p)?) {
            Some(l) => (Some(l.block), Some(cube_lua(lua, l.id)?)),
            None => (None, None),
        })
    })?;

    // 射线：从origin沿dir最远max_distance，找第一个filter(方块)为true的cube（filter不写时是不是空气）。
    // 返回{block, cube, point, normal, distance}，没打到时是nil
    let sh = s.clone();
    add(lua, &w, "raycast", move |lua, (o, d, max, filter): (Value, Value, f64, Option<Function>)| {
        let hits = hit_table(&sh, filter)?;
        let w = sh.world_ref()?;
        let Some(h) = w.world.raycast(pos(&o)?, pos(&d)?, max, &|b| hits.get(b as usize).copied().unwrap_or(false))
        else {
            return Ok(Value::Nil);
        };
        let t = lua.create_table()?;
        t.set("block", h.leaf.block)?;
        t.set("cube", cube_lua(lua, h.leaf.id)?)?;
        t.set("point", pos_lua(lua, h.point)?)?;
        let n = lua.create_table()?;
        n.set("x", h.normal[0])?;
        n.set("y", h.normal[1])?;
        n.set("z", h.normal[2])?;
        t.set("normal", n)?;
        t.set("distance", h.distance)?;
        Ok(Value::Table(t))
    })?;

    // lo到hi的长方体有没有碰到实心方块（没加载的地方算实心）
    let sh = s.clone();
    add(lua, &w, "box_collides", move |_, (lo, hi): (Value, Value)| {
        let solid = solid_table(&sh)?;
        let w = sh.world_ref()?;
        Ok(w.world.box_collides(pos(&lo)?, pos(&hi)?, &|b| solid.get(b as usize).copied().unwrap_or(true)))
    })?;

    // 移动一个长方体（底面中心pos，宽width，高height）delta，碰到实心方块就停下。
    // step大于0时，站在地上走路碰到不超过step高的台阶会迈上去。返回新的位置和{x=, y=, z=}每个轴有没有被挡住
    let sh = s.clone();
    add(lua, &w, "move_box", move |lua, (p, width, height, delta, step): (Value, f64, f64, Value, Option<f64>)| {
        let solid = solid_table(&sh)?;
        let w = sh.world_ref()?;
        let (np, hit) = w.world.move_box(pos(&p)?, width, height, pos(&delta)?, step.unwrap_or(0.0), &|b| {
            solid.get(b as usize).copied().unwrap_or(true)
        });
        let h = lua.create_table()?;
        h.set("x", hit[0])?;
        h.set("y", hit[1])?;
        h.set("z", hit[2])?;
        Ok((pos_lua(lua, np)?, h))
    })?;

    // 场：给cube附加的字符串数据，跟着存档保存，cube被覆盖时删掉
    let sh = s.clone();
    add(lua, &w, "field_write", move |_, (name, c, v): (String, Value, String)| {
        sh.world()?.world.field_write(&name, cube(&c)?, v);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &w, "field_read", move |_, (name, c): (String, Value)| {
        Ok(sh.world_ref()?.world.field_read(&name, cube(&c)?).map(str::to_string))
    })?;
    let sh = s.clone();
    add(lua, &w, "field_delete", move |_, (name, c): (String, Value)| {
        sh.world()?.world.field_delete(&name, cube(&c)?);
        Ok(())
    })?;
    // 一个场里的所有数据：{{cube=, value=}, ...}
    let sh = s.clone();
    add(lua, &w, "field_all", move |lua, name: String| {
        let w = sh.world_ref()?;
        let out = lua.create_table()?;
        for (id, v) in w.world.field_all(&name) {
            let e = lua.create_table()?;
            e.set("cube", cube_lua(lua, id)?)?;
            e.set("value", v)?;
            out.push(e)?;
        }
        Ok(out)
    })?;

    // 已加载的chunk数
    let sh = s.clone();
    add(lua, &w, "loaded_chunks", move |_, ()| Ok(sh.world_ref()?.world.loaded_count()))?;

    lua.globals().set("world", w)?;

    let c = lua.create_table()?;
    c.set("MIN_DEPTH", MIN_DEPTH)?;
    c.set("CHUNK_DEPTH", CHUNK_DEPTH)?;
    c.set("CHUNK_SIZE", CHUNK_SIZE)?;
    // 包含位置p的、深度为depth（默认0）的cube
    add(lua, &c, "at", |lua, (p, d): (Value, Option<i32>)| cube_lua(lua, CubeId::at(pos(&p)?, d.unwrap_or(0))))?;
    // 方块(x, y, z)的cube
    add(lua, &c, "block", |lua, (x, y, z): (i64, i64, i64)| cube_lua(lua, CubeId::block(x, y, z)))?;
    add(lua, &c, "min", |lua, v: Value| pos_lua(lua, cube(&v)?.min()))?;
    add(lua, &c, "max", |lua, v: Value| pos_lua(lua, cube(&v)?.max()))?;
    add(lua, &c, "center", |lua, v: Value| pos_lua(lua, cube(&v)?.center()))?;
    // 边长。可以传cube，也可以直接传深度
    add(lua, &c, "side", |_, v: Value| match v {
        Value::Integer(d) => Ok(side(d as i32)),
        v => Ok(cube(&v)?.side()),
    })?;
    // 包含它的、深度为depth的cube
    add(lua, &c, "parent", |lua, (v, d): (Value, i32)| {
        let id = cube(&v)?;
        if d < id.depth {
            return error("parent的深度不能比cube小");
        }
        cube_lua(lua, id.parent_at(d))
    })?;
    // 第o个子cube（0到7），o是卦限：x正为1，y正为2，z正为4
    add(lua, &c, "child", |lua, (v, o): (Value, usize)| cube_lua(lua, cube(&v)?.child(o & 7)))?;
    // b在不在a里面（包括b就是a）
    add(lua, &c, "contains", |_, (a, b): (Value, Value)| Ok(cube(&a)?.contains(cube(&b)?)))?;
    add(lua, &c, "chunk", |lua, v: Value| cube_lua(lua, cube(&v)?.chunk()))?;
    // 同样大小、挨着的cube，dx、dy、dz是格子数
    add(lua, &c, "neighbor", |lua, (v, dx, dy, dz): (Value, i64, i64, i64)| {
        cube_lua(lua, cube(&v)?.neighbor(dx, dy, dz))
    })?;
    add(lua, &c, "equal", |_, (a, b): (Value, Value)| Ok(cube(&a)? == cube(&b)?))?;
    // 可以当table的key用的字符串，比如"3,60,-2@0"
    add(lua, &c, "key", |_, v: Value| {
        let id = cube(&v)?;
        Ok(format!("{},{},{}@{}", id.x, id.y, id.z, id.depth))
    })?;
    lua.globals().set("cube", c)
}
