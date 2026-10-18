package world

var next_id CubeID = CubeID{
	id: [4]uint64{0, 0, 0, 0},
}

// Cube里面只储存了cubeID和关于cube位置的信息。
type Cube struct {
	children [8]*Cube
	rotation [3]float64
	offset   [3]float64
	id       CubeID
}

type CubeID struct {
	id [4]uint64
}

// 分配一个新的CubeID，按256位整数（id[0]为最低位）自增
// 零值留给根cube，所以第一个分配出去的id是1
func newCubeID() CubeID {
	for i := range next_id.id {
		next_id.id[i]++
		if next_id.id[i] != 0 {
			break
		}
	}
	return next_id
}
