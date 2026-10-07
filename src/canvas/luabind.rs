use std::{any::Any, result};

use mlua::{AnyUserData, Lua, Result, Table, Variadic};
use raylib::{audio, ffi::guiRAYGUI_ICONName};
use crate::canvas::{self, canvas::{Canvas, Canvas1d, Canvas2d, CanvasColor, DrawIns}, raylib::RaylibCanvas};


// The functions we need to bind. ->
fn lua_new_canvas1d(lua:&Lua, i:i32) -> Result<AnyUserData> {
    lua.create_any_userdata(
        Canvas1d(i)
    )
}

fn lua_new_canvas2d(lua:&Lua, (x, y):(i32, i32)) ->Result<AnyUserData>{
    lua.create_any_userdata(
        Canvas2d(x, y)
    )
}

fn lua_new_canvascolor(lua:&Lua, (r, g, b, a):(u8, u8, u8, u8)) -> Result<AnyUserData> {
    lua.create_any_userdata(
        CanvasColor(r,g,b,a)
    )
}

fn lua_new_drawins_square(lua:&Lua, (side_canvas1d, center_canvas2d, color_canvascolor, rotation_f32):(AnyUserData, AnyUserData, AnyUserData, f32)) -> Result<AnyUserData> {
    let side = *(side_canvas1d.borrow::<Canvas1d>()?);
    let center = *(center_canvas2d.borrow::<Canvas2d>()?);
    let color = *(color_canvascolor.borrow::<CanvasColor>()?);
    lua.create_any_userdata(
        DrawIns::Square { side, center, color, rotation: rotation_f32 }
    )
}

fn lua_new_drawins_line(lua:&Lua, (start_canvas2d, end_canvas2d, color_canvascolor):(AnyUserData, AnyUserData, AnyUserData)) -> Result<AnyUserData> {
    let start = *(start_canvas2d.borrow::<Canvas2d>()?);
    let end = *(end_canvas2d.borrow::<Canvas2d>()?);
    let color = *(color_canvascolor.borrow::<CanvasColor>()?);
    lua.create_any_userdata(
        DrawIns::Line { start, end, color }
    )
}

fn lua_new_drawins_text(lua:&Lua, (text_string, center_canvas2d, color_canvascolor, rotation_f32, font_string):(String, AnyUserData, AnyUserData, f32, String)) -> Result<AnyUserData> {
    let center = *(center_canvas2d.borrow::<Canvas2d>()?);
    let color = *(color_canvascolor.borrow::<CanvasColor>()?);
    lua.create_any_userdata(
        DrawIns::Text {
            text: text_string,
            center,
            color,
            rotation: rotation_f32,
            font: font_string,
        }
    )
}

fn lua_new_raylibcanvas(lua:&Lua, (shape_x, shape_y, resizable):(i32, i32, bool))->Result<AnyUserData> {
    return lua.create_any_userdata(RaylibCanvas::new((shape_x, shape_y), resizable))
}



/// 和 Rust 的 `r = r.draw(ins)` 对应：传入的 canvas 句柄会失效，返回新的句柄。
/// Lua 里写 `canvas = rust.lua_raylibcanvas_draw(canvas, ins1, ins2, ...)`。
fn lua_raylibcanvas_draw(lua: &Lua, (raylib_canvas, ins):(AnyUserData, Variadic<AnyUserData>)) -> Result<AnyUserData> {
    // 复制一份，Lua 里的指令可以重复使用。
    let mut ins_list = Vec::with_capacity(ins.len());
    for i in ins.iter() {
        ins_list.push(i.borrow::<DrawIns>()?.clone());
    }
    // 先检查完所有指令再取出 canvas，这样指令出错时 canvas 句柄不会失效。
    let rc = raylib_canvas.take::<RaylibCanvas>()?;
    lua.create_any_userdata(rc.draw(ins_list))
}
fn lua_raylibcanvas_should_close(lua:&Lua, (raylib_canvas):(AnyUserData)) -> Result<bool> {
    Ok(raylib_canvas.borrow::<RaylibCanvas>()?.should_close())
}
fn lua_raylibcanvas_get_size(lua:&Lua, (raylib_canvas):(AnyUserData)) -> Result<(usize, usize)> {
    Ok(raylib_canvas.borrow::<RaylibCanvas>()?.get_size())
}
// // The functions we need to bind. <-


/// 把上面的函数注册到 Lua 全局的 `rust` 表里。
/// 调用前 Lua 里必须已经有全局 `rust` 表，否则返回错误。
pub fn bind(lua:&Lua) ->Result<()>{
    let rust_table:Table = lua.globals().get("rust")?;
    let lua_new_drawins = lua.create_function(|lua, string:String|{
        match string {
            _ => ()
        }
        lua.create_any_userdata(())
    })?;
    rust_table.set("lua_new_drawins", lua_new_drawins)?;
    rust_table.set("lua_new_canvas1d", lua.create_function(lua_new_canvas1d)?)?;
    rust_table.set("lua_new_canvas2d", lua.create_function(lua_new_canvas2d)?)?;
    rust_table.set("lua_new_canvascolor", lua.create_function(lua_new_canvascolor)?)?;
    rust_table.set("lua_new_drawins_square", lua.create_function(lua_new_drawins_square)?)?;
    rust_table.set("lua_new_drawins_line", lua.create_function(lua_new_drawins_line)?)?;
    rust_table.set("lua_new_drawins_text", lua.create_function(lua_new_drawins_text)?)?;
    rust_table.set("lua_new_raylibcanvas", lua.create_function(lua_new_raylibcanvas)?)?;
    rust_table.set("lua_raylibcanvas_draw", lua.create_function(lua_raylibcanvas_draw)?)?;
    rust_table.set("lua_raylibcanvas_should_close", lua.create_function(lua_raylibcanvas_should_close)?)?;
    rust_table.set("lua_raylibcanvas_get_size", lua.create_function(lua_raylibcanvas_get_size)?)?;
    Ok(())
}



struct Test{
    x:usize,
    y:usize,
}
#[test]
fn t()->Result<()>{
    let lua = Lua::new();
    lua.load("print('ccc')").exec()?;
    let u = lua.create_any_userdata(Test{x:34, y:444})?;
    lua.globals().set("z", u)?;
    lua.load("print(kkk)").exec()?;
    Ok(())
}