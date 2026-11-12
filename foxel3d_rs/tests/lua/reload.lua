-- 用另一个种子打开play.lua存的世界：种子、方块、场、物品栏、玩家都是存档里的

local H = dofile(TEST_DIR .. "/helpers.lua")
local G, B = H.G, H.B

H.start(99)

assert(world.seed() == 5, "seed")
assert(G.hotbar[4] == B.BRICKS, "hotbar")
assert(G.mod_data["test:x"][3] == 3, "mod_data")
local cell, small = SAVED.cell, SAVED.small
assert(world.get(cell.x, cell.y, cell.z) == block.AIR, "broken block came back")
local b, leaf = world.leaf_at(cube.center(small))
assert(b == B.GLASS and cube.equal(leaf, small), "small block")
assert(world.field_read("test:note", small) == "hi", "field")
H.near(G.player.pos.x, SAVED.pos.x, 1e-6, "player x")
H.near(G.player.pos.z, SAVED.pos.z, 1e-6, "player z")
