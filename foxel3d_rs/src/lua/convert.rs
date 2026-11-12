//! Lua值和Rust类型之间的转换。
//!
//! - 颜色：{r, g, b}或{r, g, b, a}（0到255），也可以写成{r=, g=, b=, a=}
//! - 位置、方向：{x=, y=, z=}，也可以写成{x, y, z}
//! - cube：{x=, y=, z=, depth=}，depth省略时是0
//! - 方块：BlockId（整数）或者名字

use mlua::{Lua, Result, Table, Value};

use crate::block::Registry;
use crate::canvas::{Color, Image, Images};
use crate::world::{BlockId, CubeId, Position};

use super::Shared;

pub fn error<T>(msg: impl std::fmt::Display) -> Result<T> {
    Err(mlua::Error::runtime(msg))
}

fn table<'a>(v: &'a Value, what: &str) -> Result<&'a Table> {
    match v {
        Value::Table(t) => Ok(t),
        other => error(format!("{what}应该是table，不是{}", other.type_name())),
    }
}

/// t[named]，没有时取t[index]
fn field<T: mlua::FromLua>(t: &Table, named: &str, index: i64) -> Result<Option<T>> {
    match t.get::<Option<T>>(named)? {
        Some(v) => Ok(Some(v)),
        None => t.get(index),
    }
}

pub fn color(v: &Value) -> Result<Color> {
    let t = table(v, "颜色")?;
    let c = |name: &str, i: i64, default: Option<f64>| -> Result<u8> {
        match field::<f64>(t, name, i)?.or(default) {
            Some(x) => Ok(x.round().clamp(0.0, 255.0) as u8),
            None => error(format!("颜色缺少{name}")),
        }
    };
    Ok(Color::rgba(c("r", 1, None)?, c("g", 2, None)?, c("b", 3, None)?, c("a", 4, Some(255.0))?))
}

pub fn color_lua(lua: &Lua, c: Color) -> Result<Table> {
    lua.create_sequence_from([c.r, c.g, c.b, c.a])
}

pub fn pos(v: &Value) -> Result<Position> {
    let t = table(v, "位置")?;
    let c = |name: &str, i: i64| -> Result<f64> {
        field::<f64>(t, name, i)?.ok_or_else(|| mlua::Error::runtime(format!("位置缺少{name}")))
    };
    Ok(Position::new(c("x", 1)?, c("y", 2)?, c("z", 3)?))
}

pub fn pos_lua(lua: &Lua, p: Position) -> Result<Table> {
    let t = lua.create_table_with_capacity(0, 3)?;
    t.set("x", p.x)?;
    t.set("y", p.y)?;
    t.set("z", p.z)?;
    Ok(t)
}

pub fn cube(v: &Value) -> Result<CubeId> {
    let t = table(v, "cube")?;
    let c = |name: &str| -> Result<i64> {
        t.get::<Option<i64>>(name)?.ok_or_else(|| mlua::Error::runtime(format!("cube缺少{name}（要是整数）")))
    };
    Ok(CubeId::new(c("x")?, c("y")?, c("z")?, t.get::<Option<i32>>("depth")?.unwrap_or(0)))
}

pub fn cube_lua(lua: &Lua, id: CubeId) -> Result<Table> {
    let t = lua.create_table_with_capacity(0, 4)?;
    t.set("x", id.x)?;
    t.set("y", id.y)?;
    t.set("z", id.z)?;
    t.set("depth", id.depth)?;
    Ok(t)
}

pub fn block_id(reg: &Registry, v: &Value) -> Result<BlockId> {
    match v {
        Value::Integer(i) if *i >= 0 && (*i as usize) < reg.len() => Ok(*i as BlockId),
        Value::Number(n) if n.fract() == 0.0 && *n >= 0.0 && (*n as usize) < reg.len() => Ok(*n as BlockId),
        Value::String(s) => {
            let name = s.to_str()?;
            reg.by_name(&name).ok_or_else(|| mlua::Error::runtime(format!("没有叫{name}的方块")))
        }
        other => error(format!("不是方块：{other:?}")),
    }
}

/// 一张图片：Image、颜色、或者png的路径（相对游戏根目录）
pub fn image(s: &Shared, v: &Value) -> Result<Image> {
    match v {
        Value::UserData(ud) => Ok(ud.borrow::<Image>()?.clone()),
        Value::String(p) => Ok(Image::load(&s.path(&p.to_str()?))),
        Value::Table(_) => Ok(Image::solid(color(v)?)),
        other => error(format!("不是图片：{}", other.type_name())),
    }
}

/// 6个面的图片。可以是一张图片（6个面都用它），也可以是
/// {all=, side=, top=, bottom=, up=, down=, north=, south=, east=, west=}，后面的覆盖前面的
pub fn images(s: &Shared, v: &Value) -> Result<Images> {
    let Value::Table(t) = v else {
        return Ok(Images::all(image(s, v)?));
    };
    // 颜色也是table：有t[1]就是颜色
    if t.contains_key(1)? {
        return Ok(Images::all(image(s, v)?));
    }
    let mut out = Images::default();
    let get = |name: &str| -> Result<Option<Image>> {
        match t.get::<Value>(name)? {
            Value::Nil => Ok(None),
            v => image(s, &v).map(Some),
        }
    };
    if let Some(img) = get("all")? {
        out = Images::all(img);
    }
    if let Some(img) = get("side")? {
        out.north = Some(img.clone());
        out.south = Some(img.clone());
        out.west = Some(img.clone());
        out.east = Some(img);
    }
    for (names, slot) in [
        (["top", "up"], &mut out.up),
        (["bottom", "down"], &mut out.down),
        (["north", "north"], &mut out.north),
        (["south", "south"], &mut out.south),
        (["west", "west"], &mut out.west),
        (["east", "east"], &mut out.east),
    ] {
        for n in names {
            if let Some(img) = get(n)? {
                *slot = Some(img);
            }
        }
    }
    Ok(out)
}
