package terrain

import (
	"math"
	"math/rand/v2"
)

// 2D Perlin噪声，输出大约在[-1, 1]，坐标每变化1大约是一个起伏
type perlin struct {
	perm [512]uint8
}

func new_perlin(seed uint64) *perlin {
	p := &perlin{}
	r := rand.New(rand.NewPCG(seed, 0x5eed))
	for i := range 256 {
		p.perm[i] = uint8(i)
	}
	r.Shuffle(256, func(i, j int) { p.perm[i], p.perm[j] = p.perm[j], p.perm[i] })
	copy(p.perm[256:], p.perm[:256])
	return p
}

func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

func lerp(t, a, b float64) float64 { return a + t*(b-a) }

func grad(h uint8, x, y float64) float64 {
	switch h & 7 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x
	case 5:
		return -x
	case 6:
		return y
	default:
		return -y
	}
}

func (p *perlin) noise(x, y float64) float64 {
	fx, fy := math.Floor(x), math.Floor(y)
	xi, yi := int(fx)&255, int(fy)&255
	xf, yf := x-fx, y-fy
	u, v := fade(xf), fade(yf)
	aa := p.perm[int(p.perm[xi])+yi]
	ab := p.perm[int(p.perm[xi])+yi+1]
	ba := p.perm[int(p.perm[xi+1])+yi]
	bb := p.perm[int(p.perm[xi+1])+yi+1]
	return lerp(v,
		lerp(u, grad(aa, xf, yf), grad(ba, xf-1, yf)),
		lerp(u, grad(ab, xf, yf-1), grad(bb, xf-1, yf-1)))
}

// 分形噪声：octaves层噪声叠加，每层频率翻倍、幅度减半，输出大约在[-1, 1]
func (p *perlin) fbm(x, y float64, octaves int) float64 {
	sum, amp, total := 0.0, 1.0, 0.0
	for i := range octaves {
		// 每层错开一点，避免原点附近每层都是0
		off := float64(i) * 17.31
		sum += p.noise(x+off, y-off) * amp
		total += amp
		x, y, amp = x*2, y*2, amp/2
	}
	return sum / total
}
