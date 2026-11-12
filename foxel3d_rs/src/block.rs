//! 方块种类的注册表。world里每个cube只存一个BlockId，方块长什么样、能不能穿过都在这里查。
//!
//! 同一种方块的所有cube共用一个BlockType，所以一百万块石头只有一份配置。
//! 每个cube自己的数据（比如箱子里的东西）用world的场。
//! 存档里按名字保存方块，所以注册顺序变了、新增了方块都没关系。

use std::collections::HashMap;

use crate::canvas::Images;
use crate::world::{AIR, BlockId};

#[derive(Clone, Default)]
pub struct BlockType {
    /// 唯一的名字，比如"foxel:grass"
    pub name: String,
    /// 显示给玩家看的名字（只能用ASCII）
    pub display: String,
    pub images: Images,
    /// 有碰撞，玩家不能穿过
    pub solid: bool,
    /// 完全不透明：会挡住旁边方块的面。玻璃、树叶这种有透明像素的是false
    pub opaque: bool,
    /// 两个同样的、opaque是false的方块挨着时，省掉中间的面。玻璃是true，树叶是false
    pub cull_same: bool,
    /// 玩家挖不掉，比如基岩
    pub unbreakable: bool,
    /// 不出现在物品栏里，比如空气
    pub hidden: bool,
    /// 自定义属性，mod可以随便用
    pub props: HashMap<String, String>,
}

pub struct Registry {
    types: Vec<BlockType>,
    by_name: HashMap<String, BlockId>,
    /// 世界打开以后不能再注册
    frozen: bool,
}

impl Default for Registry {
    fn default() -> Self {
        Self::new()
    }
}

impl Registry {
    /// 只有空气（BlockId 0，名字"foxel:air"）
    pub fn new() -> Registry {
        let air = BlockType { name: "foxel:air".into(), display: "Air".into(), hidden: true, ..Default::default() };
        Registry { types: vec![air], by_name: HashMap::from([("foxel:air".to_string(), AIR)]), frozen: false }
    }

    /// 注册一种方块，返回它的BlockId
    pub fn register(&mut self, t: BlockType) -> Result<BlockId, String> {
        if self.frozen {
            return Err(format!("方块{}注册得太晚了：世界已经打开", t.name));
        }
        if self.by_name.contains_key(&t.name) {
            return Err(format!("方块{}注册了两次", t.name));
        }
        if self.types.len() > BlockId::MAX as usize {
            return Err("方块种类太多了".into());
        }
        let id = self.types.len() as BlockId;
        self.by_name.insert(t.name.clone(), id);
        self.types.push(t);
        Ok(id)
    }

    pub fn freeze(&mut self) {
        self.frozen = true;
    }

    pub fn is_frozen(&self) -> bool {
        self.frozen
    }

    /// BlockId对应的方块种类，没注册过的当成空气
    pub fn get(&self, id: BlockId) -> &BlockType {
        self.types.get(id as usize).unwrap_or(&self.types[AIR as usize])
    }

    pub fn by_name(&self, name: &str) -> Option<BlockId> {
        self.by_name.get(name).copied()
    }

    /// 注册了几种方块（包括空气）。BlockId是0到len-1
    pub fn len(&self) -> usize {
        self.types.len()
    }

    pub fn is_empty(&self) -> bool {
        false
    }

    pub fn is_solid(&self, id: BlockId) -> bool {
        self.get(id).solid
    }

    pub fn is_opaque(&self, id: BlockId) -> bool {
        self.get(id).opaque
    }
}
