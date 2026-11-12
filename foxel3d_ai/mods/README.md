# mod API

这里列出写 mod 时能用到的所有接口。完整的例子见 [flowers](flowers/flowers.go)。

- [快速开始](#快速开始)
- [坐标和格子](#坐标和格子)：`world.CubeID`、`Depth`、`PhysicalPosition3D`
- [方块](#方块)：`block`
- [贴图](#贴图)：`canvas.Images`、`canvas.Image`、`block.Pixel_texture`
- [地形](#地形)：`terrain` 包里的生物群系和 feature
- [游戏事件](#游戏事件)：`game.Mod`
- [游戏状态](#游戏状态)：`*game.Game`
- [世界](#世界)：`*world.World`
- [绘制](#绘制)：`*engine.Engine`、`canvas.Canvas`
- [输入](#输入)：`canvas.Input`
- [存档里保存什么](#存档里保存什么)
- [规矩和常见坑](#规矩和常见坑)

## 快速开始

mod 是一个 Go 包。所有东西都在 `init()` 里注册，或者在包级变量里注册：

```go
package mymod

import (
	"foxel3d_ai/block"
	"foxel3d_ai/canvas"
	"foxel3d_ai/game"
	"foxel3d_ai/terrain"
	"foxel3d_ai/world"
)

var RUBY = block.Register(block.Simple("mymod:ruby", "Ruby", canvas.Images{}.Url("assets/ruby.png")))

func init() {
	terrain.Register_feature(&terrain.Ore{Salt: 9001, Block: RUBY, Per_chunk: 2, Size: 4, Min_y: -32, Max_y: 0})

	game.Register_mod(game.Mod{
		Name: "mymod",
		On_break: func(g *game.Game, id world.CubeID, b world.BlockID) bool {
			if b == RUBY {
				g.Show_message("Shiny!")
			}
			return true
		},
	})
}
```

然后在 [main.go](../main.go) 里加一行 `import _ "foxel3d_ai/mods/mymod"`。

**注册必须在游戏开始之前完成。** 原因有两个：

- `terrain.New` 会把当时已经注册的生物群系和 feature 复制一份。
- 读档时，方块要按名字找回 BlockID。

所以 `block.Register`、`terrain.Register_biome`、`terrain.Register_feature`、`game.Register_mod` 都只能在 `init()` 或包级变量里调用。

## 坐标和格子

`import "foxel3d_ai/world"`

**坐标轴：** x 向东，y 向上，z 向南（北是 -z）。1 个单位就是 1 格。

### Depth

`world.Depth` 是 cube 的深度。深度为 d 的 cube 边长是 2^d。

| 常量 / 方法 | 值 | 说明 |
|---|---|---|
| `world.MIN_DEPTH` | -2 | 最小的 cube，边长 1/4 |
| `0` | | 普通方块，边长 1 |
| `world.CHUNK_DEPTH` | 4 | chunk，边长 16 |
| `world.CHUNK_SIZE` | 16 | |
| `world.IDENTITY_DEPTH` | 32 | 整个世界的根 |
| `d.Side()` | | 边长，`float64` 类型 |

### PhysicalPosition3D

`world.PhysicalPosition3D{X, Y, Z float64}` 是物理坐标。

- 方法：`Add(b)`、`Sub(b)`、`Scale(k)`、`Length()`。
- 简写：`game.PhyPos(x, y, z)`。

### CubeID

`world.CubeID{X, Y, Z int64; Depth}` 是八叉树里一个格子的编号。它在 x 方向覆盖 `[X*边长, (X+1)*边长)`，y、z 方向一样。

同一个位置、同一个深度永远对应同一个 CubeID，所以可以当 map 的 key，也可以存进存档。

| 函数 / 方法 | 说明 |
|---|---|
| `world.Block_id(x, y, z)` | depth 0 的普通方块格子 |
| `world.Cube_id_at(p, d)` | 包含物理坐标 p 的、深度为 d 的格子 |
| `id.Min()`、`id.Max()`、`id.Center()` | 两个角和中心，都是物理坐标 |
| `id.Side()` | 边长 |
| `id.Parent_at(d)` | 包含它的、深度为 d 的格子，要求 d ≥ id.Depth |
| `id.Child(o)` | 第 o 个子格子。o 是卦限：x 正为 1，y 正为 2，z 正为 4 |
| `id.Octant()` | 它在父格子里的卦限 |
| `id.Contains(other)` | other 在不在它里面，other 就是它自己时也算 |
| `id.Neighbor(dx, dy, dz)` | 同样大小的邻居，偏移按格子数算 |
| `id.Chunk()` | 所在的 chunk |
| `id.String()` | 比如 `(3, 60, -2)@0` |

**换算小格子的坐标：** depth -2 的格子 `{X, Y, Z, -2}` 在方块 `(X/4, Y/4, Z/4)` 里面。所以方块 `(x, y, z)` 底部的某个 1/4 小格是 `{x*4+i, y*4, z*4+k, -2}`，其中 i、k 取 0 到 3。

## 方块

`import "foxel3d_ai/block"`

`world` 里每个 cube 只存一个 `world.BlockID`（uint16）。方块长什么样、有什么属性，都到注册表里查。`world.AIR` 是 0。

### block.Type

| 字段 | 说明 |
|---|---|
| `Name` | 唯一的名字，写成 `"mymod:ruby"` 的形式。存档按名字保存方块 |
| `Display` | 给玩家看的名字，只能用 ASCII |
| `Images` | 6 个面的贴图，见[贴图](#贴图) |
| `Solid` | 有碰撞，玩家不能穿过 |
| `Opaque` | 完全不透明，会挡住旁边方块的面。有透明像素的方块（玻璃、树叶）要设成 false |
| `Cull_same` | 两个同种的不透明度为 false 的方块挨着时，省掉中间的面。玻璃设成 true，树叶设成 false |
| `Unbreakable` | 玩家挖不掉，比如基岩 |
| `Hidden` | 不出现在物品栏里 |
| `Props` | `map[string]string`，自定义属性，mod 可以随便用 |

### 函数

| 函数 | 说明 |
|---|---|
| `block.Register(t) BlockID` | 注册一种方块，名字重复时 panic |
| `block.Simple(name, display, images) Type` | 普通的实心、不透明方块 |
| `block.Get(id) *Type` | 查方块的配置。没注册过的 id 当成空气 |
| `block.By_name(name) (BlockID, bool)` | 按名字查方块 |
| `block.All() []BlockID` | 所有方块，包括空气，按 id 排序 |
| `block.Is_solid(id)`、`block.Is_opaque(id)` | |
| `block.Default_hotbar` | `[]BlockID`，新游戏的物品栏。可以在 `init()` 里改 |

### 内置方块

内置方块都是 `block.XXX` 变量，名字是 `foxel:xxx`：

STONE、COBBLESTONE、DIRT、GRASS、SNOWY_GRASS、SNOW、SAND、SANDSTONE、GRAVEL、COAL_ORE、IRON_ORE、OAK_LOG、OAK_PLANKS、OAK_LEAVES、SPRUCE_LOG、SPRUCE_PLANKS、SPRUCE_LEAVES、GLASS、BRICKS、STONE_BRICKS、BEDROCK

### 每个格子自己的数据

方块的配置是同一种方块共用的。每个格子自己的数据（比如箱子里装了什么）要用[场](#场)来存。

## 贴图

`import "foxel3d_ai/canvas"`

### Images

`canvas.Images{Up, Down, North, South, West, East Image}` 是 cube 6 个面的图片，为 nil 的面画成白色。

**图片的朝向：** 从 cube 外面看，每张图都是正的。侧面图片的上方朝上；Up 面图片的上方朝北，Down 面图片的上方朝南。

| 写法 | 说明 |
|---|---|
| `canvas.Images{}.All(img)` | 6 个面都用 img |
| `canvas.Images{}.Color(c)` | 纯色 |
| `canvas.Images{}.Url("assets/a.png")` | 读本地文件，或者读 `http(s)://` 开头的网址。支持 png、jpeg、gif；读不到时显示品红色 |
| `canvas.Images{}.Column(top, side, bottom)` | 上、侧、下三种图，比如草方块、木头 |

### Image

`canvas.Image` 只有一个方法 `Get_pixel(x_frac, y_frac) Color`。坐标是 0 到 1，`(0, 0)` 是左上角。

图片如果还实现了 `Size() (w, h int)`（也就是 `canvas.SizedImage`），canvas 会按这个大小采样，像素风贴图就不会糊。

alpha 小于 128 的像素不画，可以用来做镂空的贴图。

现成的实现：

| 实现 | 说明 |
|---|---|
| `canvas.SolidColor(c)` | 纯色 |
| `canvas.New_pixel_image(w, h, func(x, y int) Color) *PixelImage` | 按像素生成 |
| `canvas.Load_picture(url) Image` | 读图片，同一个 url 只读一次 |

### 程序生成的 16×16 贴图

| 函数 | 说明 |
|---|---|
| `block.Pixel_texture(seed, func(x, y int, r *block.Rng) Color)` | 按像素生成。r 是确定性的随机数 |
| `block.Noise_texture(seed, base, amount)` | 一种颜色，加上随机的明暗 |
| `block.New_rng(seed)` | 单独的随机数生成器 |

`block.Rng` 的方法：`Next() uint64`、`Float()`（返回 [0, 1)）、`Intn(n)`、`Vary(c, amount)`（亮度随机上下浮动 amount）。

### 颜色

`canvas.Color{R, G, B, A uint8}`

- 方法：`c.Scale(k)`（变亮或变暗，alpha 不变）、`c.With_alpha(a)`。
- 常量：`canvas.Red`、`Green`、`Blue`、`White`、`Black`、`Gray`、`LightGray`、`DarkGray`、`Yellow`、`Magenta`、`SkyBlue`、`Transparent`。

**注意：** canvas 会把同一个 Image 缓存成贴图，所以：

- Image 必须是可以比较的值，比如指针，或者不含 slice、map 的结构体。
- 画过之后不要再改它的内容。

## 地形

`import "foxel3d_ai/terrain"`

### 生成的顺序

每个 chunk 的生成分两步：

1. 先按生物群系填好地面：表层、表层下面几层、石头，最底下 `terrain.BOTTOM_Y`（-32）是基岩。
2. 再按注册的顺序执行每个 feature。

世界在竖直方向上只有 chunk y 从 `terrain.MIN_CHUNK_Y`（-2）到 `terrain.MAX_CHUNK_Y`（7），也就是方块 y 从 -32 到 127。

### 生物群系

`terrain.Biome` 的字段：

| 字段 | 说明 |
|---|---|
| `Name` | 写成 `"mymod:xxx"` 的形式 |
| `Temperature`、`Humidity` | 气候，-1 到 1。每一列选气候最接近的生物群系，边界处会平滑过渡 |
| `Base_height`、`Variation` | 地面的平均高度和起伏幅度 |
| `Ridged` | 用山脊形状的噪声，只往上起伏 |
| `Surface`、`Subsurface`、`Subsurface_depth` | 表层的方块；表层下面几层的方块和层数。再往下是石头 |
| `Rock_line` | 地面高于这个高度时露出石头。0 表示不用 |
| `Snow_line` | 地面高于这个高度时盖雪。0 表示不用 |

相关函数：

- `terrain.Register_biome(b)`：注册生物群系。名字重复时替换旧的，所以也可以用来改内置的。
- `terrain.Biome_by_name(name)`：按名字查，没有时返回 nil。

内置的生物群系：`terrain.Plains`、`Forest`、`Desert`、`Mountains`、`Taiga`。它们都是指针，可以在 `init()` 里直接改字段，比如 `terrain.Desert.Variation = 8`。

### Feature

feature 是一个 `Place(ctx *terrain.Context)` 方法，每个 chunk 调用一次。

- 注册：`terrain.Register_feature(f)`。
- 写成函数的话：`terrain.Feature_func(func(ctx *terrain.Context) { ... })`。

`*terrain.Context` 提供这些东西：

| 字段 / 方法 | 说明 |
|---|---|
| `ctx.Chunk` | 正在生成的 chunk 的 CubeID |
| `ctx.Data` | `*world.ChunkData`，正在生成的 chunk 数据，一般用下面的方法改 |
| `ctx.Gen` | `*terrain.Generator` |
| `ctx.Origin() (x, y, z)` | chunk 最小那个角的方块坐标 |
| `ctx.Get(x, y, z) (BlockID, bool)` | 读一个方块。不在这个 chunk 里时返回 false |
| `ctx.Set(x, y, z, b)` | 写一个方块。不在这个 chunk 里时自动忽略 |
| `ctx.Set_if_air(x, y, z, b)` | 只有原来是空气时才写 |
| `ctx.Set_small(id, b)` | 放一个比 1 小的 cube，最小到 depth -2 |
| `ctx.Column(x, z) terrain.Column` | 这一列的 `Height`（地面最上面那个方块的 y）和 `Biome` |
| `ctx.Ground(x, z) BlockID` | 这一列地面最上面的方块，不考虑 feature 改过的结果 |
| `ctx.Rand(x, y, z, salt) *rand.Rand` | 确定性的随机数，只由种子、位置和 salt 决定 |
| `ctx.Scatter(radius, salt, chance, fn)` | 遍历这个 chunk 和周围 radius 格以内的每一列，每列以 `chance(x, z)` 的概率调用 `fn(x, z, r)` |

**跨 chunk 的 feature：** 树冠这种东西会伸到隔壁 chunk 里。做法是：

1. 用 `Scatter`，radius 设成 feature 的最大半径。
2. 照常把整棵树写进去，`Set` 会自动丢掉不在这个 chunk 里的部分。

隔壁 chunk 生成时会算出同一棵树，两边正好拼上。

**salt：** 每个 feature 用不同的 salt，否则它们会挑中同样的位置。

### 现成的 feature 类型

**`terrain.Ore`：矿脉。** 每个 chunk 放 `Per_chunk` 条矿脉，每条 `Size` 个方块，只替换石头。字段有 `Salt`、`Block`、`Per_chunk`、`Size`、`Min_y`、`Max_y`。

**`terrain.Tree`：树。** 字段：

- `Salt`。
- `Density`：`map[群系名]每列长树的概率`。
- `Soil`：能长树的地面方块。
- `Radius`：树最多离树根多少格。
- `Grow`：`func(ctx, x, y, z, r)`，其中 y 是地面方块的高度。

内置的 feature 实例：

- `terrain.Coal`、`terrain.Iron`（`*Ore`）。
- `terrain.Oak_trees`、`terrain.Spruce_trees`（`*Tree`）。

它们都可以直接改，比如 `terrain.Oak_trees.Density[terrain.Plains.Name] = 0.02`。

### Generator

`g.Terrain` 是 `*terrain.Generator`：

- `Height(x, z)`：这一列地面的高度。
- `Biome(x, z)`：这一列的生物群系。
- `Column(x, z)`：上面两个一起取。
- `Base_block(col, y)`：只按生物群系算，这一列高度 y 处应该是什么方块。
- `Seed`：世界的种子。

## 游戏事件

`import "foxel3d_ai/game"`

`game.Register_mod(game.Mod{...})` 注册一组事件，不需要的事件留成 nil。多个 mod 按注册的顺序调用。

| 事件 | 什么时候调用 |
|---|---|
| `Name string` | mod 的名字 |
| `On_start(g)` | 游戏开始时调用一次。这时世界和存档都准备好了，但 chunk 还没加载 |
| `On_update(g, in, dt)` | 每帧调用，在玩家移动和射线检测之后。世界加载完之前不调用。dt 是秒 |
| `On_hud(g)` | 每帧画完世界和游戏自己的界面之后调用，加载时也会调用。**所有绘制都写在这里** |
| `On_break(g, id, b) bool` | 玩家挖掉 id 这个格子之前调用。b 是玩家指着的方块。返回 false 就取消 |
| `On_place(g, id, b) bool` | 玩家要在 id 这个格子放方块 b 之前调用。返回 false 就取消 |
| `On_save(g)` | 存档之前调用，可以把自己的数据写进 `g.Mod_data` |

`game.Mods()` 返回所有已注册的 mod。

**一帧的顺序：**

1. 读输入。
2. 处理游戏自己的按键。
3. 玩家移动。
4. 射线检测，结果写进 `g.Target`。
5. 处理挖和放，这一步会调用 `On_break` 和 `On_place`。
6. 调用 `On_update`。
7. 加载 chunk。
8. 画世界。
9. 画界面。
10. 调用 `On_hud`。
11. 显示到屏幕上。

## 游戏状态

`*game.Game` 的字段：

| 字段 | 说明 |
|---|---|
| `World` | `*world.World`，见[世界](#世界) |
| `Engine` | `*engine.Engine`，见[绘制](#绘制) |
| `Canvas` | `canvas.Canvas`，见[绘制](#绘制) |
| `Terrain` | `*terrain.Generator` |
| `Player` | `game.Player`，见下面 |
| `Hotbar` | `[9]world.BlockID`，物品栏 |
| `Selected` | 物品栏里选中的格子，0 到 8 |
| `Build_depth` | 挖和放的格子大小：0 是 1 格，-1 是 1/2 格，-2 是 1/4 格 |
| `Target` | `*world.Hit`，玩家正在指着的方块，没有时是 nil |
| `Mod_data` | `map[string]string`，跟着存档保存。key 写成 `"mymod:xxx"` 的形式 |
| `Time` | 游戏开始后过了多少秒 |
| `Config` | `World_name`、`Seed`、`View_distance` |

`*game.Game` 的方法：

| 方法 | 说明 |
|---|---|
| `Playing() bool` | 玩家是不是正在玩：已经加载完、鼠标锁定、没开物品栏。处理按键前先判断它 |
| `Input() canvas.Input` | 这一帧的输入 |
| `Show_message(text)` | 屏幕上方显示一条消息，2.5 秒后消失。只能用 ASCII |
| `Break(id) bool` | 像玩家一样挖：会检查挖不掉的方块，会调用 `On_break` |
| `Place(id, b) bool` | 像玩家一样放：格子必须是全空的，不能卡住玩家，会调用 `On_place` |
| `Target_cell()` | 玩家指着的方块里，按 `Build_depth` 大小算的要挖掉的格子 |
| `Place_cell()` | 玩家指着的那个面外面，要放方块的格子 |
| `Set_cube(p, depth, b) error` | 把物理坐标 p 所在的格子设成 b。不经过 `On_break` 和 `On_place` |
| `Block_at(p) BlockID` | 物理坐标 p 处的方块，没加载时返回 AIR |
| `Surface_below(x, y, z, max_depth) (int64, bool)` | 从 y 往下找第一个实心方块，返回它上面那个空格子的 y |
| `Save() error` | 立刻存档 |
| `Save_path() string` | 存档文件的路径 |

### game.Player

- 字段：
  - `Pos`：脚底中心。
  - `Velocity`。
  - `Yaw`：0 朝北，π/2 朝东。
  - `Pitch`：向上为正。
  - `Flying`、`On_ground`。
- 方法：`Eye()`（眼睛位置）、`Look_dir()`（视线方向的单位向量）。
- 常量：`PLAYER_WIDTH`、`PLAYER_HEIGHT`、`EYE_HEIGHT`、`REACH`、`WALK_SPEED`、`GRAVITY`、`JUMP_SPEED` 等。

**传送：** 直接改 `g.Player.Pos` 就行。

## 世界

`*world.World`，也就是 `g.World`。

### 读和写

| 方法 | 说明 |
|---|---|
| `Block_at(x, y, z) (b, partial, loaded)` | 读一个普通方块。这个格子被分成更小的 cube 时 partial 是 true，b 没有意义 |
| `Leaf_at(p) (world.Leaf, bool)` | 包含物理坐标 p 的最小的 cube，返回它的 `Id` 和 `Block` |
| `Find(id) (node, at, loaded)` | 找 id 这个格子。如果它在一个更大的叶子里面，返回那个叶子，at 是叶子的 id。node 是 `world.Cube`，`Children` 为 nil 就是叶子 |
| `Set_cube(id, b) error` | 把 id 这个格子整个变成 b（AIR 就是挖掉），里面原来的东西和场都删掉。深度必须在 -2 到 4 之间，chunk 没加载时返回 `world.ErrNotLoaded` |
| `Raycast(origin, dir, max_dist, hit) (world.Hit, bool)` | 沿射线找第一个 `hit(block)` 为 true 的 cube。碰到没加载的 chunk 就停下 |
| `Visit_chunk(pos, fn) bool` | 按先序遍历一个 chunk 里的 cube。fn 返回 false 时不进入子 cube |
| `On_change(fn func(id CubeID))` | 每次 `Set_cube` 之后调用 fn |

### chunk

| 方法 | 说明 |
|---|---|
| `Is_loaded(pos)` | 这个 chunk 有没有加载 |
| `Loaded_chunks()` | 所有已加载的 chunk |
| `Load_chunk(pos)` | 加载一个 chunk。一般不用管，engine 会自动加载相机附近的 |
| `In_range(pos)` | chunk 在不在世界的竖直范围里 |

`world.Hit` 的字段：

- `Leaf`：碰到的 cube。
- `Point`：碰到的点。
- `Normal`：碰到的面朝外的方向，`[3]int64`。从方块里面出发时是 0。
- `Distance`：距离。

**改动是即时的：** `Set_cube` 改完后，8 个一样的兄弟 cube 会自动合并，网格在下一帧重建。

### 场

场是给某个格子附加的字符串数据，跟着存档保存。

- 场名写成 `"mymod:xxx"` 的形式。
- 格子被 `Set_cube` 覆盖时（包括被挖掉），里面的场会自动删掉。

| 方法 | 说明 |
|---|---|
| `Field_write(name, id, value)` | 写 |
| `Field_read(name, id) (string, bool)` | 读 |
| `Field_delete(name, id)` | 删 |
| `Field_all(name) map[CubeID]string` | 这个场的所有数据，返回副本 |

需要结构化的数据时，自己编码成 JSON 再存。

## 绘制

**所有绘制都写在 `On_hud` 里。** 每帧开始时 canvas 会被清空，在 `On_update` 里画的东西会被清掉。

### 3D：g.Engine

3D 的绘制用物理坐标，engine 会自动换算成 canvas 坐标。

| 方法 | 说明 |
|---|---|
| `Add_line3d(a, b, color)` | 一条线 |
| `Add_box_lines(lo, hi, color)` | 长方体的 12 条边 |
| `Add_cube(center, side, images, rot)` | 一个会动的 cube，它不在世界里。rot 是 `world.Rotation3D`，向量方向是转轴，长度是弧度 |
| `Get_camera() (pos, dir)` | 相机的位置和朝向 |
| `To_canvas(p)` | 物理坐标换算成 canvas 坐标，直接调用 canvas 的 3D 接口时要用 |
| `View_distance` | 视距，单位是 chunk |
| `Stats` | 上一帧的统计：`Loaded`、`Meshes`、`Drawn`、`Pending` |

**性能：** `Add_cube` 每帧每个 cube 都要重新画，只适合少量的东西。要往世界里放很多方块，就用 `World.Set_cube`。

### 2D：g.Canvas

2D 的坐标是像素，`(0, 0)` 是左上角。后画的盖在上面。

| 方法 | 说明 |
|---|---|
| `Get_window_size() (w, h)` | 窗口大小 |
| `Add_2d_text(text, pos, color)` | 文字。pos 是左上角，字号 20，只支持 ASCII，可以用 `\n` 换行 |
| `Measure_2d_text(text) Canvas2D` | 文字画出来有多大 |
| `Add_2d_line(a, b, color)` | 线 |
| `Add_2d_rect(pos, size, color)` | 实心矩形 |
| `Add_2d_image(img, pos, size)` | 图片 |

### 环境

- `Set_background(color)`：天空的颜色。
- `Set_fog(color, start, end)`：雾，离相机 start 以内没有雾，到 end 完全变成 color。
- `Camera3D_vis_angle(h, v, tilt, auto)`：视角。

这几个设一次之后一直有效，一般在 `On_start` 里设。

### 鼠标

`Set_cursor_locked(bool)`、`Is_cursor_locked()`。鼠标锁定时光标隐藏，用于第一人称视角。

## 输入

`canvas.Input` 是一帧的输入：

| 字段 / 方法 | 说明 |
|---|---|
| `in.Is_pressed(k)` | 这一帧刚按下 |
| `in.Is_held(k)` | 这一帧按着 |
| `Mouse_dx`、`Mouse_dy` | 鼠标移动了多少像素，x 向右、y 向下为正 |
| `Mouse_x`、`Mouse_y` | 鼠标在窗口里的位置 |
| `Wheel` | 滚轮，向上滚为正 |

### 键

- 字母：`KEY_A` 到 `KEY_Z`。
- 数字：`KEY_0` 到 `KEY_9`。
- 功能键：`KEY_F1` 到 `KEY_F12`。
- 其他键：`KEY_SPACE`、`KEY_ENTER`、`KEY_ESCAPE`、`KEY_TAB`、`KEY_BACKSPACE`、`KEY_LEFT_SHIFT`、`KEY_RIGHT_SHIFT`、`KEY_LEFT_CTRL`、`KEY_RIGHT_CTRL`、`KEY_LEFT_ALT`、`KEY_RIGHT_ALT`、`KEY_UP`、`KEY_DOWN`、`KEY_LEFT`、`KEY_RIGHT`。
- 鼠标：`MOUSE_LEFT`、`MOUSE_RIGHT`、`MOUSE_MIDDLE`。

**已经被占用的键：**

- 游戏自己用了：WASD、空格、Shift、Ctrl、Tab、Esc、F、R、H、F3、F5、1–9、鼠标三个键、滚轮。
- flowers mod 用了 G。

## 存档里保存什么

| 内容 | 是否保存 |
|---|---|
| 改动过的 chunk | 保存，方块按名字保存 |
| 场 | 保存 |
| `g.Mod_data` | 保存 |
| 玩家、物品栏、`Build_depth`、种子 | 保存 |
| mod 自己的 Go 变量 | **不保存**。要保存的东西，在 `On_save` 里写进 `Mod_data`，在 `On_start` 里读回来 |

**增删 mod 不会弄坏存档：**

- 加了新方块，不影响已有的方块。
- 删掉一个 mod，它的方块读档时会变成空气。

## 规矩和常见坑

- **名字都加前缀：** 方块、生物群系、场、`Mod_data` 的 key 都写成 `mod名:` 开头，避免和别的 mod 重名。
- **注册只在 `init()` 里做：** 见[快速开始](#快速开始)。
- **地形必须是确定性的：** feature 里只能用 `ctx.Rand` 和 `ctx.Scatter` 给的随机数，不能用全局随机数或时间。否则同一个 chunk 卸载后再生成会不一样，跨 chunk 的树也会拼不上。
- **feature 只能改 `ctx` 里的这个 chunk：** 不要在 feature 里调用 `g.World`。
- **`World.Set_cube` 和 `g.Set_cube` 不会调用 `On_break` 和 `On_place`：** 只有 `g.Break` 和 `g.Place` 会调用。
- **注意 chunk 没加载的情况：** 远处的 chunk 可能没加载。这时 `Block_at` 和 `Find` 的 loaded 是 false，`Set_cube` 返回错误，要处理这些情况。
- **不要在 `On_update` 里画东西：** 会被清掉，要画在 `On_hud` 里。
- **按键先判断 `g.Playing()`：** 否则玩家在物品栏里点东西时也会触发你的按键。
- **文字只支持 ASCII：** `Display`、`Show_message` 和 2D 文字都一样。
- **Image 画过之后不要改：** canvas 已经把它缓存成贴图了。
- **不要用 `Add_cube` 画大量的方块：** 它每帧都要重新画，会很慢。
