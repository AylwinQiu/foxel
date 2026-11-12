-- 终端：`打开，打字，回车执行，Esc关闭

local H = dofile(TEST_DIR .. "/helpers.lua")
local G, B = H.G, H.B
local console = require("console")

H.start(5)

-- 输入一行并回车
local function type_line(s)
    H.frames(1, {}, {}, s)
    H.frames(1, {}, { "enter" })
end
local function last()
    return console.lines(1)[1]
end

-- 打开：`本身那个字符不会进到输入框里
H.frames(1, {}, { "grave" }, "`")
assert(console.open and not G.playing(), "open")

-- 表达式显示结果
type_line("1 + 2")
assert(last() == "3", last())
type_line("block.name(B.STONE)")
assert(last() == '"foxel:stone"', last())

-- 语句：全局变量留到下一行，不会进真正的_G
type_line("x = 40")
type_line("x + 2")
assert(last() == "42" and rawget(_G, "x") == nil, last())

-- print写到终端里
type_line("print('hi', 7)")
assert(last() == "hi  7", last())

-- 错误显示出来，游戏不受影响
type_line("nil + 1")
assert(last():match("arithmetic"), last())
type_line("this is not lua")
assert(last():match("console"), last())

-- 改游戏：传送、放方块
type_line("tp(3.5, 80, 3.5)")
H.near(G.player.pos.y, 80, 0.1, "tp")
type_line("give('foxel:bricks')")
assert(G.hotbar[G.selected] == B.BRICKS, "give")

-- 退格（中文按字符删）、历史
H.frames(1, {}, {}, "1+1你")
H.frames(1, {}, { "backspace" })
H.frames(1, {}, { "enter" })
assert(last() == "2", last())
H.frames(1, {}, { "up" })
H.frames(1, {}, { "enter" })
assert(last() == "2", "history")

-- 开着终端时按W不会走
G.player.flying = true
local z = G.player.pos.z
H.frames(10, { "w" })
assert(G.player.pos.z == z, "moved while console open")

-- Esc关闭终端，不会暂停游戏
H.frames(1, {}, { "escape" })
assert(not console.open and canvas.cursor_locked() and G.playing(), "close")
H.frames(10, { "w" })
assert(G.player.pos.z < z, "can't move after close")
