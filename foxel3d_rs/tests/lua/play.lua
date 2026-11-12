-- 走路、跳、撞墙、上台阶、挖、放、存档。世界是新的（种子5）

local H = dofile(TEST_DIR .. "/helpers.lua")
local G, B, P = H.G, H.B, require("player")

H.start(5)

-- 在高处铺一块10x10的石头平台，上面清空，玩家站在上面
for x = -5, 4 do
    for z = -5, 4 do
        assert(world.set(x, 60, z, B.STONE))
        for y = 61, 65 do
            world.set(x, y, z, block.AIR)
        end
    end
end
G.player = P.new({ x = 0.5, y = 61, z = 0.5 })

H.frames(30)
assert(G.player.on_ground, "not standing")
H.near(G.player.pos.y, 61, 0.01, "stand")

-- 往北走1秒
H.frames(60, { "w" })
H.near(0.5 - G.player.pos.z, P.WALK, 0.3, "walk")

-- 跳起来再落下
H.frames(5, { "space" })
assert(G.player.pos.y > 61.2, "no jump")
H.frames(60)
H.near(G.player.pos.y, 61, 0.01, "land")

-- 北边z=-2有一堵两格高的墙
G.player = P.new({ x = 0.5, y = 61, z = 0.5 })
for x = -5, 4 do
    world.set(x, 61, -2, B.STONE)
    world.set(x, 62, -2, B.STONE)
end
H.frames(60, { "w" })
H.near(G.player.pos.z, -1 + P.WIDTH / 2, 0.01, "wall")

-- 东边铺一层半格高的台阶：直接走上去
G.player = P.new({ x = 0.5, y = 61, z = 0.5 })
G.player.yaw = math.pi / 2
for z = -2, 9 do
    for x = 2, 9 do
        world.set_cube({ x = x, y = 122, z = z, depth = -1 }, B.STONE)
    end
end
H.frames(30, { "w" })
assert(G.player.pos.x > 1.5, "step x")
H.near(G.player.pos.y, 61.5, 0.01, "step y")

-- 双击W开始跑，松开W停下；按住Ctrl也能跑
local function horizontal_speed()
    return math.sqrt(G.player.vel.x ^ 2 + G.player.vel.z ^ 2)
end
G.player = P.new({ x = 0.5, y = 61, z = 2.5 })
G.player.yaw = math.pi -- 朝南
H.frames(5)
H.frames(1, {}, { "w" })
H.frames(3)
H.frames(1, {}, { "w" })
H.frames(3, { "w" })
assert(G.player.sprinting, "double tap")
H.near(horizontal_speed(), P.SPRINT, 0.01, "sprint speed")
H.frames(2)
assert(not G.player.sprinting, "still sprinting after release")
-- 两次按W隔太久不算
G.player = P.new({ x = 0.5, y = 61, z = 2.5 })
G.player.yaw = math.pi
H.frames(1, {}, { "w" })
H.frames(30)
H.frames(1, {}, { "w" })
assert(not G.player.sprinting, "slow double tap")
H.frames(2, { "w", "left_ctrl" })
H.near(horizontal_speed(), P.SPRINT, 0.01, "ctrl sprint")

-- 西边有一条1格高的缝（地面y=60，顶在y=62）：站着进不去，蹲着能钻过去
for x = -5, -2 do
    for z = 0, 4 do
        world.set(x, 62, z, B.STONE)
    end
end
G.player = P.new({ x = 0.5, y = 61, z = 2.5 })
G.player.yaw = -math.pi / 2 -- 朝西
H.frames(60, { "w" })
H.near(G.player.pos.x, -1 + P.WIDTH / 2, 0.01, "standing enters gap")
H.frames(120, { "w", "left_shift" })
assert(G.player.crouching and G.player.pos.x < -3, "crouch through gap")
H.near(P.eye(G.player).y, 61 + P.CROUCH_EYE, 0.01, "crouch eye")
-- 在缝里松开Shift：头顶有东西，站不起来
H.frames(10)
assert(G.player.crouching, "stood up under ceiling")
-- 往回走出缝，自动站起来
G.player.yaw = math.pi / 2
for _ = 1, 300 do
    if not G.player.crouching then
        break
    end
    H.frames(1, { "w" })
end
assert(not G.player.crouching, "never stood up")
-- 头刚好离开顶的时候站起来
H.near(G.player.pos.x, -1 + P.WIDTH / 2, 0.1, "stand up outside")
H.frames(30)
H.near(P.eye(G.player).y, 61 + P.EYE, 0.01, "stand eye")

-- 暂停时点击回到游戏：这次点击不挖方块，按住也不挖
G.player = P.new({ x = 0.5, y = 61, z = 0.5 })
G.player.pitch = -1.2
H.frames(10)
local cell = G.target_cell()
assert(cell and cell.y == 60, "target")
canvas.lock_cursor(false)
H.frames(1, {}, { "mouse_left" })
assert(canvas.cursor_locked(), "not resumed")
H.frames(30, { "mouse_left" })
assert(world.get(cell.x, cell.y, cell.z) == B.STONE, "resume click broke a block")
-- 松开再点：挖掉
H.frames(1)
H.frames(1, {}, { "mouse_left" })
assert(world.get(cell.x, cell.y, cell.z) == block.AIR, "click did not break")

-- R：往小切，切到最小（-4）回到0
G.build_depth = 0
H.frames(1, {}, { "r" })
assert(G.build_depth == -1, "r")
H.frames(3, {}, { "r" })
assert(G.build_depth == -4, "r to -4")
H.frames(1, {}, { "r" })
assert(G.build_depth == 0, "r wraps")
-- 1/16格的方块
local tiny = { x = -50, y = 61 * 16, z = -50, depth = -4 }
assert(G.place(tiny, B.STONE) and world.find(tiny) == B.STONE, "tiny place")
assert(world.set_cube(tiny, block.AIR))

-- 不能放在自己身上
assert(not G.place(cube.block(0, 61, 0), B.STONE), "placed inside player")
-- 1/4大小的方块
G.build_depth = cube.MIN_DEPTH
local small = { x = -16, y = 244, z = -16, depth = -2 }
assert(G.place(small, B.GLASS), "small place")
assert(not G.place(small, B.STONE), "placed twice")
world.field_write("test:note", small, "hi")
G.mod_data["test:x"] = { 1, 2, 3 }
G.hotbar[4] = B.BRICKS
G.save()

-- 给reload.lua检查用
SAVED = { cell = cell, small = small, pos = G.player.pos }
