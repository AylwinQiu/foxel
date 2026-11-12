//! 世界的数据结构：八叉树、CubeId、场、chunk编码、射线检测、碰撞。
//! world不知道方块长什么样，只存BlockId；需要知道"实不实心"时由调用者传一个函数进来。

mod cube;
pub mod encode;
mod field;
mod physics;
mod position;
mod raycast;
#[allow(clippy::module_inception)]
mod world;

#[cfg(test)]
mod tests;

pub use cube::{AIR, BlockId, Cube};
pub use physics::box_corners;
pub use position::{CHUNK_DEPTH, CHUNK_SIZE, CubeId, Depth, IDENTITY_DEPTH, MIN_DEPTH, Position, side};
pub use raycast::Hit;
pub use world::{ChunkData, Generator, Leaf, World, WorldError};
