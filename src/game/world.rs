use core::fmt;
use std::collections::HashMap;

use rustpython_vm::stdlib::{_io::Fildes, errno::errors::TPM_E_BAD_PRESENCE};

const TREE_POSITION_LEN: usize = 64;

/// Physical type.
pub type PhysicalTime = f64;
pub struct Physical1d(pub f64);
pub struct Physical2d(pub f64, pub f64);
pub struct Angle(pub f32);

pub struct PhysicalLinearVelocity(pub f64, pub f64);
pub struct PhycicalAngualrVelocity(pub f64);

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct CellId(pub u128);

/// 世界的场。表示从id可以算出各种方块的特性。对于可逆场也能算出id。
pub trait Field<T> {
    fn get_field(self: &Self, cellid: CellId) -> Option<T>;
    /// 对于没有返回场的永远返回None。
    fn get_id(self: &Self, t: T) -> Option<CellId>;
    /// 这个的k或者v设置成None可以起到删除的作用。
    fn set(self: &mut Self, k: Option<CellId>, v: Option<T>);
    /// 返回是否该场真的有返场。
    fn have_rev(self: &Self) -> bool;
}

/// 四叉树坐标。  
/// 四叉树坐标原点在左下角。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct TreePosition {
    x: [i8; TREE_POSITION_LEN], // could be +1, -1
    y: [i8; TREE_POSITION_LEN], // Could be +1, -1
}

/// 游戏世界
pub struct WorldStatus {
    tree_position: HashMap<CellId, TreePosition>,
    tree_position_rev: HashMap<TreePosition, CellId>,
}

/// 实现树位置场
impl Field<TreePosition> for WorldStatus {
    fn get_field(self: &Self, cellid: CellId) -> Option<TreePosition> {
        return match self.tree_position.get(&cellid) {
            Some(x) => Some((*x).clone()),
            None => None,
        };
    }
    fn get_id(self: &Self, t: TreePosition) -> Option<CellId> {
        return match self.tree_position_rev.get(&t) {
            Some(x) => Some((*x).clone()),
            None => None,
        };
    }
    fn set(self: &mut Self, k: Option<CellId>, v: Option<TreePosition>) {
        todo!()
    }
    fn have_rev(self: &Self) -> bool {
        return true;
    }
}
pub type Offset = Physical2d;

// 实现offset场（这个场输出的是每个square相关自己原来位置的偏移量）
impl Field<Offset> for WorldStatus {
    fn get_field(self: &Self, cellid: CellId) -> Option<Offset> {
        todo!()
    }
    fn have_rev(self: &Self) -> bool {
        return false;
    }
    fn get_id(self: &Self, t: Offset) -> Option<CellId> {
        todo!()
    }
    fn set(self: &mut Self, k: Option<CellId>, v: Option<Offset>) {
        todo!()
    }
}

// 实现角度场（这个场输出的是每个square的倾斜角度）
impl Field<Angle> for WorldStatus {
    fn get_field(self: &Self, cellid: CellId) -> Option<Angle> {
        todo!()
    }
    fn get_id(self: &Self, t: Angle) -> Option<CellId> {
        todo!()
    }
    fn have_rev(self: &Self) -> bool {
        return false;
    }
    fn set(self: &mut Self, k: Option<CellId>, v: Option<Angle>) {
        todo!()
    }
}

// 实现线速度场
impl Field<PhysicalLinearVelocity> for WorldStatus {
    fn get_field(self: &Self, cellid: CellId) -> Option<PhysicalLinearVelocity> {
        todo!()
    }
    fn get_id(self: &Self, t: PhysicalLinearVelocity) -> Option<CellId> {
        todo!()
    }
    fn have_rev(self: &Self) -> bool {
        todo!()
    }
    fn set(self: &mut Self, k: Option<CellId>, v: Option<PhysicalLinearVelocity>) {
        todo!()
    }
}

// 实现角速度场
impl Field<PhycicalAngualrVelocity> for WorldStatus {
    fn get_field(self: &Self, cellid: CellId) -> Option<PhycicalAngualrVelocity> {
        todo!()
    }
    fn get_id(self: &Self, t: PhycicalAngualrVelocity) -> Option<CellId> {
        todo!()
    }
    fn have_rev(self: &Self) -> bool {
        todo!()
    }
    fn set(self: &mut Self, k: Option<CellId>, v: Option<PhycicalAngualrVelocity>) {
        todo!()
    }
}

// 实现图层场
