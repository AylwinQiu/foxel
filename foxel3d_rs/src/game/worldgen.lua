-- 内置的地形：5种生物群系、煤矿和铁矿、橡树和云杉。
-- mod可以用同样的名字再登记一次来调整，比如
--   terrain.biome { name = "foxel:desert", variation = 8 }
--   terrain.scatter { name = "foxel:oak_trees", density = { ["foxel:plains"] = 0.02 } }

local B = require("blocks")

-- 世界的竖直范围是chunk -2到7（方块y从-32到127），最下面一层是基岩
terrain.settings { stone = B.STONE, bedrock = B.BEDROCK, snow = B.SNOW, min_chunk_y = -2, max_chunk_y = 7 }

terrain.biome {
    name = "foxel:plains", temperature = 0.2, humidity = -0.1,
    base_height = 12, variation = 5, surface = B.GRASS, subsurface = B.DIRT, subsurface_depth = 4,
}
terrain.biome {
    name = "foxel:forest", temperature = 0.25, humidity = 0.5,
    base_height = 15, variation = 9, surface = B.GRASS, subsurface = B.DIRT, subsurface_depth = 4,
}
terrain.biome {
    name = "foxel:desert", temperature = 0.8, humidity = -0.6,
    base_height = 10, variation = 4, surface = B.SAND, subsurface = B.SANDSTONE, subsurface_depth = 6,
}
terrain.biome {
    name = "foxel:mountains", temperature = -0.2, humidity = -0.5,
    base_height = 24, variation = 70, ridged = true, surface = B.GRASS, subsurface = B.DIRT, subsurface_depth = 3,
    rock_line = 58, snow_line = 78,
}
terrain.biome {
    name = "foxel:taiga", temperature = -0.7, humidity = 0.3,
    base_height = 16, variation = 10, surface = B.SNOWY_GRASS, subsurface = B.DIRT, subsurface_depth = 4,
}

terrain.ore { name = "foxel:coal", salt = 101, block = B.COAL_ORE, per_chunk = 8, size = 10, max_y = 90 }
terrain.ore { name = "foxel:iron", salt = 102, block = B.IRON_ORE, per_chunk = 4, size = 6, max_y = 30 }

-- 橡树：4到6格高的树干，顶上一团圆圆的树叶。y是地面方块的高度
local function grow_oak(ctx, x, y, z, r)
    local h = 4 + r:random(0, 2)
    local top = y + h
    for dy = -2, 1 do
        local radius = dy >= 0 and 1 or 2
        for dz = -radius, radius do
            for dx = -radius, radius do
                local corner = (dx == -radius or dx == radius) and (dz == -radius or dz == radius)
                -- 角上的叶子随机缺一些，最上面一层只留十字
                if not (corner and (dy == 1 or r:random(0, 1) == 0)) then
                    ctx:set_if_air(x + dx, top + dy, z + dz, B.OAK_LEAVES)
                end
            end
        end
    end
    for dy = 1, h do
        ctx:set(x, y + dy, z, B.OAK_LOG)
    end
    ctx:set(x, y, z, B.DIRT)
end

-- 云杉：细高的树干，树叶一层大一层小地往下变宽，像个圆锥
local function grow_spruce(ctx, x, y, z, r)
    local h = 7 + r:random(0, 3)
    local top = y + h
    ctx:set_if_air(x, top + 1, z, B.SPRUCE_LEAVES)
    for ly = top, y + 3, -1 do
        -- 离树顶越远越宽，每隔一层缩回去一点
        local d = top - ly
        local radius = math.min(1 + d // 3, 3)
        if d % 2 == 0 then
            radius = math.max(radius - 1, 1)
        end
        for dz = -radius, radius do
            for dx = -radius, radius do
                if dx * dx + dz * dz <= radius * radius + 1 then
                    ctx:set_if_air(x + dx, ly, z + dz, B.SPRUCE_LEAVES)
                end
            end
        end
    end
    for dy = 1, h do
        ctx:set(x, y + dy, z, B.SPRUCE_LOG)
    end
    ctx:set(x, y, z, B.DIRT)
end

terrain.scatter {
    name = "foxel:oak_trees", salt = 201, radius = 2, height = 20, soil = { B.GRASS },
    density = { ["foxel:plains"] = 0.004, ["foxel:forest"] = 0.045, ["foxel:mountains"] = 0.004 },
    place = grow_oak,
}
terrain.scatter {
    name = "foxel:spruce_trees", salt = 202, radius = 3, height = 20, soil = { B.GRASS, B.SNOWY_GRASS },
    density = { ["foxel:taiga"] = 0.03, ["foxel:mountains"] = 0.012 },
    place = grow_spruce,
}
