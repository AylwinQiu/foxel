use std::collections::HashMap;
use std::fmt;

use super::{AIR, BlockId, CHUNK_DEPTH, CHUNK_SIZE, Cube, CubeId, MIN_DEPTH, Position};

/// 地形生成器
pub trait Generator {
    /// 生成一个chunk。pos是chunk的CubeId（深度是CHUNK_DEPTH）
    fn generate_chunk(&mut self, pos: CubeId) -> ChunkData;
    /// 世界在竖直方向上有哪些chunk（chunk的y，两端都包括）。范围外永远是空气，不会生成也不会保存
    fn vertical_range(&self) -> (i64, i64);
}

const CHUNK_VOLUME: usize = (CHUNK_SIZE * CHUNK_SIZE * CHUNK_SIZE) as usize;

/// 生成器填写的一个chunk
pub struct ChunkData {
    /// depth 0的方块，下标是index(x, y, z)，x、y、z是chunk里的坐标0到15
    pub blocks: Box<[BlockId; CHUNK_VOLUME]>,
    /// 比1小的方块，在blocks之后放上去
    pub small: Vec<(CubeId, BlockId)>,
}

impl ChunkData {
    /// 全是空气
    pub fn new() -> ChunkData {
        ChunkData { blocks: Box::new([AIR; CHUNK_VOLUME]), small: Vec::new() }
    }

    pub fn index(x: usize, y: usize, z: usize) -> usize {
        x + z * CHUNK_SIZE as usize + y * (CHUNK_SIZE * CHUNK_SIZE) as usize
    }

    pub fn get(&self, x: usize, y: usize, z: usize) -> BlockId {
        self.blocks[Self::index(x, y, z)]
    }

    pub fn set(&mut self, x: usize, y: usize, z: usize, b: BlockId) {
        self.blocks[Self::index(x, y, z)] = b;
    }
}

impl Default for ChunkData {
    fn default() -> Self {
        Self::new()
    }
}

/// 一个叶子cube：位置（包括大小）和方块
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Leaf {
    pub id: CubeId,
    pub block: BlockId,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum WorldError {
    /// chunk没有加载
    NotLoaded,
    /// cube的深度必须在MIN_DEPTH到CHUNK_DEPTH之间
    BadDepth,
}

impl fmt::Display for WorldError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match self {
            WorldError::NotLoaded => write!(f, "chunk没有加载"),
            WorldError::BadDepth => write!(f, "cube的深度必须在{MIN_DEPTH}到{CHUNK_DEPTH}之间"),
        }
    }
}

impl std::error::Error for WorldError {}

pub(crate) struct Chunk {
    pub(crate) node: Cube,
    /// 和生成器生成的不一样了，存档时需要保存
    pub(crate) modified: bool,
}

static AIR_CUBE: Cube = Cube::leaf(AIR);

/// 世界：一棵根边长2^IDENTITY_DEPTH的八叉树。
///
/// chunk以上的那几层只是容器，没有内容，所以不真的建出来：已加载的chunk直接放在HashMap里，
/// 卸载的chunk整个删掉（改动过的先编码保存），走得再远内存也不会变大。
pub struct World {
    generator: Box<dyn Generator>,
    pub(crate) chunks: HashMap<CubeId, Chunk>,
    /// 卸载了的、改动过的chunk（编码后的数据）
    pub(crate) saved: HashMap<CubeId, Vec<u8>>,
    /// 场：名字 -> cube -> 值
    pub(crate) fields: HashMap<String, HashMap<CubeId, String>>,
    /// 上次take_changes以后被set_cube改动的cube
    changes: Vec<CubeId>,
}

impl World {
    pub fn new(generator: Box<dyn Generator>) -> World {
        World { generator, chunks: HashMap::new(), saved: HashMap::new(), fields: HashMap::new(), changes: Vec::new() }
    }

    pub fn generator(&self) -> &dyn Generator {
        self.generator.as_ref()
    }

    /// chunk在不在世界的竖直范围里
    pub fn in_range(&self, pos: CubeId) -> bool {
        let (lo, hi) = self.generator.vertical_range();
        pos.y >= lo && pos.y <= hi
    }

    pub fn is_loaded(&self, pos: CubeId) -> bool {
        self.chunks.contains_key(&pos)
    }

    /// 所有已加载的chunk
    pub fn loaded_chunks(&self) -> Vec<CubeId> {
        self.chunks.keys().copied().collect()
    }

    pub fn loaded_count(&self) -> usize {
        self.chunks.len()
    }

    /// 加载一个chunk：有保存的改动就读出来，否则用生成器生成。已经加载了、或者在竖直范围外时什么都不做
    pub fn load_chunk(&mut self, pos: CubeId) {
        assert_eq!(pos.depth, CHUNK_DEPTH, "load_chunk: {pos}不是chunk");
        if self.is_loaded(pos) || !self.in_range(pos) {
            return;
        }
        if let Some(data) = self.saved.remove(&pos) {
            match super::encode::decode_chunk(&data) {
                Ok(node) => {
                    self.chunks.insert(pos, Chunk { node, modified: true });
                    return;
                }
                Err(e) => eprintln!("foxel: chunk {pos}的存档坏了，重新生成：{e}"),
            }
        }
        let data = self.generator.generate_chunk(pos);
        let mut node = build_cube(&data, 0, 0, 0, CHUNK_SIZE as usize);
        for &(id, b) in &data.small {
            if pos.contains(id) && id.depth >= MIN_DEPTH {
                set_in(&mut node, pos, id, b);
            }
        }
        self.chunks.insert(pos, Chunk { node, modified: false });
    }

