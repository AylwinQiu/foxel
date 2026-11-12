# foxel3d_rs

一个方块沙盒游戏，玩法类似我的世界。它是 [foxel3d_ai](../foxel3d_ai) 的重构版，玩法一样，结构改成了两层：

- **底层是 Rust：** 世界的八叉树、地形生成、网格、物理碰撞、存档、窗口。
- **上层是 Lua 5.5：** 方块、生物群系、树都在 Lua 里定义；玩家怎么动、界面长什么样、按键做什么也都是 Lua 写的。

游戏入口是 [src/game/main.lua](src/game/main.lua)。Rust 开好窗口、把 API 装进 Lua 以后，剩下的都交给它。

```sh
cd foxel3d_rs
cargo run --release                                      # 打开（或新建）存档 saves/default
cargo run --release -- --world test --seed 42 --view 12  # 另一个存档、种子、视距
cargo test                                               # 测试，不需要开窗口
```

**需要的环境：**

- Rust 1.88 或更新的版本（edition 2024）。
- cmake 和 C 编译器，用来编译 raylib 和 Lua。
- macOS 上装 Xcode Command Line Tools 和 `brew install cmake` 就行。

第一次编译大约要 30 秒，因为 raylib 和 Lua 都要从源码编译。

在视距 8 和 12 下，游戏都能跑满 60 帧（垂直同步）。每帧的 Lua 和 Rust 逻辑加起来不到 0.5 毫秒。

## 操作

