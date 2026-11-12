---@meta
-- foxel3d_rs的Lua API类型定义，给编辑器（Lua Language Server）做补全和检查用，不会被游戏加载。
-- 说明和例子见API.md。

-- ---------------------------------------------------------------- 通用类型

---方块：BlockId（整数），或者方块的名字（比如"foxel:stone"）
---@alias foxel.Block integer|string

---颜色：{r, g, b}或{r, g, b, a}，0到255；也可以写{r=, g=, b=, a=}
---@alias foxel.Color integer[]|{r: integer, g: integer, b: integer, a?: integer}

---位置或方向（物理坐标，1就是1格）。x向东，y向上，z向南
---@class foxel.Vec
---@field x number
---@field y number
---@field z number

---八叉树的一个格子：深度为depth的第(x, y, z)个格子，边长2^depth。depth 0是普通方块，-4是1/16格（最小）
---@class foxel.Cube
---@field x integer
---@field y integer
---@field z integer
---@field depth? integer 省略时是0

---图片：Image、颜色（纯色）、或者png的路径（相对游戏根目录）
---@alias foxel.ImageLike foxel.Image|foxel.Color|string

---6个面的图片：一张图片（6个面都用），或者分面写。后面的覆盖前面的：all < side < 具体的面
---@alias foxel.Images foxel.ImageLike|{all?: foxel.ImageLike, side?: foxel.ImageLike, top?: foxel.ImageLike, bottom?: foxel.ImageLike, up?: foxel.ImageLike, down?: foxel.ImageLike, north?: foxel.ImageLike, south?: foxel.ImageLike, east?: foxel.ImageLike, west?: foxel.ImageLike}

---@class foxel.Image
local Image = {}
---@return integer width, integer height
function Image:size() end
---@param x integer
---@param y integer
---@return integer[] color
function Image:pixel(x, y) end

---确定性的随机数，同一个种子永远得到同样的序列
---@class foxel.Rng
local Rng = {}
---和math.random一样：random()是[0, 1)的小数，random(n)是1到n，random(m, n)是m到n
---@param m? integer
---@param n? integer
---@return number
function Rng:random(m, n) end
---64位随机整数
---@return integer
function Rng:next() end

---2D Perlin噪声，输出大约在[-1, 1]
---@class foxel.Noise
local Noise = {}
---@return number
function Noise:noise(x, y) end
---分形噪声：octaves层（默认4）叠加，每层频率翻倍、幅度减半
---@return number
function Noise:fbm(x, y, octaves) end

-- ---------------------------------------------------------------- foxel

---杂项
foxel = {}
---游戏的根目录（里面有src/game和mods）
---@type string
foxel.root = ""
---@type string
foxel.version = ""
---@param seed integer
---@return foxel.Rng
function foxel.rng(seed) end
---@param seed integer
---@return foxel.Noise
function foxel.noise(seed) end
---游戏开始后过了多少秒
---@return number
function foxel.time() end
---mods/下面所有有init.lua的文件夹的名字，按名字排序
---@return string[]
function foxel.mods() end

-- ---------------------------------------------------------------- block

---@class foxel.BlockDef
---@field name string 唯一的名字，写成"mod名:方块名"
---@field display? string 给玩家看的名字（只能用ASCII），默认是name
---@field images? foxel.Images
---@field solid? boolean 有碰撞，玩家不能穿过。默认true
---@field opaque? boolean 完全不透明，会挡住旁边方块的面。有透明像素的方块要写false。默认true
---@field cull_same? boolean 两个同种的不透明度为false的方块挨着时省掉中间的面（玻璃）。默认false
---@field unbreakable? boolean 玩家挖不掉。默认false
---@field hidden? boolean 不出现在背包里。默认false
---@field props? table<string, string> 自定义属性

