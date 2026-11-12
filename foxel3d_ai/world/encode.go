// chunk的编码，用于存档。
//
// 先序遍历八叉树，每个cube写一个uvarint：0表示分成8个子cube（后面跟着8个子cube的编码），
// 否则是叶子，值是Block+1。没有改动过的chunk不保存，加载时重新生成。
package world

import (
	"encoding/binary"
	"errors"
)

var errBadChunk = errors.New("chunk数据格式不对")

func encode_chunk(n *Cube) []byte {
	return encode_cube(nil, n)
}

func encode_cube(buf []byte, n *Cube) []byte {
	if n.Children == nil {
		return binary.AppendUvarint(buf, uint64(n.Block)+1)
	}
	buf = append(buf, 0)
	for i := range n.Children {
		buf = encode_cube(buf, &n.Children[i])
	}
	return buf
}

func decode_chunk(data []byte, n *Cube) error {
	rest, err := decode_cube(data, n, CHUNK_DEPTH)
	if err == nil && len(rest) != 0 {
		err = errBadChunk
	}
	return err
}

func decode_cube(data []byte, n *Cube, depth Depth) ([]byte, error) {
	v, k := binary.Uvarint(data)
	if k <= 0 || v > 1<<16 {
		return nil, errBadChunk
	}
	data = data[k:]
	if v != 0 {
		*n = Cube{Block: BlockID(v - 1)}
		return data, nil
	}
	if depth <= MIN_DEPTH {
		return nil, errBadChunk
	}
	*n = Cube{Children: new([8]Cube)}
	for i := range n.Children {
		var err error
		if data, err = decode_cube(data, &n.Children[i], depth-1); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// 所有改动过的chunk的编码，包括已加载的和卸载了的（存档用）
func (w *World) Saved_chunks() map[CubeID][]byte {
	out := make(map[CubeID][]byte, len(w.saved))
	for pos, data := range w.saved {
		out[pos] = data
	}
	for pos, ch := range w.chunks {
		if ch.modified {
			out[pos] = encode_chunk(ch.node)
		}
	}
	return out
}

// 换成存档里的chunk。remap把存档里的BlockID换成现在的BlockID（注册顺序可能变了），
// 数据坏了的chunk会被丢掉（之后重新生成），返回丢掉了几个
func (w *World) Set_saved_chunks(chunks map[CubeID][]byte, remap func(BlockID) BlockID) (dropped int) {
	w.saved = map[CubeID][]byte{}
	for pos, data := range chunks {
		var n Cube
		if pos.Depth != CHUNK_DEPTH || decode_chunk(data, &n) != nil {
			dropped++
			continue
		}
		remap_cube(&n, remap)
		w.saved[pos] = encode_chunk(&n)
	}
	return dropped
}

func remap_cube(n *Cube, remap func(BlockID) BlockID) {
	if n.Children == nil {
		n.Block = remap(n.Block)
		return
	}
	for i := range n.Children {
		remap_cube(&n.Children[i], remap)
	}
	n.try_merge()
}
