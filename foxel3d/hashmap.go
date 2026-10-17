package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"slices"
)

//实现我们自己的hashmap算法。
//对应的struct需要实现Hashable这个interface

const KEY_LENGTH = 16

type InnerHashKey [KEY_LENGTH]byte

type Hashable interface {
	//todo
	Get_hash() InnerHashKey
	Equal(other Hashable) bool
}

type HashMap[K Hashable, V Hashable] struct {
	m map[InnerHashKey]struct {
		k K
		v V
	}
}

// 方法都是值接收者，没法在nil map上分配，所以必须用NewHashMap创建
func NewHashMap[K Hashable, V Hashable]() HashMap[K, V] {
	return HashMap[K, V]{m: map[InnerHashKey]struct {
		k K
		v V
	}{}}
}

// 查询k，返回对应的value和是否成功，在不存在k时直接返回false。
func (self HashMap[K, V]) Get(k Hashable) (Hashable, bool) {
	e, ok := self.m[k.Get_hash()]
	if !ok || !e.k.Equal(k) {
		return nil, false
	}
	return e.v, true
}

// 设置kv对，返回是否成功
// k和v的类型不是K和V时失败；已有k时覆盖旧值
// 两个不相等的key算出同一个hash时panic（128位hash下基本不会发生）
func (self HashMap[K, V]) Set(k Hashable, v Hashable) bool {
	kk, ok := k.(K)
	if !ok {
		return false
	}
	vv, ok := v.(V)
	if !ok {
		return false
	}
	h := kk.Get_hash()
	if e, ok := self.m[h]; ok && !e.k.Equal(kk) {
		panic(fmt.Sprintf("HashMap: hash collision between keys %v and %v", e.k, kk))
	}
	self.m[h] = struct {
		k K
		v V
	}{kk, vv}
	return true
}

// 删除kv对，返回是否成功，k不存在时返回false
func (self HashMap[K, V]) Del(k Hashable) bool {
	h := k.Get_hash()
	e, ok := self.m[h]
	if !ok || !e.k.Equal(k) {
		return false
	}
	delete(self.m, h)
	return true
}

// 我们实现一些常用的类型的hashable
type MyInt int64
type MyFloat float64
type MyString string
type MyBool bool
type MyList[T Hashable] []T

// 每种类型的hash第一个字节是类型标记，保证不同类型的值不会得到相同的key
const (
	hashTagInt byte = iota + 1
	hashTagFloat
	hashTagString
	hashTagBool
	hashTagList
	hashTagMap
)

// 定长的类型直接把值编码进key，不会有碰撞
func fixedKey(tag byte, bits uint64) InnerHashKey {
	var k InnerHashKey
	k[0] = tag
	binary.LittleEndian.PutUint64(k[1:9], bits)
	return k
}

// 变长的类型用FNV-128a，正好16字节
func fnvKey(tag byte, data ...[]byte) InnerHashKey {
	h := fnv.New128a()
	h.Write([]byte{tag})
	for _, d := range data {
		h.Write(d)
	}
	var k InnerHashKey
	copy(k[:], h.Sum(nil))
	return k
}

func (self MyInt) Get_hash() InnerHashKey {
	return fixedKey(hashTagInt, uint64(self))
}

func (self MyInt) Equal(other Hashable) bool {
	o, ok := other.(MyInt)
	return ok && self == o
}

// -0和0视为相等，所有NaN视为相等（否则NaN作为key永远取不出来）
func (self MyFloat) Get_hash() InnerHashKey {
	f := float64(self)
	switch {
	case f == 0:
		f = 0
	case math.IsNaN(f):
		f = math.NaN()
	}
	return fixedKey(hashTagFloat, math.Float64bits(f))
}

func (self MyFloat) Equal(other Hashable) bool {
	o, ok := other.(MyFloat)
	if !ok {
		return false
	}
	return self == o || (math.IsNaN(float64(self)) && math.IsNaN(float64(o)))
}

func (self MyString) Get_hash() InnerHashKey {
	return fnvKey(hashTagString, []byte(self))
}

func (self MyString) Equal(other Hashable) bool {
	o, ok := other.(MyString)
	return ok && self == o
}

func (self MyBool) Get_hash() InnerHashKey {
	var bits uint64
	if self {
		bits = 1
	}
	return fixedKey(hashTagBool, bits)
}

func (self MyBool) Equal(other Hashable) bool {
	o, ok := other.(MyBool)
	return ok && self == o
}

// 把长度和每个元素的hash依次喂进FNV，长度保证嵌套list不会因为拼接方式不同而撞上
func (self MyList[T]) Get_hash() InnerHashKey {
	parts := make([][]byte, 0, len(self)+1)
	parts = append(parts, binary.LittleEndian.AppendUint64(nil, uint64(len(self))))
	for _, e := range self {
		k := e.Get_hash()
		parts = append(parts, k[:])
	}
	return fnvKey(hashTagList, parts...)
}

// 只有元素类型相同的MyList才可能相等
func (self MyList[T]) Equal(other Hashable) bool {
	o, ok := other.(MyList[T])
	if !ok || len(self) != len(o) {
		return false
	}
	for i := range self {
		if !self[i].Equal(o[i]) {
			return false
		}
	}
	return true
}

// 和插入顺序无关：每个键值对先算出一个entry hash，排序后再整体hash
func (self HashMap[K, V]) Get_hash() InnerHashKey {
	entries := make([]InnerHashKey, 0, len(self.m))
	for k, e := range self.m {
		vh := e.v.Get_hash()
		entries = append(entries, fnvKey(hashTagMap, k[:], vh[:]))
	}
	slices.SortFunc(entries, func(a, b InnerHashKey) int {
		return bytes.Compare(a[:], b[:])
	})

	parts := make([][]byte, 0, len(entries)+1)
	parts = append(parts, binary.LittleEndian.AppendUint64(nil, uint64(len(entries))))
	for i := range entries {
		parts = append(parts, entries[i][:])
	}
	return fnvKey(hashTagMap, parts...)
}

// 键和值都用Equal比较
func (self HashMap[K, V]) Equal(other Hashable) bool {
	o, ok := other.(HashMap[K, V])
	if !ok || len(self.m) != len(o.m) {
		return false
	}
	for k, e := range self.m {
		oe, ok := o.m[k]
		if !ok || !e.k.Equal(oe.k) || !e.v.Equal(oe.v) {
			return false
		}
	}
	return true
}
