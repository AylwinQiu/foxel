//! foxel3d_rs：方块沙盒游戏。底层是Rust（世界、地形、网格、窗口），上层逻辑是Lua（src/game/）。
//!
//! 模块按依赖从下往上：
//!   rng、world  八叉树、CubeId、场、chunk编码、射线、碰撞
//!   canvas      窗口接口（Canvas trait）+ raylib实现，换前端只要实现Canvas
//!   block       方块注册表
//!   terrain     地形生成
//!   engine      把world画到canvas上
//!   save        存档
//!   lua         Lua绑定，运行src/game/main.lua

pub mod block;
pub mod canvas;
pub mod engine;
pub mod lua;
pub mod rng;
pub mod save;
pub mod terrain;
pub mod world;
