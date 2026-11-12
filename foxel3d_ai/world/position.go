// 物理坐标、深度和CubeID。
//
// 坐标轴：x正方向是东，y正方向是上，z正方向是南（北是-z），右手系。
// 整个世界是一个八叉树：根cube以原点为中心，边长2^IDENTITY_DEPTH。
// 深度为d的cube边长是2^d：depth 0是边长1的普通方块，depth越大cube越大，越小cube越小。
package world

import (
	"fmt"
	"math"
)

// 根cube的深度，边长2^32，所以坐标必须在[-2^31, 2^31)以内
const IDENTITY_DEPTH = 32

// cube的深度，边长是2^Depth
type Depth int

const (
	// 最小的cube：边长1/4
	MIN_DEPTH Depth = -2
	// 一个chunk是边长16的cube，世界按chunk加载、保存、画
	CHUNK_DEPTH Depth = 4
	CHUNK_SIZE        = 1 << CHUNK_DEPTH
)

// 这个深度的cube的边长
func (d Depth) Side() float64 { return math.Ldexp(1, int(d)) }

// 物理坐标
type PhysicalPosition3D struct {
	X float64
	Y float64
	Z float64
}

func (a PhysicalPosition3D) Add(b PhysicalPosition3D) PhysicalPosition3D {
	return PhysicalPosition3D{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

func (a PhysicalPosition3D) Sub(b PhysicalPosition3D) PhysicalPosition3D {
	return PhysicalPosition3D{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

func (a PhysicalPosition3D) Scale(k float64) PhysicalPosition3D {
	return PhysicalPosition3D{a.X * k, a.Y * k, a.Z * k}
}

func (a PhysicalPosition3D) Length() float64 {
	return math.Sqrt(a.X*a.X + a.Y*a.Y + a.Z*a.Z)
}

// 旋转
// 这个是旋转的轴向，使用右手定则，向量模长代表的是旋转弧度，使用弧度制。
type Rotation3D struct {
	X float64
	Y float64
	Z float64
}

// CubeID是八叉树里一个格子的编号：深度为Depth的格子，x方向覆盖物理坐标[X*边长, (X+1)*边长)，y、z一样。
// 同一个位置、同一个深度永远是同一个CubeID，所以可以存档，也可以当map的key。
type CubeID struct {
	X, Y, Z int64
	Depth   Depth
}

// 包含物理坐标p的、深度为d的格子
func Cube_id_at(p PhysicalPosition3D, d Depth) CubeID {
	s := d.Side()
	return CubeID{int64(math.Floor(p.X / s)), int64(math.Floor(p.Y / s)), int64(math.Floor(p.Z / s)), d}
}

// depth 0的格子(x, y, z)，也就是坐标[x, x+1)×[y, y+1)×[z, z+1)的普通方块
func Block_id(x, y, z int64) CubeID {
	return CubeID{x, y, z, 0}
}

func (id CubeID) Side() float64 { return id.Depth.Side() }

// 最小的角（x、y、z都最小）
func (id CubeID) Min() PhysicalPosition3D {
	s := id.Side()
	return PhysicalPosition3D{float64(id.X) * s, float64(id.Y) * s, float64(id.Z) * s}
}

// 最大的角
func (id CubeID) Max() PhysicalPosition3D {
	s := id.Side()
	return PhysicalPosition3D{float64(id.X+1) * s, float64(id.Y+1) * s, float64(id.Z+1) * s}
}

func (id CubeID) Center() PhysicalPosition3D {
	return id.Min().Add(id.Max()).Scale(0.5)
}

// 包含这个cube的、深度为d的cube，d必须不小于id.Depth
func (id CubeID) Parent_at(d Depth) CubeID {
	shift := uint(d - id.Depth)
	return CubeID{id.X >> shift, id.Y >> shift, id.Z >> shift, d}
}

// 第o个子cube，o是卦限：x正为1，y正为2，z正为4
func (id CubeID) Child(o int) CubeID {
	return CubeID{id.X*2 + int64(o&1), id.Y*2 + int64(o>>1&1), id.Z*2 + int64(o>>2&1), id.Depth - 1}
}

// 这个cube在父cube里的卦限
func (id CubeID) Octant() int {
	return int(id.X&1) | int(id.Y&1)<<1 | int(id.Z&1)<<2
}

// other是不是在这个cube里面（包括other就是它自己）
func (id CubeID) Contains(other CubeID) bool {
	return other.Depth <= id.Depth && other.Parent_at(id.Depth) == id
}

// 同样大小、挨着的cube，dx、dy、dz是格子数
func (id CubeID) Neighbor(dx, dy, dz int64) CubeID {
	return CubeID{id.X + dx, id.Y + dy, id.Z + dz, id.Depth}
}

// 这个cube所在的chunk，id.Depth必须不大于CHUNK_DEPTH
func (id CubeID) Chunk() CubeID {
	return id.Parent_at(CHUNK_DEPTH)
}

func (id CubeID) String() string {
	return fmt.Sprintf("(%d, %d, %d)@%d", id.X, id.Y, id.Z, id.Depth)
}
