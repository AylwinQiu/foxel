package world

// 方块种类的编号，每种编号是什么方块在block包里注册。world只关心编号
type BlockID uint16

// 空气，也就是没有方块
const AIR BlockID = 0

// 八叉树的节点。
//
// Children是nil时这是叶子：整个cube都是Block（AIR表示空的）。
// Children不是nil时cube分成8个子cube，Block没有意义。子cube的下标是卦限：x正为1，y正为2，z正为4。
//
// 8个子cube都是同样的叶子时会自动合并回一个叶子，所以一大片石头只占一个节点。
// cube的位置不存在节点里，由它在树里的路径决定（见CubeID）。
type Cube struct {
	Block    BlockID
	Children *[8]Cube
}

func (c *Cube) Is_leaf() bool { return c.Children == nil }

// 把叶子分成8个和它一样的子cube
func (c *Cube) split() {
	if c.Children != nil {
		return
	}
	c.Children = new([8]Cube)
	for i := range c.Children {
		c.Children[i].Block = c.Block
	}
}

// 8个子cube都是同样的叶子时合并成一个叶子，返回是否合并了
func (c *Cube) try_merge() bool {
	if c.Children == nil {
		return false
	}
	b := c.Children[0].Block
	for i := range c.Children {
		if ch := &c.Children[i]; ch.Children != nil || ch.Block != b {
			return false
		}
	}
	*c = Cube{Block: b}
	return true
}
