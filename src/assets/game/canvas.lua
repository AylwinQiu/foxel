--- 这个是lua的屏幕接口定义，定义了lua如何和屏幕交互，他们都是纯函数。

---@class Canvas

---@alias Int number
---@alias Float number

---@class Canvas2d
---@field x number
---@field y number

---这个返回屏幕的坐标。
---@param x Float
---@param y Float
---@return Canvas2d
local Canvas2d = function(x, y) 
    return {x=x, y=y}
end

---@class Color
---@field r Int
---@field g Int
---@field b Int
---@field a Int

---返回颜色
---@param r Int
---@param g Int
---@param b Int
---@param a Int
---@return Color
local Color = function(r,g,b,a)
    return {r=r,g=g,b=b,a=a}
end

--- 这个函数负责Canvas句柄的生成，包括设置窗口宽长和是否允许调节大小。
--- @param x number 
--- @param y number
--- @param resizable boolean
--- @return Canvas
local Canvas = function (x, y, resizable) 
    -- todo
    return {

    }
end

---添加一个方块到canvas等待绘制。
---@param canvas Canvas
---@param side number
---@param center Canvas2d
---@param color Color
---@param rotation number
local add_square = function(canvas, side, center, color, rotation)
    -- todo
end

---添加一个线到canvas等待绘制
---@param canvas Canvas
---@param start_pt Canvas2d
---@param end_pt Canvas2d
---@param color Color
local add_line = function(canvas, start_pt, end_pt, color)

end
---添加一个文字到canvas等待绘制
---@param canvas Canvas
---@param text string
---@param center Canvas2d
---@param color Color
---@param font string
local add_text = function(canvas, text, center, color, font)
    --
end

---清除屏幕缓存。
---@param canvas Canvas
local clean= function(canvas) 
    --todo
end

---绘制
---@param canvas Canvas
local draw = function(canvas)
    --todo
end

---是否需要退出
---@param canvas Canvas
---@return boolean
local should_close = function(canvas) 
    --
end


return {
    Color = Color,
    Canvas = Canvas,
    add_square = add_square,
    add_line = add_line,
    add_text = add_text,
    should_close = should_close,
    draw = draw,
}