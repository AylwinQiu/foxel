// 基础材料。这些方块在block包初始化时注册，mod的方块在它们之后。
package block

import (
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

var (
	dirt_img  = dirt_texture(3)
	snow_img  = Noise_texture(9, rgb(240, 244, 250), 0.03)
	sand_img  = Noise_texture(10, rgb(220, 206, 160), 0.05)
	grass_img = Noise_texture(4, rgb(98, 160, 58), 0.12)
)

var (
	STONE       = Register(Simple("foxel:stone", "Stone", canvas.Images{}.All(stone_texture(1))))
	COBBLESTONE = Register(Simple("foxel:cobblestone", "Cobblestone", canvas.Images{}.All(cobblestone_texture(2))))
	DIRT        = Register(Simple("foxel:dirt", "Dirt", canvas.Images{}.All(dirt_img)))
	GRASS       = Register(Simple("foxel:grass", "Grass", canvas.Images{}.Column(
		grass_img, topped_texture(5, dirt_img, rgb(98, 160, 58)), dirt_img)))
	SNOWY_GRASS = Register(Simple("foxel:snowy_grass", "Snowy Grass", canvas.Images{}.Column(
		snow_img, topped_texture(6, dirt_img, rgb(240, 244, 250)), dirt_img)))
	SNOW      = Register(Simple("foxel:snow", "Snow", canvas.Images{}.All(snow_img)))
	SAND      = Register(Simple("foxel:sand", "Sand", canvas.Images{}.All(sand_img)))
	SANDSTONE = Register(Simple("foxel:sandstone", "Sandstone", canvas.Images{}.Column(
		Noise_texture(11, rgb(222, 208, 160), 0.03), layered_texture(12, rgb(216, 200, 150)), Noise_texture(11, rgb(222, 208, 160), 0.03))))
	GRAVEL   = Register(Simple("foxel:gravel", "Gravel", canvas.Images{}.All(gravel_texture(13))))
	COAL_ORE = Register(Simple("foxel:coal_ore", "Coal Ore", canvas.Images{}.All(ore_texture(14, rgb(35, 35, 35)))))
	IRON_ORE = Register(Simple("foxel:iron_ore", "Iron Ore", canvas.Images{}.All(ore_texture(15, rgb(214, 172, 140)))))

	OAK_LOG = Register(Simple("foxel:oak_log", "Oak Log", canvas.Images{}.Column(
		rings_texture(20, rgb(105, 82, 50), rgb(182, 148, 92), rgb(150, 118, 70)), bark_texture(21, rgb(105, 82, 50)),
		rings_texture(20, rgb(105, 82, 50), rgb(182, 148, 92), rgb(150, 118, 70)))))
	OAK_PLANKS = Register(Simple("foxel:oak_planks", "Oak Planks", canvas.Images{}.All(planks_texture(22, rgb(165, 133, 80)))))
	OAK_LEAVES = Register(Type{Name: "foxel:oak_leaves", Display: "Oak Leaves",
		Images: canvas.Images{}.All(leaves_texture(23, rgb(70, 140, 45), 0.22)), Solid: true})
	SPRUCE_LOG = Register(Simple("foxel:spruce_log", "Spruce Log", canvas.Images{}.Column(
		rings_texture(24, rgb(62, 44, 26), rgb(140, 102, 62), rgb(115, 82, 48)), bark_texture(25, rgb(66, 46, 26)),
		rings_texture(24, rgb(62, 44, 26), rgb(140, 102, 62), rgb(115, 82, 48)))))
	SPRUCE_PLANKS = Register(Simple("foxel:spruce_planks", "Spruce Planks", canvas.Images{}.All(planks_texture(26, rgb(112, 82, 50)))))
	SPRUCE_LEAVES = Register(Type{Name: "foxel:spruce_leaves", Display: "Spruce Leaves",
		Images: canvas.Images{}.All(leaves_texture(27, rgb(48, 96, 58), 0.18)), Solid: true})

	GLASS = Register(Type{Name: "foxel:glass", Display: "Glass",
		Images: canvas.Images{}.All(glass_texture()), Solid: true, Cull_same: true})
	BRICKS       = Register(Simple("foxel:bricks", "Bricks", canvas.Images{}.All(brick_texture(30, rgb(150, 70, 55), rgb(178, 172, 162), 8, 4))))
	STONE_BRICKS = Register(Simple("foxel:stone_bricks", "Stone Bricks", canvas.Images{}.All(brick_texture(31, rgb(122, 122, 122), rgb(90, 90, 90), 16, 8))))

	BEDROCK = Register(Type{Name: "foxel:bedrock", Display: "Bedrock",
		Images: canvas.Images{}.All(bedrock_texture(40)), Solid: true, Opaque: true, Unbreakable: true, Hidden: true})
)

// 玩家一开始物品栏里的方块
var Default_hotbar = []world.BlockID{GRASS, DIRT, STONE, COBBLESTONE, OAK_LOG, OAK_PLANKS, GLASS, BRICKS, SAND}
