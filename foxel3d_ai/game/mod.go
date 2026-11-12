// mod：给游戏加内容的Go包。
//
// 写一个mod（完整的例子见mods/flowers）：
//
//  1. 新建一个包，在init()里注册内容：
//     - 方块：block.Register(block.Type{Name: "mymod:ruby", ...})
//     - 地形：terrain.Register_feature(...) / terrain.Register_biome(...)
//     - 游戏事件：game.Register_mod(game.Mod{Name: "mymod", On_update: ...})
//  2. 在main.go里加一行 import _ "foxel3d_ai/mods/mymod"
//
// 注册都要在init()里做：游戏开始时会用已经注册的方块读存档、用已经注册的地形生成世界。
// 名字（方块、生物群系、场、Mod_data的key）请都加上"mod名:"前缀，避免和别的mod重名。
package game

import (
	"foxel3d_ai/canvas"
	"foxel3d_ai/world"
)

// mod的事件，不需要的留成nil
type Mod struct {
	Name string
	// 游戏开始时调用一次，这时世界和存档都准备好了（chunk还没开始加载）
	On_start func(g *Game)
	// 每一帧调用，在玩家移动之后、画之前。in是这一帧的输入
	On_update func(g *Game, in canvas.Input, dt float64)
	// 画界面时调用，在游戏自己的界面上面画
	On_hud func(g *Game)
	// 玩家要挖掉id这个格子之前调用，b是玩家指着的方块。返回false取消
	On_break func(g *Game, id world.CubeID, b world.BlockID) bool
	// 玩家要在id这个格子放方块b之前调用。返回false取消
	On_place func(g *Game, id world.CubeID, b world.BlockID) bool
	// 存档之前调用，可以把自己的数据写进g.Mod_data
	On_save func(g *Game)
}

var mods []Mod

func Register_mod(m Mod) {
	mods = append(mods, m)
}

// 已经注册的mod
func Mods() []Mod {
	return append([]Mod(nil), mods...)
}
