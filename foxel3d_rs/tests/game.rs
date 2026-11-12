//! 不开窗口把整个Lua游戏跑起来：用NullCanvas代替raylib，存档放在临时文件夹里

use std::path::{Path, PathBuf};

use foxel3d_rs::canvas::{Input, Key, NullCanvas};
use foxel3d_rs::lua;

fn root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

fn temp_saves(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!("foxel_test_{}_{name}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    dir
}

/// 在新的Lua里运行tests/lua下的脚本。shared里的全局变量SAVED从上一个脚本带过来
fn run_script(saves: &Path, script: &str, carry: Option<String>) -> Option<String> {
    let (l, _) = lua::new_lua(&root(), saves, Box::new(NullCanvas::new())).unwrap();
    let dir = root().join("tests/lua");
    l.globals().set("TEST_DIR", dir.to_string_lossy().to_string()).unwrap();
    // 两个Lua之间用Lua代码字符串传SAVED
    if let Some(code) = carry {
        l.load(code).exec().unwrap();
    }
    let code = std::fs::read_to_string(dir.join(script)).unwrap();
    if let Err(e) = l.load(code).set_name(format!("@tests/lua/{script}")).exec() {
        panic!("{script}失败了：\n{e}");
    }
    let dump: String = l
        .load(
            r#"
            local s = SAVED
            if not s then return "" end
            local function c(t) return ("{x=%d, y=%d, z=%d, depth=%d}"):format(t.x, t.y, t.z, t.depth) end
            return ("SAVED = {cell=%s, small=%s, pos={x=%.17g, y=%.17g, z=%.17g}}"):format(
                c(s.cell), c(s.small), s.pos.x, s.pos.y, s.pos.z)
            "#,
        )
        .eval()
        .unwrap();
    (!dump.is_empty()).then_some(dump)
}

#[test]
fn play_save_and_reload() {
    let saves = temp_saves("play");
    let carry = run_script(&saves, "play.lua", None);
    assert!(saves.join("test/world.json").is_file());
    assert!(saves.join("test/chunks.bin").is_file());
    run_script(&saves, "reload.lua", carry);
    let _ = std::fs::remove_dir_all(&saves);
}

#[test]
fn console() {
    let saves = temp_saves("console");
    run_script(&saves, "console.lua", None);
    let _ = std::fs::remove_dir_all(&saves);
}

/// 真正的入口main.lua：解析参数、加载mod、跑几百帧、关闭时存档
#[test]
fn main_lua_runs() {
    let saves = temp_saves("main");
    let mut inputs = vec![Input::default(); 200];
    // 点一下锁定鼠标，然后往前走
    inputs[150].pressed.insert(Key::MouseLeft);
    for i in &mut inputs[151..] {
        i.held.insert(Key::W);
    }
    let canvas = NullCanvas::with_inputs(inputs);
    let args: Vec<String> = ["--world", "smoke", "--seed", "42", "--view", "3"].iter().map(|s| s.to_string()).collect();
    lua::run(&root(), &saves, Box::new(canvas), &args).unwrap();
    let json = std::fs::read_to_string(saves.join("smoke/world.json")).unwrap();
    assert!(json.contains("\"seed\": 42"), "{json}");
    let _ = std::fs::remove_dir_all(&saves);
}
