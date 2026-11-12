// 方块种类的注册表。world里每个cube只存一个BlockID，方块长什么样、能不能穿过都在这里查。
//
// 方块配置：同一种方块的所有cube共用一个Type，所以一百万块石头只有一份配置。
// 每个cube自己的数据（比如箱子里的东西）用world的场（Field_write）。
//
// mod注册新方块：在自己包的init()里调用block.Register，名字用"mod名:方块名"。
// 存档里按名字保存方块，所以注册顺序变了、新增了方块都没关系。
package block

import (
	"fmt"

	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

type Type struct {
	// 唯一的名字，比如"foxel:grass"
	Name string
	// 显示给玩家看的名字（只能用ASCII）
	Display string
	Images  canvas.Images
	// 有碰撞，玩家不能穿过
	Solid bool
	// 完全不透明：会挡住旁边方块的面。玻璃、树叶这种有透明像素的是false
	Opaque bool
	// 两个同样的、Opaque是false的方块挨着时，省掉中间的面。玻璃是true，树叶是false（能透过叶子看到里面的叶子）
	Cull_same bool
	// 玩家挖不掉，比如基岩
	Unbreakable bool
	// 不出现在物品栏里，比如空气
	Hidden bool
	// 自定义属性，mod可以随便用
	Props map[string]string
}

var (
	types   = []Type{{Name: "foxel:air", Display: "Air", Hidden: true}}
	by_name = map[string]world.BlockID{"foxel:air": world.AIR}
)

// 注册一种方块，返回它的BlockID。名字重复时panic
func Register(t Type) world.BlockID {
	if _, ok := by_name[t.Name]; ok {
		panic(fmt.Sprintf("block: %q注册了两次", t.Name))
	}
	if len(types) >= 1<<16 {
		panic("block: 方块种类太多了")
	}
	id := world.BlockID(len(types))
	types = append(types, t)
	by_name[t.Name] = id
	return id
}

// BlockID对应的方块种类，没注册过的当成空气
func Get(id world.BlockID) *Type {
	if int(id) >= len(types) {
		return &types[world.AIR]
	}
	return &types[id]
}

func By_name(name string) (world.BlockID, bool) {
	id, ok := by_name[name]
	return id, ok
}

// 所有注册过的方块（包括空气），按BlockID排序
func All() []world.BlockID {
	out := make([]world.BlockID, len(types))
	for i := range types {
		out[i] = world.BlockID(i)
	}
	return out
}

func Is_solid(id world.BlockID) bool  { return Get(id).Solid }
func Is_opaque(id world.BlockID) bool { return Get(id).Opaque }

// 普通的不透明实心方块
func Simple(name string, display string, images canvas.Images) Type {
	return Type{Name: name, Display: display, Images: images, Solid: true, Opaque: true}
}
