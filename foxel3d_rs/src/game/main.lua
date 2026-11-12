-- 游戏入口。Rust开好窗口、装好API（见API.md）以后运行这个文件。
--
--   cargo run --release -- --world 名字 --seed 123 --view 8
--
-- 顺序很重要：方块和地形要在world.open之前登记完，所以先加载内置内容，再加载mod，最后打开世界。

-- 命令行参数：--world 存档名（默认default）、--seed 新世界的种子、--view 视距（chunk数，默认8）
local function parse_args(args)
    local config = { world = "default", view = 8 }
    local i = 1
    while i <= #args do
        local key, value = args[i]:match("^%-%-(%w+)$"), args[i + 1]
        if key == "world" then
            config.world = value
        elseif key == "seed" or key == "view" then
            config[key] = math.tointeger(tonumber(value)) or error(("--%s要是整数"):format(key))
        else
            error("不认识的参数：" .. args[i])
        end
        i = i + 2
    end
    return config
end

local config = parse_args(arg or {})

-- 内置内容：方块、地形
require("blocks")
require("worldgen")

-- mod：mods/下面每个有init.lua的文件夹，按名字顺序加载
for _, name in ipairs(foxel.mods()) do
    require(name)
end

require("game").run(config)