| 键 | 作用 | 键 | 作用 |
|---|---|---|---|
| WASD | 移动 | 空格 | 跳 / 飞行时上升 |
| 鼠标 | 看 | Shift | 蹲下（能钻过 1 格高的缝）/ 飞行时下降 |
| 左键 | 挖 | 双击 W | 跑（按住 Ctrl 也行） |
| 右键 | 放 | F | 切换飞行 |
| 中键 | 选中指着的方块 | R | 挖和放的深度：0（1 格）→ -1 → -2 → -3 → -4（1/16 格）→ 0 |
| 1-9 / 滚轮 | 选物品栏 | Tab | 背包（所有方块） |
| F5 | 存档 | Esc | 暂停 |
| F3 | 调试信息 | F2 | 截图（存到 screenshots/） |
| H | 帮助 | G | 在身边种花（flowers mod） |
| `` ` `` | Lua 终端（见下面） | | |

关闭窗口时会自动存档，另外每 5 分钟也会自动存一次。

## Lua 终端

在游戏里按 `` ` `` 打开一个小终端，输入 Lua 代码回车执行，`Esc` 或 `` ` `` 关闭。终端开着时游戏不响应按键。

```lua
> G.player.pos                 -- 写表达式会显示结果
{ x = 0.5, y = 15, z = 0.5 }
> tp(100, 80, 100)             -- 传送
> give("foxel:glass")          -- 把方块放进选中的格子
> for x = 0, 9 do world.set(x, 20, 0, B.BRICKS) end
> x = 40                       -- 全局变量会留到下一行（local不会）
```

终端里能用所有 API，另外还有 `G`、`P`、`B`、`V`、`events`（对应的模块）以及 `here()`、`tp()`、`give()`、`clear()`、`help()`。

- **翻历史：** 上下键。
- **出错不会崩：** 出错时显示红字，游戏不受影响。
- **小心死循环：** 写了死循环会卡住游戏。

## 结构

```
mods/                Lua mod，每个文件夹一个，启动时自动加载
src/game/   (Lua)    游戏本身：方块、地形、玩家、界面、事件      ← 游戏逻辑都在这里
─────────────────────────────── Lua API（API.md） ───────────────────────────────
src/lua/    (Rust)   把下面的功能做成Lua的全局模块
src/engine/          按相机加载chunk、生成网格、视锥剔除
src/terrain/         噪声、生物群系、feature（矿石、散布、自定义）
src/block.rs         方块注册表
src/save.rs          存档
src/world/           八叉树、CubeId、场、chunk编码、射线、碰撞
src/canvas/          Canvas接口 + raylib实现           ← 换前端只换这里
```

每一层只依赖它下面的层。`world` 不知道方块长什么样，`terrain` 不知道 Lua 的存在，`engine` 只认 `Canvas` 这个接口。

### 为什么这样分

**放在 Rust 的：** 每帧、每个方块都要跑的东西。

- 八叉树的读写；
- 地形噪声；
- 网格生成（一个 chunk 要检查 4096 个格子的 6 个面）；
- 碰撞（每帧几百次查询）；
- 存档编码。

**放在 Lua 的：** 想改、想加、想让 mod 动的东西。

- 有哪些方块，贴图长什么样；
- 有哪些生物群系和树，树是什么形状；
- 玩家怎么动，按键做什么；
- 界面怎么画。

**两层之间的边界按"调用次数"划：**

- **Lua 每帧只调用几十次 API：** 实测每帧只花 0.16 毫秒。
- **地形回调例外，但次数也不多：** Lua 只在"这里要长一棵树"时才被调用。逐列判断要不要长树的循环在 Rust 里，所以 Lua 被调用的次数和树的数量一样。
- **碰撞整个在 Rust 里：** 玩家物理只把"想往哪走"告诉 Rust 的 `world.move_box`，碰撞检测不经过 Lua。

### 换前端

[src/canvas/mod.rs](src/canvas/mod.rs) 里的 `Canvas` trait 是唯一和图形库打交道的地方。换成 wgpu、网页之类的前端，只要做两件事：

1. 实现这个 trait（24 个方法，其中 `screenshot` 可以不实现）。
2. 在 [src/main.rs](src/main.rs) 里换掉 `RaylibCanvas`。

Lua 和游戏逻辑都不用动。

现在有两个实现：

| 实现 | 用途 |
|---|---|
| `RaylibCanvas` | 真正的窗口 |
| `NullCanvas` | 什么都不画，测试用。输入可以一帧一帧预先写好 |

### 一帧里发生了什么

一帧的流程写在 [src/game/game.lua](src/game/game.lua) 的 `G.frame`：

1. `canvas.input()`：读输入。
2. 处理按键，更新玩家。玩家移动调用 Rust 的 `world.move_box`。
3. `world.raycast`：找玩家指着的方块，然后处理挖和放。
4. `events.emit("update")`：通知 mod。
5. `engine.update(8)`：加载附近的 chunk、卸载远处的、重建改动过的网格，最多花 8 毫秒，没做完的留到下一帧。
6. `canvas.clean()`、`engine.draw_world()`、`hud.draw()`、`events.emit("hud")`：画 3D 和 2D。
7. `canvas.draw()`：显示出来。

## 核心概念

### 坐标

- x 向东，y 向上，z 向南（北是 -z）。
- 1 个单位就是 1 格。
- 世界在竖直方向上是 y = -32 到 127，最底下是基岩。
- 水平方向是无限的。

### 八叉树和 CubeId

整个世界是一棵八叉树，根节点的边长是 2^32。

**深度：** 深度 d 的格子边长是 2^d。

| 深度 | 是什么 |
|---|---|
| 4 | chunk，边长 16，是加载、保存、生成网格的单位 |
| 0 | 普通方块 |
| -1 | 1/2 格 |
| -2 | 1/4 格 |
| -4 | 1/16 格，最小的方块 |

**CubeId：** `CubeId{x, y, z, depth}` 表示"深度为 depth 的第 (x, y, z) 个格子"。它就是位置本身，所以：

- 可以直接当 HashMap 的 key；
- 可以直接写进存档；
- 不会因为树的结构变化而失效。

**节点：** 每个节点只存方块编号和子节点指针，一共 16 字节。8 个一样的子节点会自动合并成一个，所以一整块实心的石头只占一个节点。

**chunk 以上不建节点：** chunk 以上的那几层只是容器，不真的建出来。已加载的 chunk 直接放在 HashMap 里，卸载时整个删掉。所以走得再远，内存也不会变大。Go 版会在树里留下空的容器节点，这里没有这个问题。

### 方块注册表和场

**注册表：** world 里只存 `BlockId`。方块叫什么、长什么样、能不能穿过、透不透明，都在注册表里查（Lua：`block.get`）。

**场：** 每个格子自己的数据用场来存（Lua：`world.field_write`），跟着存档保存。

### 地形

**地面高度：** 每一列先用 Perlin 噪声算出温度和湿度，再按气候和每个生物群系的距离加权，混合出地面高度。所以生物群系交界处的高度是平滑过渡的。

**feature：** 地面填好以后，按登记的顺序执行每个 feature。

| feature | 是什么 | 实现 |
|---|---|---|
| ore | 矿脉 | 整个在 Rust 里 |
| scatter | 树、花这种在地面上放东西的 | 位置的选择在 Rust 里，放什么由 Lua 回调决定 |
| feature | 每个 chunk 调用一次的 Lua 函数 | Lua |

**确定性：** 随机数只由种子和位置决定，所以同一个 chunk 每次生成都一样。跨 chunk 的树，在两边 chunk 里会算出同一棵，正好拼上。

### 存档

存档在 `saves/<名字>/`，里面有两个文件：

| 文件 | 内容 |
|---|---|
| `world.json` | 版本、种子、方块名字表、场，以及 Lua 给的数据（玩家、物品栏、mod 数据）。人可以直接看和改 |
| `chunks.bin` | gzip 压缩的二进制，只有改动过的 chunk。每个 chunk 按先序把八叉树编码成一串 varint |

- **不存没改过的 chunk：** 读档时重新生成，所以存档很小。
- **方块按名字保存：** 增删 mod、改变方块的注册顺序，都不会弄乱存档。找不到的方块变成空气。
- **写入是原子的：** 每个文件都先写到临时文件，写完再改名。存到一半出错，也不会把旧存档弄坏。

## Lua API 和 mod

- **完整的 API：** [API.md](API.md)。
- **编辑器补全：** [api/foxel.lua](api/foxel.lua) 是给 Lua Language Server 用的类型定义。

一个最小的 mod：

```lua
-- mods/mymod/init.lua
local events = require("events")

