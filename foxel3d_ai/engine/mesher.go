// 把一个chunk变成canvas的网格：只生成露在外面的面（旁边是不透明方块的面被省掉），
// 大的cube按1x1的格子切开，所以贴图在大cube上也是一格一张。
package engine

import (
	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

// cube的6个面
type face struct {
	normal [3]int64
	// 4个角（从外面看逆时针：左上、左下、右下、右上），每个坐标0表示取最小值，1表示取最大值
	corners [4][3]int
	// 明暗：上面最亮，下面最暗
	tint uint8
	// 一个角在它所在的1x1格子里的相对位置(lx, ly, lz) -> 图片上的(u, v)
	uv func(lx, ly, lz float64) (float64, float64)
	// 这个面用Images里的哪张图片
	image func(img *canvas.Images) canvas.Image
}

var faces = [6]face{
	{[3]int64{1, 0, 0}, [4][3]int{{1, 1, 1}, {1, 0, 1}, {1, 0, 0}, {1, 1, 0}}, 170,
		func(lx, ly, lz float64) (float64, float64) { return 1 - lz, 1 - ly },
		func(img *canvas.Images) canvas.Image { return img.East }},
	{[3]int64{-1, 0, 0}, [4][3]int{{0, 1, 0}, {0, 0, 0}, {0, 0, 1}, {0, 1, 1}}, 170,
		func(lx, ly, lz float64) (float64, float64) { return lz, 1 - ly },
		func(img *canvas.Images) canvas.Image { return img.West }},
	{[3]int64{0, 1, 0}, [4][3]int{{0, 1, 0}, {0, 1, 1}, {1, 1, 1}, {1, 1, 0}}, 255,
		func(lx, ly, lz float64) (float64, float64) { return lx, lz },
		func(img *canvas.Images) canvas.Image { return img.Up }},
	{[3]int64{0, -1, 0}, [4][3]int{{0, 0, 1}, {0, 0, 0}, {1, 0, 0}, {1, 0, 1}}, 130,
		func(lx, ly, lz float64) (float64, float64) { return lx, 1 - lz },
		func(img *canvas.Images) canvas.Image { return img.Down }},
	{[3]int64{0, 0, 1}, [4][3]int{{0, 1, 1}, {0, 0, 1}, {1, 0, 1}, {1, 1, 1}}, 210,
		func(lx, ly, lz float64) (float64, float64) { return lx, 1 - ly },
		func(img *canvas.Images) canvas.Image { return img.South }},
	{[3]int64{0, 0, -1}, [4][3]int{{1, 1, 0}, {1, 0, 0}, {0, 0, 0}, {0, 1, 0}}, 210,
		func(lx, ly, lz float64) (float64, float64) { return 1 - lx, 1 - ly },
		func(img *canvas.Images) canvas.Image { return img.North }},
}

// b旁边是nb时，b朝nb的那个面是不是被挡住了
func hidden(b world.BlockID, nb world.BlockID) bool {
	if nb == world.AIR {
		return false
	}
	if block.Is_opaque(nb) {
		return true
	}
	return nb == b && block.Get(b).Cull_same
}

// 一个面的四边形。lo、hi是cube相对chunk原点的两个对角，cell是它所在的1x1格子的最小角
func face_quad(f *face, t *block.Type, lo, hi, cell [3]float64) canvas.Quad {
	q := canvas.Quad{Img: f.image(&t.Images), Tint: canvas.Color{R: f.tint, G: f.tint, B: f.tint, A: 255}}
	for k, c := range f.corners {
		var p [3]float64
		for a := range 3 {
			p[a] = lo[a]
			if c[a] == 1 {
				p[a] = hi[a]
			}
		}
		q.Corners[k] = canvas.Canvas3D{X: p[0], Y: p[1], Z: p[2]}
		u, v := f.uv(p[0]-cell[0], p[1]-cell[1], p[2]-cell[2])
		q.UV[k] = canvas.Canvas2D{X: u, Y: v}
	}
	return q
}

// 1x1格子（最小角是cell）的f面外面紧挨着的是被分成小cube的格子时，看那一层小cube是不是把这个面整个挡住了
func face_covered(w *world.World, b world.BlockID, cell world.PhysicalPosition3D, f *face) bool {
	const n = 1 << -world.MIN_DEPTH
	// 法线所在的轴，和面上的另外两个轴
	an := 0
	for a := range 3 {
		if f.normal[a] != 0 {
			an = a
		}
	}
	t1, t2 := (an+1)%3, (an+2)%3
	for i := range n {
		for j := range n {
			// 面上第(i, j)个最小格子的中心，往外挪半个最小格子
			var p [3]float64
			p[t1], p[t2] = (float64(i)+0.5)/n, (float64(j)+0.5)/n
			p[an] = -0.5 / n
			if f.normal[an] > 0 {
				p[an] = 1 + 0.5/n
			}
			nb, ok := w.Leaf_at(cell.Add(world.PhysicalPosition3D{X: p[0], Y: p[1], Z: p[2]}))
			if !ok || !hidden(b, nb.Block) {
				return false
			}
		}
	}
	return true
}

const pad = world.CHUNK_SIZE + 2

// chunk和它外面一圈的1x1格子，下标是x、y、z加1
type cell_grid struct {
	block   [pad * pad * pad]world.BlockID
	partial [pad * pad * pad]bool
}

func grid_index(x, y, z int) int { return (x + 1) + (z+1)*pad + (y+1)*pad*pad }

// 生成pos这个chunk的所有面，坐标相对chunk最小的角。chunk没有加载时返回nil
func Build_quads(w *world.World, pos world.CubeID) []canvas.Quad {
	var g cell_grid
	type sub struct {
		id world.CubeID
		n  *world.Cube
	}
	var subdivided []sub
	origin := pos.Min()
	x0, y0, z0 := pos.X*world.CHUNK_SIZE, pos.Y*world.CHUNK_SIZE, pos.Z*world.CHUNK_SIZE

	// chunk里面：边长>=1的叶子按格子填满，分成更小cube的格子记下来单独处理
	loaded := w.Visit_chunk(pos, func(id world.CubeID, n *world.Cube) bool {
		if id.Depth < 0 {
			return false
		}
		s := int64(1) << id.Depth
		lx, ly, lz := int(id.X*s-x0), int(id.Y*s-y0), int(id.Z*s-z0)
		if n.Children == nil {
			for y := ly; y < ly+int(s); y++ {
				for z := lz; z < lz+int(s); z++ {
					for x := lx; x < lx+int(s); x++ {
						g.block[grid_index(x, y, z)] = n.Block
					}
				}
			}
			return false
		}
		if id.Depth == 0 {
			g.partial[grid_index(lx, ly, lz)] = true
			subdivided = append(subdivided, sub{id, n})
			return false
		}
		return true
	})
	if !loaded {
		return nil
	}

	// 外面一圈：只需要6个面上的格子
	border := func(lx, ly, lz int) {
		b, partial, _ := w.Block_at(x0+int64(lx), y0+int64(ly), z0+int64(lz))
		i := grid_index(lx, ly, lz)
		g.block[i], g.partial[i] = b, partial
	}
	for i := range world.CHUNK_SIZE {
		for j := range world.CHUNK_SIZE {
			border(-1, i, j)
			border(world.CHUNK_SIZE, i, j)
			border(i, -1, j)
			border(i, world.CHUNK_SIZE, j)
			border(i, j, -1)
			border(i, j, world.CHUNK_SIZE)
		}
	}

	var quads []canvas.Quad
	for y := range world.CHUNK_SIZE {
		for z := range world.CHUNK_SIZE {
			for x := range world.CHUNK_SIZE {
				i := grid_index(x, y, z)
				b := g.block[i]
				if b == world.AIR || g.partial[i] {
					continue
				}
				t := block.Get(b)
				lo := [3]float64{float64(x), float64(y), float64(z)}
				hi := [3]float64{lo[0] + 1, lo[1] + 1, lo[2] + 1}
				for fi := range faces {
					f := &faces[fi]
					j := grid_index(x+int(f.normal[0]), y+int(f.normal[1]), z+int(f.normal[2]))
					if g.partial[j] {
						if face_covered(w, b, origin.Add(world.PhysicalPosition3D{X: lo[0], Y: lo[1], Z: lo[2]}), f) {
							continue
						}
					} else if hidden(b, g.block[j]) {
						continue
					}
					quads = append(quads, face_quad(f, t, lo, hi, lo))
				}
			}
		}
	}

	// 分成小cube的格子：每个小cube查一下面外面紧挨着的叶子，
	// 那个叶子不比它小（能把这个面整个挡住）而且挡得住时才省掉这个面
	var small func(id world.CubeID, n *world.Cube, cell [3]float64)
	small = func(id world.CubeID, n *world.Cube, cell [3]float64) {
		if n.Children != nil {
			for o := range n.Children {
				small(id.Child(o), &n.Children[o], cell)
			}
			return
		}
		if n.Block == world.AIR {
			return
		}
		t := block.Get(n.Block)
		half := id.Side() / 2
		center := id.Center()
		lo3, hi3 := id.Min().Sub(origin), id.Max().Sub(origin)
		lo := [3]float64{lo3.X, lo3.Y, lo3.Z}
		hi := [3]float64{hi3.X, hi3.Y, hi3.Z}
		for fi := range faces {
			f := &faces[fi]
			d := half + world.MIN_DEPTH.Side()/2
			p := center.Add(world.PhysicalPosition3D{X: float64(f.normal[0]) * d, Y: float64(f.normal[1]) * d, Z: float64(f.normal[2]) * d})
			if nb, ok := w.Leaf_at(p); ok && nb.Id.Depth >= id.Depth && hidden(n.Block, nb.Block) {
				continue
			}
			quads = append(quads, face_quad(f, t, lo, hi, cell))
		}
	}
	for _, s := range subdivided {
		c := s.id.Min().Sub(origin)
		small(s.id, s.n, [3]float64{c.X, c.Y, c.Z})
	}
	return quads
}
