-- 示例mod：草地上长着小花，按G在身边种一圈花。
--
-- 演示了mod的三种接口：
--   - 注册新方块（花茎、花瓣）
--   - 登记地形feature（生成世界时在草地上种花）
--   - 登记游戏事件（按键种花，挖到花时显示消息）
--
-- 花是用1/4格大小的方块拼起来的：一格茎，上面一格花瓣。玩家可以穿过花（solid = false）。

local B = require("blocks")
local events = require("events")

local function flower(name, display, color)
    return block.register { name = name, display = display, images = color, solid = false }
end

local STEM = flower("flowers:stem", "Flower Stem", { 70, 150, 50 })
local PETALS = {
    flower("flowers:red", "Red Petal", { 220, 40, 50 }),
    flower("flowers:yellow", "Yellow Petal", { 250, 215, 40 }),
    flower("flowers:blue", "Blue Petal", { 80, 120, 240 }),
}

-- 在方块(x, y, z)这个格子的底部种一朵花，y是地面上面那个空格子。set(x, y, z, b)放一个1/4格的方块
local function plant(x, y, z, r, set)
    local n = 4 -- 一格分成4份
    local sx, sz = x * n + 1 + r:random(0, 1), z * n + 1 + r:random(0, 1)
    local height = 1 + r:random(0, 1)
    for i = 0, height - 1 do
        set(sx, y * n + i, sz, STEM)
    end
    set(sx, y * n + height, sz, PETALS[r:random(#PETALS)])
end

-- 生成世界时：平原和森林的草地上，每列有一定概率长一朵花
terrain.scatter {
    name = "flowers:flowers", salt = 0xf10e, height = 1, soil = { B.GRASS },
    density = { ["foxel:plains"] = 0.05, ["foxel:forest"] = 0.02 },
    place = function(ctx, x, y, z, r)
        -- 上面已经有东西了（比如树干），或者不在这个chunk里
        if ctx:get(x, y + 1, z) ~= block.AIR then
            return
        end
        plant(x, y + 1, z, r, function(cx, cy, cz, b)
            ctx:set_small(cx, cy, cz, -2, b)
        end)
    end,
}

-- 按G：在身边的草地上种几朵花
events.on("update", function(G, input)
    if not G.playing() or not input.pressed.g then
        return
    end
    local p = G.player.pos
    local r = foxel.rng(math.floor(G.time * 1000))
    local planted = 0
    for _ = 1, 16 do
        local x = math.floor(p.x) + r:random(-4, 4)
        local z = math.floor(p.z) + r:random(-4, 4)
        local y = G.surface_below(x, math.floor(p.y) + 3, z, 8)
        if y then
            local below = world.get(x, y - 1, z)
            local here, split = world.get(x, y, z)
            if (below == B.GRASS or below == B.SNOWY_GRASS) and here == block.AIR and not split then
                plant(x, y, z, r, function(cx, cy, cz, b)
                    world.set_cube({ x = cx, y = cy, z = cz, depth = -2 }, b)
                end)
                planted = planted + 1
            end
        end
    end
    G.show_message(("Planted %d flowers"):format(planted))
end)

-- 挖到花瓣时显示消息
events.on("break", function(G, _, b)
    for _, p in ipairs(PETALS) do
        if b == p then
            G.show_message("You picked a " .. G.info(b).display)
        end
    end
end)