---@class foxel.BlockInfo
---@field id integer
---@field name string
---@field display string
---@field solid boolean
---@field opaque boolean
---@field cull_same boolean
---@field unbreakable boolean
---@field hidden boolean
---@field props table<string, string>
---@field images {up?: foxel.Image, down?: foxel.Image, north?: foxel.Image, south?: foxel.Image, west?: foxel.Image, east?: foxel.Image}
---@field icon? foxel.Image 背包里显示的图标（south面）

---方块注册表
block = {}
---空气的BlockId（0）
block.AIR = 0
---注册一种方块，返回BlockId。只能在world.open之前调用
---@param def foxel.BlockDef
---@return integer
function block.register(def) end
---@param b foxel.Block
---@return foxel.BlockInfo
function block.get(b) end
---名字 -> BlockId，没有时是nil
---@param name string
---@return integer?
function block.id(name) end
---BlockId -> 名字
---@param b foxel.Block
---@return string
function block.name(b) end
---@param b foxel.Block
---@return boolean
function block.is_solid(b) end
---所有方块的BlockId（包括空气），从0开始
---@return integer[]
function block.all() end

-- ---------------------------------------------------------------- image

---图片
image = {}
---纯色图片。同一个颜色总是同一张
---@param c foxel.Color
---@return foxel.Image
function image.solid(c) end
---按像素生成：f(x, y)返回r, g, b[, a]或者一个颜色，(0, 0)是左上角，按行依次调用
---@param w integer
---@param h integer
---@param f fun(x: integer, y: integer): (foxel.Color|integer), integer?, integer?, integer?
---@return foxel.Image
function image.pixels(w, h, f) end
---读png（路径相对游戏根目录）。同一个路径只读一次，读不了时是品红色
---@param path string
---@return foxel.Image
function image.load(path) end

-- ---------------------------------------------------------------- terrain

---@class foxel.TerrainSettings
---@field stone? foxel.Block
---@field bedrock? foxel.Block
---@field snow? foxel.Block
---@field min_chunk_y? integer 世界最下面的chunk（方块y = min_chunk_y*16是基岩）
---@field max_chunk_y? integer 世界最上面的chunk

---@class foxel.BiomeDef
---@field name string
---@field temperature? number -1到1
---@field humidity? number -1到1
---@field base_height? number 地面的平均高度
---@field variation? number 起伏
---@field ridged? boolean 山脊形状的起伏（只往上），适合山脉
---@field surface? foxel.Block 最上面一层
---@field subsurface? foxel.Block 下面几层
---@field subsurface_depth? integer
---@field rock_line? integer 地面高于它时露出石头
---@field snow_line? integer 地面高于它时盖雪

---@class foxel.OreDef
---@field name string
---@field salt integer 每个feature不一样，否则会挑中同样的位置
---@field block foxel.Block
---@field replace? foxel.Block 只替换这种方块，默认石头
---@field per_chunk? integer 每个chunk几条矿脉，默认4
---@field size? integer 每条几个方块，默认6
---@field min_y? integer
---@field max_y? integer

---@class foxel.ScatterDef
---@field name string
---@field salt integer
---@field density table<string, number> 生物群系名 -> 每列放一个的概率
---@field soil? foxel.Block[] 只放在这些方块上，不写就是哪里都行
---@field radius? integer 东西最多离中心多少格（水平），默认0
---@field height? integer 东西最多比地面高多少格，默认16
---@field place fun(ctx: foxel.Context, x: integer, y: integer, z: integer, rng: foxel.Rng) y是地面方块的高度

---@class foxel.FeatureDef
---@field name string
---@field generate fun(ctx: foxel.Context)

