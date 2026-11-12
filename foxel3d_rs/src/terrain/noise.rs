use crate::rng::Rng;

/// 2D Perlin噪声，输出大约在[-1, 1]，坐标每变化1大约是一个起伏
pub struct Perlin {
    perm: [u8; 512],
}

fn fade(t: f64) -> f64 {
    t * t * t * (t * (t * 6.0 - 15.0) + 10.0)
}

fn lerp(t: f64, a: f64, b: f64) -> f64 {
    a + t * (b - a)
}

fn grad(h: u8, x: f64, y: f64) -> f64 {
    match h & 7 {
        0 => x + y,
        1 => -x + y,
        2 => x - y,
        3 => -x - y,
        4 => x,
        5 => -x,
        6 => y,
        _ => -y,
    }
}

impl Perlin {
    pub fn new(seed: u64) -> Perlin {
        let mut perm = [0u8; 512];
        for (i, p) in perm.iter_mut().take(256).enumerate() {
            *p = i as u8;
        }
        let mut r = Rng::new(seed);
        for i in (1..256).rev() {
            let j = r.range(0, i as i64) as usize;
            perm.swap(i, j);
        }
        perm.copy_within(0..256, 256);
        Perlin { perm }
    }

    pub fn noise(&self, x: f64, y: f64) -> f64 {
        let (fx, fy) = (x.floor(), y.floor());
        let (xi, yi) = ((fx as i64 & 255) as usize, (fy as i64 & 255) as usize);
        let (xf, yf) = (x - fx, y - fy);
        let (u, v) = (fade(xf), fade(yf));
        let p = &self.perm;
        let aa = p[p[xi] as usize + yi];
        let ab = p[p[xi] as usize + yi + 1];
        let ba = p[p[xi + 1] as usize + yi];
        let bb = p[p[xi + 1] as usize + yi + 1];
        lerp(
            v,
            lerp(u, grad(aa, xf, yf), grad(ba, xf - 1.0, yf)),
            lerp(u, grad(ab, xf, yf - 1.0), grad(bb, xf - 1.0, yf - 1.0)),
        )
    }

    /// 分形噪声：octaves层噪声叠加，每层频率翻倍、幅度减半，输出大约在[-1, 1]
    pub fn fbm(&self, mut x: f64, mut y: f64, octaves: u32) -> f64 {
        let (mut sum, mut amp, mut total) = (0.0, 1.0, 0.0);
        for i in 0..octaves {
            // 每层错开一点，避免原点附近每层都是0
            let off = i as f64 * 17.31;
            sum += self.noise(x + off, y - off) * amp;
            total += amp;
            x *= 2.0;
            y *= 2.0;
            amp /= 2.0;
        }
        sum / total
    }
}
