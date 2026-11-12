// 定义世界：一棵八叉树，按chunk（边长16的cube）加载。
//
// 没加载的chunk什么都不知道；加载时如果存档里有就用存档的，否则交给Generator生成。
// 改动世界只有一个入口Set_cube，改动后会通知On_change注册的函数（engine用它重新生成网格）。
package world

import (
	"errors"
	"fmt"
	"os"
)

var (
	// chunk没有加载
	ErrNotLoaded = errors.New("world: chunk没有加载")
	// 深度不在[MIN_DEPTH, CHUNK_DEPTH]里
	ErrBadDepth = errors.New("world: cube的深度必须在MIN_DEPTH到CHUNK_DEPTH之间")
)

// 地形生成器
type Generator interface {
	// 生成一个chunk。pos是chunk的CubeID（深度是CHUNK_DEPTH），out一开始全是AIR
	Generate_chunk(pos CubeID, out *ChunkData)
	// 世界在竖直方向上有哪些chunk（chunk的Y，两端都包括）。范围外永远是空气，不会生成也不会保存
	Vertical_range() (min_chunk_y int64, max_chunk_y int64)
}

// 生成器填写的一个chunk
type ChunkData struct {
	// depth 0的方块，下标是Index(x, y, z)，x、y、z是chunk里的坐标0到15
	Blocks [CHUNK_SIZE * CHUNK_SIZE * CHUNK_SIZE]BlockID
	// 比1小的方块，在Blocks之后放上去
	Small []Placement
}

// 在某个位置放一个cube
type Placement struct {
	Id    CubeID
	Block BlockID
}

func Index(x, y, z int) int { return x + z*CHUNK_SIZE + y*CHUNK_SIZE*CHUNK_SIZE }

func (d *ChunkData) Get(x, y, z int) BlockID    { return d.Blocks[Index(x, y, z)] }
func (d *ChunkData) Set(x, y, z int, b BlockID) { d.Blocks[Index(x, y, z)] = b }

// 一个叶子cube：位置（包括大小）和方块
type Leaf struct {
	Id    CubeID
	Block BlockID
}

// 定义了世界的接口
type WorldInterface interface {
	// 加载一个chunk，已经加载了什么都不做
	Load_chunk(pos CubeID)
	// 卸载一个chunk，改动过的会先编码保存在内存里，之后再加载时用
	Unload_chunk(pos CubeID)
	Is_loaded(pos CubeID) bool
	// 把id这个cube整个变成方块b（AIR就是挖掉），里面原来的东西和场都删掉
	Set_cube(id CubeID, b BlockID) error
	// 找到id这个位置的cube，如果id在一个更大的叶子里面，返回那个叶子
	Find(id CubeID) (node Cube, at CubeID, loaded bool)
	// 包含物理坐标p的叶子cube
	Leaf_at(p PhysicalPosition3D) (Leaf, bool)
	// 沿射线找第一个满足hit的叶子cube
	Raycast(origin PhysicalPosition3D, dir PhysicalPosition3D, max_distance float64, hit func(BlockID) bool) (Hit, bool)
	// 修改一个cube的对应的field
	Field_write(field_name string, id CubeID, value string)
	Field_read(field_name string, id CubeID) (string, bool)
}

// 实现世界
type World struct {
	// 整棵八叉树的根。chunk以上的节点只当容器，不会合并
	Root      Cube
	generator Generator
	chunks    map[CubeID]*chunk
	// 卸载了的、改动过的chunk（编码后的数据）
	saved map[CubeID][]byte
	// 场：名字 -> cube -> 值
	fields    map[string]map[CubeID]string
	listeners []func(id CubeID)
}

type chunk struct {
	node *Cube
	// 和生成器生成的不一样了，存档时需要保存
	modified bool
}

var _ WorldInterface = (*World)(nil)

func New_world(generator Generator) *World {
	return &World{
		generator: generator,
		chunks:    map[CubeID]*chunk{},
		saved:     map[CubeID][]byte{},
		fields:    map[string]map[CubeID]string{},
	}
}

func (w *World) Generator() Generator { return w.generator }

// chunk在不在世界的竖直范围里
func (w *World) In_range(pos CubeID) bool {
	lo, hi := w.generator.Vertical_range()
	return pos.Y >= lo && pos.Y <= hi
}

// 世界改动时调用fn，id是被Set_cube改动的cube
func (w *World) On_change(fn func(id CubeID)) {
	w.listeners = append(w.listeners, fn)
}

func (w *World) Is_loaded(pos CubeID) bool {
	_, ok := w.chunks[pos]
	return ok
}

// 所有已加载的chunk
func (w *World) Loaded_chunks() []CubeID {
	out := make([]CubeID, 0, len(w.chunks))
	for pos := range w.chunks {
		out = append(out, pos)
	}
	return out
}

func (w *World) Load_chunk(pos CubeID) {
	if pos.Depth != CHUNK_DEPTH {
		panic(fmt.Sprintf("Load_chunk: %v不是chunk", pos))
	}
	if w.Is_loaded(pos) || !w.In_range(pos) {
		return
	}
	node := w.node_for(pos)
	ch := &chunk{node: node}
	if data, ok := w.saved[pos]; ok {
		if err := decode_chunk(data, node); err == nil {
			delete(w.saved, pos)
			ch.modified = true
			w.chunks[pos] = ch
			return
		} else {
			fmt.Fprintf(os.Stderr, "foxel: chunk %v的存档坏了，重新生成：%v\n", pos, err)
		}
	}
	var data ChunkData
	w.generator.Generate_chunk(pos, &data)
	*node = build_cube(&data, 0, 0, 0, CHUNK_SIZE)
	for _, p := range data.Small {
		if pos.Contains(p.Id) && p.Id.Depth >= MIN_DEPTH {
			set_in(node, pos, p.Id, p.Block)
		}
	}
	w.chunks[pos] = ch
}

