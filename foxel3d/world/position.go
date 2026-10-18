package world

import (
	"errors"
	"fmt"
	"math"
)

// 树坐标的最大深度
const TREE_MAX_DEPTH = 64

// 在树坐标列表里，非8数字为此时方块边长对应1.0
const IDENTITY_DEPTH = 32

// 这里定义了深度，对于identity深度Depth是0
// cube越浅（越靠近根）Depth越大：该深度的cube边长为 2^Depth，树坐标里非8的数字个数为 IDENTITY_DEPTH-Depth
// 合法范围[-(TREE_MAX_DEPTH-IDENTITY_DEPTH), IDENTITY_DEPTH-1]，对应树坐标里1到TREE_MAX_DEPTH个非8数字（不含根）
type Depth struct {
	depth int32
}

func NewDepth(depth int64) (Depth, error) {
	if depth < -(TREE_MAX_DEPTH-IDENTITY_DEPTH) || depth > (IDENTITY_DEPTH-1) {
		// 接收到不合法参数，返回错误。
		return Depth{}, errors.New("nsupported range of depth input")
	}
	return Depth{
		depth: int32(depth),
	}, nil
}

//这里是定义的是物理距离

type PhysicalPosition1D struct {
	X float64
}

type PhysicalPosition2D struct {
	X float64
	Y float64
}
type PhysicalPosition3D struct {
	X float64
	Y float64
	Z float64
}

// 八叉树坐标
// pos里面的数字是从0-8这9个数字，0-7是7个卦限。8表示树的结束。
// x，y，z的正负号正为1负为0，卦限对应值算法为x*1+y*2+z*4
type TreePosition struct {
	Pos    [TREE_MAX_DEPTH]int8
	offset struct {
		x float64
		y float64
		z float64
	}
}

// 旋转
// 这个是旋转的轴向，使用右手定则，向量模长代表的是旋转弧度，使用弧度制。
type Rotation3D struct {
	X float64
	Y float64
	Z float64
}

// 将一个树坐标转化成物理坐标
// 根节点（树坐标里没有任何非8数字）是以原点为中心、边长 2^IDENTITY_DEPTH 的立方体
// 返回的是树坐标对应cube的中心加上offset（offset以该cube的边长为单位）
func Tree2phy(tree TreePosition) PhysicalPosition3D {
	var phy PhysicalPosition3D
	depth := 0
	for ; depth < TREE_MAX_DEPTH; depth++ {
		o := tree.Pos[depth]
		if o == 8 {
			break
		}
		// 树坐标第depth位（从0数）选出的子cube边长为 2^(IDENTITY_DEPTH-depth-1)，其中心相对父cube中心偏移半个子边长
		half := math.Ldexp(1, IDENTITY_DEPTH-depth-2)
		phy.X += octantSign(o&1) * half
		phy.Y += octantSign(o&2) * half
		phy.Z += octantSign(o&4) * half
	}

	side := math.Ldexp(1, IDENTITY_DEPTH-depth)
	phy.X += tree.offset.x * side
	phy.Y += tree.offset.y * side
	phy.Z += tree.offset.z * side
	return phy
}

// 将一个物理坐标转化成对应深度的树坐标，offset一并算好存在树坐标里
// 生成的树坐标里非8的数字个数为 IDENTITY_DEPTH-depth，对应cube边长为 2^depth
// offset以该深度cube的边长为单位，正好落在边界上时归入正向卦限，因此范围是[-0.5, 0.5)
// 物理坐标必须落在根cube [-2^(IDENTITY_DEPTH-1), 2^(IDENTITY_DEPTH-1)) 内，树深度必须在[0, TREE_MAX_DEPTH]内，否则panic
func Phy2tree(phy PhysicalPosition3D, depth Depth) TreePosition {
	if !inRootCube(phy.X) || !inRootCube(phy.Y) || !inRootCube(phy.Z) {
		panic(fmt.Sprintf("phy2tree: position %+v out of root cube", phy))
	}
	treeDepth := IDENTITY_DEPTH - int(depth.depth)
	if treeDepth < 0 || treeDepth > TREE_MAX_DEPTH {
		panic(fmt.Sprintf("phy2tree: depth %d out of tree range", depth.depth))
	}

	var tree TreePosition
	for i := range tree.Pos {
		tree.Pos[i] = 8
	}

	var center PhysicalPosition3D
	for i := range treeDepth {
		half := math.Ldexp(1, IDENTITY_DEPTH-i-2)
		var o int8
		if phy.X >= center.X {
			o |= 1
		}
		if phy.Y >= center.Y {
			o |= 2
		}
		if phy.Z >= center.Z {
			o |= 4
		}
		center.X += octantSign(o&1) * half
		center.Y += octantSign(o&2) * half
		center.Z += octantSign(o&4) * half
		tree.Pos[i] = o
	}

	side := math.Ldexp(1, IDENTITY_DEPTH-treeDepth)
	tree.offset.x = (phy.X - center.X) / side
	tree.offset.y = (phy.Y - center.Y) / side
	tree.offset.z = (phy.Z - center.Z) / side
	return tree
}

// NaN也会在这里被判为不合法
func inRootCube(v float64) bool {
	half := math.Ldexp(1, IDENTITY_DEPTH-1)
	return v >= -half && v < half
}

func octantSign(bit int8) float64 {
	if bit != 0 {
		return 1
	}
	return -1
}
