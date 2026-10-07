use crate::canvas::{self, luabind as canvas_luabind};
use crate::others::get_assets_root;
use mlua::{ExternalResult, Lua, Result, Table, Variadic, Value};


/// 读取 assets 目录下的文件。path 以 / 开头，比如 "/script/main.lua"。
fn read_string(lua:&Lua, path:String) -> Result<String>{
    // path 以 / 开头，直接拼接；用 Path::join 的话会被当成绝对路径。
    std::fs::read_to_string(format!("{}{}", get_assets_root(), path)).into_lua_err()
}


/// 保证lua里面已经有rust这个global table。
pub fn bind(lua:&Lua) -> Result<()> {
    canvas_luabind::bind(lua)?;
    let rust_table:Table = lua.globals().get("rust")?;
    rust_table.set("read_string", lua.create_function(read_string)?)?;
    return Ok(());
}
