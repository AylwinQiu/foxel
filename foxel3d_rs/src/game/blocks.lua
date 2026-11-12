-- 基础材料。返回一个表：B.STONE、B.GRASS……是BlockId，B.hotbar是新游戏的物品栏。
-- mod的方块在这些之后注册。

local T = require("textures")
local rgb, img = T.rgb, T.image

local B = {}

-- 普通的不透明实心方块
local function simple(key, name, display, images)
    B[key] = block.register { name = name, display = display, images = images }
end

local dirt = T.dirt(3)
local snow = T.noise(9, rgb(240, 244, 250), 0.03)
local sand = T.noise(10, rgb(220, 206, 160), 0.05)
local grass = T.noise(4, rgb(98, 160, 58), 0.12)
local dirt_img = img(dirt)

simple("STONE", "foxel:stone", "Stone", img(T.stone(1)))
simple("COBBLESTONE", "foxel:cobblestone", "Cobblestone", img(T.cobblestone(2)))
simple("DIRT", "foxel:dirt", "Dirt", dirt_img)
simple("GRASS", "foxel:grass", "Grass",
    { top = img(grass), side = img(T.topped(5, dirt, rgb(98, 160, 58))), bottom = dirt_img })
simple("SNOWY_GRASS", "foxel:snowy_grass", "Snowy Grass",
    { top = img(snow), side = img(T.topped(6, dirt, rgb(240, 244, 250))), bottom = dirt_img })
simple("SNOW", "foxel:snow", "Snow", img(snow))
simple("SAND", "foxel:sand", "Sand", img(sand))
local sandstone_top = img(T.noise(11, rgb(222, 208, 160), 0.03))
simple("SANDSTONE", "foxel:sandstone", "Sandstone",
    { top = sandstone_top, side = img(T.layered(12, rgb(216, 200, 150))), bottom = sandstone_top })
simple("GRAVEL", "foxel:gravel", "Gravel", img(T.gravel(13)))
simple("COAL_ORE", "foxel:coal_ore", "Coal Ore", img(T.ore(14, rgb(35, 35, 35))))
simple("IRON_ORE", "foxel:iron_ore", "Iron Ore", img(T.ore(15, rgb(214, 172, 140))))

local oak_rings = img(T.rings(20, rgb(105, 82, 50), rgb(182, 148, 92), rgb(150, 118, 70)))
simple("OAK_LOG", "foxel:oak_log", "Oak Log",
    { top = oak_rings, side = img(T.bark(21, rgb(105, 82, 50))), bottom = oak_rings })
simple("OAK_PLANKS", "foxel:oak_planks", "Oak Planks", img(T.planks(22, rgb(165, 133, 80))))
-- 树叶有透明像素，不能挡住旁边的面；挨着的树叶之间的面也要画（能透过叶子看到里面的叶子）
B.OAK_LEAVES = block.register {
    name = "foxel:oak_leaves", display = "Oak Leaves", opaque = false,
    images = img(T.leaves(23, rgb(70, 140, 45), 0.22)),
}
local spruce_rings = img(T.rings(24, rgb(62, 44, 26), rgb(140, 102, 62), rgb(115, 82, 48)))
simple("SPRUCE_LOG", "foxel:spruce_log", "Spruce Log",
    { top = spruce_rings, side = img(T.bark(25, rgb(66, 46, 26))), bottom = spruce_rings })
simple("SPRUCE_PLANKS", "foxel:spruce_planks", "Spruce Planks", img(T.planks(26, rgb(112, 82, 50))))
B.SPRUCE_LEAVES = block.register {
    name = "foxel:spruce_leaves", display = "Spruce Leaves", opaque = false,
    images = img(T.leaves(27, rgb(48, 96, 58), 0.18)),
}

-- 两块玻璃挨着时省掉中间的面，看起来是一整块
B.GLASS = block.register {
    name = "foxel:glass", display = "Glass", opaque = false, cull_same = true, images = img(T.glass()),
}
simple("BRICKS", "foxel:bricks", "Bricks", img(T.bricks(30, rgb(150, 70, 55), rgb(178, 172, 162), 8, 4)))
simple("STONE_BRICKS", "foxel:stone_bricks", "Stone Bricks", img(T.bricks(31, rgb(122, 122, 122), rgb(90, 90, 90), 16, 8)))

B.BEDROCK = block.register {
    name = "foxel:bedrock", display = "Bedrock", unbreakable = true, hidden = true, images = img(T.bedrock(40)),
}

-- 玩家一开始物品栏里的方块
B.hotbar = { B.GRASS, B.DIRT, B.STONE, B.COBBLESTONE, B.OAK_LOG, B.OAK_PLANKS, B.GLASS, B.BRICKS, B.SAND }

return B
