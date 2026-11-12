//! 确定性的随机数和哈希：同一个种子永远得到同样的结果，地形和贴图都靠它保证每次生成的一样。

/// splitmix64随机数生成器
#[derive(Clone, Debug)]
pub struct Rng {
    state: u64,
}

impl Rng {
    pub fn new(seed: u64) -> Rng {
        Rng { state: seed }
    }

    pub fn next_u64(&mut self) -> u64 {
        self.state = self.state.wrapping_add(0x9e3779b97f4a7c15);
        let mut z = self.state;
        z = (z ^ (z >> 30)).wrapping_mul(0xbf58476d1ce4e5b9);
        z = (z ^ (z >> 27)).wrapping_mul(0x94d049bb133111eb);
        z ^ (z >> 31)
    }

    /// [0, 1)
    pub fn float(&mut self) -> f64 {
        (self.next_u64() >> 11) as f64 / (1u64 << 53) as f64
    }

    /// [lo, hi]，hi < lo时返回lo
    pub fn range(&mut self, lo: i64, hi: i64) -> i64 {
        if hi <= lo {
            return lo;
        }
        let n = (hi - lo) as u64 + 1;
        lo + (self.next_u64() % n) as i64
    }
}

/// 把几个数混成一个均匀分布的64位哈希
pub fn hash(values: &[u64]) -> u64 {
    let mut h: u64 = 0x243f6a8885a308d3;
    for &v in values {
        h ^= v.wrapping_add(0x9e3779b97f4a7c15).wrapping_add(h << 6).wrapping_add(h >> 2);
        h = (h ^ (h >> 31)).wrapping_mul(0x7fb5d329728ea185);
        h = (h ^ (h >> 27)).wrapping_mul(0x81dadef4bc2dd44d);
        h ^= h >> 33;
    }
    h
}

/// 哈希的高53位换成[0, 1)
pub fn unit(h: u64) -> f64 {
    (h >> 11) as f64 / (1u64 << 53) as f64
}
