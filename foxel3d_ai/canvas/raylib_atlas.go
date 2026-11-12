// 该文件完全ai自主控制
package canvas

import (
	"fmt"
	"os"
	"reflect"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// 贴图集：所有图片都采样成atlas_tile见方的小块，放进一张大贴图里。
// 每块四周多放1像素（复制边上的像素），这样贴图边缘不会混进隔壁图片的颜色。
const (
	atlas_size   = 2048
	atlas_tile   = 32
	atlas_stride = atlas_tile + 2
	atlas_per    = atlas_size / atlas_stride
)

type atlas struct {
	tex   rl.Texture2D
	tiles map[Image]int
	next  int
	full  bool
}

const (
	tile_white   = 0
	tile_magenta = 1
)

func new_atlas() *atlas {
	img := rl.GenImageColor(atlas_size, atlas_size, rl.Blank)
	a := &atlas{tex: rl.LoadTextureFromImage(img), tiles: map[Image]int{}}
	rl.UnloadImage(img)
	// 不插值，像素风的小图片放大后不会糊
	rl.SetTextureFilter(a.tex, rl.FilterPoint)
	a.add(SolidColor(White))
	a.add(SolidColor(Magenta))
	return a
}

// img在贴图集里的编号，第一次见到时采样放进去。nil是白色，不能比较的图片是品红色
func (a *atlas) tile_of(img Image) int {
	if img == nil {
		return tile_white
	}
	if !reflect.TypeOf(img).Comparable() {
		return tile_magenta
	}
	if t, ok := a.tiles[img]; ok {
		return t
	}
	return a.add(img)
}

func (a *atlas) add(img Image) int {
	if a.next >= atlas_per*atlas_per {
		if !a.full {
			fmt.Fprintln(os.Stderr, "foxel: 贴图集满了，之后的图片都显示成品红色")
			a.full = true
		}
		a.tiles[img] = tile_magenta
		return tile_magenta
	}
	t := a.next
	a.next++
	data := make([]byte, atlas_stride*atlas_stride*4)
	for py := range atlas_stride {
		for px := range atlas_stride {
			// 外面一圈复制最边上的像素
			sx := min(max(px-1, 0), atlas_tile-1)
			sy := min(max(py-1, 0), atlas_tile-1)
			c := img.Get_pixel((float64(sx)+0.5)/atlas_tile, (float64(sy)+0.5)/atlas_tile)
			p := (py*atlas_stride + px) * 4
			data[p], data[p+1], data[p+2], data[p+3] = c.R, c.G, c.B, c.A
		}
	}
	x, y := a.tile_origin(t)
	rl.UpdateTextureRec(a.tex, rl.NewRectangle(float32(x-1), float32(y-1), atlas_stride, atlas_stride), data)
	a.tiles[img] = t
	return t
}

// 第t块图片（不含外面一圈）左上角的像素坐标
func (a *atlas) tile_origin(t int) (int, int) {
	return (t%atlas_per)*atlas_stride + 1, (t/atlas_per)*atlas_stride + 1
}

// 第t块图片上的相对位置(u, v)在贴图集里的坐标
func (a *atlas) uv(t int, u, v float64) (float32, float32) {
	x, y := a.tile_origin(t)
	return float32((float64(x) + u*atlas_tile) / atlas_size), float32((float64(y) + v*atlas_tile) / atlas_size)
}
