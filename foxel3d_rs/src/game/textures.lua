-- 程序生成的16x16像素贴图，不需要任何图片文件。
--
-- 贴图先做成"格子"（256个颜色的数组，按行存，g[y*16+x+1]是(x, y)），这样可以在别的贴图上叠东西，
-- 最后用T.image(g)变成图片。颜色是{r, g, b, a}。
-- 随机数用foxel.rng(seed)：同一个seed永远生成同一张贴图。

local T = {}

local SIZE = 16
T.SIZE = SIZE

local function clamp(v)
    return math.max(0, math.min(255, math.floor(v + 0.5)))
end

function T.rgb(r, g, b, a)
    return { r, g, b, a or 255 }
end
local rgb = T.rgb

-- 颜色乘上k（0到1之间变暗），alpha不变
function T.scale(c, k)
    return { clamp(c[1] * k), clamp(c[2] * k), clamp(c[3] * k), c[4] }
end
local scale = T.scale

-- 亮度随机上下浮动amount
function T.vary(r, c, amount)
    return scale(c, 1 + (r:random() * 2 - 1) * amount)
end
local vary = T.vary

local TRANSPARENT = { 0, 0, 0, 0 }

-- 用f(x, y, r)生成格子，f按行从左上到右下依次调用，r是这张贴图的随机数
function T.grid(seed, f)
    local r = foxel.rng(seed)
    local g = {}
    for y = 0, SIZE - 1 do
        for x = 0, SIZE - 1 do
            g[y * SIZE + x + 1] = f(x, y, r)
        end
    end
    return g
end
local grid = T.grid

function T.image(g)
    return image.pixels(SIZE, SIZE, function(x, y)
        return g[y * SIZE + x + 1]
    end)
end

-- 一种颜色加上随机的明暗
function T.noise(seed, base, amount)
    return grid(seed, function(_, _, r)
        return vary(r, base, amount)
    end)
end

function T.stone(seed)
    return grid(seed, function(_, _, r)
        local c = vary(r, rgb(128, 128, 128), 0.07)
        if r:random() < 0.08 then
            c = scale(c, 0.82)
        end
        return c
    end)
end

-- 石头上带一些颜色的斑点
function T.ore(seed, spot)
    local g = T.stone(seed)
    local r = foxel.rng(seed * 31)
    local centers = {}
    for i = 1, 5 do
        centers[i] = { 2 + r:random() * 12, 2 + r:random() * 12 }
    end
    for y = 0, SIZE - 1 do
        for x = 0, SIZE - 1 do
            for _, c in ipairs(centers) do
                local d = math.sqrt((x - c[1]) ^ 2 + (y - c[2]) ^ 2)
                if d < 1.4 and r:random() < 0.85 then
                    g[y * SIZE + x + 1] = vary(r, spot, 0.1)
                end
            end
        end
    end
    return g
end

function T.cobblestone(seed)
    local r = foxel.rng(seed)
    local cells = {}
    for i = 1, 10 do
        cells[i] = { x = r:random() * 16, y = r:random() * 16, shade = 0.75 + r:random() * 0.4 }
    end
    -- 贴图左右、上下是接起来的，所以距离按环绕算
    local function dist(ax, ay, bx, by)
        local dx = math.min(math.abs(ax - bx), 16 - math.abs(ax - bx))
        local dy = math.min(math.abs(ay - by), 16 - math.abs(ay - by))
        return math.sqrt(dx * dx + dy * dy)
    end
    return grid(seed, function(x, y, rr)
        local px, py = x + 0.5, y + 0.5
        local d1, d2, best = 99, 99, 1
        for i, c in ipairs(cells) do
            local d = dist(px, py, c.x, c.y)
            if d < d1 then
                d2, d1, best = d1, d, i
            elseif d < d2 then
                d2 = d
            end
        end
        if d2 - d1 < 1.1 then
            return vary(rr, rgb(78, 78, 78), 0.08)
        end
        return vary(rr, scale(rgb(128, 128, 128), cells[best].shade), 0.06)
    end)
