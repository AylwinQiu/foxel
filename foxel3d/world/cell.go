package world

var next_id CellID = CellID{
	id: [4]uint64{0, 0, 0, 0},
}

type Cell struct {
	children [8]*Cell
	id       CellID
}

type CellID struct {
	id [4]uint64
}
