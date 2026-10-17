// 定义一个空白的世界框架是什么样的
// 包括世界接口
package world

type WorldInterface[Self any] interface {
	//创建一个新的world
	//Init() Self
	//添加一个cell。
	//Add_cube()
	//添加最简单的cell，只在world里的tree添加一个cell，定义位置。fields和revfields里不添加任何性质。
	//Add_simple_cube()
}

type World struct {
	tree      Cell
	fields    map[string]Field[any]
	revfields map[string]RevField[any]
}

// 删除cube，返回是否成功.
// 删除tree里的并且删除每个fieldl里面对应的kv
func (world World) delete_cube(id CellID) bool {
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
