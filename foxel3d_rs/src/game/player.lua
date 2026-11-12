-- 玩家：移动、跳、飞、跑、蹲。碰撞由Rust的world.move_box做，这里只管速度和姿势。

local V = require("vec")

local P = {
    WIDTH = 0.6,
    HEIGHT = 1.8,
    EYE = 1.62,
    -- 蹲下时的高度和眼睛高度：比1格矮，能钻过1格高的缝
    CROUCH_HEIGHT = 0.95,
    CROUCH_EYE = 0.8,
    REACH = 6.0,
    WALK = 4.3,
    SPRINT = 6.5,
    SNEAK = 1.5,
    FLY = 11.0,
    GRAVITY = 30.0,
    JUMP = 8.6,
    MAX_FALL = 60.0,
    -- 能直接走上去的台阶高度
    STEP = 0.5,
    MOUSE_SENSITIVITY = 0.0025,
    -- 两次按W间隔小于这个时间（秒）就开始跑
    DOUBLE_TAP = 0.3,
    -- 蹲下、站起时眼睛移动的快慢
    EYE_SPEED = 15,
}

-- pos是脚底中心。yaw是0时朝北（-z），pi/2朝东；pitch向上为正
function P.new(pos)
    return {
        pos = pos, vel = V.new(), yaw = 0, pitch = 0,
        flying = false, on_ground = false,
        -- 蹲着（Shift，头顶有东西时松开Shift也站不起来）、跑着（双击W或者按住Ctrl）
        crouching = false, sprinting = false,
        -- 现在眼睛的高度，蹲下、站起时慢慢变过去
        eye_height = P.EYE,
        -- 上一次按下W的时间，判断双击用
        clock = 0, last_w = -1,
    }
end

-- 现在的身高
function P.height(p)
    return p.crouching and P.CROUCH_HEIGHT or P.HEIGHT
end

-- 眼睛的位置
function P.eye(p)
    return V.add(p.pos, V.new(0, p.eye_height or P.EYE, 0))
end

-- 视线方向（单位向量）
function P.look_dir(p)
    local cp = math.cos(p.pitch)
    return V.new(math.sin(p.yaw) * cp, math.sin(p.pitch), -math.cos(p.yaw) * cp)
end

-- 站在pos、身高height（默认站着的高度）时的碰撞盒
function P.box(pos, height)
    local h = P.WIDTH / 2
    return V.new(pos.x - h, pos.y, pos.z - h), V.new(pos.x + h, pos.y + (height or P.HEIGHT), pos.z + h)
end

function P.collides(pos, height)
    return world.box_collides(P.box(pos, height))
end

-- 刚进游戏时如果卡在方块里，往上挪到空的地方
function P.unstuck(p)
    for _ = 1, 256 do
        if not P.collides(p.pos, P.height(p)) then
            return
        end
        p.pos.y = p.pos.y + 1
    end
end

-- 蹲下、站起：想蹲就蹲；不想蹲时，头顶有空间才站起来
local function update_posture(p, want, dt)
    if want then
        p.crouching = true
    elseif p.crouching and not P.collides(p.pos, P.HEIGHT) then
        p.crouching = false
    end
    local target = p.crouching and P.CROUCH_EYE or P.EYE
    p.eye_height = p.eye_height + (target - p.eye_height) * math.min(1, dt * P.EYE_SPEED)
end

-- 双击W开始跑，松开W、蹲下时停下。按住Ctrl也算跑
local function update_sprint(p, held, pressed)
    if pressed.w then
        if p.clock - p.last_w < P.DOUBLE_TAP then
            p.sprinting = true
        end
        p.last_w = p.clock
    end
    if not held.w or p.crouching then
        p.sprinting = false
    end
    return p.sprinting or held.left_ctrl
end

-- playing是false时（暂停、开着物品栏）不响应输入，但重力照样有
function P.update(p, input, dt, playing)
    local held = playing and input.held or {}
    local pressed = playing and input.pressed or {}
    p.clock = p.clock + dt
    if playing then
        p.yaw = p.yaw + input.mouse_dx * P.MOUSE_SENSITIVITY
        p.pitch = math.max(-1.55, math.min(1.55, p.pitch - input.mouse_dy * P.MOUSE_SENSITIVITY))
    end

    local function axis(plus, minus)
        return (held[plus] and 1 or 0) - (held[minus] and 1 or 0)
    end
    -- 想往哪里走（水平方向）
    local forward = V.new(math.sin(p.yaw), 0, -math.cos(p.yaw))
    local right = V.new(math.cos(p.yaw), 0, math.sin(p.yaw))
    local wish = V.normalize(V.add(V.scale(forward, axis("w", "s")), V.scale(right, axis("d", "a"))))

    -- 飞的时候Shift是下降，不是蹲
    update_posture(p, held.left_shift and not p.flying, dt)
    local sprint = update_sprint(p, held, pressed)

    if p.flying then
        local speed = sprint and P.FLY * 2 or P.FLY
        p.vel = V.scale(wish, speed)
        p.vel.y = axis("space", "left_shift") * speed
    else
        local speed = p.crouching and P.SNEAK or (sprint and P.SPRINT or P.WALK)
        p.vel.x, p.vel.z = wish.x * speed, wish.z * speed
        p.vel.y = math.max(p.vel.y - P.GRAVITY * dt, -P.MAX_FALL)
        if p.on_ground and held.space then
            p.vel.y = P.JUMP
        end
    end

    local delta = V.scale(p.vel, dt)
    local step = p.flying and 0 or P.STEP
    local pos, hit = world.move_box(p.pos, P.WIDTH, P.height(p), delta, step)
    p.pos = pos
    p.on_ground = hit.y and delta.y < 0
    for _, a in ipairs({ "x", "y", "z" }) do
        if hit[a] then
            p.vel[a] = 0
        end
    end
end

-- 存档用。蹲着也要存：存档时在矮的地方，读档后站着会卡进方块里
function P.save(p)
    return { pos = p.pos, yaw = p.yaw, pitch = p.pitch, flying = p.flying, crouching = p.crouching }
end

function P.load(t)
    local p = P.new(V.copy(t.pos))
    p.yaw, p.pitch, p.flying = t.yaw or 0, t.pitch or 0, t.flying or false
    p.crouching = t.crouching or false
    p.eye_height = p.crouching and P.CROUCH_EYE or P.EYE
    return p
end

return P
