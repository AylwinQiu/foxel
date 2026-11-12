//! Lua绑定：把Rust这边的功能做成Lua的全局模块，然后运行src/game/main.lua。
//!
//! 全局模块（完整说明见API.md）：
//!   foxel    杂项：随机数、噪声、mod列表
//!   block    方块注册表
//!   image    图片
//!   terrain  生物群系和feature
//!   cube     CubeId的计算
//!   world    世界：读写方块、射线、碰撞、场、存档
//!   engine   相机、加载chunk、画世界和3D辅助线
//!   canvas   窗口、2D绘制、输入
//!
//! 所有状态放在Shared里，每个子系统一个RefCell。地形生成的回调里world正在被修改，
//! 这时调用world.*会报错，而不是把程序弄崩。

mod api_block;
mod api_canvas;
mod api_terrain;
mod api_world;
mod convert;

use std::cell::{Ref, RefCell, RefMut};
use std::path::{Path, PathBuf};
use std::rc::Rc;
use std::time::Instant;

use mlua::{Lua, Result, Table};

use crate::block::Registry;
use crate::canvas::Canvas;
use crate::engine::Engine;
use crate::terrain::TerrainCore;
use crate::world::World;

pub use api_terrain::TerrainDefs;

/// 打开了的世界
pub struct OpenWorld {
    pub world: World,
    pub core: Rc<TerrainCore>,
    pub save_dir: PathBuf,
    pub seed: i64,
}

pub struct Shared {
    /// 游戏的根目录，里面有src/game和mods
    pub root: PathBuf,
    /// 存档放在这里，每个世界一个文件夹
    pub saves: PathBuf,
    pub blocks: RefCell<Registry>,
    pub terrain: RefCell<TerrainDefs>,
    pub world: RefCell<Option<OpenWorld>>,
    pub engine: RefCell<Engine>,
    pub canvas: RefCell<Box<dyn Canvas>>,
    pub start: Instant,
}

fn busy(what: &str) -> mlua::Error {
    mlua::Error::runtime(format!("{what}正在使用中（比如地形生成的回调里不能用world.*，请用ctx）"))
}

impl Shared {
    pub fn world(&self) -> Result<RefMut<'_, OpenWorld>> {
        let w = self.world.try_borrow_mut().map_err(|_| busy("world"))?;
        RefMut::filter_map(w, Option::as_mut).map_err(|_| mlua::Error::runtime("世界还没有打开，先调用world.open"))
    }

    pub fn world_ref(&self) -> Result<Ref<'_, OpenWorld>> {
        let w = self.world.try_borrow().map_err(|_| busy("world"))?;
        Ref::filter_map(w, Option::as_ref).map_err(|_| mlua::Error::runtime("世界还没有打开，先调用world.open"))
    }

    pub fn blocks(&self) -> Result<Ref<'_, Registry>> {
        self.blocks.try_borrow().map_err(|_| busy("block"))
    }

    pub fn canvas(&self) -> Result<RefMut<'_, Box<dyn Canvas>>> {
        self.canvas.try_borrow_mut().map_err(|_| busy("canvas"))
    }

    pub fn engine(&self) -> Result<RefMut<'_, Engine>> {
        self.engine.try_borrow_mut().map_err(|_| busy("engine"))
    }

    /// 相对根目录的路径
    pub fn path(&self, p: &str) -> PathBuf {
        self.root.join(p)
    }
}

/// 在模块表t里加一个函数
pub(crate) fn add<A, R, F>(lua: &Lua, t: &Table, name: &str, f: F) -> Result<()>
where
    A: mlua::FromLuaMulti,
    R: mlua::IntoLuaMulti,
    F: Fn(&Lua, A) -> Result<R> + 'static,
{
    t.set(name, lua.create_function(f)?)
}

/// 新建一个装好所有模块的Lua。root是游戏的根目录，saves是存档目录
pub fn new_lua(root: &Path, saves: &Path, canvas: Box<dyn Canvas>) -> Result<(Lua, Rc<Shared>)> {
    let lua = Lua::new();
    let shared = Rc::new(Shared {
        root: root.to_path_buf(),
        saves: saves.to_path_buf(),
        blocks: RefCell::new(Registry::new()),
        terrain: RefCell::new(TerrainDefs::default()),
        world: RefCell::new(None),
        engine: RefCell::new(Engine::new()),
        canvas: RefCell::new(canvas),
        start: Instant::now(),
    });
    lua.set_app_data(shared.clone());

    let package: Table = lua.globals().get("package")?;
    let r = root.to_string_lossy();
    package.set("path", format!("{r}/src/game/?.lua;{r}/mods/?/init.lua;{r}/mods/?.lua"))?;

    install_foxel(&lua, &shared)?;
    api_block::install(&lua, &shared)?;
    api_terrain::install(&lua, &shared)?;
    api_world::install(&lua, &shared)?;
    api_canvas::install(&lua, &shared)?;
    Ok((lua, shared))
}

/// 运行src/game/main.lua，args是命令行参数（Lua里的arg[1], arg[2], ...）
pub fn run(root: &Path, saves: &Path, canvas: Box<dyn Canvas>, args: &[String]) -> std::result::Result<(), String> {
    run_file(root, saves, &root.join("src/game/main.lua"), canvas, args)
}

/// 用别的Lua文件代替main.lua（测试、工具脚本用）
pub fn run_file(
    root: &Path,
    saves: &Path,
    file: &Path,
    canvas: Box<dyn Canvas>,
    args: &[String],
) -> std::result::Result<(), String> {
    let (lua, _shared) = new_lua(root, saves, canvas).map_err(|e| e.to_string())?;
    let code = std::fs::read_to_string(file).map_err(|e| format!("读不了{}：{e}", file.display()))?;
    let name = file.strip_prefix(root).unwrap_or(file).to_string_lossy().to_string();
    let set_args = || -> Result<()> {
        let arg = lua.create_sequence_from(args.iter().cloned())?;
        arg.set(0, name.as_str())?;
        lua.globals().set("arg", arg)
    };
    set_args().map_err(|e| e.to_string())?;
    lua.load(code).set_name(format!("@{name}")).exec().map_err(|e| e.to_string())
}

fn install_foxel(lua: &Lua, s: &Rc<Shared>) -> Result<()> {
    let t = lua.create_table()?;
    t.set("root", s.root.to_string_lossy().to_string())?;
    t.set("version", env!("CARGO_PKG_VERSION"))?;

    // 确定性的随机数
    add(lua, &t, "rng", |_, seed: i64| Ok(crate::rng::Rng::new(seed as u64)))?;
    // 2D Perlin噪声
    add(lua, &t, "noise", |_, seed: i64| Ok(crate::terrain::Perlin::new(seed as u64)))?;
    // 游戏开始后过了多少秒
    let sh = s.clone();
    add(lua, &t, "time", move |_, ()| Ok(sh.start.elapsed().as_secs_f64()))?;
    // mods/下面所有有init.lua的文件夹，按名字排序
    let sh = s.clone();
    add(lua, &t, "mods", move |lua, ()| {
        let mut names = Vec::new();
        if let Ok(dir) = std::fs::read_dir(sh.path("mods")) {
            for e in dir.flatten() {
                if e.path().join("init.lua").is_file() {
                    names.push(e.file_name().to_string_lossy().to_string());
                }
            }
        }
        names.sort();
        lua.create_sequence_from(names)
    })?;
    lua.globals().set("foxel", t)
}
