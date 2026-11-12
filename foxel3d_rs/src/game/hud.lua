-- 界面：准星、物品栏、帮助、调试信息、消息、背包、暂停。所有函数的第一个参数是game表

local hud = {}

local PANEL = { 0, 0, 0, 130 }
local HIGHLIGHT = { 255, 255, 255, 230 }
local WHITE = { 255, 255, 255 }
local SLOT = 52
local INV_COLS = 9

local HELP = [[WASD     move            Space   jump / fly up
Mouse    look            Shift   crouch / fly down
Left     break block     W W     sprint (or Ctrl)
Right    place block     F       toggle flying
Middle   pick block      R       depth 0 .. -4
1-9      select slot     Tab     all blocks
Wheel    select slot     F5      save
F3       debug info      Esc     pause
F2       screenshot      `       Lua console
H        hide this help]]

-- 文字加一个半透明的底
function hud.text_box(text, x, y)
    local w, h = canvas.measure(text)
    canvas.rect(x - 6, y - 4, w + 12, h + 8, PANEL)
    canvas.text(text, x, y, WHITE)
end

-- 屏幕中间横着居中的文字
function hud.centered_text(text, y)
    local sw = canvas.size()
    local w = canvas.measure(text)
    hud.text_box(text, (sw - w) / 2, y)
end

-- 空心的框
function hud.frame(x, y, w, h, width, color)
    canvas.rect(x, y, w, width, color)
    canvas.rect(x, y + h - width, w, width, color)
    canvas.rect(x, y, width, h, color)
    canvas.rect(x + w - width, y, width, h, color)
end

local function icon(G, b, x, y)
    local img = G.info(b).icon
    if img then
        canvas.image(img, x + 8, y + 8, SLOT - 16, SLOT - 16)
    end
end

-- 指着的格子画一个框
local function draw_target(G)
    if not G.playing() then
        return
    end
    local c = G.target_cell()
    if not c then
        return
    end
    local lo, hi = cube.min(c), cube.max(c)
    local e = 0.003
    engine.box({ x = lo.x - e, y = lo.y - e, z = lo.z - e }, { x = hi.x + e, y = hi.y + e, z = hi.z + e }, { 20, 20, 20 })
end

local function draw_hotbar(G)
    local sw, sh = canvas.size()
    local x0 = (sw - SLOT * G.HOTBAR_SIZE) / 2
    local y0 = sh - SLOT - 14
    canvas.rect(x0 - 4, y0 - 4, SLOT * G.HOTBAR_SIZE + 8, SLOT + 8, PANEL)
    for i = 1, G.HOTBAR_SIZE do
        local x = x0 + (i - 1) * SLOT
        if i == G.selected then
            hud.frame(x, y0, SLOT, SLOT, 3, HIGHLIGHT)
        end
        local b = G.hotbar[i]
        if b and b ~= block.AIR then
            icon(G, b, x, y0)
        end
    end
    hud.centered_text(G.info(G.hotbar[G.selected] or block.AIR).display, y0 - 40)
    hud.text_box(G.depth_name(G.build_depth), x0 + SLOT * G.HOTBAR_SIZE + 18, y0 + SLOT / 2 - 10)
end

local function debug_text(G)
    local p = G.player.pos
    local chunk = cube.at(p, cube.CHUNK_DEPTH)
    local _, biome = terrain.column(math.floor(p.x), math.floor(p.z))
    local st = engine.stats()
    local lines = {
        ("FPS %.0f"):format(1 / math.max(canvas.frame_time(), 1e-6)),
        ("XYZ %.2f %.2f %.2f"):format(p.x, p.y, p.z),
        ("Chunk %d %d %d"):format(chunk.x, chunk.y, chunk.z),
        ("Biome %s"):format(biome),
        ("Chunks loaded %d, meshes %d, drawn %d, pending %d"):format(st.loaded, st.meshes, st.drawn, st.pending),
        ("Seed %d"):format(world.seed()),
    }
    if G.target then
        local c = G.target.cube
        table.insert(lines, ("Looking at %s (%d, %d, %d)@%d"):format(G.info(G.target.block).name, c.x, c.y, c.z, c.depth))
    end
    return table.concat(lines, "\n")
end

-- 背包里的方块：所有不隐藏的方块
local inventory_cache
local function inventory_blocks(G)
    if not inventory_cache then
        inventory_cache = {}
        for _, b in ipairs(block.all()) do
            if not G.info(b).hidden then
                table.insert(inventory_cache, b)
            end
        end
    end
    return inventory_cache
end

local function inventory_origin(n)
    local sw, sh = canvas.size()
    local rows = (n + INV_COLS - 1) // INV_COLS
    return (sw - SLOT * INV_COLS) / 2, (sh - SLOT * rows) / 2 - 40
end

-- 鼠标指着背包里的第几个方块，没有时是nil
local function inventory_hover(input, n)
    local ox, oy = inventory_origin(n)
    local mx, my = input.mouse_x or 0, input.mouse_y or 0
    if mx < ox or my < oy then
        return nil
    end
    local col, row = (mx - ox) // SLOT, (my - oy) // SLOT
    if col >= INV_COLS then
        return nil
    end
    local i = math.floor(row * INV_COLS + col) + 1
    if i <= n then
        return i
    end
    return nil
end

local function draw_inventory(G)
    local sw, sh = canvas.size()
    canvas.rect(0, 0, sw, sh, { 0, 0, 0, 120 })
    local blocks = inventory_blocks(G)
    local ox, oy = inventory_origin(#blocks)
    local rows = (#blocks + INV_COLS - 1) // INV_COLS
    canvas.rect(ox - 8, oy - 8, SLOT * INV_COLS + 16, SLOT * rows + 16, PANEL)
    hud.centered_text("Click a block to put it in the selected slot (Tab to close)", oy - 50)
    local hover = inventory_hover(G.input, #blocks)
    for i, b in ipairs(blocks) do
        local x = ox + ((i - 1) % INV_COLS) * SLOT
        local y = oy + ((i - 1) // INV_COLS) * SLOT
        if i == hover then
            hud.frame(x, y, SLOT, SLOT, 2, HIGHLIGHT)
        end
        icon(G, b, x, y)
    end
    if hover then
        hud.text_box(G.info(blocks[hover]).display, G.input.mouse_x + 16, G.input.mouse_y + 16)
    end
end

-- 背包打开时点了一个方块：放进选中的那一格
function hud.inventory_click(G, input)
    if not input.pressed.mouse_left then
        return
    end
    local blocks = inventory_blocks(G)
    local i = inventory_hover(input, #blocks)
    if i then
        G.hotbar[G.selected] = blocks[i]
    end
end

function hud.draw(G)
    local sw, sh = canvas.size()
    if G.loading then
        canvas.rect(0, 0, sw, sh, { 40, 60, 90 })
        hud.centered_text("Generating world...", sh / 2 - 10)
        return
    end
    draw_target(G)

    -- 准星
    local cx, cy = sw / 2, sh / 2
    canvas.rect(cx - 9, cy - 1, 18, 2, HIGHLIGHT)
    canvas.rect(cx - 1, cy - 9, 2, 18, HIGHLIGHT)

    draw_hotbar(G)

    if G.show_debug then
        hud.text_box(debug_text(G), 12, 12)
    elseif G.show_help then
        hud.text_box(HELP, 12, 12)
    else
        hud.text_box("H: help", 12, 12)
    end
    if G.message and G.time < G.message_until then
        hud.centered_text(G.message, 60)
    end

    if G.inventory_open then
        draw_inventory(G)
    elseif not canvas.cursor_locked() then
        canvas.rect(0, 0, sw, sh, { 0, 0, 0, 90 })
        hud.centered_text("Paused - click to play", sh / 2 - 40)
    end
end

return hud
