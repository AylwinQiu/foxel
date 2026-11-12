# Lua API

foxel3d_rs 的底层是 Rust，上层的游戏逻辑全部是 Lua。Rust 开好窗口以后，把下面这些**全局模块**装进 Lua，然后运行 [src/game/main.lua](src/game/main.lua)。

| 模块 | 做什么 | 什么时候能用 |
|---|---|---|
| [`foxel`](#foxel) | 随机数、噪声、mod 列表 | 随时 |
| [`block`](#block) | 方块注册表 | 注册只能在 `world.open` 之前 |
| [`image`](#image) | 图片 | 随时 |
| [`terrain`](#terrain) | 生物群系、feature | 登记只能在 `world.open` 之前 |
| [`cube`](#cube) | 八叉树格子的计算 | 随时 |
| [`world`](#world) | 读写方块、射线、碰撞、场、存档 | `world.open` 之后 |
| [`engine`](#engine) | 相机、加载 chunk、画世界和 3D 线 | `world.open` 之后 |
| [`canvas`](#canvas) | 窗口、2D 绘制、输入、截图 | 随时 |

在这些 Rust 模块之上，[src/game/](src/game/) 里是用 Lua 写的游戏本身：[`game`](#game游戏本身)、[`events`](#events事件) 等。mod 主要和这一层打交道，见[写 mod](#写-mod)。

**编辑器补全：** 所有函数的类型定义在 [api/foxel.lua](api/foxel.lua)。装了 Lua Language Server（VS Code 的 sumneko.lua 插件）就能补全，也能检查参数。

## 约定

| 东西 | Lua 里怎么写 |
|---|---|
| 方块 | BlockId（整数），或者名字 `"foxel:stone"`。两种写法哪里都能用 |
| 颜色 | `{r, g, b}` 或 `{r, g, b, a}`，0 到 255；也可以写 `{r=, g=, b=, a=}` |
| 位置、方向 | `{x=, y=, z=}`，物理坐标，1 就是 1 格。x 向东，y 向上，z 向南（北是 -z） |
| 格子（cube） | `{x=, y=, z=, depth=}`：深度为 depth 的第 (x, y, z) 个格子，边长 2^depth，depth 省略时是 0 |
| 图片 | Image、颜色（纯色）、或者 png 的路径（相对游戏根目录） |
| 名字 | 方块、生物群系、feature、场、`mod_data` 的 key，都写成 `mod名:名字` |

**depth：** 最小是 -4（1/16 格），-2 是 1/4 格，-1 是 1/2 格，0 是普通方块，4 是 chunk（16 格）。比如 `{x=12, y=244, z=12, depth=-2}` 在方块 (3, 61, 3) 里面。

**出错：** 参数不对会抛 Lua 错误。操作可能失败的函数（比如 `world.set`）返回 `false, 原因`，可以 `assert(world.set(...))`。

## foxel

| 函数 | 说明 |
|---|---|
| `foxel.rng(seed)` | 确定性的随机数：`r:random()` 是 [0, 1) 的小数，`r:random(n)` 是 1 到 n，`r:random(m, n)` 是 m 到 n，`r:next()` 是 64 位整数 |
| `foxel.noise(seed)` | 2D Perlin 噪声：`n:noise(x, y)`，`n:fbm(x, y, octaves)`，输出大约在 [-1, 1] |
| `foxel.time()` | 游戏开始后过了多少秒 |
| `foxel.mods()` | `mods/` 下面所有有 `init.lua` 的文件夹名字，按名字排序 |
| `foxel.root` | 游戏的根目录 |

地形和贴图里不要用 `math.random`，它每次运行都不一样。用 `foxel.rng`，同一个种子永远得到同样的结果。

## block

```lua
local RUBY = block.register {
    name = "mymod:ruby", display = "Ruby",
    images = "mods/mymod/ruby.png",
}
```

`block.register(def)` 注册一种方块，返回 BlockId。`def` 的字段：

| 字段 | 默认 | 说明 |
|---|---|---|
| `name` | 必填 | 唯一的名字。存档按名字保存方块，所以注册顺序变了、增删了 mod 都不会弄乱存档 |
| `display` | `name` | 给玩家看的名字，只能用 ASCII |
| `images` | 白色 | 6 个面的贴图，见下面 |
| `solid` | `true` | 有碰撞，玩家不能穿过 |
| `opaque` | `true` | 完全不透明，会挡住旁边方块的面。**有透明像素的方块（树叶、玻璃）要写 false**，否则会看到后面的东西被挖空 |
| `cull_same` | `false` | 两个同种的、不透明度为 false 的方块挨着时，省掉中间的面。玻璃写 true，树叶写 false |
| `unbreakable` | `false` | 玩家挖不掉 |
| `hidden` | `false` | 不出现在背包里 |
| `props` | `{}` | 自定义属性，`{[字符串] = 字符串}` |

**images：** 可以是一张图片，6 个面都用它；也可以分面写，后面的覆盖前面的：`all` < `side` < 具体的面。

```lua
images = { top = grass, side = grass_side, bottom = dirt }   -- top也可以写up，bottom也可以写down
images = { all = stone, north = stone_face }                 -- 只有北面不一样
images = { 220, 40, 50 }                                     -- 纯色
```

图片朝向：从 cube 外面看都是正的。侧面图片的上方朝上；up 面图片的上方朝北，down 面图片的上方朝南。

其他函数：

| 函数 | 说明 |
|---|---|
| `block.get(b)` | 方块信息：`{id, name, display, solid, opaque, cull_same, unbreakable, hidden, props, images, icon}`。`images` 是 6 个面的 Image，`icon` 是背包里显示的那张 |
| `block.id(name)` | 名字换成 BlockId，没有时返回 nil |
| `block.name(b)` | BlockId 换成名字 |
| `block.is_solid(b)` | 有没有碰撞 |
| `block.all()` | 所有 BlockId，包括空气，从 0 开始 |
| `block.AIR` | 空气，就是 0 |

## image

| 函数 | 说明 |
|---|---|
| `image.solid(color)` | 纯色图片。同一个颜色总是同一张 |
| `image.pixels(w, h, f)` | 按像素生成：`f(x, y)` 返回 `r, g, b[, a]` 或者一个颜色，(0, 0) 是左上角，按行依次调用 |
| `image.load(path)` | 读 png，路径相对游戏根目录。同一个路径只读一次，读不了时是品红色 |
| `img:size()` | 宽、高 |
| `img:pixel(x, y)` | 那个像素的颜色 |

alpha 小于 128 的像素不画，可以拿来做镂空的贴图。

**图片不要每帧新建。** canvas 会把每张新图片放进显卡里的贴图集，贴图集满了以后，新图片都显示成品红色。所以贴图要在加载时做好，存在变量里反复用。`image.solid` 有缓存，每帧调用没关系。

程序生成贴图的例子见 [src/game/textures.lua](src/game/textures.lua)。

## terrain

**生成的顺序：** 每个 chunk 先按生物群系填好地面，再按登记的顺序执行每个 feature：

- 地面从上往下依次是：表层、表层下面几层、石头；
- 最底下一层是基岩。

**改内置内容：** 同名的生物群系、feature 再登记一次时，只改给出的字段。所以 mod 可以这样调整内置的地形：

```lua
terrain.biome { name = "foxel:desert", variation = 8 }                          -- 沙漠起伏大一点
terrain.scatter { name = "foxel:oak_trees", density = { ["foxel:plains"] = 0.02 } } -- 平原多长点树
terrain.remove("foxel:iron")                                                     -- 不要铁矿
```

### terrain.settings

`terrain.settings { stone=, bedrock=, snow=, min_chunk_y=, max_chunk_y= }`

- `stone`、`bedrock`、`snow`：石头、基岩、雪用哪种方块。
- `min_chunk_y`、`max_chunk_y`：世界的竖直范围，单位是 chunk，两端都包括。默认 -2 到 7，也就是方块 y 从 -32 到 127。

### terrain.biome

每一列选气候最接近的生物群系，交界处的高度会平滑过渡。

| 字段 | 说明 |
|---|---|
| `name` | 名字 |
| `temperature`、`humidity` | 气候，-1 到 1 |
| `base_height`、`variation` | 地面的平均高度和起伏 |
| `ridged` | 用山脊形状的起伏（只往上起伏），适合山脉 |
| `surface` | 最上面一层的方块 |
| `subsurface`、`subsurface_depth` | 下面几层的方块和层数 |
| `rock_line` | 地面高于这个高度时，表层露出石头 |
| `snow_line` | 地面高于这个高度时，表层盖上雪 |

### terrain.ore

矿脉。每个 chunk 放 `per_chunk` 条（默认 4），每条 `size` 个方块（默认 6），只出现在 `min_y` 到 `max_y` 之间，只替换 `replace`（默认石头）。

```lua
terrain.ore { name = "mymod:ruby", salt = 9001, block = RUBY, per_chunk = 2, size = 4, max_y = 0 }
```

### terrain.scatter

树、花这种"在地面上某些位置放一个东西"的 feature，最常用。

```lua
terrain.scatter {
    name = "mymod:rocks", salt = 777,
    density = { ["foxel:plains"] = 0.01 },  -- 每列放一个的概率，没写的生物群系是0
    soil = { "foxel:grass" },               -- 只放在这些方块上，不写就是哪里都行
    radius = 1,                             -- 东西最多离中心多少格（水平方向），默认0
    height = 2,                             -- 东西最多比地面高多少格，默认16
    place = function(ctx, x, y, z, rng)     -- y是地面方块的高度
        ctx:set(x, y + 1, z, "foxel:cobblestone")
    end,
}
```

**跨 chunk 的东西：** 树冠这种东西会伸到隔壁 chunk 里。不用特别处理，照常写就行：

- 不在这个 chunk 里的方块，`ctx:set` 会自动忽略。
- 隔壁 chunk 生成时，会用同样的随机数算出同一棵树，两边正好拼上。
- 前提是 `radius` 和 `height` 写对，否则树会在 chunk 边上被切掉。

**salt：** 每个 feature 用不同的 salt，否则它们会挑中同样的位置。

### terrain.feature

`terrain.feature { name=, generate = function(ctx) ... end }`，每个 chunk 调用一次 `generate`。

它比 scatter 慢，因为不管有没有东西要放，每个 chunk 都要进一次 Lua。能用 scatter 时尽量用 scatter。

### ctx（生成中的 chunk）

feature 的回调里只能通过 `ctx` 读写世界。这时世界正在生成，调用 `world.*` 会报错。

| 方法 | 说明 |
|---|---|
| `ctx:get(x, y, z)` | 那个方块现在是什么。不在这个 chunk 里时返回 nil |
| `ctx:set(x, y, z, b)` | 写一个方块。不在这个 chunk 里时什么都不做 |
| `ctx:set_if_air(x, y, z, b)` | 只有原来是空气时才写（树叶不要盖掉别的树干） |
| `ctx:set_small(x, y, z, depth, b)` | 放一个比 1 小的 cube |
| `ctx:column(x, z)` | 这一列地面的高度、生物群系名字 |
| `ctx:ground(x, z)` | 这一列地面最上面的方块（不考虑 feature 改过的） |
| `ctx:rand(x, y, z, salt)` | 确定性的随机数 |
| `ctx:origin()` | chunk 最小那个角的方块坐标 |
| `ctx:chunk()` | chunk 的 cube |
| `ctx:seed()` | 世界的种子 |

**一定要确定：** 只能用 `rng`、`ctx:rand` 给的随机数，不要用 `math.random`，也不要读时间。否则同一个 chunk 卸载以后再生成会不一样。

feature 出错时会打印错误，这个 feature 之后就停用了，游戏不会崩。

### 其他

| 函数 | 说明 |
|---|---|
| `terrain.biomes()`、`terrain.features()` | 已登记的名字 |
| `terrain.column(x, z)` | 世界打开以后：这一列地面的高度和生物群系名字 |

## cube

| 函数 | 说明 |
|---|---|
| `cube.at(pos, depth)` | 包含位置 pos 的、深度为 depth（默认 0）的格子 |
| `cube.block(x, y, z)` | 方块 (x, y, z) 的格子 |
| `cube.min(c)`、`cube.max(c)`、`cube.center(c)` | 两个角和中心 |
| `cube.side(c)` | 边长。也可以直接传深度：`cube.side(-2)` 是 0.25 |
| `cube.parent(c, depth)` | 包含它的、深度为 depth 的格子 |
| `cube.child(c, o)` | 第 o 个子格子（0 到 7），x 正为 1，y 正为 2，z 正为 4 |
| `cube.contains(a, b)` | b 在不在 a 里面（包括 b 就是 a） |
| `cube.chunk(c)` | 所在的 chunk |
| `cube.neighbor(c, dx, dy, dz)` | 同样大小、挨着的格子 |
| `cube.equal(a, b)` | 两个格子是不是同一个 |
| `cube.key(c)` | 能当 table key 的字符串，比如 `"3,60,-2@0"`。Lua 的 table 不能直接拿来当 key 比较 |
| `cube.MIN_DEPTH`、`cube.CHUNK_DEPTH`、`cube.CHUNK_SIZE` | -4、4、16 |

## world

### 打开和存档

```lua
local meta = world.open { save = "default", seed = 123 }
-- meta：上次world.save存的东西；新世界时是nil
world.save { player = ..., hotbar = ... }
```

- **`world.open`：** 有存档时读存档，种子用存档里的；否则用 `seed` 生成新世界，不写 `seed` 就随机。
  - 存档放在 `saves/<save>/`。
  - 打开以后，方块和地形都不能再注册了。
- **`world.save(meta)`：** 保存改动过的 chunk、场，再加上 `meta`。
  - `meta` 是任何能转成 JSON 的 Lua 值，游戏用它存玩家、物品栏、mod 数据。
  - 函数和 userdata 不能放进去。
- `world.is_open()`、`world.seed()`。

### 读和写

| 函数 | 说明 |
|---|---|
| `world.get(x, y, z)` | 方块 (x, y, z) 的 BlockId，和它是不是被分成了更小的格子（这时 BlockId 没有意义）。没加载时返回 nil |
| `world.set(x, y, z, b)` | 设成方块 b。成功返回 true，失败返回 `false, 原因`（比如 chunk 没加载） |
| `world.set_cube(c, b)` | 把格子 c 整个设成 b，`block.AIR` 就是挖掉。里面原来的东西和场都会删掉。深度要在 -4 到 4 之间 |
| `world.find(c)` | 格子 c：方块、真正的叶子格子（c 在更大的叶子里时是那个叶子）、是不是被分开了 |
| `world.leaf_at(pos)` | 包含位置 pos 的最小的格子：方块和格子 |
| `world.loaded_chunks()` | 已加载的 chunk 数 |

**自动合并：** 8 个一样的兄弟格子会自动合并成一个。所以你放下去的 1/2 格子，`world.find` 可能找到的是更大的格子。

**挖和放时的事件：** `world.set`、`world.set_cube` 不会触发 `break`、`place` 事件。想像玩家一样挖和放，用 [`G.break_cube`、`G.place`](#game游戏本身)。

### 射线

```lua
local hit = world.raycast(eye, dir, 6)  -- 第三个参数是最远距离
if hit then print(hit.block, hit.cube, hit.point, hit.normal, hit.distance) end
```

`world.raycast(origin, dir, max_distance, filter)` 找第一个 `filter(方块)` 为 true 的格子。`filter` 不写时是"不是空气"，写了的话每种方块只会问一次。

返回值的字段：

- `normal`：打到的那个面朝外的方向，比如 `{x=0, y=1, z=0}` 是上面。
- 碰到没加载的 chunk 就停下，返回 nil。

### 碰撞

| 函数 | 说明 |
|---|---|
| `world.box_collides(lo, hi)` | lo 到 hi 的长方体有没有碰到实心方块。没加载的地方算实心 |
| `world.move_box(pos, width, height, delta, step)` | 移动一个长方体，碰到方块就停在边上。返回新位置和 `{x=, y=, z=}`（每个轴有没有被挡住） |

`move_box` 的长方体用"底面中心 pos、宽 width、高 height"表示。

- **台阶：** `step` 大于 0 时，站在地上走路碰到不超过 `step` 高的台阶会迈上去。
- **不会穿墙：** 每次最多走 0.05 格，速度再快也不会穿过 1/16 格的方块。
- **用法：** 玩家就是用它移动的（[src/game/player.lua](src/game/player.lua)），以后的动物也可以用。

### 场

场是给某个格子附加的字符串，跟着存档保存。格子被 `set_cube` 覆盖（包括挖掉）时，里面的场会自动删掉。

| 函数 | 说明 |
|---|---|
| `world.field_write(name, c, value)` | 写 |
| `world.field_read(name, c)` | 读，没有时返回 nil |
| `world.field_delete(name, c)` | 删 |
| `world.field_all(name)` | 这个场里的所有数据：`{{cube=, value=}, ...}` |

## engine

engine 管相机、加载 chunk 和画世界，坐标都是物理坐标。

| 函数 | 说明 |
|---|---|
| `engine.set_camera(pos, dir)`、`engine.camera()` | 设置、读取相机的位置和朝向 |
| `engine.set_view_distance(n)`、`engine.view_distance()` | 视距，单位是 chunk，1 到 64 |
| `engine.update(budget_ms)` | 加载相机附近的 chunk、卸载远处的、重新生成改动过的网格，最多花 `budget_ms` 毫秒（默认 8）。没做完的留到下一帧 |
| `engine.ready(radius)` | 相机周围 radius（默认 2）个 chunk 以内是否都加载好、网格也生成好了 |
| `engine.draw_world()` | 把视锥里的 chunk 画出来 |
| `engine.line(a, b, color)` | 3D 线 |
| `engine.box(lo, hi, color)` | 长方体的 12 条边 |
| `engine.cube(center, side, images, rotation)` | 一个会动的 cube，不在世界里。`rotation` 的方向是转轴，长度是转角（弧度） |
| `engine.stats()` | `{loaded, meshes, drawn, pending}` |

`engine.cube` 每帧都要重新画，只适合少量会动的东西。要放很多方块，用 `world.set`。

## canvas

**每帧的流程：**

```lua
canvas.clean()        -- 清掉上一帧的东西
engine.draw_world()   -- 3D
canvas.text(...)      -- 2D，按顺序画，后画的盖在上面
canvas.draw()         -- 显示出来，然后读下一帧的输入
```

### 窗口和画面

| 函数 | 说明 |
|---|---|
| `canvas.should_close()` | 窗口是否需要关闭 |
| `canvas.size()` | 窗口的宽、高（像素） |
| `canvas.frame_time()` | 上一帧用了多少秒 |
| `canvas.set_background(color)` | 天空的颜色 |
| `canvas.set_fog(color, start, end)` | 雾：离相机 `start` 以内没有雾，到 `end` 完全变成雾的颜色 |
| `canvas.view_angle(h, v, tilt, auto)` | 视角，角度制：横向、纵向、倾斜。`auto`（默认 true）时纵向按窗口宽高比算 |
| `canvas.screenshot(path)` | 下一次 `canvas.draw` 时把画面存成 png |

### 2D

坐标是像素，(0, 0) 是左上角。

| 函数 | 说明 |
|---|---|
| `canvas.text(text, x, y, color)` | 文字，字号 20，只支持 ASCII，可以用 `\n` 换行。颜色默认白色 |
| `canvas.measure(text)` | 文字画出来的宽、高 |
| `canvas.line(x1, y1, x2, y2, color)` | 线 |
| `canvas.rect(x, y, w, h, color)` | 实心矩形 |
| `canvas.image(img, x, y, w, h)` | 图片 |

### 输入

`canvas.input()` 返回这一帧的输入：

```lua
local input = canvas.input()
if input.pressed.g then ... end         -- 这一帧刚按下G
if input.held.left_shift then ... end   -- 按着Shift
```

| 字段 | 说明 |
|---|---|
| `held` | 按着的键，`{[键名] = true}` |
| `pressed` | 这一帧刚按下的键 |
| `mouse_dx`、`mouse_dy` | 鼠标这一帧移动了多少像素，x 向右、y 向下为正 |
| `mouse_x`、`mouse_y` | 鼠标在窗口里的位置 |
| `wheel` | 滚轮，向上为正 |
| `text` | 这一帧输入的文字，已经按键盘布局转换好了，给聊天框、命令行用 |

**键名：**

- 字母和数字：`"a"`–`"z"`、`"0"`–`"9"`。
- 功能键：`"f1"`–`"f12"`。
- 其他键：`"space"`、`"enter"`、`"escape"`、`"tab"`、`"backspace"`、`"left_shift"`、`"right_shift"`、`"left_ctrl"`、`"right_ctrl"`、`"left_alt"`、`"right_alt"`、`"up"`、`"down"`、`"left"`、`"right"`、`"slash"`、`"minus"`、`"equal"`、`"comma"`、`"period"`、`"grave"`。
- 鼠标：`"mouse_left"`、`"mouse_right"`、`"mouse_middle"`。

**鼠标锁定：** `canvas.lock_cursor(true)` 隐藏光标，鼠标可以一直往一个方向移动，第一人称视角要用。`canvas.cursor_locked()` 读当前状态。

---

## game（游戏本身）

[src/game/](src/game/) 里是用上面的 API 写的游戏，mod 通过 `require` 拿到它们：

| 文件 | `require` | 内容 |
|---|---|---|
| [main.lua](src/game/main.lua) | | 入口：解析参数，加载内置内容和 mod，`game.run` |
| [blocks.lua](src/game/blocks.lua) | `require("blocks")` | 内置方块，返回 `B.STONE`、`B.GRASS`……和 `B.hotbar` |
| [textures.lua](src/game/textures.lua) | `require("textures")` | 程序生成贴图的工具 |
| [worldgen.lua](src/game/worldgen.lua) | | 内置的生物群系、矿石、树 |
| [game.lua](src/game/game.lua) | `require("game")` | 游戏状态和一帧里做什么 |
| [player.lua](src/game/player.lua) | `require("player")` | 玩家的移动 |
| [hud.lua](src/game/hud.lua) | `require("hud")` | 界面 |
| [events.lua](src/game/events.lua) | `require("events")` | 事件 |
| [console.lua](src/game/console.lua) | `require("console")` | 按 `` ` `` 打开的 Lua 终端。`console.exec(G, code)` 可以直接执行一行代码 |
| [vec.lua](src/game/vec.lua) | `require("vec")` | 向量计算 |

`require("game")` 拿到的是同一个表，我们叫它 `G`。

`G` 的字段：

| 字段 | 说明 |
|---|---|
| `G.player` | 玩家：`pos`（脚底中心）、`vel`、`yaw`（0 朝北，π/2 朝东）、`pitch`（向上为正）、`flying`、`on_ground`、`crouching`、`sprinting`。传送就是直接改 `G.player.pos` |
| `G.hotbar`、`G.selected` | 物品栏（9 个 BlockId），以及选中的第几格（1 到 9） |
| `G.build_depth` | 挖和放的格子深度：0 到 -4，按 R 切换。改它用 `G.set_build_depth(d)`，会限制在范围里并显示消息 |
| `G.target` | 玩家指着的东西（`world.raycast` 的结果），没有时是 nil |
| `G.mod_data` | mod 的数据，跟着存档保存。可以放任何能转成 JSON 的值 |
| `G.time` | 游戏开始后过了多少秒 |

`G` 的函数：

| 函数 | 说明 |
|---|---|
| `G.playing()` | 玩家正在玩：加载完、鼠标锁定、没开背包。处理按键前先判断它 |
| `G.show_message(text)` | 屏幕上方显示一条消息，2.5 秒后消失 |
| `G.break_cube(c)` | 像玩家一样挖：检查挖不掉的方块，触发 `break` 事件 |
| `G.place(c, b)` | 像玩家一样放：格子要是空的，不能卡住玩家，触发 `place` 事件 |
| `G.target_cell()` | 玩家指着的方块里，按 `build_depth` 大小算的要挖掉的格子 |
| `G.place_cell()` | 玩家指着的那个面外面，要放方块的格子 |
| `G.surface_below(x, y, z, max)` | 从 y 往下找第一个实心方块，返回它上面那个空格子的 y |
| `G.info(b)` | 有缓存的 `block.get(b)` |
| `G.save()` | 立刻存档 |

## events（事件）

```lua
local events = require("events")
events.on("break", function(G, c, b)
    if b == RUBY then G.show_message("Shiny!") end
end)
```

| 事件 | 参数 | 什么时候 |
|---|---|---|
| `"start"` | `G` | 世界打开、存档读完之后，chunk 还没开始加载 |
| `"update"` | `G, input, dt` | 每一帧，玩家移动之后、画之前 |
| `"hud"` | `G` | 画完游戏自己的界面之后，可以在上面画东西 |
| `"break"` | `G, cube, block` | 玩家要挖掉 cube 之前。**返回 false 就取消** |
| `"place"` | `G, cube, block` | 玩家要在 cube 放方块 block 之前。**返回 false 就取消** |
| `"save"` | `G` | 存档之前，可以把数据写进 `G.mod_data` |

一个处理函数出错时，会打印出来，不影响别的函数和游戏本身。

**画东西要在 `"hud"` 里画。** 每帧画之前 canvas 会被清空，在 `"update"` 里画的东西会被清掉。

## 写 mod

mod 就是 `mods/` 下面的一个文件夹，里面有 `init.lua`。游戏启动时按文件夹名字的顺序 `require` 每个 mod，时机在内置方块和地形之后、`world.open` 之前。

```lua
-- mods/mymod/init.lua
local events = require("events")

local RUBY = block.register { name = "mymod:ruby", display = "Ruby", images = { 200, 30, 60 } }
terrain.ore { name = "mymod:ruby", salt = 9001, block = RUBY, per_chunk = 2, size = 4, max_y = 0 }

events.on("update", function(G, input)
    if G.playing() and input.pressed.k then
        G.player.pos.y = G.player.pos.y + 10
        G.show_message("Jump!")
    end
end)
```

完整的例子见 [mods/flowers](mods/flowers/init.lua)，大约 90 行。它演示了这几样东西：

- 新方块；
- 地形 scatter；
- 按键事件；
- 挖方块事件。

**可以用 Lua 写，也可以用 Rust 写：**

- **Lua：** 像上面那样，大部分东西都适合用 Lua。
- **Rust：** 需要性能的东西（新的地形算法、大量方块的计算）用 Rust 实现，再在 [src/lua/](src/lua/) 里加一个函数绑定给 Lua。Lua 这边用起来和内置的 API 一样。

## 注意

- **注册的时机：** 方块、生物群系、feature 只能在 `world.open` 之前注册，`world.open` 之后再调用会报错。
- **地形回调里只能用 ctx：** feature 的回调里世界正在生成，调用 `world.*` 会报错。
- **不要每帧新建图片：** 见 [image](#image)。
- **Lua 5.5 和 5.4 不一样的地方：** `for` 循环的控制变量是只读的，不能在循环里给它赋值。