---正在生成的chunk。只在feature的回调里有效，回调里不能用world.*
---@class foxel.Context
local Context = {}
---chunk最小的角的方块坐标
---@return integer x, integer y, integer z
function Context:origin() end
---@return foxel.Cube
function Context:chunk() end
---@return integer
function Context:seed() end
---方块坐标(x, y, z)是什么，不在这个chunk里时是nil
---@return integer?
function Context:get(x, y, z) end
---不在这个chunk里时什么都不做
---@param b foxel.Block
function Context:set(x, y, z, b) end
---只有原来是空气时才设
---@param b foxel.Block
function Context:set_if_air(x, y, z, b) end
---放一个比1小的cube：深度为depth的格子(x, y, z)
---@param b foxel.Block
function Context:set_small(x, y, z, depth, b) end
---这一列地面的高度和生物群系的名字
---@return integer height, string biome
function Context:column(x, z) end
---这一列地面最上面的方块（不考虑feature）
---@return integer
function Context:ground(x, z) end
---确定性的随机数：同一个世界、同一个位置、同一个salt总是一样的
---@return foxel.Rng
function Context:rand(x, y, z, salt) end

---地形。除了column，都只能在world.open之前调用；同名的再登记一次时只改给出的字段
terrain = {}
---@param def foxel.TerrainSettings
function terrain.settings(def) end
---@param def foxel.BiomeDef
function terrain.biome(def) end
---@param def foxel.OreDef
function terrain.ore(def) end
---树、花这种在地面上放东西的feature。再登记一次同名的时，density按生物群系合并
---@param def foxel.ScatterDef
function terrain.scatter(def) end
---每个chunk调用一次generate(ctx)。比scatter慢，能用scatter时尽量用scatter
---@param def foxel.FeatureDef
function terrain.feature(def) end
---删掉一个生物群系或feature
---@param name string
function terrain.remove(name) end
---@return string[]
function terrain.biomes() end
---@return string[]
function terrain.features() end
---世界打开以后：(x, z)这一列地面的高度和生物群系的名字
---@return integer height, string biome
function terrain.column(x, z) end

-- ---------------------------------------------------------------- cube

---CubeId的计算
cube = {}
cube.MIN_DEPTH = -4
cube.CHUNK_DEPTH = 4
cube.CHUNK_SIZE = 16
---包含位置p的、深度为depth（默认0）的cube
---@param p foxel.Vec
---@param depth? integer
---@return foxel.Cube
function cube.at(p, depth) end
---方块(x, y, z)的cube（depth 0）
---@return foxel.Cube
function cube.block(x, y, z) end
---最小的角
---@param c foxel.Cube
---@return foxel.Vec
function cube.min(c) end
---最大的角
---@param c foxel.Cube
---@return foxel.Vec
function cube.max(c) end
---@param c foxel.Cube
---@return foxel.Vec
function cube.center(c) end
---边长。可以传cube，也可以直接传深度
---@param c foxel.Cube|integer
---@return number
function cube.side(c) end
---包含它的、深度为depth的cube
---@param c foxel.Cube
---@param depth integer
---@return foxel.Cube
function cube.parent(c, depth) end
---第o个子cube（0到7），o是卦限：x正为1，y正为2，z正为4
---@param c foxel.Cube
---@param o integer
---@return foxel.Cube
function cube.child(c, o) end
---b在不在a里面（包括b就是a）
---@param a foxel.Cube
---@param b foxel.Cube
---@return boolean
function cube.contains(a, b) end
---所在的chunk
---@param c foxel.Cube
---@return foxel.Cube
function cube.chunk(c) end
---同样大小、挨着的cube，dx、dy、dz是格子数
---@param c foxel.Cube
---@return foxel.Cube
function cube.neighbor(c, dx, dy, dz) end
---@param a foxel.Cube
---@param b foxel.Cube
---@return boolean
function cube.equal(a, b) end
---可以当table的key用的字符串，比如"3,60,-2@0"
---@param c foxel.Cube
---@return string
function cube.key(c) end

-- ---------------------------------------------------------------- world

---@class foxel.Hit
---@field block integer 打到的方块
---@field cube foxel.Cube 打到的叶子cube
---@field point foxel.Vec 射线进入这个cube的点
---@field normal {x: integer, y: integer, z: integer} 打到的面朝外的方向；从cube里面出发时是0
---@field distance number

