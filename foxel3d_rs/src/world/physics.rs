//! 长方体和世界的碰撞：玩家、以后的动物都用它移动。
//!
//! 长方体用"底面中心pos、宽width（x和z方向）、高height"表示。
//! 没加载的地方当成实心的，这样不会掉进还没生成的世界里。

use super::{BlockId, Cube, CubeId, MIN_DEPTH, Position, World};

/// 每次最多走这么远，比最小的方块（1/16格）小，所以不会穿过去
const MAX_STEP: f64 = 0.05;

/// 碰撞盒的两个对角
pub fn box_corners(pos: Position, width: f64, height: f64) -> (Position, Position) {
    let h = width / 2.0;
    (Position::new(pos.x - h, pos.y, pos.z - h), Position::new(pos.x + h, pos.y + height, pos.z + h))
}

/// 节点n（位置是id）里有没有和长方体重叠的实心方块
fn node_collides(n: &Cube, id: CubeId, lo: Position, hi: Position, solid: &dyn Fn(BlockId) -> bool) -> bool {
    let (a, b) = (id.min(), id.max());
    if !(lo.x < b.x && hi.x > a.x && lo.y < b.y && hi.y > a.y && lo.z < b.z && hi.z > a.z) {
        return false;
    }
    match &n.children {
        None => solid(n.block),
        Some(children) => children.iter().enumerate().any(|(o, c)| node_collides(c, id.child(o), lo, hi, solid)),
    }
}

impl World {
    /// lo到hi的长方体和实心方块重叠吗（只碰到边不算）。
    /// 先找和长方体重叠的普通方块格子，格子被分成小cube时才沿八叉树往下找，
    /// 所以最小方块再小，一大片普通方块的地方也只要查十几个格子
    pub fn box_collides(&self, lo: Position, hi: Position, solid: &dyn Fn(BlockId) -> bool) -> bool {
        let cells = |a: f64, b: f64| (a.floor() as i64, b.ceil() as i64 - 1);
        let (x0, x1) = cells(lo.x, hi.x);
        let (y0, y1) = cells(lo.y, hi.y);
        let (z0, z1) = cells(lo.z, hi.z);
        for y in y0..=y1 {
            for z in z0..=z1 {
                for x in x0..=x1 {
                    match self.find(CubeId::block(x, y, z)) {
                        Some((n, at)) if !node_collides(n, at, lo, hi, solid) => {}
                        _ => return true,
                    }
                }
            }
        }
        false
    }

    /// 把长方体移动delta，碰到方块就停在方块边上。先走y，再走x、z。
    /// step大于0时，水平方向碰到不超过step高的台阶会自动迈上去（走路时传，飞行、在空中时传0）。
    /// 返回新的位置和每个轴有没有被挡住
    pub fn move_box(
        &self,
        pos: Position,
        width: f64,
        height: f64,
        delta: Position,
        step: f64,
        solid: &dyn Fn(BlockId) -> bool,
    ) -> (Position, [bool; 3]) {
        let collides = |p: Position| {
            let (lo, hi) = box_corners(p, width, height);
            self.box_collides(lo, hi, solid)
        };
        let mut p = pos;
        let mut hit = [false; 3];
        for a in [1, 0, 2] {
            let d = delta.axis(a);
            let steps = (d.abs() / MAX_STEP).ceil() as usize;
            // 往下走时碰到了地面（站在地上）才能迈台阶
            let can_step = a != 1 && step > 0.0 && hit[1] && delta.y < 0.0;
            for _ in 0..steps {
                let part = d / steps as f64;
                let next = p.add_axis(a, part);
                if !collides(next) {
                    p = next;
                    continue;
                }
                if can_step {
                    let q = super::side(MIN_DEPTH);
                    let mut up = q;
                    let mut stepped = false;
                    while up <= step + 1e-9 {
                        let lifted = next.add_axis(1, up);
                        if !collides(lifted) && !collides(p.add_axis(1, up)) {
                            p = lifted;
                            stepped = true;
                            break;
                        }
                        up += q;
                    }
                    if stepped {
                        continue;
                    }
                }
                // 二分找到能走到的最远的地方
                let (mut lo, mut hi) = (0.0, part);
                for _ in 0..10 {
                    let mid = (lo + hi) / 2.0;
                    if collides(p.add_axis(a, mid)) {
                        hi = mid;
                    } else {
                        lo = mid;
                    }
                }
                p = p.add_axis(a, lo);
                hit[a] = true;
                break;
            }
        }
        (p, hit)
    }
}
