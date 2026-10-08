---@class CellID
---@field a Int
---@field b Int
---@field c Int
---@field d Int

--- 这个函数返回的是一个cell的id
---@param a Int
---@param b Int
---@param c Int
---@param d Int
---@return CellID
local CellID = function(a, b, c, d)
    return tostring(a)..";"..tostring(b)..";"..tostring(c)..";"..tostring(d)
end
---四叉树算法
---@class TreePosition2d
---@field x Int[]
---@field y Int[]

local TreePosition2d = function(x, y) 
    return {
        x = x,
        y = y
    }
end

---定义物理坐标
---@class PhyPosition2d
---@field x Float
---@field y Float
---物理坐标
---@param x Float
---@param y Float
---@return PhyPosition2d
local PhyPosition2d = function(x, y)
    return {
        x=x, y=y
    }
end

---定义旋转
---@class Rotation
---@field angle Float
---旋转角度（角度制）
---@param angle Float
---@return Rotation
local Rotation = function(angle)
    return {
        angle = angle
    }
end 


--- 新建一个world模型，里面保存有所有的field的状态
local World = function()
    return {
        status = {},
        fields = {
            ---从id获得树坐标
            ---@param id CellID
            ---@return TreePosition2d|nil
            tree_position = function(id)
                -- todo
            end,
            ---从树坐标获得回id
            ---@param treepos TreePosition2d
            ---@return CellID|nil
            tree_position_rev = function(treepos)
                --todo
            end,
            ---从id获得rotation.
            ---@param id CellID
            ---@return Rotation | nil
            rotation = function(id)
                --todo
            end,
            ---从id获得
        },
    }
end


--- 位置场：通过id获得cell的位置信息



return {
    World = World,

}