local RUBY = block.register { name = "mymod:ruby", display = "Ruby", images = { 200, 30, 60 } }
terrain.ore { name = "mymod:ruby", salt = 9001, block = RUBY, per_chunk = 2, size = 4, max_y = 0 }

events.on("break", function(G, c, b)
    if b == RUBY then G.show_message("Shiny!") end
end)
```

把文件夹放进 `mods/` 就会被加载，不用改任何代码。完整的例子见 [mods/flowers](mods/flowers/init.lua)。

**内置内容和 mod 的写法完全一样：** 内置的方块和地形（[blocks.lua](src/game/blocks.lua)、[worldgen.lua](src/game/worldgen.lua)）用的也是 mod 能用的这套 API。想知道某个东西怎么做，直接看内置的就行。

**用 Rust 扩展：** 需要性能的东西在 Rust 里实现，然后在 [src/lua/](src/lua/) 里加一个函数绑定给 Lua。

## 测试

```sh
cargo test
```

| 测试 | 内容 |
|---|---|
| `world` | 八叉树的拆分和合并、chunk 编码、场、射线、碰撞（撞墙、上台阶） |
| `terrain` | 地形是确定性的，5 种生物群系都会出现，树能跨 chunk 拼上，矿石只替换石头 |
| `engine` | 哪些面该省掉（玻璃、树叶、小方块、chunk 边界），视锥剔除，卸载再加载后改动还在 |
| [tests/game.rs](tests/game.rs) | 用 `NullCanvas` 把整个 Lua 游戏跑起来（见下面） |

[tests/game.rs](tests/game.rs) 跑两种测试：

- **用测试脚本控制输入：** [tests/lua/play.lua](tests/lua/play.lua) 按顺序测这些操作：走路、跳、撞墙、上台阶、点击回到游戏不挖方块、放 1/4 和 1/16 的方块、存档。然后 [tests/lua/reload.lua](tests/lua/reload.lua) 换一个种子读同一个存档，检查所有东西都还在。
- **运行真正的 `main.lua`：** 跑 200 帧后关闭窗口，检查存档写出来了。

**开发用：** `cargo run -- --main 别的文件.lua` 会用别的 Lua 文件代替 `main.lua`，可以用来写工具脚本、截图、测性能。

## 和 foxel3d_ai 的关系

玩法、地形参数、贴图都和 Go 版一样，方块和地形的定义是从 Go 原样移植的。

但两边的存档不通用：存档格式不一样，噪声的随机数也换了，所以同一个种子生成的地形和 Go 版不同。

不一样的地方：

| | foxel3d_ai（Go） | foxel3d_rs |
|---|---|---|
| 游戏逻辑 | Go | Lua，改了不用重新编译 |
| mod | Go 包，要改 main.go 重新编译 | 放进 mods/ 就行；也可以用 Rust 扩展 |
| 方块、地形 | Go 代码 | Lua，用的就是 mod 的 API |
| 世界树 | chunk 以上建真的节点，会留下空节点 | chunk 以上不建节点，内存不会增长 |
| 存档 | 一个 gob 文件 | world.json（人能读）+ chunks.bin |
| 截图 | 没有 | F2 |