// 从稠密数组建树，size是这个cube的边长（格子数），x、y、z是它最小的角
func build_cube(d *ChunkData, x, y, z, size int) Cube {
	if size == 1 {
		return Cube{Block: d.Get(x, y, z)}
	}
	h := size / 2
	c := Cube{Children: new([8]Cube)}
	for o := range 8 {
		c.Children[o] = build_cube(d, x+(o&1)*h, y+(o>>1&1)*h, z+(o>>2&1)*h, h)
	}
	c.try_merge()
	return c
}

func (w *World) Unload_chunk(pos CubeID) {
	ch, ok := w.chunks[pos]
	if !ok {
		return
	}
	if ch.modified {
		w.saved[pos] = encode_chunk(ch.node)
	}
	*ch.node = Cube{}
	delete(w.chunks, pos)
}

// 从根往下走到id，路上缺的节点都建好（chunk以上的节点用）
func (w *World) node_for(id CubeID) *Cube {
	n := &w.Root
	for d := Depth(IDENTITY_DEPTH - 1); d >= id.Depth; d-- {
		n.split()
		n = &n.Children[octant_in_parent(id.Parent_at(d))]
	}
	return n
}

// 根cube以原点为中心，不在深度32的格子上，所以第一层按正负号算卦限；再往下就是坐标的最低位
func octant_in_parent(id CubeID) int {
	if id.Depth != IDENTITY_DEPTH-1 {
		return id.Octant()
	}
	o := 0
	if id.X >= 0 {
		o |= 1
	}
	if id.Y >= 0 {
		o |= 2
	}
	if id.Z >= 0 {
		o |= 4
	}
	return o
}

// 找到id这个位置的cube：返回的node就是id这个cube；如果id在一个更大的叶子里面，返回那个叶子，at是它的id。
// 竖直范围外永远是空气（loaded是true）；chunk没有加载时loaded是false。
// id的深度必须在[MIN_DEPTH, CHUNK_DEPTH]里
func (w *World) Find(id CubeID) (node Cube, at CubeID, loaded bool) {
	pos := id.Chunk()
	ch, ok := w.chunks[pos]
	if !ok {
		return Cube{}, pos, !w.In_range(pos)
	}
	n, at := ch.node, pos
	for at.Depth > id.Depth && n.Children != nil {
		at = id.Parent_at(at.Depth - 1)
		n = &n.Children[at.Octant()]
	}
	return *n, at, true
}

// 包含物理坐标p的叶子cube（最小到MIN_DEPTH）
func (w *World) Leaf_at(p PhysicalPosition3D) (Leaf, bool) {
	node, at, ok := w.Find(Cube_id_at(p, MIN_DEPTH))
	return Leaf{at, node.Block}, ok
}

// depth 0格子(x, y, z)的方块。这个格子被分成更小的cube时，partial是true，b没有意义
func (w *World) Block_at(x, y, z int64) (b BlockID, partial bool, loaded bool) {
	node, _, loaded := w.Find(Block_id(x, y, z))
	return node.Block, node.Children != nil, loaded
}

// 把id这个cube整个变成方块b（AIR就是挖掉），里面原来的东西都删掉，里面的场也删掉。
// 改完后8个一样的兄弟cube会自动合并。chunk没有加载时返回ErrNotLoaded
func (w *World) Set_cube(id CubeID, b BlockID) error {
	if id.Depth < MIN_DEPTH || id.Depth > CHUNK_DEPTH {
		return ErrBadDepth
	}
	ch, ok := w.chunks[id.Chunk()]
	if !ok {
		return ErrNotLoaded
	}
	if node, _, _ := w.Find(id); node.Children == nil && node.Block == b {
		return nil
	}
	set_in(ch.node, id.Chunk(), id, b)
	ch.modified = true
	for _, f := range w.fields {
		for k := range f {
			if id.Contains(k) {
				delete(f, k)
			}
		}
	}
	for _, fn := range w.listeners {
		fn(id)
	}
	return nil
}

// 在root（id是root_id）这棵子树里把id设成叶子b，然后往上合并
func set_in(root *Cube, root_id CubeID, id CubeID, b BlockID) {
	var path [CHUNK_DEPTH - MIN_DEPTH]*Cube
	depth := 0
	n, at := root, root_id
	for at.Depth > id.Depth {
		n.split()
		path[depth] = n
		depth++
		at = id.Parent_at(at.Depth - 1)
		n = &n.Children[at.Octant()]
	}
	*n = Cube{Block: b}
	for i := depth - 1; i >= 0 && path[i].try_merge(); i-- {
	}
}

// 按先序遍历一个已加载chunk里的cube，fn返回false时不进入这个cube的子cube。chunk没有加载时返回false
func (w *World) Visit_chunk(pos CubeID, fn func(id CubeID, n *Cube) bool) bool {
	ch, ok := w.chunks[pos]
	if !ok {
		return false
	}
	visit(ch.node, pos, fn)
	return true
}

func visit(n *Cube, id CubeID, fn func(id CubeID, n *Cube) bool) {
	if !fn(id, n) || n.Children == nil {
		return
	}
	for o := range n.Children {
		visit(&n.Children[o], id.Child(o), fn)
	}
}
