-- 事件：游戏在这些时候通知所有登记了的函数（mod主要靠它）。
--
--   "start"  (game)              世界打开、存档读完之后，chunk还没开始加载
--   "update" (game, input, dt)   每一帧，玩家移动之后、画之前
--   "hud"    (game)              画完游戏自己的界面之后，在上面画东西
--   "break"  (game, cube, block) 玩家要挖掉cube之前，返回false取消
--   "place"  (game, cube, block) 玩家要在cube放方块之前，返回false取消
--   "save"   (game)              存档之前，可以把数据写进game.mod_data
--
-- 一个函数出错时打印出来，不影响别的函数和游戏本身。

local events = {}

local handlers = {}

-- 登记fn，按登记的顺序调用
function events.on(name, fn)
    handlers[name] = handlers[name] or {}
    table.insert(handlers[name], fn)
end

local function call(name, fn, ...)
    local ok, result = pcall(fn, ...)
    if not ok then
        print(("foxel: 事件%s的处理函数出错了：%s"):format(name, result))
        return nil
    end
    return result
end

-- 通知所有函数
function events.emit(name, ...)
    for _, fn in ipairs(handlers[name] or {}) do
        call(name, fn, ...)
    end
end

-- 问所有函数行不行：有一个返回false就是不行（出错不算反对）
function events.allow(name, ...)
    for _, fn in ipairs(handlers[name] or {}) do
        if call(name, fn, ...) == false then
            return false
        end
    end
    return true
end

return events
