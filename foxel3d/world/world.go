// 定义一个空白的世界框架是什么样的
// 包括世界接口，同时实现了一个World
package world

import (
	"errors"
	"fmt"
	"math"
)

var (
	//未查询到错误。
	ErrCubeNotFound error = errors.New("not found the cube")
)

// 定义了世界的接口
type WorldInterface[Self any] interface {
	//删除一个cube
	Delete_cube(id CubeID) bool
	//添加最简单的cube，只在world里的tree添加一个cube，定义位置。fields和revfields里不添加任何性质。这个会分配一个cubeid，返回新建的CubeID
	Add_simple_cube(tree_pos TreePosition) CubeID
	//围绕cube中心旋转，这个会考虑到offset旋转（会同时修改offset），直接修改rotation，而不是叠加在旧的rotation上。
	//abs为true时rotation是cube在世界里的朝向（已经叠加了所有父cube的rotation），为false时是相对父cube的rotation。没有这个id返回ErrCubeNotFound
	Set_rotation(id CubeID, rotation Rotation3D, abs bool) error
	//重整，这个功能可以把一个范围内的选定的cubes在另一个地方重整到一个大cube里。
	Rebuild(id_list []CubeID, other_place TreePosition) CubeID
	//修改一个cube的对应的field
	Field_write(field_name string, id CubeID, value any)
	//用tree_position获得一个cube（tree_pos 的offset没有用）如果没有就返回error
	Get_cube_by_tree_pos(tree_pos TreePosition) (*Cube, error)
	//用物理位置获得cube，没有的话返回错误。只会获得对应depth的cube，如果层数不对也会返回错误。另外不考虑offset和rotation，只是看格子的位置（这样算法更快）。
	Get_cube_by_phy_pos(phy_pos PhysicalPosition3D, depth Depth) (*Cube, error)
	//通过id获得某个cube中心的物理位置。（我们会叠加所有的父cube的offset来计算，父cube的rotation也会带着子cube绕父cube中心转），没有这个id返回ErrCubeNotFound
	Get_cube_phy_pos_by_id(id CubeID) (PhysicalPosition3D, error)
	//通过id获得cube的rotation，需要叠加所有的父cube的rotation来计算，没有这个id返回ErrCubeNotFound
	Get_cube_rotation_by_id(id CubeID) (Rotation3D, error)
	// TODO:我们准备添加一些和canvas交互的
}

// 实现世界
type World struct {
	tree      Cube
	fields    map[string]Field[any]
	revfields map[string]RevField[any]
}

func New_World() World {
	return World{
		tree:      Cube{},
		fields:    make(map[string]Field[any]),
		revfields: make(map[string]RevField[any]),
	}
}

// 删除cube，返回是否成功.
// 删除tree里的并且删除每个fields里面对应的kv
func (world *World) Delete_cube(id CubeID) bool {
	//遍历fields
	for name, field := range world.fields {
		//是否存在返场
		revfield, ok := world.revfields[name]
		if !ok {
			//没找到对应的返场，只删除field
			field.Del(id)
			continue
		}
		value, ok := field.Get(world, id)
		if ok {
			//我们找到了field中的值现在删除
			field.Del(id)
			revfield.Del(world, value)
		}
	}
	return true
}

// 指针接收者：要修改world.tree，值接收者的修改会丢失
// 路径上缺少的中间层cube会一并创建，每个新建的cube都分配id；tree_pos的offset只写给最终位置上新建的cube
//
// 在对应的tree_pos位置上添加一个cube，并且返回新cube的id
//
// 如果位置上已经有cube就不动了，直接返回这个位置的CubeID
//
// 有可能生成多层cube，只用返回tree_pos位置上的cubeID
func (world *World) Add_simple_cube(tree_pos TreePosition) CubeID {
	node := &world.tree
	created := false
	for _, o := range tree_pos.Pos {
		if o == 8 {
			break
		}
		if o < 0 || o > 8 {
			panic(fmt.Sprintf("Add_simple_cube: invalid octant %d in %v", o, tree_pos.Pos))
		}
		child := node.Children[o]
		created = child == nil
		if created {
			child = &Cube{Id: newCubeID()}
			node.Children[o] = child
		}
		node = child
	}
	if created {
		node.Offset = [3]float64{tree_pos.offset.x, tree_pos.offset.y, tree_pos.offset.z}
	}
	return node.Id
}