    /// 卸载一个chunk，改动过的会先编码保存在内存里，之后再加载时用
    pub fn unload_chunk(&mut self, pos: CubeId) {
        if let Some(ch) = self.chunks.remove(&pos)
            && ch.modified
        {
            self.saved.insert(pos, super::encode::encode_chunk(&ch.node));
        }
    }

    /// 找到id这个位置的cube：返回的就是id这个cube；如果id在一个更大的叶子里面，返回那个叶子和它的id。
    /// 竖直范围外永远是空气；chunk没有加载时返回None。id的深度必须在[MIN_DEPTH, CHUNK_DEPTH]里
    pub fn find(&self, id: CubeId) -> Option<(&Cube, CubeId)> {
        let pos = id.chunk();
        let Some(ch) = self.chunks.get(&pos) else {
            return if self.in_range(pos) { None } else { Some((&AIR_CUBE, pos)) };
        };
        let (mut n, mut at) = (&ch.node, pos);
        while at.depth > id.depth {
            let Some(children) = &n.children else { break };
            at = id.parent_at(at.depth - 1);
            n = &children[at.octant()];
        }
        Some((n, at))
    }

    /// 包含物理坐标p的叶子cube（最小到MIN_DEPTH），没加载时返回None
    pub fn leaf_at(&self, p: Position) -> Option<Leaf> {
        self.find(CubeId::at(p, MIN_DEPTH)).map(|(n, id)| Leaf { id, block: n.block })
    }

    /// depth 0格子(x, y, z)的方块和它是不是被分成了更小的cube（这时方块没有意义）。没加载时返回None
    pub fn block_at(&self, x: i64, y: i64, z: i64) -> Option<(BlockId, bool)> {
        self.find(CubeId::block(x, y, z)).map(|(n, _)| (n.block, !n.is_leaf()))
    }

    /// 把id这个cube整个变成方块b（AIR就是挖掉），里面原来的东西和场都删掉。
    /// 改完后8个一样的兄弟cube会自动合并
    pub fn set_cube(&mut self, id: CubeId, b: BlockId) -> Result<(), WorldError> {
        if id.depth < MIN_DEPTH || id.depth > CHUNK_DEPTH {
            return Err(WorldError::BadDepth);
        }
        let pos = id.chunk();
        let Some(ch) = self.chunks.get_mut(&pos) else {
            return Err(WorldError::NotLoaded);
        };
        set_in(&mut ch.node, pos, id, b);
        ch.modified = true;
        for f in self.fields.values_mut() {
            f.retain(|k, _| !id.contains(*k));
        }
        self.changes.push(id);
        Ok(())
    }

    /// 上次调用以后被set_cube改动过的cube，engine用它来重新生成网格
    pub fn take_changes(&mut self) -> Vec<CubeId> {
        std::mem::take(&mut self.changes)
    }

    /// 按先序遍历一个已加载chunk里的cube，f返回false时不进入这个cube的子cube。chunk没有加载时返回false
    pub fn visit_chunk(&self, pos: CubeId, f: &mut dyn FnMut(CubeId, &Cube) -> bool) -> bool {
        let Some(ch) = self.chunks.get(&pos) else {
            return false;
        };
        visit(&ch.node, pos, f);
        true
    }
}

fn visit(n: &Cube, id: CubeId, f: &mut dyn FnMut(CubeId, &Cube) -> bool) {
    if !f(id, n) {
        return;
    }
    if let Some(children) = &n.children {
        for (o, c) in children.iter().enumerate() {
            visit(c, id.child(o), f);
        }
    }
}

/// 从稠密数组建树，size是这个cube的边长（格子数），x、y、z是它最小的角
fn build_cube(d: &ChunkData, x: usize, y: usize, z: usize, size: usize) -> Cube {
    if size == 1 {
        return Cube::leaf(d.get(x, y, z));
    }
    let h = size / 2;
    let mut c = Cube {
        block: AIR,
        children: Some(Box::new(std::array::from_fn(|o| {
            build_cube(d, x + (o & 1) * h, y + (o >> 1 & 1) * h, z + (o >> 2 & 1) * h, h)
        }))),
    };
    c.try_merge();
    c
}

/// 在n（id是at）这棵子树里把id设成叶子b，然后往上合并
fn set_in(n: &mut Cube, at: CubeId, id: CubeId, b: BlockId) {
    if at.depth == id.depth {
        *n = Cube::leaf(b);
        return;
    }
    let child = id.parent_at(at.depth - 1);
    set_in(&mut n.split()[child.octant()], child, id, b);
    n.try_merge();
}
