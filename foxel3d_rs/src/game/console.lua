-- 终端：按`打开，输入Lua代码回车执行。
--
-- 能直接用所有API（world、block……），另外有这些快捷的东西：
--   G、P、B、V、events   game、player、blocks、vec、events模块
--   here()              玩家脚下的位置
--   tp(x, y, z)         传送
--   give(b)             把方块放进选中的物品栏格子
--   clear()             清屏
--   help()              显示这些说明
--
-- 写表达式会显示结果（比如输入 G.player.pos）。
-- 每一行是单独执行的，local变量下一行就没了；要留着的东西写成全局变量（x = 1），
-- 它们放在终端自己的环境里，不会影响游戏。

local console = {
    open = false,
    -- 最多保留多少行输出、多少条历史
    MAX_LINES = 200,
    MAX_HISTORY = 50,
}

local LINE_HEIGHT = 22
local PANEL = { 10, 12, 16, 225 }
local COLORS = {
    input = { 200, 200, 200 },
    output = { 255, 255, 255 },
    result = { 140, 220, 255 },
    error = { 255, 110, 110 },
}

local lines = {}   -- {text=, kind=}
local history = {}
local text = ""    -- 正在输入的
local browsing     -- 正在看第几条历史，nil表示没在看
local backspace_timer = 0
local env

local function add_line(s, kind)
    for line in (tostring(s) .. "\n"):gmatch("(.-)\n") do
        table.insert(lines, { text = line, kind = kind or "output" })
    end
    while #lines > console.MAX_LINES do
        table.remove(lines, 1)
    end
end

-- 把一个值变成好读的字符串，table只展开一层
local function show(v)
    if type(v) == "string" then
        return ("%q"):format(v)
    end
    if type(v) ~= "table" then
        return tostring(v)
    end
    local keys = {}
    for k in pairs(v) do
        table.insert(keys, k)
    end
    table.sort(keys, function(a, b)
        if type(a) == type(b) and (type(a) == "number" or type(a) == "string") then
            return a < b
        end
        return type(a) < type(b)
    end)
    local parts = {}
    for i, k in ipairs(keys) do
        if i > 12 then
            table.insert(parts, ("... (%d more)"):format(#keys - 12))
            break
        end
        local value = v[k]
        local vs = type(value) == "table" and "{...}" or show(value)
        if math.type(k) == "integer" and k == i then
            table.insert(parts, vs)
        else
            table.insert(parts, ("%s = %s"):format(type(k) == "string" and k or "[" .. show(k) .. "]", vs))
        end
    end
    return "{ " .. table.concat(parts, ", ") .. " }"
end

local function make_env(G)
    local e = {
        G = G,
        P = require("player"),
        B = require("blocks"),
        V = require("vec"),
        events = require("events"),
        show = show,
    }
    -- print写到终端里（也照样打印到命令行）
    function e.print(...)
        local parts = {}
        for i = 1, select("#", ...) do
            parts[i] = tostring((select(i, ...)))
        end
        add_line(table.concat(parts, "  "))
        print(...)
    end
    function e.here()
        return G.player.pos
    end
    function e.tp(x, y, z)
        G.player.pos = { x = x, y = y, z = z }
        G.player.vel = { x = 0, y = 0, z = 0 }
    end
    function e.give(b)
        G.hotbar[G.selected] = type(b) == "string" and assert(block.id(b), "没有这个方块") or b
    end
    function e.clear()
        lines = {}
    end
    function e.help()
        add_line("G P B V events: game, player, blocks, vec, events")
        add_line("here()  tp(x, y, z)  give(block)  clear()  show(v)")
        add_line("expressions print their value; use globals (x = 1) to keep values")
    end
    return setmetatable(e, { __index = _G })
end

-- 执行一行代码：先当成表达式试一下，不行再当成语句
local function run(G, code)
    add_line("> " .. code, "input")
    env = env or make_env(G)
    local chunk = load("return " .. code, "=console", "t", env)
    local is_expr = chunk ~= nil
    if not chunk then
        local err
        chunk, err = load(code, "=console", "t", env)
        if not chunk then
            add_line(err, "error")
            return
        end
    end
    local result = table.pack(pcall(chunk))
    if not result[1] then
        add_line(tostring(result[2]), "error")
        return
    end
    if is_expr and result.n > 1 then
        local parts = {}
        for i = 2, result.n do
            table.insert(parts, show(result[i]))
        end
        add_line(table.concat(parts, ", "), "result")
    end
end

local function remove_last_char(s)
    local n = utf8.offset(s, -1)
    return n and s:sub(1, n - 1) or ""
end

-- 处理这一帧的输入。返回true表示终端用掉了输入，游戏不要再处理
function console.handle(G, input, dt)
    local pressed = input.pressed
    if not console.open then
        if pressed.grave and not G.loading then
            console.open = true
            browsing = nil
            return true
        end
        return false
    end
    if pressed.grave or pressed.escape then
        console.open = false
        return true
    end

    -- 打字。`本身也会作为文字输进来，去掉
    local typed = (input.text or ""):gsub("[`·]", "")
    text = text .. typed

    -- 退格：按住时0.4秒后开始连续删
    if pressed.backspace then
        text = remove_last_char(text)
        backspace_timer = -0.4
    elseif input.held.backspace then
        backspace_timer = backspace_timer + dt
        while backspace_timer > 0 do
            text = remove_last_char(text)
            backspace_timer = backspace_timer - 0.04
        end
    end

    -- 上下翻历史
    if pressed.up and #history > 0 then
        browsing = math.max(1, (browsing or #history + 1) - 1)
        text = history[browsing]
    elseif pressed.down and browsing then
        browsing = browsing + 1
        if browsing > #history then
            browsing, text = nil, ""
        else
            text = history[browsing]
        end
    end

    if pressed.enter and text:match("%S") then
        if history[#history] ~= text then
            table.insert(history, text)
            if #history > console.MAX_HISTORY then
                table.remove(history, 1)
            end
        end
        local code = text
        text, browsing = "", nil
        run(G, code)
    end
    return true
end

-- 直接执行一行代码（测试、mod用）
function console.exec(G, code)
    run(G, code)
end

-- 最近的n行输出
function console.lines(n)
    local out = {}
    for i = math.max(1, #lines - n + 1), #lines do
        table.insert(out, lines[i].text)
    end
    return out
end

function console.draw(G)
    if not console.open then
        return
    end
    local sw, sh = canvas.size()
    local h = math.floor(sh * 0.45)
    canvas.rect(0, 0, sw, h, PANEL)
    -- 输入行在最下面，光标一闪一闪
    local cursor = (G.time * 2) % 2 < 1 and "_" or ""
    canvas.text("> " .. text .. cursor, 10, h - LINE_HEIGHT - 6, COLORS.output)
    canvas.line(0, h - LINE_HEIGHT - 12, sw, h - LINE_HEIGHT - 12, { 255, 255, 255, 60 })
    -- 输出从下往上画，画到面板顶为止
    local y = h - 2 * LINE_HEIGHT - 16
    for i = #lines, 1, -1 do
        if y < 4 then
            break
        end
        canvas.text(lines[i].text, 10, y, COLORS[lines[i].kind])
        y = y - LINE_HEIGHT
    end
end

return console
