// Image的几种实现：从文件或网络读取的图片，纯色。
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

// 纯色图片，每个位置都是这个颜色
type SolidColor Color

func (s SolidColor) Get_pixel(x_frac float64, y_frac float64) Color {
	return Color(s)
}

func (s SolidColor) size() (int, int) {
	return 1, 1
}

// 读进内存的图片
type picture struct {
	img image.Image
}

// 取(x_frac, y_frac)所在的那个像素，不做插值
func (p *picture) Get_pixel(x_frac float64, y_frac float64) Color {
	b := p.img.Bounds()
	x := b.Min.X + min(max(int(x_frac*float64(b.Dx())), 0), b.Dx()-1)
	y := b.Min.Y + min(max(int(y_frac*float64(b.Dy())), 0), b.Dy()-1)
	c := color.NRGBAModel.Convert(p.img.At(x, y)).(color.NRGBA)
	return Color{c.R, c.G, c.B, c.A}
}

// 图片的像素大小，canvas用它决定贴图分辨率
func (p *picture) size() (int, int) {
	return p.img.Bounds().Dx(), p.img.Bounds().Dy()
}

// 读取失败时用的颜色
var loadFailedColor = SolidColor{R: 255, G: 0, B: 255, A: 255}

var (
	picturesMu sync.Mutex
	// url -> 读取结果（包括失败），每个url只读一次
	pictures = map[string]Image{}
)

func loadPicture(url string) Image {
	picturesMu.Lock()
	defer picturesMu.Unlock()
	if img, ok := pictures[url]; ok {
		return img
	}
	var img Image = loadFailedColor
	if decoded, err := readPicture(url); err != nil {
		fmt.Fprintf(os.Stderr, "foxel: 读取图片失败：%s：%v\n", url, err)
	} else {
		img = &picture{decoded}
	}
	pictures[url] = img
	return img
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

func readPicture(url string) (image.Image, error) {
	var data []byte
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		resp, err := httpClient.Get(url)
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
	} else {
		var err error
		if data, err = os.ReadFile(url); err != nil {
			return nil, err
		}
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
