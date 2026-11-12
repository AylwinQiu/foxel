// Image的几种实现：纯色、按像素定义的图片、从文件或网络读取的图片。还有常用颜色。
package canvas

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// 常用颜色
var (
	Red         = Color{R: 255, G: 0, B: 0, A: 255}
	Green       = Color{R: 0, G: 255, B: 0, A: 255}
	Blue        = Color{R: 0, G: 0, B: 255, A: 255}
	White       = Color{R: 255, G: 255, B: 255, A: 255}
	Black       = Color{R: 0, G: 0, B: 0, A: 255}
	Gray        = Color{R: 128, G: 128, B: 128, A: 255}
	LightGray   = Color{R: 200, G: 200, B: 200, A: 255}
	DarkGray    = Color{R: 80, G: 80, B: 80, A: 255}
	Yellow      = Color{R: 255, G: 255, B: 0, A: 255}
	Magenta     = Color{R: 255, G: 0, B: 255, A: 255}
	SkyBlue     = Color{R: 140, G: 190, B: 255, A: 255}
	Transparent = Color{R: 0, G: 0, B: 0, A: 0}
)

// 颜色乘上k（0到1之间变暗），alpha不变
func (c Color) Scale(k float64) Color {
	f := func(v uint8) uint8 { return uint8(min(max(float64(v)*k, 0), 255)) }
	return Color{f(c.R), f(c.G), f(c.B), c.A}
}

// alpha换成a
func (c Color) With_alpha(a uint8) Color {
	c.A = a
	return c
}

// 纯色图片，每个位置都是这个颜色
type SolidColor Color

func (s SolidColor) Get_pixel(x_frac float64, y_frac float64) Color { return Color(s) }
func (s SolidColor) Size() (int, int)                               { return 1, 1 }

// 按像素定义的图片，适合程序生成的像素风贴图。Pix按行存，Pix[y*W+x]是(x, y)的颜色
type PixelImage struct {
	W, H int
	Pix  []Color
}

// 用f(x, y)生成每个像素
func New_pixel_image(w int, h int, f func(x int, y int) Color) *PixelImage {
	p := &PixelImage{W: w, H: h, Pix: make([]Color, w*h)}
	for y := range h {
		for x := range w {
			p.Pix[y*w+x] = f(x, y)
		}
	}
	return p
}

// 取(x_frac, y_frac)所在的那个像素，不做插值
func (p *PixelImage) Get_pixel(x_frac float64, y_frac float64) Color {
	x := min(max(int(x_frac*float64(p.W)), 0), p.W-1)
	y := min(max(int(y_frac*float64(p.H)), 0), p.H-1)
	return p.Pix[y*p.W+x]
}

func (p *PixelImage) Size() (int, int) { return p.W, p.H }

// 读进内存的图片
type picture struct {
	img image.Image
}

func (p *picture) Get_pixel(x_frac float64, y_frac float64) Color {
	b := p.img.Bounds()
	x := b.Min.X + min(max(int(x_frac*float64(b.Dx())), 0), b.Dx()-1)
	y := b.Min.Y + min(max(int(y_frac*float64(b.Dy())), 0), b.Dy()-1)
	c := color.NRGBAModel.Convert(p.img.At(x, y)).(color.NRGBA)
	return Color{c.R, c.G, c.B, c.A}
}

func (p *picture) Size() (int, int) { return p.img.Bounds().Dx(), p.img.Bounds().Dy() }

var (
	pictures_mu sync.Mutex
	// url -> 读取结果（包括失败），每个url只读一次
	pictures = map[string]Image{}
)

// 读取一张图片，http(s)开头的从网络下载，其他的当作本地路径。同一个url只读一次，失败时返回品红色并打印原因
func Load_picture(url string) Image {
	pictures_mu.Lock()
	defer pictures_mu.Unlock()
	if img, ok := pictures[url]; ok {
		return img
	}
	var img Image = SolidColor(Magenta)
	if decoded, err := read_picture(url); err != nil {
		fmt.Fprintf(os.Stderr, "foxel: 读取图片失败：%s：%v\n", url, err)
	} else {
		img = &picture{decoded}
	}
	pictures[url] = img
	return img
}

var http_client = &http.Client{Timeout: 20 * time.Second}

func read_picture(url string) (image.Image, error) {
	var data []byte
	var err error
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		resp, err := http_client.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %s", resp.Status)
		}
		if data, err = io.ReadAll(resp.Body); err != nil {
			return nil, err
		}
	} else if data, err = os.ReadFile(url); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if b := img.Bounds(); b.Dx() == 0 || b.Dy() == 0 {
		return nil, fmt.Errorf("图片是空的")
	}
	return img, nil
}