// abs为false时rotation是相对父cube的旋转，直接写进Cube.Rotation
// abs为true时rotation是世界里的朝向W，先抵消父cube们的累计旋转P，换算成相对父cube的 P⁻¹·W 再写入
// 转动中心是cube所在格子的中心：从旧旋转转到新旋转的差 ΔR = R新·R旧⁻¹ 同样作用在Offset上
// 子cube在这个cube的坐标系里，会被Rotation自动带着转，不用改
// 没有这个id返回ErrCubeNotFound
func (world *World) Set_rotation(id CubeID, rotation Rotation3D, abs bool) error {
	path, _, ok := world.findPath(id)
	if !ok {
		return ErrCubeNotFound
	}
	c := path[len(path)-1]
	newRot := [3]float64{rotation.X, rotation.Y, rotation.Z}
	if abs {
		parentRot := quatIdentity
		for _, p := range path[:len(path)-1] {
			parentRot = parentRot.mul(quatFromRotation(p.Rotation))
		}
		r := parentRot.conj().mul(quatFromRotation(newRot)).toRotation()
		newRot = [3]float64{r.X, r.Y, r.Z}
	}
	delta := quatFromRotation(newRot).mul(quatFromRotation(c.Rotation).conj())
	c.Offset = delta.rotate(c.Offset)
	c.Rotation = newRot
	return nil
}

func (world *World) Rebuild(id_list []CubeID, other_place TreePosition) CubeID {
	//todo
	panic("todo: the function is not supported yet")
}

func (world *World) Field_write(field_name string, id CubeID, value any) {
	//todo
}

// 沿树坐标从根往下走，路径上任何一层没有cube都返回ErrCubeNotFound
// 树坐标里没有非8数字时返回根cube；出现0-8以外的数字返回错误
func (world *World) Get_cube_by_tree_pos(tree_pos TreePosition) (*Cube, error) {
	node := &world.tree
	for _, o := range tree_pos.Pos {
		if o == 8 {
			break
		}
		if o < 0 || o > 8 {
			return nil, fmt.Errorf("Get_cube_by_tree_pos: invalid octant %d in %v", o, tree_pos.Pos)
		}
		node = node.Children[o]
		if node == nil {
			return nil, ErrCubeNotFound
		}
	}
	return node, nil
}

// 找物理位置所在的、边长为2^depth的那个格子里的cube，必须正好在这个深度上有cube，不会返回更大或更小的cube
// 只按格子找，不考虑cube的Offset和Rotation：cube偏移到别的格子里也找不到
// 物理位置在根cube外面时返回ErrCubeNotFound
func (world *World) Get_cube_by_phy_pos(phy_pos PhysicalPosition3D, depth Depth) (*Cube, error) {
	if !inRootCube(phy_pos.X) || !inRootCube(phy_pos.Y) || !inRootCube(phy_pos.Z) {
		return nil, ErrCubeNotFound
	}
	return world.Get_cube_by_tree_pos(Phy2tree(phy_pos, depth))
}

// 从根往下逐层算：子cube中心 = 父cube中心 + 父cube累计旋转作用在(格子偏移 + 子cube的Offset)上
// Offset以该cube自己的边长为单位；没有旋转时结果和Tree2phy一致
// 目前靠遍历整棵树找id，复杂度O(cube数)
func (world *World) Get_cube_phy_pos_by_id(id CubeID) (PhysicalPosition3D, error) {
	path, octants, ok := world.findPath(id)
	if !ok {
		return PhysicalPosition3D{}, ErrCubeNotFound
	}
	root := path[0]
	side := math.Ldexp(1, IDENTITY_DEPTH)
	pos := [3]float64{root.Offset[0] * side, root.Offset[1] * side, root.Offset[2] * side}
	rot := quatFromRotation(root.Rotation)
	for i, o := range octants {
		child := path[i+1]
		side /= 2
		local := [3]float64{
			octantSign(o&1)*side/2 + child.Offset[0]*side,
			octantSign(o&2)*side/2 + child.Offset[1]*side,
			octantSign(o&4)*side/2 + child.Offset[2]*side,
		}
		d := rot.rotate(local)
		pos = [3]float64{pos[0] + d[0], pos[1] + d[1], pos[2] + d[2]}
		rot = rot.mul(quatFromRotation(child.Rotation))
	}
	return PhysicalPosition3D{X: pos[0], Y: pos[1], Z: pos[2]}, nil
}

// 累计旋转 = 根的旋转 ∘ ... ∘ 父cube的旋转 ∘ 自己的旋转（先转自己，再依次被外层带着转）
// 返回的旋转角在[0, pi]内
func (world *World) Get_cube_rotation_by_id(id CubeID) (Rotation3D, error) {
	path, _, ok := world.findPath(id)
	if !ok {
		return Rotation3D{}, ErrCubeNotFound
	}
	rot := quatIdentity
	for _, c := range path {
		rot = rot.mul(quatFromRotation(c.Rotation))
	}
	return rot.toRotation(), nil
}

// 深度优先找id，返回从根到这个cube的所有cube（含根和它自己）以及每一步走的卦限
func (world *World) findPath(id CubeID) (path []*Cube, octants []int8, ok bool) {
	var walk func(c *Cube) bool
	walk = func(c *Cube) bool {
		path = append(path, c)
		if c.Id == id {
			return true
		}
		for o, child := range c.Children {
			if child == nil {
				continue
			}
			octants = append(octants, int8(o))
			if walk(child) {
				return true
			}
			octants = octants[:len(octants)-1]
		}
		path = path[:len(path)-1]
		return false
	}
	ok = walk(&world.tree)
	return path, octants, ok
}
