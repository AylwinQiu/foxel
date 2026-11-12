-- 游戏本身：状态、一帧里做什么、挖和放、存档。
--
-- require("game")拿到的是同一个表G，mod可以直接读写里面的东西：
--   G.player       玩家（见player.lua）
--   G.hotbar       物品栏，9个BlockId，G.selected是选中的第几格（1到9）
--   G.build_depth  挖和放的格子深度：0是1格，-1是1/2格……-4是1/16格
--   G.target       现在指着的方块（world.raycast的结果），没有时是nil
--   G.mod_data     mod的数据，跟着存档保存，key请用"mod名:名字"
--   G.time         游戏开始后过了多少秒

local B = require("blocks")
local P = require("player")
local V = require("vec")
local events = require("events")
local hud = require("hud")
local console = require("console")

local G = {
    HOTBAR_SIZE = 9,
    -- 多久自动存档一次（秒）
    AUTOSAVE_INTERVAL = 300,
    -- 按住鼠标时多久挖、放一次（秒）
    REPEAT_INTERVAL = 0.22,

    player = nil,
    hotbar = {},
    selected = 1,
    build_depth = 0,
    target = nil,
    mod_data = {},
    time = 0,

    loading = true,
    inventory_open = false,
    show_debug = false,
    show_help = false,
    message = nil,
    message_until = 0,
    cooldown = 0,
    -- 刚点击回到游戏：松开鼠标左键之前不挖、不放
    wait_release = false,
    last_save = 0,
    input = {},
}

-- block.get的结果不会变（世界打开以后不能再注册方块），缓存起来
local infos = {}
function G.info(b)
    local i = infos[b]
    if not i then
        i = block.get(b)
        infos[b] = i
    end
    return i
end

-- 在屏幕上方显示一条消息，几秒后消失（只能用ASCII）
function G.show_message(text)
    G.message = text
    G.message_until = G.time + 2.5
end

-- 玩家正在玩：加载完了、鼠标锁定、没有打开物品栏
function G.playing()
    return not G.loading and not G.inventory_open and not console.open and canvas.cursor_locked()
end

-- ---------------------------------------------------------------- 存档

function G.save()
    events.emit("save", G)
    local hotbar = {}
    for i = 1, G.HOTBAR_SIZE do
        hotbar[i] = block.name(G.hotbar[i] or block.AIR)
    end
    world.save {
        player = P.save(G.player),
        hotbar = hotbar,
        selected = G.selected,
        build_depth = G.build_depth,
        mod_data = G.mod_data,
    }
    G.last_save = G.time
end

local function load_meta(meta)
    G.player = P.load(meta.player)
    for i = 1, G.HOTBAR_SIZE do
        G.hotbar[i] = block.id(meta.hotbar[i] or "") or block.AIR
    end
    G.selected = math.max(1, math.min(G.HOTBAR_SIZE, meta.selected or 1))
    G.build_depth = math.max(cube.MIN_DEPTH, math.min(G.MAX_BUILD_DEPTH, meta.build_depth or 0))
    G.mod_data = meta.mod_data or {}
end

-- ---------------------------------------------------------------- 挖和放

-- 指着的方块里要挖掉的格子（build_depth大小）
function G.target_cell()
    local t = G.target
    if not t then
        return nil
    end
    return cube.at(V.sub(t.point, V.scale(t.normal, 1e-4)), G.build_depth)
end

-- 指着的那个面外面要放方块的格子（build_depth大小）
function G.place_cell()
    local t = G.target
    if not t or (t.normal.x == 0 and t.normal.y == 0 and t.normal.z == 0) then
        return nil
    end
    return cube.at(V.add(t.point, V.scale(t.normal, cube.side(G.build_depth) / 2)), G.build_depth)
end

-- c这个格子里有没有挖不掉的方块（没加载也算）
local function contains_unbreakable(c)
    local b, _, split = world.find(c)
    if b == nil then
        return true
    end
    if not split then
        return G.info(b).unbreakable
    end
    for o = 0, 7 do
        if contains_unbreakable(cube.child(c, o)) then
            return true
        end
    end
    return false
end

-- 像玩家一样挖掉c（里面的东西全部变成空气）。挖不掉、mod不让挖时返回false
function G.break_cube(c)
    if contains_unbreakable(c) then
        return false
    end
    local b = G.target and G.target.block or block.AIR
    if not events.allow("break", G, c, b) then
        return false
    end
    return world.set_cube(c, block.AIR)
end

-- 像玩家一样在c放方块b。格子不是全空的、会卡住玩家、mod不让放时返回false
function G.place(c, b)
    if b == block.AIR then
        return false
    end
    local here, _, split = world.find(c)
    if here ~= block.AIR or split then
        return false
    end
    if G.info(b).solid then
        local lo, hi = P.box(G.player.pos, P.height(G.player))
        local clo, chi = cube.min(c), cube.max(c)
        if lo.x < chi.x and hi.x > clo.x and lo.y < chi.y and hi.y > clo.y and lo.z < chi.z and hi.z > clo.z then
            return false
        end
    end
    if not events.allow("place", G, c, b) then
        return false
    end
    return world.set_cube(c, b)
end

-- (x, z)这一列从y往下数第一个实心方块上面的空格子的y，找不到时返回nil
function G.surface_below(x, y, z, max_depth)
    for d = 0, max_depth - 1 do
        local b, split = world.get(x, y - d, z)
        if b == nil then
            return nil
        end
        if split or b ~= block.AIR then
            return y - d + 1
        end
    end
    return nil
end

