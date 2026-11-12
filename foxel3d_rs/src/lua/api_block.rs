//! block、image模块，以及Image、Rng、Perlin这几种userdata

use std::collections::HashMap;
use std::rc::Rc;

use mlua::{Lua, MultiValue, Result, Table, UserData, UserDataMethods, Value};

use super::convert::{self, color_lua, error};
use super::{Shared, add};
use crate::block::BlockType;
use crate::canvas::{Color, Image};
use crate::rng::Rng;
use crate::terrain::Perlin;
use crate::world::{AIR, BlockId};

impl UserData for Image {
    fn add_methods<M: UserDataMethods<Self>>(m: &mut M) {
        // 宽和高（像素）
        m.add_method("size", |_, img, ()| Ok(img.size()));
        // 像素(x, y)的颜色，(0, 0)是左上角
        m.add_method("pixel", |lua, img, (x, y): (u32, u32)| color_lua(lua, img.pixel(x, y)));
    }
}

impl UserData for Rng {
    fn add_methods<M: UserDataMethods<Self>>(m: &mut M) {
        // 和math.random一样：random()是[0, 1)的小数，random(n)是1到n，random(m, n)是m到n
        m.add_method_mut("random", |_, r, (a, b): (Option<i64>, Option<i64>)| {
            Ok(match (a, b) {
                (None, _) => Value::Number(r.float()),
                (Some(n), None) => Value::Integer(r.range(1, n)),
                (Some(lo), Some(hi)) => Value::Integer(r.range(lo, hi)),
            })
        });
        // 64位随机整数
        m.add_method_mut("next", |_, r, ()| Ok(r.next_u64() as i64));
    }
}

impl UserData for Perlin {
    fn add_methods<M: UserDataMethods<Self>>(m: &mut M) {
        m.add_method("noise", |_, p, (x, y): (f64, f64)| Ok(p.noise(x, y)));
        m.add_method("fbm", |_, p, (x, y, octaves): (f64, f64, Option<u32>)| Ok(p.fbm(x, y, octaves.unwrap_or(4))));
    }
}

fn block_info(lua: &Lua, id: BlockId, t: &BlockType) -> Result<Table> {
    let info = lua.create_table()?;
    info.set("id", id)?;
    info.set("name", t.name.as_str())?;
    info.set("display", t.display.as_str())?;
    info.set("solid", t.solid)?;
    info.set("opaque", t.opaque)?;
    info.set("cull_same", t.cull_same)?;
    info.set("unbreakable", t.unbreakable)?;
    info.set("hidden", t.hidden)?;
    info.set("props", lua.create_table_from(t.props.clone())?)?;
    let images = lua.create_table()?;
    let i = &t.images;
    for (name, img) in [
        ("up", &i.up),
        ("down", &i.down),
        ("north", &i.north),
        ("south", &i.south),
        ("west", &i.west),
        ("east", &i.east),
    ] {
        images.set(name, img.clone())?;
    }
    info.set("images", images)?;
    // 物品栏里显示的图标
    info.set("icon", i.south.clone().or(i.up.clone()))?;
    Ok(info)
}

pub fn install(lua: &Lua, s: &Rc<Shared>) -> Result<()> {
    let block = lua.create_table()?;
    block.set("AIR", AIR)?;

    // 注册一种方块，返回BlockId
    let sh = s.clone();
    add(lua, &block, "register", move |_, def: Table| {
        let name: String = def.get::<Option<String>>("name")?.ok_or_else(|| mlua::Error::runtime("方块缺少name"))?;
        let images = match def.get::<Value>("images")? {
            Value::Nil => Default::default(),
            v => convert::images(&sh, &v)?,
        };
        let flag = |k: &str, default: bool| -> Result<bool> { Ok(def.get::<Option<bool>>(k)?.unwrap_or(default)) };
        let props: HashMap<String, String> = def.get::<Option<HashMap<String, String>>>("props")?.unwrap_or_default();
        let t = BlockType {
            display: def.get::<Option<String>>("display")?.unwrap_or_else(|| name.clone()),
            name,
            images,
            solid: flag("solid", true)?,
            opaque: flag("opaque", true)?,
            cull_same: flag("cull_same", false)?,
            unbreakable: flag("unbreakable", false)?,
            hidden: flag("hidden", false)?,
            props,
        };
        let mut reg = sh.blocks.try_borrow_mut().map_err(|_| mlua::Error::runtime("现在不能注册方块"))?;
        reg.register(t).map_err(mlua::Error::runtime)
    })?;

    // 方块的信息：{id, name, display, solid, opaque, cull_same, unbreakable, hidden, props, images, icon}
    let sh = s.clone();
    add(lua, &block, "get", move |lua, b: Value| {
        let reg = sh.blocks()?;
        let id = convert::block_id(&reg, &b)?;
        block_info(lua, id, reg.get(id))
    })?;

    // 名字 -> BlockId，没有时是nil
    let sh = s.clone();
    add(lua, &block, "id", move |_, name: String| Ok(sh.blocks()?.by_name(&name)))?;

    // BlockId -> 名字
    let sh = s.clone();
    add(lua, &block, "name", move |_, b: Value| {
        let reg = sh.blocks()?;
        let id = convert::block_id(&reg, &b)?;
        Ok(reg.get(id).name.clone())
    })?;

    // 有碰撞吗（不能穿过）
    let sh = s.clone();
    add(lua, &block, "is_solid", move |_, b: Value| {
        let reg = sh.blocks()?;
        Ok(reg.is_solid(convert::block_id(&reg, &b)?))
    })?;

    // 所有方块的BlockId（包括空气），从0开始
    let sh = s.clone();
    add(lua, &block, "all", move |lua, ()| lua.create_sequence_from(0..sh.blocks()?.len() as BlockId))?;

    lua.globals().set("block", block)?;

    let image = lua.create_table()?;

    // 纯色图片
    add(lua, &image, "solid", |_, c: Value| Ok(Image::solid(convert::color(&c)?)))?;

    // 按像素生成：f(x, y)返回r, g, b[, a]或者一个颜色，(0, 0)是左上角
    add(lua, &image, "pixels", |_, (w, h, f): (u32, u32, mlua::Function)| {
        if w == 0 || h == 0 || w > 1024 || h > 1024 {
            return error("图片的大小要在1到1024之间");
        }
        let mut pixels = Vec::with_capacity((w * h) as usize);
        for y in 0..h {
            for x in 0..w {
                let r: MultiValue = f.call((x, y))?;
                let c = match r.len() {
                    0 => return error("image.pixels的函数要返回颜色"),
                    1 => convert::color(&r[0])?,
                    _ => {
                        let n = |i: usize, d: f64| match r.get(i) {
                            Some(Value::Integer(v)) => *v as f64,
                            Some(Value::Number(v)) => *v,
                            _ => d,
                        };
                        let b = |v: f64| v.round().clamp(0.0, 255.0) as u8;
                        Color::rgba(b(n(0, 0.0)), b(n(1, 0.0)), b(n(2, 0.0)), b(n(3, 255.0)))
                    }
                };
                pixels.push(c);
            }
        }
        Ok(Image::new(w, h, pixels))
    })?;

    // 读png，路径相对游戏根目录。读不了时是品红色
    let sh = s.clone();
    add(lua, &image, "load", move |_, path: String| Ok(Image::load(&sh.path(&path))))?;

    lua.globals().set("image", image)
}
