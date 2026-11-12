/// 方块种类的编号，每种编号是什么方块在block模块的注册表里查。world只关心编号
pub type BlockId = u16;

/// 空气，也就是没有方块
pub const AIR: BlockId = 0;

/// 八叉树的节点，16字节。
///
/// children是None时这是叶子：整个cube都是block（AIR表示空的）。
/// children不是None时cube分成8个子cube，block没有意义。子cube的下标是卦限：x正为1，y正为2，z正为4。
///
/// 8个子cube都是同样的叶子时会自动合并回一个叶子，所以一大片石头只占一个节点。
/// cube的位置不存在节点里，由它在树里的路径决定（见CubeId）。
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Cube {
    pub block: BlockId,
    pub children: Option<Box<[Cube; 8]>>,
}

impl Cube {
    pub const fn leaf(block: BlockId) -> Cube {
        Cube { block, children: None }
    }

    pub fn is_leaf(&self) -> bool {
        self.children.is_none()
    }

    /// 把叶子分成8个和它一样的子cube
    pub(crate) fn split(&mut self) -> &mut [Cube; 8] {
        let b = self.block;
        self.children.get_or_insert_with(|| Box::new(std::array::from_fn(|_| Cube::leaf(b))))
    }

    /// 8个子cube都是同样的叶子时合并成一个叶子，返回是否合并了
    pub(crate) fn try_merge(&mut self) -> bool {
        let Some(children) = &self.children else {
            return false;
        };
        let b = children[0].block;
        if children.iter().any(|c| c.children.is_some() || c.block != b) {
            return false;
        }
        *self = Cube::leaf(b);
        true
    }
}
