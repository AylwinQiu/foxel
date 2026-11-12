// 玩家：移动、碰撞、挖和放方块
package game

import (
	"math"

	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

const (
	PLAYER_WIDTH  = 0.6
	PLAYER_HEIGHT = 1.8
	EYE_HEIGHT    = 1.62
	// 能挖到、放到多远的方块
	REACH = 6.0

	WALK_SPEED   = 4.3
	SPRINT_SPEED = 6.5
	SNEAK_SPEED  = 1.5
	FLY_SPEED    = 11.0
	GRAVITY      = 30.0
	JUMP_SPEED   = 8.6
	MAX_FALL     = 60.0
	// 走路时能自动迈上多高的台阶
	STEP_HEIGHT = 0.5
	// 鼠标每移动一个像素转多少弧度
	MOUSE_SENSITIVITY = 0.0025
	// 按住鼠标时每隔多少秒挖/放一次
	REPEAT_INTERVAL = 0.22
)

type Player struct {
	// 脚底中心
	Pos      world.PhysicalPosition3D
	Velocity world.PhysicalPosition3D
	// 弧度。Yaw=0看向北（-z），往右转为正；Pitch往上看为正
	Yaw   float64
	Pitch float64
	// 飞行时没有重力，空格上升、Shift下降
	Flying    bool
	On_ground bool
}

// 眼睛的位置
func (p *Player) Eye() world.PhysicalPosition3D {
	return p.Pos.Add(world.PhysicalPosition3D{Y: EYE_HEIGHT})
}

// 视线方向（单位向量）
func (p *Player) Look_dir() world.PhysicalPosition3D {
	cp := math.Cos(p.Pitch)
	return world.PhysicalPosition3D{X: math.Sin(p.Yaw) * cp, Y: math.Sin(p.Pitch), Z: -math.Cos(p.Yaw) * cp}
}

// 玩家站在pos时的碰撞盒
func player_box(pos world.PhysicalPosition3D) (lo, hi world.PhysicalPosition3D) {
	h := PLAYER_WIDTH / 2
	return pos.Sub(world.PhysicalPosition3D{X: h, Z: h}), pos.Add(world.PhysicalPosition3D{X: h, Y: PLAYER_HEIGHT, Z: h})
}

// 碰撞盒和实心方块重叠吗。没加载的地方当成实心的，玩家不会掉进还没生成的世界里
func (g *Game) box_collides(lo, hi world.PhysicalPosition3D) bool {
	s := world.MIN_DEPTH.Side()
	cells := func(a, b float64) (int64, int64) {
		return int64(math.Floor(a / s)), int64(math.Ceil(b/s)) - 1
	}
	x0, x1 := cells(lo.X, hi.X)
	y0, y1 := cells(lo.Y, hi.Y)
	z0, z1 := cells(lo.Z, hi.Z)
	for y := y0; y <= y1; y++ {
		for z := z0; z <= z1; z++ {
			for x := x0; x <= x1; x++ {
				leaf, ok := g.World.Leaf_at(world.CubeID{X: x, Y: y, Z: z, Depth: world.MIN_DEPTH}.Center())
				if !ok || block.Is_solid(leaf.Block) {
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) player_collides(pos world.PhysicalPosition3D) bool {
	return g.box_collides(player_box(pos))
}

// 刚进游戏时如果卡在方块里，往上挪到空的地方
func (g *Game) unstuck_player() {
	for range 256 {
		if !g.player_collides(g.Player.Pos) {
			return
		}
		g.Player.Pos.Y += 1
	}
}

func (g *Game) update_player(in canvas.Input, dt float64, playing bool) {
	p := &g.Player
	if playing {
		p.Yaw += in.Mouse_dx * MOUSE_SENSITIVITY
		p.Pitch = math.Max(-1.55, math.Min(1.55, p.Pitch-in.Mouse_dy*MOUSE_SENSITIVITY))
	} else {
		in = canvas.Input{}
	}

	// 想往哪里走（水平方向）
	forward := world.PhysicalPosition3D{X: math.Sin(p.Yaw), Z: -math.Cos(p.Yaw)}
	right := world.PhysicalPosition3D{X: math.Cos(p.Yaw), Z: math.Sin(p.Yaw)}
	axis := func(plus, minus canvas.Key) float64 {
		v := 0.0
		if in.Is_held(plus) {
			v++
		}
		if in.Is_held(minus) {
			v--
		}
		return v
	}
	wish := forward.Scale(axis(canvas.KEY_W, canvas.KEY_S)).Add(right.Scale(axis(canvas.KEY_D, canvas.KEY_A)))
	if l := wish.Length(); l > 0 {
		wish = wish.Scale(1 / l)
	}
	sprint := in.Is_held(canvas.KEY_LEFT_CTRL)
	sneak := in.Is_held(canvas.KEY_LEFT_SHIFT)

	if p.Flying {
		speed := FLY_SPEED
		if sprint {
			speed *= 2
		}
		v := wish.Scale(speed)
		v.Y = axis(canvas.KEY_SPACE, canvas.KEY_LEFT_SHIFT) * speed
		p.Velocity = v
	} else {
		speed := WALK_SPEED
		switch {
		case sneak:
			speed = SNEAK_SPEED
		case sprint:
			speed = SPRINT_SPEED
		}
		p.Velocity.X, p.Velocity.Z = wish.X*speed, wish.Z*speed
		p.Velocity.Y = math.Max(p.Velocity.Y-GRAVITY*dt, -MAX_FALL)
		if p.On_ground && in.Is_held(canvas.KEY_SPACE) {
			p.Velocity.Y = JUMP_SPEED
		}
	}

	p.On_ground = false
	g.move_axis(1, p.Velocity.Y*dt)
	g.move_axis(0, p.Velocity.X*dt)
	g.move_axis(2, p.Velocity.Z*dt)
}

func add_axis(v world.PhysicalPosition3D, a int, d float64) world.PhysicalPosition3D {
	switch a {
	case 0:
		v.X += d
	case 1:
		v.Y += d
	default:
		v.Z += d
	}
	return v
}

// 沿第a个轴（0是x，1是y，2是z）移动d，碰到方块就停在方块边上。
// 每次最多走0.2格，比最小的方块（1/4格）小，所以不会穿过去
func (g *Game) move_axis(a int, d float64) {
	p := &g.Player
	steps := int(math.Ceil(math.Abs(d) / 0.2))
	for range steps {
		part := d / float64(steps)
		next := add_axis(p.Pos, a, part)
		if !g.player_collides(next) {
			p.Pos = next
			continue
		}
		// 走路时碰到矮的台阶自动迈上去
		if a != 1 && p.On_ground && !p.Flying {
			stepped := false
			for up := world.MIN_DEPTH.Side(); up <= STEP_HEIGHT; up += world.MIN_DEPTH.Side() {
				lifted := add_axis(next, 1, up)
				if !g.player_collides(lifted) && !g.player_collides(add_axis(p.Pos, 1, up)) {
					p.Pos = lifted
					stepped = true
					break
				}
			}
			if stepped {
				continue
			}
		}
		// 二分找到能走到的最远的地方
		lo, hi := 0.0, part
		for range 10 {
			mid := (lo + hi) / 2
			if g.player_collides(add_axis(p.Pos, a, mid)) {
				hi = mid
			} else {
				lo = mid
			}
		}
		p.Pos = add_axis(p.Pos, a, lo)
		if a == 1 && part < 0 {
			p.On_ground = true
		}
		switch a {
		case 0:
			p.Velocity.X = 0
		case 1:
			p.Velocity.Y = 0
		default:
			p.Velocity.Z = 0
		}
		return
	}
}

// ---------------------------------------------------------------- 挖和放

// 指着的方块里要挖掉的格子（Build_depth大小）
func (g *Game) Target_cell() (world.CubeID, bool) {
	if g.Target == nil {
		return world.CubeID{}, false
	}
	n := g.Target.Normal
	inside := g.Target.Point.Sub(world.PhysicalPosition3D{X: float64(n[0]), Y: float64(n[1]), Z: float64(n[2])}.Scale(1e-4))
	return world.Cube_id_at(inside, g.Build_depth), true
}

// 指着的那个面外面要放方块的格子（Build_depth大小）
func (g *Game) Place_cell() (world.CubeID, bool) {
	if g.Target == nil || g.Target.Normal == [3]int64{} {
		return world.CubeID{}, false
	}
	n := g.Target.Normal
	d := g.Build_depth.Side() / 2
	outside := g.Target.Point.Add(world.PhysicalPosition3D{X: float64(n[0]), Y: float64(n[1]), Z: float64(n[2])}.Scale(d))
	return world.Cube_id_at(outside, g.Build_depth), true
}

func (g *Game) interact(in canvas.Input, dt float64) {
	g.cooldown -= dt
	if g.wait_release {
		if in.Is_held(canvas.MOUSE_LEFT) {
			return
		}
		g.wait_release = false
	}
	if g.Target == nil {
		return
	}
	act := func(k canvas.Key) bool {
		return in.Is_pressed(k) || (in.Is_held(k) && g.cooldown <= 0)
	}
	switch {
	case act(canvas.MOUSE_LEFT):
		if id, ok := g.Target_cell(); ok {
			g.Break(id)
		}
		g.cooldown = REPEAT_INTERVAL
	case act(canvas.MOUSE_RIGHT):
		if id, ok := g.Place_cell(); ok {
			g.Place(id, g.Hotbar[g.Selected])
		}
		g.cooldown = REPEAT_INTERVAL
	case in.Is_pressed(canvas.MOUSE_MIDDLE):
		if b := g.Target.Leaf.Block; !block.Get(b).Hidden {
			g.Hotbar[g.Selected] = b
		}
	}
}

// id这个格子里有没有挖不掉的方块
func (g *Game) contains_unbreakable(id world.CubeID) bool {
	node, _, ok := g.World.Find(id)
	if !ok {
		return true
	}
	var check func(n *world.Cube) bool
	check = func(n *world.Cube) bool {
		if n.Children == nil {
			return block.Get(n.Block).Unbreakable
		}
		for i := range n.Children {
			if check(&n.Children[i]) {
				return true
			}
		}
		return false
	}
	return check(&node)
}

// 挖掉id这个格子（里面的东西全部变成空气）。挖不掉、mod不让挖时返回false
func (g *Game) Break(id world.CubeID) bool {
	if g.contains_unbreakable(id) {
		return false
	}
	var b world.BlockID
	if g.Target != nil {
		b = g.Target.Leaf.Block
	}
	for _, m := range mods {
		if m.On_break != nil && !m.On_break(g, id, b) {
			return false
		}
	}
	return g.World.Set_cube(id, world.AIR) == nil
}

// 在id这个格子放方块b。格子不是全空的、会卡住玩家、mod不让放时返回false
func (g *Game) Place(id world.CubeID, b world.BlockID) bool {
	if b == world.AIR {
		return false
	}
	node, _, ok := g.World.Find(id)
	if !ok || node.Children != nil || node.Block != world.AIR {
		return false
	}
	if block.Is_solid(b) {
		lo, hi := player_box(g.Player.Pos)
		clo, chi := id.Min(), id.Max()
		if lo.X < chi.X && hi.X > clo.X && lo.Y < chi.Y && hi.Y > clo.Y && lo.Z < chi.Z && hi.Z > clo.Z {
			return false
		}
	}
	for _, m := range mods {
		if m.On_place != nil && !m.On_place(g, id, b) {
			return false
		}
	}
	return g.World.Set_cube(id, b) == nil
}