end

function T.dirt(seed)
    return grid(seed, function(_, _, r)
        local c = vary(r, rgb(134, 96, 67), 0.1)
        local f = r:random()
        if f < 0.1 then
            c = scale(c, 0.75)
        elseif f > 0.95 then
            c = scale(c, 1.2)
        end
        return c
    end)
end

-- 下面是side，上面盖着top颜色的一层（草、雪），每列厚度不一样
function T.topped(seed, side, top)
    local r = foxel.rng(seed)
    local depth = {}
    for x = 0, SIZE - 1 do
        depth[x] = 2 + r:random(0, 2)
    end
    return grid(seed, function(x, y, rr)
        if y < depth[x] then
            return vary(rr, top, 0.1)
        end
        return side[y * SIZE + x + 1]
    end)
end

function T.gravel(seed)
    local palette = { rgb(120, 115, 110), rgb(150, 144, 138), rgb(95, 90, 88), rgb(170, 160, 150), rgb(130, 120, 112) }
    return grid(seed, function(_, _, r)
        return vary(r, palette[r:random(#palette)], 0.05)
    end)
end

function T.bedrock(seed)
    local palette = { rgb(40, 40, 40), rgb(85, 85, 85), rgb(20, 20, 20), rgb(120, 120, 120) }
    return grid(seed, function(_, _, r)
        return palette[r:random(#palette)]
    end)
end

-- 树皮：竖条纹
function T.bark(seed, base)
    local r = foxel.rng(seed)
    local stripe = {}
    for x = 0, SIZE - 1 do
        stripe[x] = 0.8 + r:random() * 0.3
    end
    return grid(seed, function(x, _, rr)
        return vary(rr, scale(base, stripe[x]), 0.06)
    end)
end

-- 木头的横截面：年轮，外面一圈是树皮
function T.rings(seed, bark, light, dark)
    return grid(seed, function(x, y, r)
        if x == 0 or y == 0 or x == SIZE - 1 or y == SIZE - 1 then
            return vary(r, bark, 0.08)
        end
        local d = math.sqrt((x - 7.5) ^ 2 + (y - 7.5) ^ 2)
        if math.floor(d) % 3 == 0 then
            return vary(r, dark, 0.05)
        end
        return vary(r, light, 0.05)
    end)
end

function T.planks(seed, base)
    local r = foxel.rng(seed)
    local shade = {}
    for i = 0, 3 do
        shade[i] = 0.9 + r:random() * 0.2
    end
    return grid(seed, function(x, y, rr)
        local plank = y // 4
        if y % 4 == 3 or x == (plank * 5 + 3) % SIZE then
            return scale(base, 0.6)
        end
        return vary(rr, scale(base, shade[plank]), 0.05)
    end)
end

-- 有一部分像素是透明的
function T.leaves(seed, base, holes)
    return grid(seed, function(_, _, r)
        if r:random() < holes then
            return TRANSPARENT
        end
        return vary(r, base, 0.2)
    end)
end

function T.glass()
    local frame = rgb(215, 235, 240)
    return grid(0, function(x, y)
        local edge = x == 0 or y == 0 or x == SIZE - 1 or y == SIZE - 1
        local shine = (x + y == 6 or x + y == 7) and x >= 2 and x <= 5
        if edge or shine then
            return frame
        end
        return TRANSPARENT
    end)
end

-- 砖墙：w×h的砖，每行错开半块
function T.bricks(seed, brick, mortar, w, h)
    return grid(seed, function(x, y, r)
        local offset = (y // h) % 2 * (w // 2)
        if y % h == h - 1 or (x + offset) % w == w - 1 then
            return vary(r, mortar, 0.05)
        end
        return vary(r, brick, 0.08)
    end)
end

-- 横条纹，比如砂岩
function T.layered(seed, base)
    return grid(seed, function(_, y, r)
        local c = vary(r, base, 0.04)
        if y % 5 == 4 then
            c = scale(c, 0.88)
        end
        return c
    end)
end

return T
