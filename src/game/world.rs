use std::collections::HashMap;

const TREE_POSITION_LEN:usize=64;


#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct CellId(pub u128);

/// 世界的场。表示从id可以算出各种方块的特性。对于可逆场也能算出id。
pub trait Field<T> {
    fn get_field(self:&Self, cellid:CellId)->Option<T>;
    /// 对于没有返回场的永远返回None。
    fn get_id(self:&Self, t:T) -> Option<CellId>;
    /// 这个的k或者v设置成None可以起到删除的作用。
    fn set(self:&mut Self, k:Option<CellId>, v:Option<T>);
    /// 返回是否该场真的有返场。
    fn have_rev(self:&Self) -> bool;
}

/// 四叉树坐标。  
/// 四叉树坐标原点在左下角。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct TreePosition {
    x:[i8; TREE_POSITION_LEN], // could be +1, -1
    y:[i8; TREE_POSITION_LEN], // Could be +1, -1
}

/// 游戏世界
pub struct WorldStatus {
    tree_position:HashMap<CellId, TreePosition>,
    tree_position_rev:HashMap<TreePosition, CellId>,
}

/// 实现位置场
impl Field<TreePosition> for WorldStatus {
    fn get_field(self:&Self, cellid:CellId)->Option<TreePosition>{
        return match self.tree_position.get(&cellid){
            Some(x) => Some((*x).clone()),
            None => None,
        }
    }
    fn get_id(self:&Self, t:TreePosition) -> Option<CellId>{
        return match self.tree_position_rev.get(&t) {
            Some(x) => Some((*x).clone()),
            None => None
        }
    }
    fn set(self:&mut Self, k:Option<CellId>, v:Option<TreePosition>) {
        
    }
    fn have_rev(self:&Self) -> bool {
        return true
    }
}

