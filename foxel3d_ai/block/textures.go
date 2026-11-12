// 程序生成的16x16像素贴图，不需要任何图片文件
package block

import (
	"math"

	"foxel3d_ai/canvas"
)

const tex_size = 16

// 确定性的随机数（splitmix64），同一个seed永远生成同一张贴图。Pixel_texture把它传给每个像素
type Rng struct{ s uint64 }

func New_rng(seed uint64) *Rng { return &Rng{seed} }

func (r *Rng) Next() uint64 {
	r.s += 0x9e3779b97f4a7c15
	z := r.s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// [0, 1)
func (r *Rng) Float() float64 { return float64(r.Next()>>11) / (1 << 53) }

func (r *Rng) Intn(n int) int { return int(r.Next() % uint64(n)) }

// 亮度随机上下浮动amount
func (r *Rng) Vary(c canvas.Color, amount float64) canvas.Color {
	return c.Scale(1 + (r.Float()*2-1)*amount)
}

// 用f生成一张16x16的贴图，f按行从左上到右下依次调用
func Pixel_texture(seed uint64, f func(x, y int, r *Rng) canvas.Color) *canvas.PixelImage {
	r := &Rng{seed}
	return canvas.New_pixel_image(tex_size, tex_size, func(x, y int) canvas.Color { return f(x, y, r) })
}

// 一种颜色加上随机的明暗
func Noise_texture(seed uint64, base canvas.Color, amount float64) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color { return r.Vary(base, amount) })
}

func rgb(r, g, b uint8) canvas.Color { return canvas.Color{R: r, G: g, B: b, A: 255} }

func stone_texture(seed uint64) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		c := r.Vary(rgb(128, 128, 128), 0.07)
		if r.Float() < 0.08 {
			c = c.Scale(0.82)
		}
		return c
	})
}

// 石头上带一些颜色的斑点
func ore_texture(seed uint64, spot canvas.Color) *canvas.PixelImage {
	base := stone_texture(seed)
	r := &Rng{seed * 31}
	var centers [5][2]float64
	for i := range centers {
		centers[i] = [2]float64{2 + r.Float()*12, 2 + r.Float()*12}
	}
	for y := range tex_size {
		for x := range tex_size {
			for _, c := range centers {
				if math.Hypot(float64(x)-c[0], float64(y)-c[1]) < 1.4 && r.Float() < 0.85 {
					base.Pix[y*tex_size+x] = r.Vary(spot, 0.1)
				}
			}
		}
	}
	return base
}

func cobblestone_texture(seed uint64) *canvas.PixelImage {
	r := &Rng{seed}
	type cell struct{ x, y, shade float64 }
	cells := make([]cell, 10)
	for i := range cells {
		cells[i] = cell{r.Float() * 16, r.Float() * 16, 0.75 + r.Float()*0.4}
	}
	// 贴图左右、上下是接起来的，所以距离按环绕算
	dist := func(ax, ay, bx, by float64) float64 {
		dx := math.Min(math.Abs(ax-bx), 16-math.Abs(ax-bx))
		dy := math.Min(math.Abs(ay-by), 16-math.Abs(ay-by))
		return math.Hypot(dx, dy)
	}
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		px, py := float64(x)+0.5, float64(y)+0.5
		d1, d2, best := 99.0, 99.0, 0
		for i, c := range cells {
			d := dist(px, py, c.x, c.y)
			if d < d1 {
				d2, d1, best = d1, d, i
			} else if d < d2 {
				d2 = d
			}
		}
		if d2-d1 < 1.1 {
			return r.Vary(rgb(78, 78, 78), 0.08)
		}
		return r.Vary(rgb(128, 128, 128).Scale(cells[best].shade), 0.06)
	})
}

func dirt_texture(seed uint64) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		c := r.Vary(rgb(134, 96, 67), 0.1)
		if f := r.Float(); f < 0.1 {
			c = c.Scale(0.75)
		} else if f > 0.95 {
			c = c.Scale(1.2)
		}
		return c
	})
}

