-- 3D向量{x=, y=, z=}的计算

local V = {}

function V.new(x, y, z)
    return { x = x or 0, y = y or 0, z = z or 0 }
end

function V.add(a, b)
    return { x = a.x + b.x, y = a.y + b.y, z = a.z + b.z }
end

function V.sub(a, b)
    return { x = a.x - b.x, y = a.y - b.y, z = a.z - b.z }
end

function V.scale(a, k)
    return { x = a.x * k, y = a.y * k, z = a.z * k }
end

function V.length(a)
    return math.sqrt(a.x * a.x + a.y * a.y + a.z * a.z)
end

function V.normalize(a)
    local l = V.length(a)
    if l == 0 then
        return V.new()
    end
    return V.scale(a, 1 / l)
end

function V.copy(a)
    return { x = a.x, y = a.y, z = a.z }
end

return V
