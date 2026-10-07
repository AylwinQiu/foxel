use core::fmt;
use std::collections::HashMap;


//use rustpython_vm::stdlib::{_io::Fildes, errno::errors::TPM_E_BAD_PRESENCE};

const TREE_POSITION_LEN: usize = 64;
//在这个tree位置上对应的physical大小是1.0量级。
const IDENTITY_SCALE: usize = 32;

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
    fn set(self: &mut Self, k: CellId, v: Option<T>);
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

impl TreePosition {
    /// 该格子中心的物理坐标。
    pub fn to_physical(self:&Self) -> Physical2d {
        let half_root = 2f64.powi(IDENTITY_SCALE as i32 - 1);
        let (mut px, mut py) = (half_root, half_root); // 根节点中心
        let mut step = half_root / 2.0;
        for i in 0..TREE_POSITION_LEN {
            if self.x[i] == 0 {
                break; // 全 0 表示结束
            }
            px += self.x[i] as f64 * step;
            py += self.y[i] as f64 * step;
            step /= 2.0;
        }
        return Physical2d(px, py)
    }
    /// 找到物理坐标在第 depth 层所在的格子，并返回该点相对格子中心的偏移。
    pub fn from_physical(phy:Physical2d, depth:usize) -> (TreePosition, Offset){
        let depth = depth.min(TREE_POSITION_LEN);
        let half_root = 2f64.powi(IDENTITY_SCALE as i32 - 1);
        let mut pos = TreePosition { x: [0; TREE_POSITION_LEN], y: [0; TREE_POSITION_LEN] };
        // 相对当前格子中心的残差
        let (mut rx, mut ry) = (phy.0 - half_root, phy.1 - half_root);
        let mut step = half_root / 2.0;
        for i in 0..depth {
            // 正好在分界线上时归到右/上（格子左闭右开）
            pos.x[i] = if rx >= 0.0 { 1 } else { -1 };
            pos.y[i] = if ry >= 0.0 { 1 } else { -1 };
            rx -= pos.x[i] as f64 * step;
            ry -= pos.y[i] as f64 * step;
            step /= 2.0;
        }
        return (pos, Physical2d(rx, ry))
    }

}

/// 图层坐标, 游戏里面不同物体所在的图层。
pub struct Layer(pub i64);

/// 我们以后让游戏玩家可以自己添加field。
pub trait FieldStatus{
    // TODO
}

/// 游戏世界
pub struct WorldStatus {
    tree_position: HashMap<CellId, TreePosition>,
    tree_position_rev: HashMap<TreePosition, CellId>,
    // 这个不一定用得到。
    dyn_fields_status:HashMap<String, Box<dyn FieldStatus>>, 
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
    fn set(self: &mut Self, k: CellId, v: Option<TreePosition>) {
        if v == None {
            // delete the pair in the field and rev field if rev field exist.
            // Get the value of k
            match self.tree_position.get(&k){
                Some(vv) => {
                    // remove the rev field pair.
                    self.tree_position_rev.remove(vv);
                    // remove the field.
                    self.tree_position.remove(&k);
                },
                None => ()
            };
            self.tree_position.remove(&k);
        }
        return ();
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
    fn set(self: &mut Self, k: CellId, v: Option<Offset>) {
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
    fn set(self: &mut Self, k: CellId, v: Option<Angle>) {
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
        return false;
    }
    fn set(self: &mut Self, k: CellId, v: Option<PhysicalLinearVelocity>) {
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
        return false;
    }
    fn set(self: &mut Self, k: CellId, v: Option<PhycicalAngualrVelocity>) {
        todo!()
    }
}

// 实现图层场
impl Field<Layer> for WorldStatus {
    fn get_field(self: &Self, cellid: CellId) -> Option<Layer> {
        todo!()
    }
    fn get_id(self: &Self, t: Layer) -> Option<CellId> {
        todo!()
    }
    fn have_rev(self: &Self) -> bool {
        return false;
    }
    fn set(self: &mut Self, k: CellId, v: Option<Layer>) {
        todo!()
    }
} 

#[test]
fn __test(){
    
}