---世界
world = {}
---打开世界：有存档时读存档（种子用存档里的），否则用seed生成新世界。
---返回上次world.save存的meta，新世界时是nil。打开以后不能再注册方块和地形
---@param opts? {save?: string, seed?: integer}
---@return any meta
function world.open(opts) end
---存档：改动过的chunk、场，加上meta（能转成JSON的Lua值）
---@param meta any
function world.save(meta) end
---@return boolean
function world.is_open() end
---@return integer
function world.seed() end
---方块(x, y, z)：BlockId和它是不是被分成了更小的cube（这时BlockId没有意义）。没加载时是nil
---@return integer? block, boolean? split
function world.get(x, y, z) end
---把方块(x, y, z)设成b。成功返回true，失败返回false和原因
---@param b foxel.Block
---@return boolean ok, string? err
function world.set(x, y, z, b) end
---把cube整个设成b（AIR就是挖掉），里面原来的东西和场都删掉。深度要在-4到4之间
---@param c foxel.Cube
---@param b foxel.Block
---@return boolean ok, string? err
function world.set_cube(c, b) end
---找到cube：方块、真正的叶子cube（cube在更大的叶子里时是那个叶子）、是不是被分开了。没加载时是nil
---@param c foxel.Cube
---@return integer? block, foxel.Cube? leaf, boolean? split
function world.find(c) end
---包含位置p的最小的cube：方块和cube。没加载时是nil
---@param p foxel.Vec
---@return integer? block, foxel.Cube? leaf
function world.leaf_at(p) end
---从origin沿dir找第一个filter(方块)为true的cube（filter不写时是"不是空气"）。碰到没加载的chunk就停下
---@param origin foxel.Vec
---@param dir foxel.Vec
---@param max_distance number
---@param filter? fun(b: integer): boolean
---@return foxel.Hit?
function world.raycast(origin, dir, max_distance, filter) end
---lo到hi的长方体有没有碰到实心方块（没加载的地方算实心）
---@param lo foxel.Vec
---@param hi foxel.Vec
---@return boolean
function world.box_collides(lo, hi) end
---移动一个长方体（底面中心pos，宽width，高height）delta，碰到实心方块就停在边上。
---step大于0时，站在地上走路碰到不超过step高的台阶会迈上去
---@param pos foxel.Vec
---@param width number
---@param height number
---@param delta foxel.Vec
---@param step? number
---@return foxel.Vec pos, {x: boolean, y: boolean, z: boolean} hit
function world.move_box(pos, width, height, delta, step) end
---场：给cube附加的字符串，跟着存档保存，cube被覆盖时自动删掉。场名请用"mod名:场名"
---@param name string
---@param c foxel.Cube
---@param value string
function world.field_write(name, c, value) end
---@param name string
---@param c foxel.Cube
---@return string?
function world.field_read(name, c) end
---@param name string
---@param c foxel.Cube
function world.field_delete(name, c) end
---@param name string
---@return {cube: foxel.Cube, value: string}[]
function world.field_all(name) end
---已加载的chunk数
---@return integer
function world.loaded_chunks() end

-- ---------------------------------------------------------------- engine

---@class foxel.Stats
---@field loaded integer 已加载的chunk
---@field meshes integer 有网格的chunk
---@field drawn integer 上一帧画了的chunk
---@field pending integer 还在排队生成网格的chunk