// 下面是side，上面盖着top颜色的一层（草、雪），每列厚度不一样
func topped_texture(seed uint64, side *canvas.PixelImage, top canvas.Color) *canvas.PixelImage {
	r := &Rng{seed}
	var depth [tex_size]int
	for x := range depth {
		depth[x] = 2 + r.Intn(3)
	}
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		if y < depth[x] {
			return r.Vary(top, 0.1)
		}
		return side.Pix[y*tex_size+x]
	})
}

func gravel_texture(seed uint64) *canvas.PixelImage {
	palette := []canvas.Color{rgb(120, 115, 110), rgb(150, 144, 138), rgb(95, 90, 88), rgb(170, 160, 150), rgb(130, 120, 112)}
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		return r.Vary(palette[r.Intn(len(palette))], 0.05)
	})
}

func bedrock_texture(seed uint64) *canvas.PixelImage {
	palette := []canvas.Color{rgb(40, 40, 40), rgb(85, 85, 85), rgb(20, 20, 20), rgb(120, 120, 120)}
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		return palette[r.Intn(len(palette))]
	})
}

// 树皮：竖条纹
func bark_texture(seed uint64, base canvas.Color) *canvas.PixelImage {
	r := &Rng{seed}
	var stripe [tex_size]float64
	for x := range stripe {
		stripe[x] = 0.8 + r.Float()*0.3
	}
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		return r.Vary(base.Scale(stripe[x]), 0.06)
	})
}

// 木头的横截面：年轮，外面一圈是树皮
func rings_texture(seed uint64, bark, light, dark canvas.Color) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		if x == 0 || y == 0 || x == tex_size-1 || y == tex_size-1 {
			return r.Vary(bark, 0.08)
		}
		d := math.Hypot(float64(x)-7.5, float64(y)-7.5)
		if int(d)%3 == 0 {
			return r.Vary(dark, 0.05)
		}
		return r.Vary(light, 0.05)
	})
}

func planks_texture(seed uint64, base canvas.Color) *canvas.PixelImage {
	r := &Rng{seed}
	var shade [4]float64
	for i := range shade {
		shade[i] = 0.9 + r.Float()*0.2
	}
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		plank := y / 4
		if y%4 == 3 || x == (plank*5+3)%tex_size {
			return base.Scale(0.6)
		}
		return r.Vary(base.Scale(shade[plank]), 0.05)
	})
}

// 有一部分像素是透明的
func leaves_texture(seed uint64, base canvas.Color, holes float64) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		if r.Float() < holes {
			return canvas.Transparent
		}
		return r.Vary(base, 0.2)
	})
}

func glass_texture() *canvas.PixelImage {
	frame := rgb(215, 235, 240)
	return Pixel_texture(0, func(x, y int, r *Rng) canvas.Color {
		edge := x == 0 || y == 0 || x == tex_size-1 || y == tex_size-1
		shine := (x+y == 6 || x+y == 7) && x >= 2 && x <= 5
		if edge || shine {
			return frame
		}
		return canvas.Transparent
	})
}

// 砖墙：w×h的砖，每行错开半块
func brick_texture(seed uint64, brick, mortar canvas.Color, w, h int) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		offset := (y / h) % 2 * (w / 2)
		if y%h == h-1 || (x+offset)%w == w-1 {
			return r.Vary(mortar, 0.05)
		}
		return r.Vary(brick, 0.08)
	})
}

// 横条纹，比如砂岩
func layered_texture(seed uint64, base canvas.Color) *canvas.PixelImage {
	return Pixel_texture(seed, func(x, y int, r *Rng) canvas.Color {
		c := r.Vary(base, 0.04)
		if y%5 == 4 {
			c = c.Scale(0.88)
		}
		return c
	})
}
