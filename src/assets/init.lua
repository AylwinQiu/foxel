print("zzzzzekjhwfkdjhkdsfsd")

local canvas = rust.lua_new_raylibcanvas(800, 600, true)

-- 指令只创建一次，每帧重复使用。
local color = rust.lua_new_canvascolor(200, 30, 30, 255)
local line = rust.lua_new_drawins_line(
    rust.lua_new_canvas2d(100, 100),
    rust.lua_new_canvas2d(700, 500),
    color
)

while not rust.lua_raylibcanvas_should_close(canvas) do
    -- draw 会让旧句柄失效，必须接住返回的新句柄。
    canvas = rust.lua_raylibcanvas_draw(canvas, line)
end
