package world

import (
	"math"
	"testing"
)

const stone BlockID = 1
const dirt BlockID = 2

// y < 0是石头，其他是空气
type flat struct{}

func (flat) Vertical_range() (int64, int64) { return -2, 1 }
func (flat) Generate_chunk(pos CubeID, out *ChunkData) {
	for y := range CHUNK_SIZE {
		if pos.Y*CHUNK_SIZE+int64(y) >= 0 {
			continue
		}
		for z := range CHUNK_SIZE {
			for x := range CHUNK_SIZE {
				out.Set(x, y, z, stone)
			}
		}
	}
}

func load_around(w *World) {
	for y := int64(-2); y <= 1; y++ {
		for z := int64(-1); z <= 0; z++ {
			for x := int64(-1); x <= 0; x++ {
				w.Load_chunk(CubeID{x, y, z, CHUNK_DEPTH})
			}
		}
	}
}

func TestCubeID(t *testing.T) {
	id := Cube_id_at(PhysicalPosition3D{-0.3, 1.7, 5.2}, 0)
	if id != (CubeID{-1, 1, 5, 0}) {
		t.Fatal(id)
	}
	if id.Chunk() != (CubeID{-1, 0, 0, CHUNK_DEPTH}) {
		t.Fatal(id.Chunk())
	}
	small := Cube_id_at(PhysicalPosition3D{-0.3, 1.7, 5.2}, -2)
	if !id.Contains(small) || small.Contains(id) || small.Parent_at(0) != id {
		t.Fatal(small)
	}
	for o := range 8 {
		if c := id.Child(o); c.Octant() != o || c.Parent_at(0) != id {
			t.Fatal(o, c)
		}
	}
	if octant_in_parent(CubeID{-1, 0, -1, IDENTITY_DEPTH - 1}) != 2 {
		t.Fatal("top octant")
	}
}

func TestGenerateAndMerge(t *testing.T) {
	w := New_world(flat{})
	load_around(w)
	// 全是石头的chunk合并成了一个叶子
	n, at, ok := w.Find(Block_id(-3, -5, -3))
	if !ok || n.Block != stone || at.Depth != CHUNK_DEPTH {
		t.Fatal(n, at, ok)
	}
	if b, partial, _ := w.Block_at(0, 0, 0); b != AIR || partial {
		t.Fatal("air")
	}
	// 范围外是空气
	if l, ok := w.Leaf_at(PhysicalPosition3D{0, 1000, 0}); !ok || l.Block != AIR {
		t.Fatal("above")
	}
	// 没加载的chunk
	if _, ok := w.Leaf_at(PhysicalPosition3D{100, 0, 0}); ok {
		t.Fatal("not loaded")
	}
}

