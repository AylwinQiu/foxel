// 游戏入口。在foxel3d_ai目录下 go run . ，存档在saves/下面。
//
//	go run . -world 名字 -seed 123 -view 8
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"foxel3d_ai/canvas"
	"foxel3d_ai/game"

	// mod：import进来就会注册，不想要哪个就删掉那一行
	_ "foxel3d_ai/mods/flowers"
)

func main() {
	name := flag.String("world", "default", "存档的名字")
	seed := flag.Int64("seed", time.Now().UnixNano()%1000000, "新世界的种子（读存档时用存档里的）")
	view := flag.Int64("view", 8, "视距（chunk数，一个chunk是16格）")
	flag.Parse()

	c := &canvas.Raylib{}
	c.Init(1280, 720)
	defer c.Close()

	g, err := game.New_game(game.Config{World_name: *name, Seed: *seed, View_distance: *view}, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	g.Run()
}
