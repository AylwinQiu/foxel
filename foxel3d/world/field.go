//该文件定义了场和返场的接口。

package world

// 定义场的接口
type Field[T any] interface {
	//通过id返回场的类型同时返回是否成功
	Get(WorldInterface[any], CellID) (T, bool)
	//添加kv对，同时返回是否成功
	Add(WorldInterface[any], CellID, T) bool
	//如果不存在kv，什么都不用做
	Del(CellID)
}

// 定义反向场的接口
type RevField[T any] interface {
	//返回场的id和是否成功
	Get(WorldInterface[any], T) (CellID, bool)
	//添加T和id的键值对，然后返回是否成功
	Add(WorldInterface[any], CellID, T) bool
	//如果不存在kv，什么都不用做
	Del(WorldInterface[any], T)
}