func TestSetCube(t *testing.T) {
	w := New_world(flat{})
	load_around(w)
	var changed []CubeID
	w.On_change(func(id CubeID) { changed = append(changed, id) })

	// 挖掉一个方块：石头chunk被分开
	if err := w.Set_cube(Block_id(2, -1, 3), AIR); err != nil {
		t.Fatal(err)
	}
	if b, _, _ := w.Block_at(2, -1, 3); b != AIR {
		t.Fatal("dig")
	}
	if b, _, _ := w.Block_at(3, -1, 3); b != stone {
		t.Fatal("neighbor")
	}
	// 再填回去，又合并成一个叶子
	w.Set_cube(Block_id(2, -1, 3), stone)
	if _, at, _ := w.Find(Block_id(2, -1, 3)); at.Depth != CHUNK_DEPTH {
		t.Fatal("not merged", at)
	}
	// 放一个1/4的小方块
	small := Cube_id_at(PhysicalPosition3D{0.1, 0.1, 0.1}, MIN_DEPTH)
	w.Field_write("test:x", small, "hello")
	w.Set_cube(small, dirt)
	if _, partial, _ := w.Block_at(0, 0, 0); !partial {
		t.Fatal("partial")
	}
	if l, _ := w.Leaf_at(PhysicalPosition3D{0.2, 0.2, 0.2}); l.Block != dirt || l.Id != small {
		t.Fatal(l)
	}
	if l, _ := w.Leaf_at(PhysicalPosition3D{0.3, 0.2, 0.2}); l.Block != AIR {
		t.Fatal(l)
	}
	// 场被覆盖时删掉
	w.Field_write("test:x", small, "hello")
	w.Set_cube(Block_id(0, 0, 0), AIR)
	if _, ok := w.Field_read("test:x", small); ok {
		t.Fatal("field not deleted")
	}
	if len(changed) != 4 {
		t.Fatal(changed)
	}
	if w.Set_cube(Block_id(100, 0, 0), stone) != ErrNotLoaded {
		t.Fatal("not loaded")
	}
	if w.Set_cube(CubeID{0, 0, 0, -3}, stone) != ErrBadDepth {
		t.Fatal("depth")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	w := New_world(flat{})
	load_around(w)
	w.Set_cube(Block_id(1, 0, 1), dirt)
	w.Set_cube(Cube_id_at(PhysicalPosition3D{5.6, 0.1, 1}, -1), stone)
	saved := w.Saved_chunks()
	if len(saved) != 1 {
		t.Fatal(len(saved))
	}
	// 卸载再加载
	w.Unload_chunk(CubeID{0, 0, 0, CHUNK_DEPTH})
	w.Load_chunk(CubeID{0, 0, 0, CHUNK_DEPTH})
	if b, _, _ := w.Block_at(1, 0, 1); b != dirt {
		t.Fatal("reload")
	}
	// 新世界读存档，dirt和stone换了编号
	w2 := New_world(flat{})
	swap := func(b BlockID) BlockID { return map[BlockID]BlockID{stone: dirt, dirt: stone}[b] }
	if w2.Set_saved_chunks(saved, swap) != 0 {
		t.Fatal("dropped")
	}
	load_around(w2)
	if b, _, _ := w2.Block_at(1, 0, 1); b != stone {
		t.Fatal("remap")
	}
	if l, _ := w2.Leaf_at(PhysicalPosition3D{5.6, 0.1, 1}); l.Block != dirt || l.Id.Depth != -1 {
		t.Fatal(l)
	}
	// 坏数据
	if decode_chunk([]byte{0, 1}, &Cube{}) == nil {
		t.Fatal("bad data accepted")
	}
}

func TestRaycast(t *testing.T) {
	w := New_world(flat{})
	load_around(w)
	solid := func(b BlockID) bool { return b != AIR }
	// 往下看：打到y=0的地面（石头的上面）
	hit, ok := w.Raycast(PhysicalPosition3D{0.5, 3.3, 0.5}, PhysicalPosition3D{0, -1, 0}, 10, solid)
	if !ok || hit.Normal != [3]int64{0, 1, 0} || math.Abs(hit.Point.Y) > 1e-9 || math.Abs(hit.Distance-3.3) > 1e-9 {
		t.Fatal(hit, ok)
	}
	// 斜着打到一个小方块的侧面
	small := Cube_id_at(PhysicalPosition3D{3.1, 0.1, 0.6}, MIN_DEPTH)
	w.Set_cube(small, dirt)
	hit, ok = w.Raycast(PhysicalPosition3D{1, 0.15, 0.6}, PhysicalPosition3D{1, 0, 0}, 10, solid)
	if !ok || hit.Leaf.Id != small || hit.Normal != [3]int64{-1, 0, 0} || math.Abs(hit.Point.X-3) > 1e-9 {
		t.Fatal(hit, ok)
	}
	// 太远
	if _, ok := w.Raycast(PhysicalPosition3D{0.5, 3.3, 0.5}, PhysicalPosition3D{0, -1, 0}, 2, solid); ok {
		t.Fatal("too far")
	}
}
