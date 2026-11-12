use super::{BlockId, CubeId, Leaf, MIN_DEPTH, Position, World};

/// 射线打到的东西
#[derive(Clone, Copy, Debug)]
pub struct Hit {
    /// 打到的叶子cube
    pub leaf: Leaf,
    /// 射线进入这个cube的点
    pub point: Position,
    /// 打到的那个面朝外的方向，比如[0, 1, 0]是上面。射线从cube里面出发时是[0, 0, 0]
    pub normal: [i64; 3],
    /// 从起点到point的距离
    pub distance: f64,
}

impl World {
    /// 从origin沿dir找第一个hit(方块)为true的叶子cube，最远max_distance。
    /// 按MIN_DEPTH的格子一格一格走，碰到没加载的chunk就停下
    pub fn raycast(
        &self,
        origin: Position,
        dir: Position,
        max_distance: f64,
        hit: &dyn Fn(BlockId) -> bool,
    ) -> Option<Hit> {
        let l = dir.length();
        if l == 0.0 {
            return None;
        }
        let d = [dir.x / l, dir.y / l, dir.z / l];
        let o = [origin.x, origin.y, origin.z];
        let s = super::side(MIN_DEPTH);
        let start = CubeId::at(origin, MIN_DEPTH);
        let mut cell = [start.x, start.y, start.z];

        let mut step = [0i64; 3];
        let mut t_max = [f64::INFINITY; 3];
        let mut t_delta = [f64::INFINITY; 3];
        for a in 0..3 {
            if d[a] > 0.0 {
                step[a] = 1;
                t_max[a] = ((cell[a] + 1) as f64 * s - o[a]) / d[a];
            } else if d[a] < 0.0 {
                step[a] = -1;
                t_max[a] = (cell[a] as f64 * s - o[a]) / d[a];
            }
            t_delta[a] = s / d[a].abs();
        }

        let mut t = 0.0;
        let mut normal = [0i64; 3];
        while t <= max_distance {
            let id = CubeId::new(cell[0], cell[1], cell[2], MIN_DEPTH);
            let leaf = self.leaf_at(id.center())?;
            if hit(leaf.block) {
                let point = Position::new(o[0] + d[0] * t, o[1] + d[1] * t, o[2] + d[2] * t);
                return Some(Hit { leaf, point, normal, distance: t });
            }
            let mut a = 0;
            if t_max[1] < t_max[a] {
                a = 1;
            }
            if t_max[2] < t_max[a] {
                a = 2;
            }
            t = t_max[a];
            cell[a] += step[a];
            t_max[a] += t_delta[a];
            normal = [0; 3];
            normal[a] = -step[a];
        }
        None
    }
}
