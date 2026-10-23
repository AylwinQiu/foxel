package world

import "math"

// 旋转用的单位四元数，只在world包内部用来组合旋转
type quat struct {
	w, x, y, z float64
}

var quatIdentity = quat{w: 1}

// r是旋转轴，长度是转动弧度（右手定则），和Cube.Rotation、Rotation3D的约定一致
func quatFromRotation(r [3]float64) quat {
	angle := math.Sqrt(r[0]*r[0] + r[1]*r[1] + r[2]*r[2])
	if angle == 0 {
		return quatIdentity
	}
	s := math.Sin(angle/2) / angle
	return quat{w: math.Cos(angle / 2), x: r[0] * s, y: r[1] * s, z: r[2] * s}
}

// a.mul(b)：先转b再转a
func (a quat) mul(b quat) quat {
	return quat{
		w: a.w*b.w - a.x*b.x - a.y*b.y - a.z*b.z,
		x: a.w*b.x + a.x*b.w + a.y*b.z - a.z*b.y,
		y: a.w*b.y - a.x*b.z + a.y*b.w + a.z*b.x,
		z: a.w*b.z + a.x*b.y - a.y*b.x + a.z*b.w,
	}
}

// 单位四元数的逆
func (q quat) conj() quat {
	return quat{w: q.w, x: -q.x, y: -q.y, z: -q.z}
}

func (q quat) rotate(v [3]float64) [3]float64 {
	p := q.mul(quat{x: v[0], y: v[1], z: v[2]}).mul(q.conj())
	return [3]float64{p.x, p.y, p.z}
}

// 转回轴*弧度的表示，转角取[0, pi]
func (q quat) toRotation() Rotation3D {
	if q.w < 0 {
		q = quat{-q.w, -q.x, -q.y, -q.z}
	}
	n := math.Sqrt(q.x*q.x + q.y*q.y + q.z*q.z)
	if n < 1e-12 {
		return Rotation3D{X: 2 * q.x, Y: 2 * q.y, Z: 2 * q.z}
	}
	k := 2 * math.Atan2(n, q.w) / n
	return Rotation3D{X: q.x * k, Y: q.y * k, Z: q.z * k}
}
