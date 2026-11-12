//! 游戏入口：开一个raylib窗口，然后把控制权交给src/game/main.lua。
//!
//!     cargo run --release -- --world 名字 --seed 123 --view 8
//!
//! 命令行参数原样传给Lua（arg表），由main.lua解析。
//! 只有一个参数Rust自己处理：--main 文件.lua，用这个文件代替src/game/main.lua（开发、测试用）。

use std::path::PathBuf;

use foxel3d_rs::canvas::raylib::RaylibCanvas;

/// 游戏的根目录（里面有src/game、mods，存档放在saves）：当前目录有src/game/main.lua就用当前目录，否则用源码所在的目录
fn find_root() -> PathBuf {
    let cwd = std::env::current_dir().unwrap_or_default();
    if cwd.join("src/game/main.lua").is_file() {
        return cwd;
    }
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

fn main() {
    let root = find_root();
    let mut args: Vec<String> = std::env::args().skip(1).collect();
    let mut main = root.join("src/game/main.lua");
    if let Some(i) = args.iter().position(|a| a == "--main") {
        if i + 1 < args.len() {
            main = PathBuf::from(args.remove(i + 1));
        }
        args.remove(i);
    }
    let canvas = RaylibCanvas::new(1280, 720, "foxel3d");
    if let Err(e) = foxel3d_rs::lua::run_file(&root, &root.join("saves"), &main, Box::new(canvas), &args) {
        eprintln!("{e}");
        std::process::exit(1);
    }
}