---相机、加载、画世界。坐标都是物理坐标
engine = {}
---@param pos foxel.Vec
---@param dir foxel.Vec
function engine.set_camera(pos, dir) end
---@return foxel.Vec pos, foxel.Vec dir
function engine.camera() end
---视距（chunk数，1到64）
---@param n integer
function engine.set_view_distance(n) end
---@return integer
function engine.view_distance() end
---加载相机附近的chunk、卸载远处的、重新生成改动过的网格，最多花budget毫秒（默认8）
---@param budget_ms? number
function engine.update(budget_ms) end
---相机周围radius（默认2）个chunk以内是否都加载好、网格也生成好了
---@param radius? integer
---@return boolean
function engine.ready(radius) end
---把视锥里的chunk画出来（每帧canvas.clean之后调用）
function engine.draw_world() end
---3D线
---@param a foxel.Vec
---@param b foxel.Vec
---@param c foxel.Color
function engine.line(a, b, c) end
---长方体的12条边
---@param lo foxel.Vec
---@param hi foxel.Vec
---@param c foxel.Color
function engine.box(lo, hi, c) end
---一个会动的cube（不在世界里，每帧都要画）：中心、边长、贴图、旋转（方向是转轴，长度是弧度）
---@param center foxel.Vec
---@param side number
---@param images foxel.Images
---@param rotation? foxel.Vec
function engine.cube(center, side, images, rotation) end
---@return foxel.Stats
function engine.stats() end

-- ---------------------------------------------------------------- canvas

---一帧的输入。键名："a"-"z"、"0"-"9"、"space"、"enter"、"escape"、"tab"、"backspace"、
---"left_shift"、"right_shift"、"left_ctrl"、"right_ctrl"、"left_alt"、"right_alt"、
---"up"、"down"、"left"、"right"、"slash"、"minus"、"equal"、"comma"、"period"、"grave"、
---"f1"-"f12"、"mouse_left"、"mouse_right"、"mouse_middle"
---@class foxel.Input
---@field held table<string, boolean> 这一帧按着的键
---@field pressed table<string, boolean> 这一帧刚按下的键
---@field mouse_dx number 鼠标移动（像素），x向右、y向下为正
---@field mouse_dy number
---@field mouse_x number 鼠标在窗口里的位置
---@field mouse_y number
---@field wheel number 滚轮，向上为正
---@field text string 这一帧输入的文字

---窗口、2D绘制、输入。每帧：canvas.clean() -> 画 -> canvas.draw()
canvas = {}
---@return boolean
function canvas.should_close() end
---窗口大小（像素）
---@return integer w, integer h
function canvas.size() end
---上一帧用了多少秒
---@return number
function canvas.frame_time() end
---清掉这一帧添加的东西（网格不会删）
function canvas.clean() end
---画出这一帧，然后读下一帧的输入
function canvas.draw() end
---@param c foxel.Color
function canvas.set_background(c) end
---雾：离相机start以内没有雾，到end完全变成颜色c
---@param c foxel.Color
function canvas.set_fog(c, start, finish) end
---视角（角度制）：横向、纵向、倾斜（顺时针为正）。auto（默认true）时纵向按窗口宽高比算
---@param horizontal number
---@param vertical? number
---@param tilt? number
---@param auto? boolean
function canvas.view_angle(horizontal, vertical, tilt, auto) end
---文字，(x, y)是左上角，字号20，只支持ASCII，可以用\n换行。颜色默认白色
---@param text string
---@param color? foxel.Color
function canvas.text(text, x, y, color) end
---文字画出来有多大
---@return number w, number h
function canvas.measure(text) end
---@param c foxel.Color
function canvas.line(x1, y1, x2, y2, c) end
---实心矩形
---@param c foxel.Color
function canvas.rect(x, y, w, h, c) end
---@param img foxel.Image
function canvas.image(img, x, y, w, h) end
---这一帧的输入
---@return foxel.Input
function canvas.input() end
---锁定鼠标：光标隐藏，鼠标可以一直往一个方向移动
---@param locked boolean
function canvas.lock_cursor(locked) end
---@return boolean
function canvas.cursor_locked() end
---下一次canvas.draw时把画面存成png，路径相对游戏根目录
---@param path string
function canvas.screenshot(path) end
