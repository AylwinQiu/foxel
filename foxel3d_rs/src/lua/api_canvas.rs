//! engine、canvas模块：相机、加载chunk、画世界，窗口、2D绘制、输入

use std::rc::Rc;
use std::time::Duration;

use mlua::{AnyUserData, Lua, Result, Table, Value};

use super::convert::{self, color, pos, pos_lua};
use super::{Shared, add};
use crate::canvas::{Image, Vec2, Vec3};

pub fn install(lua: &Lua, s: &Rc<Shared>) -> Result<()> {
    let e = lua.create_table()?;

    // 相机的位置和朝向（物理坐标）
    let sh = s.clone();
    add(lua, &e, "set_camera", move |_, (p, d): (Value, Value)| {
        sh.engine()?.set_camera(pos(&p)?, pos(&d)?);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &e, "camera", move |lua, ()| {
        let (p, d) = sh.engine()?.camera();
        Ok((pos_lua(lua, p)?, pos_lua(lua, d)?))
    })?;

    // 视距（chunk数）
    let sh = s.clone();
    add(lua, &e, "set_view_distance", move |_, n: i64| {
        sh.engine()?.view_distance = n.clamp(1, 64);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &e, "view_distance", move |_, ()| Ok(sh.engine()?.view_distance))?;

    // 加载相机附近的chunk、卸载远处的、重新生成改动过的网格，最多花budget毫秒（默认8）
    let sh = s.clone();
    add(lua, &e, "update", move |_, budget: Option<f64>| {
        let mut w = sh.world()?;
        let reg = sh.blocks()?;
        let budget = Duration::from_secs_f64(budget.unwrap_or(8.0).max(0.0) / 1000.0);
        sh.engine()?.update(&mut w.world, sh.canvas()?.as_mut(), &reg, budget);
        Ok(())
    })?;

    // 相机周围radius个chunk以内是否都加载好、网格也生成好了
    let sh = s.clone();
    add(lua, &e, "ready", move |_, radius: Option<i64>| {
        Ok(sh.engine()?.ready(&sh.world_ref()?.world, radius.unwrap_or(2)))
    })?;

    // 把视锥里的chunk画出来
    let sh = s.clone();
    add(lua, &e, "draw_world", move |_, ()| {
        sh.engine()?.draw_world(sh.canvas()?.as_mut());
        Ok(())
    })?;

    // 3D线（物理坐标）
    let sh = s.clone();
    add(lua, &e, "line", move |_, (a, b, c): (Value, Value, Value)| {
        sh.engine()?.add_line3d(sh.canvas()?.as_mut(), pos(&a)?, pos(&b)?, color(&c)?);
        Ok(())
    })?;
    // 长方体的12条边
    let sh = s.clone();
    add(lua, &e, "box", move |_, (lo, hi, c): (Value, Value, Value)| {
        sh.engine()?.add_box_lines(sh.canvas()?.as_mut(), pos(&lo)?, pos(&hi)?, color(&c)?);
        Ok(())
    })?;
    // 一个会动的cube（不在世界里）：中心、边长、贴图、旋转（方向是转轴，长度是弧度）
    let sh = s.clone();
    add(lua, &e, "cube", move |_, (center, side, images, rot): (Value, f64, Value, Option<Value>)| {
        let images = convert::images(&sh, &images)?;
        let r = match rot {
            Some(v) => pos(&v)?,
            None => Default::default(),
        };
        sh.engine()?.add_cube(sh.canvas()?.as_mut(), pos(&center)?, side, &images, Vec3::new(r.x, r.y, r.z));
        Ok(())
    })?;

    // 上一帧的统计：{loaded, meshes, drawn, pending}
    let sh = s.clone();
    add(lua, &e, "stats", move |lua, ()| {
        let st = sh.engine()?.stats;
        let t = lua.create_table()?;
        t.set("loaded", st.loaded)?;
        t.set("meshes", st.meshes)?;
        t.set("drawn", st.drawn)?;
        t.set("pending", st.pending)?;
        Ok(t)
    })?;
    lua.globals().set("engine", e)?;

    let c = lua.create_table()?;

    let sh = s.clone();
    add(lua, &c, "should_close", move |_, ()| Ok(sh.canvas()?.should_close()))?;
    let sh = s.clone();
    add(lua, &c, "size", move |_, ()| Ok(sh.canvas()?.window_size()))?;
    let sh = s.clone();
    add(lua, &c, "frame_time", move |_, ()| Ok(sh.canvas()?.frame_time()))?;
    // 清掉这一帧添加的东西（网格不会删）
    let sh = s.clone();
    add(lua, &c, "clean", move |_, ()| {
        sh.canvas()?.clean();
        Ok(())
    })?;
    // 画出这一帧，然后读下一帧的输入
    let sh = s.clone();
    add(lua, &c, "draw", move |_, ()| {
        sh.canvas()?.draw();
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &c, "set_background", move |_, v: Value| {
        sh.canvas()?.set_background(color(&v)?);
        Ok(())
    })?;
    // 雾：离相机start以内没有雾，到end完全变成颜色c
    let sh = s.clone();
    add(lua, &c, "set_fog", move |_, (v, start, end): (Value, f64, f64)| {
        sh.canvas()?.set_fog(color(&v)?, start, end);
        Ok(())
    })?;
    // 视角（角度制）：横向、纵向、倾斜。auto为true时纵向按窗口宽高比算
    let sh = s.clone();
    add(lua, &c, "view_angle", move |_, (h, v, tilt, auto): (f64, Option<f64>, Option<f64>, Option<bool>)| {
        sh.canvas()?.view_angle(h, v.unwrap_or(0.0), tilt.unwrap_or(0.0), auto.unwrap_or(true));
        Ok(())
    })?;

    // 2D：坐标是像素，(0, 0)是左上角，后画的盖在上面
    let sh = s.clone();
    add(lua, &c, "text", move |_, (t, x, y, col): (String, f64, f64, Option<Value>)| {
        let col = match col {
            Some(v) => color(&v)?,
            None => crate::canvas::Color::WHITE,
        };
        sh.canvas()?.add_text(&t, Vec2::new(x, y), col);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &c, "measure", move |_, t: String| {
        let v = sh.canvas()?.measure_text(&t);
        Ok((v.x, v.y))
    })?;
    let sh = s.clone();
    add(lua, &c, "line", move |_, (x1, y1, x2, y2, col): (f64, f64, f64, f64, Value)| {
        sh.canvas()?.add_line2d(Vec2::new(x1, y1), Vec2::new(x2, y2), color(&col)?);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &c, "rect", move |_, (x, y, w, h, col): (f64, f64, f64, f64, Value)| {
        sh.canvas()?.add_rect(Vec2::new(x, y), Vec2::new(w, h), color(&col)?);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &c, "image", move |_, (img, x, y, w, h): (AnyUserData, f64, f64, f64, f64)| {
        let img = img.borrow::<Image>()?;
        sh.canvas()?.add_image(&img, Vec2::new(x, y), Vec2::new(w, h));
        Ok(())
    })?;

    // 这一帧的输入：{held = {[键名] = true}, pressed = {...}, mouse_dx, mouse_dy, mouse_x, mouse_y, wheel, text}
    let sh = s.clone();
    add(lua, &c, "input", move |lua, ()| {
        let canvas = sh.canvas()?;
        let i = canvas.input();
        let t = lua.create_table()?;
        let set = |keys: &std::collections::HashSet<crate::canvas::Key>| -> Result<Table> {
            lua.create_table_from(keys.iter().map(|k| (k.name(), true)))
        };
        t.set("held", set(&i.held)?)?;
        t.set("pressed", set(&i.pressed)?)?;
        t.set("mouse_dx", i.mouse_dx)?;
        t.set("mouse_dy", i.mouse_dy)?;
        t.set("mouse_x", i.mouse_x)?;
        t.set("mouse_y", i.mouse_y)?;
        t.set("wheel", i.wheel)?;
        t.set("text", i.text.as_str())?;
        Ok(t)
    })?;
    let sh = s.clone();
    add(lua, &c, "lock_cursor", move |_, locked: bool| {
        sh.canvas()?.set_cursor_locked(locked);
        Ok(())
    })?;
    let sh = s.clone();
    add(lua, &c, "cursor_locked", move |_, ()| Ok(sh.canvas()?.is_cursor_locked()))?;
    // 下一次canvas.draw时把画面存成png，路径相对游戏根目录
    let sh = s.clone();
    add(lua, &c, "screenshot", move |_, path: String| {
        let p = sh.path(&path);
        sh.canvas()?.screenshot(&p);
        Ok(())
    })?;

    lua.globals().set("canvas", c)
}
