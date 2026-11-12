# foxel3d_ai

一个用 Go 写的方块沙盒游戏，玩法类似我的世界：

- **地形：** 世界是程序生成的，有平原、森林、沙漠、雪山、针叶林 5 种地形，长着橡树和云杉，地下有煤矿和铁矿。
- **建造：** 可以挖方块、放方块。方块有 1 格、1/2 格、1/4 格三种大小，可以混在一起搭。
- **存档：** 可以存档，下次打开接着玩。
- **mod：** 可以用 Go 写 mod，添加方块、地形和游戏逻辑。

整个项目大约 5000 行，只依赖 [raylib-go](https://github.com/gen2brain/raylib-go)，用它开窗口和画图。它是 [foxel3d](../foxel3d) 设计思路的完整实现，见[和 foxel3d 的关系](#和-foxel3d-的关系)。

## 运行

需要 Go 1.27 或更新的版本，还需要一个 C 编译器，因为 raylib 要用 cgo 编译：

- **macOS：** 装 Xcode Command Line Tools 就行。
- **Linux：** 还要装 X11 和 OpenGL 的开发包。具体的包名见 raylib-go 的说明。

```sh
cd foxel3d_ai
go run .                        # 打开（或新建）存档 saves/default
go run . -world test -seed 42   # 另一个存档；种子只对新世界有效
go run . -view 12               # 视距（单位是 chunk，1 个 chunk 是 16 格），默认 8
go test ./...                   # 测试，不需要开窗口
```

第一次编译 raylib 大约要一分钟，之后几秒就能启动。

在视距 8 的情况下，游戏能跑满 60 帧（vsync）；启动后大约 1 秒就能玩。

## 操作

| 键 | 作用 | 键 | 作用 |
|---|---|---|---|
| WASD | 移动 | 空格 | 跳 / 飞行时上升 |
| 鼠标 | 看 | Shift | 潜行 / 飞行时下降 |
| 左键 | 挖 | Ctrl | 跑 |
| 右键 | 放 | F | 切换飞行 |
| 中键 | 选中指着的方块 | R | 方块大小 1 → 1/2 → 1/4 |
| 1-9 / 滚轮 | 选物品栏 | Tab | 打开所有方块 |
| F5 | 存档 | Esc | 暂停（松开鼠标） |
| F3 | 调试信息 | H | 帮助 |
| G | 在身边种花（flowers mod） | | |

关闭窗口时会自动存档，另外每 5 分钟也会自动存一次。

## 代码结构

一共 6 个包，按依赖从下往上排，下面的包不知道上面的包存在：

```
mods/      mod（Go 包），main.go 里 import 进来
  ↓
game       玩家、物理、界面、存档、mod 事件
  ↓
engine     把 world 画到 canvas 上：按相机加载 chunk、生成网格、视锥剔除
  ↓
terrain    地形生成：噪声、生物群系、feature（树、矿石）
  ↓
block      方块注册表 + 内置方块（贴图是程序生成的）
  ↓
world      八叉树、CubeID、场、chunk 编码、射线检测      （只用标准库）
canvas     窗口接口 + raylib 实现                         （只依赖 world 的几个基础类型）
```

想看懂代码，建议按这个顺序读：

1. [world/position.go](world/position.go) 和 [world/world.go](world/world.go)：数据怎么存。
2. [engine/engine.go](engine/engine.go)：世界怎么画出来。
3. [game/game.go](game/game.go)：一帧里发生了什么。

| 包 | 主要文件 | 说明 |
|---|---|---|
| `world` | `position.go` `cube.go` `world.go` `field.go` `encode.go` `raycast.go` | 世界的数据结构，不知道方块长什么样 |
| `canvas` | `canvas_interface.go` `canvas_raylib.go` `raylib_atlas.go` `raylib_input.go` `images.go` | `Canvas` 接口是唯一和图形库打交道的地方，换引擎只要重新实现它 |
| `block` | `block.go` `materials.go` `textures.go` | 21 种内置方块，16×16 的像素贴图都是代码生成的，没有图片文件 |
| `terrain` | `terrain.go` `biomes.go` `features.go` `noise.go` | 实现 `world.Generator` 接口 |
| `engine` | `engine.go` `mesher.go` | 连接 world 和 canvas |
| `game` | `game.go` `player.go` `hud.go` `save.go` `mod.go` `game_aux.go` | 游戏本身 |

## 核心概念

### 坐标

- x 向东，y 向上，z 向南（北是 -z）。
- 1 个单位就是 1 格。
- 地面大约在 y = 10 到 30，山最高超过 100。
- 世界的竖直范围是 y = -32 到 127，最底下是基岩。
- 水平方向是无限的，实际上限是 ±2^31。

### 八叉树和 CubeID

整个世界是一棵八叉树。根节点的边长是 2^32，每个节点可以分成 8 个子节点。

**Depth：** 节点的深度 d 表示它的边长是 2^d。

| Depth | 边长 | 是什么 |
|---|---|---|
| 32 | 2^32 | 根 |
| 4 | 16 | chunk，加载、保存、生成网格的单位 |
| 0 | 1 | 普通方块 |
| -1 | 1/2 | 小方块 |
| -2 | 1/4 | 最小的方块 |

**CubeID：** `CubeID{X, Y, Z, Depth}` 表示"深度为 Depth 的第 (X, Y, Z) 个格子"。它就是位置本身，所以：

- 可以直接当 map 的 key。
- 可以直接写进存档。
- 不会因为树的结构变化而失效。

**节点：** 每个节点（`world.Cube`）只存一个方块编号 `BlockID` 和子节点指针，一共 16 字节。

**自动合并：** 8 个一样的子节点会自动合并成一个。所以一整块实心的石头，在树里只是一个节点。

### 方块注册表

world 里只存 `BlockID`（uint16）。方块叫什么、长什么样、能不能穿过、透不透明，都在 `block` 包的注册表里查（`block.Get(id)`）。

所以一百万块石头只共用一份配置。

### 场

每个格子自己的数据，比如箱子里装了什么，用"场"来存：`World.Field_write(场名, CubeID, 字符串)`。

- 场跟着存档保存。
- 格子被覆盖或被挖掉时，它里面的场会自动删掉。

### chunk 的生命周期

1. **生成：** 相机附近的 chunk 还没有时，按需生成。地形生成是确定性的：同一个种子、同一个位置，永远生成一样的 chunk。所以没改过的 chunk 不用保存。
2. **生成网格：** chunk 和它的邻居都加载好之后，生成网格。网格只包含露在外面的面，上传到显卡里。
3. **重建网格：** chunk 被改动时（`World.On_change`），重新生成它的网格；如果改动在边界上，邻居的网格也一起重建。
4. **卸载：** 离开视距之后卸载。如果玩家改动过这个 chunk，先把它编码进内存，存档时写进文件。

engine 每帧最多花 8 毫秒加载 chunk 和生成网格，超过了就留到下一帧，所以走得快的时候也不会卡。

### 画面

**网格：** 画地形不是每帧一个方块一个方块地画，而是每个 chunk 一个网格，常驻在显卡里。

**贴图：** 所有方块的贴图放在一张贴图集上。用像素风采样，不做插值，看起来不会糊。

**透明：** 着色器只做一件事：丢掉透明的像素。树叶和玻璃就是这样实现的。

**雾：** 远处有雾，盖住视距的边缘。

**浮点原点：** 渲染时相机永远在原点。这样走到很远的地方，坐标也不会因为浮点数精度不够而抖动。

### 玩家物理

玩家的碰撞盒是 0.6 × 1.8 × 0.6。

**移动：** x、y、z 三个方向分开移动。每一步不超过 0.2 格，所以速度再快也不会穿墙。

**台阶：** 半格以内的台阶可以直接走上去。所以 1/2 格的方块能拿来铺楼梯。

**飞行：** 按 F 进入飞行模式。飞行时没有重力，但仍然有碰撞。

## 存档

存档在 `saves/<名字>/world.foxel`，这个文件夹已经写进了 `.gitignore`。

**格式：** 存档是 gzip 压缩的 Go `gob` 格式，里面保存了：

- 版本号和种子；
- 玩家的位置、视角、飞行状态；
- 物品栏和选中的格子、方块大小；
- **改动过的 chunk**：每个 chunk 按先序把八叉树编码成一串 varint；
- 场；
- mod 的数据（`Mod_data`）。

没改过的 chunk 不保存，读档时重新生成。所以存档一般只有几 KB 到几百 KB。

**写入是原子的：** 存档先写到一个临时文件，写完再改名替换原来的存档。存到一半崩溃，也不会把旧存档弄坏。

**方块按名字保存：** 存档里有一张"编号 → 名字"的表，读档时按名字找回方块。所以：

- 增删 mod、改变方块的注册顺序，都不会弄乱存档。
- 找不到的方块（比如删掉的 mod 里的方块）变成空气。

## 写 mod

mod 是一个普通的 Go 包，在 `init()` 里注册方块、地形和事件，然后在 [main.go](main.go) 里 import 它：

```go
var RUBY = block.Register(block.Simple("mymod:ruby", "Ruby", canvas.Images{}.Url("assets/ruby.png")))

func init() {
	terrain.Register_feature(&terrain.Ore{Salt: 9001, Block: RUBY, Per_chunk: 2, Size: 4, Min_y: -32, Max_y: 0})
	game.Register_mod(game.Mod{
		Name:     "mymod",
		On_break: func(g *game.Game, id world.CubeID, b world.BlockID) bool { return true },
	})
}
```

- **例子：** [mods/flowers](mods/flowers/flowers.go)，大约 100 行。它用 1/4 格的方块拼出小花，演示了新方块、地形 feature、按键事件、挖方块事件。
- **完整的 API 文档：** [mods/README.md](mods/README.md)。
- **以后改用 Starlark：** 计划用 [Starlark](https://github.com/google/starlark-go)（纯 Go 实现）写配置和 mod，这样不用重新编译游戏就能装 mod，代码放在 [lark_engine/](lark_engine/)，还在开发中。

## 测试

```sh
go test ./...
```

测试里用一个不画图的假 canvas 代替 raylib，所以不会开窗口。测试覆盖了这些东西：

| 包 | 测试的内容 |
|---|---|
| `world` | 八叉树的拆分和合并、chunk 的编码和读回、场、射线检测 |
| `terrain` | 地形是确定性的，生物群系都会出现，树能跨 chunk 拼上 |
| `engine` | 哪些面该省掉（包括玻璃、树叶、小方块和 chunk 边界），视锥剔除，卸载再加载后改动还在 |
| `game` | 走路、跳跃、撞墙、上台阶、挖、放、存档读档 |

## 和 foxel3d 的关系

这个项目是从 [foxel3d](../foxel3d) 的设计出发写的。foxel3d 本身没有改动。

**保留下来的设计：**

- `Canvas` 接口加上 raylib 实现；
- `Images` 6 个面的贴图，`Image.Get_pixel`；
- 八叉树世界，根节点边长 2^32，`Depth`，`CubeID`；
- 场；
- engine 用浮点原点和视锥剔除；
- 方块配置共用。

**改了的地方：**

| 改动 | 原因 |
|---|---|
| Canvas 加了网格接口（`Set_mesh` / `Add_mesh`） | 地形有几十万个方块，每帧每个方块调一次 `Add_cube` 根本画不动。`Add_cube` 还保留着，用来画少量会动的东西 |
| `Get_keys()` 换成了 `Get_input()`，加了 `Set_cursor_locked` | 输入还要带上鼠标移动、鼠标按键和滚轮 |
| `CubeID` 从自增计数器改成了位置 | 用计数器的话，找一个格子要遍历整棵树；节点自动合并、拆分时编号也会乱掉 |
| `Cube` 节点去掉了 `Rotation` 和 `Offset` | 方块都对齐格子，碰撞、网格、射线检测都依赖这一点；而且每个节点能省下 48 字节 |
| 方块属性放进了注册表（`block.Type`） | 这就是"同一个配置反复传入"的想法 |
| 去掉了 `TreePosition`、`Phy2tree` 和 `CANVAS_DIV_PHY_3D` 缩放 | 有了 `CubeID` 和 `Cube_id_at(p, depth)` 就够了，1 格就是 canvas 里的 1 |

## 还没有的东西

- 水
- 动物
- 光照（现在每个面按朝向固定明暗）
- 合成
- 多人

这些都是有意没做的，不是 bug。
