-- 测试用：加载内置内容和mod，打开世界，替换canvas.input让测试自己控制输入

require("blocks")
require("worldgen")
for _, name in ipairs(foxel.mods()) do
    require(name)
end

local H = {}
H.G = require("game")
H.B = require("blocks")

local current
function H.input(held, pressed, text, wheel)
    current = { held = {}, pressed = {}, mouse_dx = 0, mouse_dy = 0, mouse_x = 0, mouse_y = 0, wheel = wheel or 0, text = text or "" }
    for _, k in ipairs(held or {}) do
        current.held[k] = true
    end
    for _, k in ipairs(pressed or {}) do
        current.pressed[k] = true
        current.held[k] = true
    end
    return current
end
H.input()
canvas.input = function()
    return current
end

-- 用这个输入跑n帧
function H.frames(n, held, pressed, text, wheel)
    for _ = 1, n do
        H.input(held, pressed, text, wheel)
        H.G.frame()
    end
end

-- 开一个新游戏，等世界加载好
function H.start(seed)
    H.G.start { world = "test", seed = seed, view = 3 }
    local n = 0
    while H.G.loading do
        H.G.frame()
        n = n + 1
        assert(n < 1000, "loading forever")
    end
    canvas.lock_cursor(true)
end

function H.near(a, b, eps, what)
    if math.abs(a - b) > (eps or 0.01) then
        error(("%s: %s 和 %s 差太多"):format(what or "near", a, b), 2)
    end
end

return H
