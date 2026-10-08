use core::fmt;
use std::{cell::Cell, collections::HashMap, hash::Hash};


//use rustpython_vm::stdlib::{_io::Fildes, errno::errors::TPM_E_BAD_PRESENCE};

const TREE_POSITION_LEN: usize = 64;
//在这个tree位置上对应的physical大小是1.0量级。
const IDENTITY_SCALE: usize = 32;

/// Physical type.
pub type PhysicalTime = f64;
#[derive(Debug, Clone, Copy)]
pub struct Physical1d(pub f64);

#[derive(Debug, Clone, Copy)]
pub struct Physical2d(pub f64, pub f64);

#[derive(Debug, Clone, Copy)]
pub struct Angle(pub f32);

#[derive(Debug, Clone, Copy)]
pub struct PhysicalLinearVelocity(pub f64, pub f64);
#[derive(Debug, Clone, Copy)]
pub struct PhycicalAngualrVelocity(pub f64);

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct CellId(pub u128);

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
#[derive(Debug, Clone, Copy)]
pub struct Layer(pub i64);

pub type Offset = Physical2d;

/// 一个场的数据：每个格子最多一个值。没有返场。
pub struct FieldStore<T> {
    map: HashMap<CellId, T>,
}

impl<T: Copy> FieldStore<T> {
    pub fn new() -> Self {
        return Self { map: HashMap::new() };
    }
    pub fn get(self: &Self, id: CellId) -> Option<T> {
        return self.map.get(&id).copied();
    }
    /// v 为 None 时删除。
    pub fn set(self: &mut Self, id: CellId, v: Option<T>) {
        match v {
            Some(v) => { self.map.insert(id, v); },
            None => { self.map.remove(&id); },
        };
    }
    pub fn iter(self: &Self) -> impl Iterator<Item = (CellId, T)> + '_ {
        return self.map.iter().map(|(k, v)| (*k, *v));
    }
}

/// 带返场的场：值和格子一一对应，可以从值反查格子。
pub struct IndexedFieldStore<T> {
    map: HashMap<CellId, T>,
    rev: HashMap<T, CellId>,
}

impl<T: Copy + Eq + Hash> IndexedFieldStore<T> {
    pub fn new() -> Self {
        return Self { map: HashMap::new(), rev: HashMap::new() };
    }
    pub fn get(self: &Self, id: CellId) -> Option<T> {
        return self.map.get(&id).copied();
    }
    pub fn get_id(self: &Self, t: T) -> Option<CellId> {
        return self.rev.get(&t).copied();
    }
    /// v 为 None 时删除。
    pub fn set(self: &mut Self, id: CellId, v: Option<T>) {
        // 先删掉 id 原来的值，正反两张表一起删。
        if let Some(old) = self.map.remove(&id) {
            self.rev.remove(&old);
        }
        if let Some(v) = v {
            // 新值如果被别的格子占着，把那个格子的记录也删掉，保证正反表一一对应。
            if let Some(other) = self.rev.insert(v, id) {
                self.map.remove(&other);
            }
            self.map.insert(id, v);
        }
    }
    pub fn iter(self: &Self) -> impl Iterator<Item = (CellId, T)> + '_ {
        return self.map.iter().map(|(k, v)| (*k, *v));
    }
}

/// 世界的所有数据。场之间只能通过这里交流。
pub struct Status {
    pub tree_position: IndexedFieldStore<TreePosition>,
    pub offset: FieldStore<Offset>,
    pub angle: FieldStore<Angle>,
    pub linear_velocity: FieldStore<PhysicalLinearVelocity>,
    pub angular_velocity: FieldStore<PhycicalAngualrVelocity>,
    pub layer: FieldStore<Layer>,
    next_id: u128,
}

impl Status {
    pub fn new() -> Self {
        return Self {
            tree_position: IndexedFieldStore::new(),
            offset: FieldStore::new(),
            angle: FieldStore::new(),
            linear_velocity: FieldStore::new(),
            angular_velocity: FieldStore::new(),
            layer: FieldStore::new(),
            next_id: 0,
        };
    }
    /// 分配一个新的格子 id，不写入任何场。
    pub fn new_cell(self: &mut Self) -> CellId {
        let id = CellId(self.next_id);
        self.next_id += 1;
        return id;
    }
    /// 从所有场里删掉这个格子。新增场时这里也要加。
    pub fn remove_cell(self: &mut Self, id: CellId) {
        self.tree_position.set(id, None);
        self.offset.set(id, None);
        self.angle.set(id, None);
        self.linear_velocity.set(id, None);
        self.angular_velocity.set(id, None);
        self.layer.set(id, None);
    }
}

/// 场的逻辑。只能读写 Status，看不到其他场的逻辑。
pub trait FieldLogic {
    fn update(self: &mut Self, status: &mut Status, dt: PhysicalTime);
}

/// 游戏世界
pub struct WorldStatus {
    pub status: Status,
    fields: Vec<Box<dyn FieldLogic>>,
}

impl WorldStatus {
    pub fn new() -> Self {
        return Self { status: Status::new(), fields: Vec::new() };
    }
    /// 按添加顺序执行。
    pub fn add_field(self: &mut Self, field: Box<dyn FieldLogic>) {
        self.fields.push(field);
    }
    pub fn update(self: &mut Self, dt: PhysicalTime) {
        for f in self.fields.iter_mut() {
            f.update(&mut self.status, dt);
        }
    }
}

#[test]
fn __test(){
    
}