local function interact(input, dt)
    G.cooldown = G.cooldown - dt
    if G.wait_release then
        if input.held.mouse_left then
            return
        end
        G.wait_release = false
    end
    if not G.target then
        return
    end
    local function act(k)
        return input.pressed[k] or (input.held[k] and G.cooldown <= 0)
    end
    if act("mouse_left") then
        local c = G.target_cell()
        if c then
            G.break_cube(c)
        end
        G.cooldown = G.REPEAT_INTERVAL
    elseif act("mouse_right") then
        local c = G.place_cell()
        if c then
            G.place(c, G.hotbar[G.selected])
        end
        G.cooldown = G.REPEAT_INTERVAL
    elseif input.pressed.mouse_middle then
        local b = G.target.block
        if not G.info(b).hidden then
            G.hotbar[G.selected] = b
        end
    end
end

-- ---------------------------------------------------------------- 一帧

-- 挖和放的格子深度：0是1格，-1是1/2格……最小是cube.MIN_DEPTH（-4，1/16格）
G.MAX_BUILD_DEPTH = 0

function G.depth_name(d)
    return ("Depth %d"):format(d)
end

-- 改挖和放的深度，超出范围时停在边上
function G.set_build_depth(d)
    d = math.max(cube.MIN_DEPTH, math.min(G.MAX_BUILD_DEPTH, d))
    if d ~= G.build_depth then
        G.build_depth = d
        G.show_message(G.depth_name(d))
    end
end

local function update(dt)
    local input = G.input
    local pressed = input.pressed
    local locked = canvas.cursor_locked()

    if pressed.tab then
        G.inventory_open = not G.inventory_open
        canvas.lock_cursor(not G.inventory_open)
    elseif pressed.escape then
        G.inventory_open = false
        canvas.lock_cursor(false)
    elseif not locked and not G.inventory_open and pressed.mouse_left then
        -- 点一下回到游戏，这次点击不算挖方块
        canvas.lock_cursor(true)
        G.wait_release = true
    end
    if pressed.f3 then
        G.show_debug = not G.show_debug
    end
    if pressed.h then
        G.show_help = not G.show_help
    end
    if pressed.f5 then
        local ok, err = pcall(G.save)
        G.show_message(ok and "World saved" or ("Save failed: " .. tostring(err)))
    end
    if pressed.f2 then
        local name = ("screenshots/%s.png"):format(os.date("%Y%m%d-%H%M%S"))
        canvas.screenshot(name)
        G.show_message("Screenshot " .. name)
    end
    if pressed.f then
        G.player.flying = not G.player.flying
        G.player.vel = V.new()
        G.show_message(G.player.flying and "Flying" or "Walking")
    end
    if pressed.r then
        -- 往小切，切到最小以后回到最大
        G.set_build_depth(G.build_depth > cube.MIN_DEPTH and G.build_depth - 1 or G.MAX_BUILD_DEPTH)
    end
    for i = 1, G.HOTBAR_SIZE do
        if pressed[tostring(i)] then
            G.selected = i
        end
    end
    if input.wheel ~= 0 and not G.inventory_open then
        local step = input.wheel > 0 and -1 or 1
        G.selected = (G.selected - 1 + step) % G.HOTBAR_SIZE + 1
    end

    P.update(G.player, input, dt, G.playing())
    G.target = world.raycast(P.eye(G.player), P.look_dir(G.player), P.REACH)
    if G.playing() then
        interact(input, dt)
    end
    if G.inventory_open then
        hud.inventory_click(G, input)
    end
    events.emit("update", G, input, dt)
    if G.time - G.last_save > G.AUTOSAVE_INTERVAL then
        local ok, err = pcall(G.save)
        if not ok then
            G.show_message("Autosave failed: " .. tostring(err))
        end
    end
end

-- 运行一帧：处理输入、移动玩家、加载世界、画出来
function G.frame()
    local dt = math.min(canvas.frame_time(), 0.05)
    G.time = G.time + dt
    G.input = canvas.input()
    -- 终端开着时输入都给它，游戏收到的是空的输入
    if console.handle(G, G.input, dt) then
        G.input = { held = {}, pressed = {}, mouse_dx = 0, mouse_dy = 0,
            mouse_x = G.input.mouse_x, mouse_y = G.input.mouse_y, wheel = 0, text = "" }
    end

    if G.loading then
        -- 刚开始时多花点时间加载，加载好了再让玩家动
        engine.set_camera(P.eye(G.player), P.look_dir(G.player))
        engine.update(50)
        if engine.ready(2) then
            G.loading = false
            P.unstuck(G.player)
        end
    else
        update(dt)
        engine.set_camera(P.eye(G.player), P.look_dir(G.player))
        engine.update(8)
    end

    canvas.clean()
    engine.draw_world()
    hud.draw(G)
    events.emit("hud", G)
    console.draw(G)
    canvas.draw()
end

-- 打开（或新建）世界。config：{world = 存档名, seed = 新世界的种子, view = 视距}
function G.start(config)
    engine.set_view_distance(config.view or 8)
    local meta = world.open { save = config.world or "default", seed = config.seed }
    for i = 1, G.HOTBAR_SIZE do
        G.hotbar[i] = B.hotbar[i] or block.AIR
    end
    if meta then
        load_meta(meta)
    else
        local h = terrain.column(0, 0)
        G.player = P.new(V.new(0.5, h + 1, 0.5))
    end

    local sky = { 140, 190, 255 }
    local far = engine.view_distance() * cube.CHUNK_SIZE
    canvas.set_background(sky)
    canvas.set_fog(sky, far * 0.55, far - 4)
    canvas.view_angle(100)
    events.emit("start", G)
end

-- 一直运行到窗口关闭，关闭时存档
function G.run(config)
    G.start(config)
    while not canvas.should_close() do
        G.frame()
    end
    G.save()
end

return G
