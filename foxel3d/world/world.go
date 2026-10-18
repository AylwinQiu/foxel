// 定义一个空白的世界框架是什么样的
// 包括世界接口，同时实现了一个World
package world

import (
	"errors"
	"fmt"
)

var (
	//未查询到错误。
	ErrCubeNotFound error = errors.New("not found the error")
)

// 定义了世界的接口
type WorldInterface[Self any] interface {
	//删除一个cube
	Delete_cube(id CubeID) bool
	//添加最简单的cube，只在world里的tree添加一个cube，定义位置。fields和revfields里不添加任何性质。这个会分配一个cubeid，返回新建的CubeID
	Add_simple_cube(tree_pos TreePosition) CubeID
	//围绕cube中心旋转，这个会考虑到offset旋转。
	Rotate(id CubeID, rotation Rotation3D)
	//重整，这个功能可以把一个范围内的选定的cubes在另一个地方重整到一个大cube里。
	Rebuild(id_list []CubeID, other_place TreePosition) CubeID
	//修改一个cube的对应的field
	Field_write(field_name string, id CubeID, value any)
	//用tree_position获得一个cube（tree_pos 的offset没有用）如果没有就返回error
	Get_cube_by_tree_pos(tree_pos TreePosition) (*Cube, error)
	//用物理位置获得cube，没有的话返回错误
	Get_cube_by_phy_pos(phy_pos PhysicalPosition3D, depth Depth)
	//用ID获得cube, 没有找到会返回ErrCubeNotFound错误。
	Get_cube_by_id(cubeid CubeID) (*Cube, error)
	// TODO:我们准备添加一些和canvas交互的
}

// 实现世界
type World struct {
	tree      Cube
	fields    map[string]Field[any]
	revfields map[string]RevField[any]
}

// 删除cube，返回是否成功.
// 删除tree里的并且删除每个fieldl里面对应的kv
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
		child := node.children[o]
		created = child == nil
		if created {
			child = &Cube{id: newCubeID()}
			node.children[o] = child
		}
		node = child
	}
	if created {
		node.offset = [3]float64{tree_pos.offset.x, tree_pos.offset.y, tree_pos.offset.z}
	}
	return node.id
}

func (world *World) Rotate(id CubeID, rotation Rotation3D) {
	//todo
}

func (world *World) Rebuild(id_list []CubeID, other_place TreePosition) CubeID {
	//todo
	panic("todo")
}

func (world *World) Field_write(field_name string, id CubeID, value any) {
	//todo
}

func (world *World) Get_cube_by_tree_pos(tree_pos TreePosition) (*Cube, error) {
	panic("todo")
}

func (world *World) Get_cube_by_phy_pos(phy_pos PhysicalPosition3D, depth Depth) {
	panic("todo")
}

func (world *World) Get_cube_by_id(cubeid CubeID) (*Cube, error) {
	panic("todo")
}
