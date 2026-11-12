// 场：给cube附加的自定义数据，比如箱子里装了什么、mod的状态。
//
// 场按名字分组（mod请用"mod名:场名"，避免和别人重名），key是CubeID，value是字符串，
// 需要结构化的数据时自己编码成JSON之类的。
// 场会跟着存档保存；cube被Set_cube覆盖（包括挖掉）时，它里面的场会被删掉。
package world

func (w *World) Field_write(field_name string, id CubeID, value string) {
	f, ok := w.fields[field_name]
	if !ok {
		f = map[CubeID]string{}
		w.fields[field_name] = f
	}
	f[id] = value
}

func (w *World) Field_read(field_name string, id CubeID) (string, bool) {
	v, ok := w.fields[field_name][id]
	return v, ok
}

// 不存在时什么都不做
func (w *World) Field_delete(field_name string, id CubeID) {
	delete(w.fields[field_name], id)
}

// 某个场里的所有数据，返回的是副本
func (w *World) Field_all(field_name string) map[CubeID]string {
	out := map[CubeID]string{}
	for k, v := range w.fields[field_name] {
		out[k] = v
	}
	return out
}

// 所有场（存档用），返回的是副本
func (w *World) Fields() map[string]map[CubeID]string {
	out := map[string]map[CubeID]string{}
	for name := range w.fields {
		out[name] = w.Field_all(name)
	}
	return out
}

// 换成存档里的场
func (w *World) Set_fields(fields map[string]map[CubeID]string) {
	w.fields = map[string]map[CubeID]string{}
	for name, f := range fields {
		for k, v := range f {
			w.Field_write(name, k, v)
		}
	}
}
