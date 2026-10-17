package world

import (
	"fmt"
	"math"
)

// 树坐标的最大深度
const TREE_MAX_DEPTH = 64

// 在树坐标列表里，非8数字为此时方块边长对应1.0
const IDENTITY_DEPTH = 32

//这里是定义的是物理距离

type PhysicalPosition1D struct {
	x float64
}

type PhysicalPosition2D struct {
	x float64
	y float64
}
type PhysicalPosition3D struct {
	x float64
	y float64
	z float64
}

// 八叉树坐标
// pos里面的数字是从0-8这9个数字，0-7是7个卦限。8表示树的结束。
// x，y，z的正负号正为1负为0，卦限对应值算法为x*1+y*2+z*4
type TreePosition struct {
	pos [TREE_MAX_DEPTH]int8
}

// 偏移量
// 偏移量是对应tree坐标的cube中心相对位移xyz的取值是（-0.6， 0.6)这样是为了保证不会在0.5的时候在界限之间震荡。
type Offset3D struct {
	x float64
	y float64
	z float64
}

// 将一个树坐标转化成物理坐标
// 根节点（depth=0）是以原点为中心、边长 2^IDENTITY_DEPTH 的立方体，返回的是树坐标对应cube的中心
func Tree2phy(tree TreePosition) PhysicalPosition3D {
	var phy PhysicalPosition3D
	for i := 0; i < TREE_MAX_DEPTH; i++ {
		o := tree.pos[i]
		if o == 8 {
			break
		}
		// 第i位选出的子cube边长为 2^(IDENTITY_DEPTH-i-1)，其中心相对父cube中心偏移半个子边长
		half := math.Ldexp(1, IDENTITY_DEPTH-i-2)
		phy.x += octantSign(o&1) * half
		phy.y += octantSign(o&2) * half
		phy.z += octantSign(o&4) * half
	}
	return phy
}

// 将一个物理坐标转化成对应深度的树坐标和对应offset
// depth就是tree坐标里非8的数字个数
// offset以该深度cube的边长为单位，正好落在边界上时归入正向卦限，因此范围是[-0.5, 0.5)
// 物理坐标必须落在根cube [-2^(IDENTITY_DEPTH-1), 2^(IDENTITY_DEPTH-1)) 内，否则panic
func Phy2tree(phy PhysicalPosition3D, depth int) (TreePosition, Offset3D) {
	if !inRootCube(phy.x) || !inRootCube(phy.y) || !inRootCube(phy.z) {
		panic(fmt.Sprintf("phy2tree: position %+v out of root cube", phy))
	}
	depth = max(0, min(depth, TREE_MAX_DEPTH))

	var tree TreePosition
	for i := range tree.pos {
		tree.pos[i] = 8
	}

	var center PhysicalPosition3D
	for i := 0; i < depth; i++ {
		half := math.Ldexp(1, IDENTITY_DEPTH-i-2)
		var o int8
		if phy.x >= center.x {
			o |= 1
		}
		if phy.y >= center.y {
			o |= 2
		}
		if phy.z >= center.z {
			o |= 4
		}
		center.x += octantSign(o&1) * half
		center.y += octantSign(o&2) * half
		center.z += octantSign(o&4) * half
		tree.pos[i] = o
	}

	side := math.Ldexp(1, IDENTITY_DEPTH-depth)
	return tree, Offset3D{
		x: (phy.x - center.x) / side,
		y: (phy.y - center.y) / side,
		z: (phy.z - center.z) / side,
	}
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
