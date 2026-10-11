import raylib as rl

class Canvas1D:
    def __init__(self, x:float):
        self.x = x

#屏幕坐标左上是0，0
class Canvas2D:
    def __init__(self, x:float, y:float):
        self.x = x
        self.y = y

class Color:
    def __init__(self, r:int, g:int, b:int, a:int):
        self.r, self.g, self.b, self.a = r, g, b, a


    
#屏幕接口。
class Canvas:
    def __init__(self, width:int, height:int, resizable:bool=True):
        return
    def add_square(self, side:Canvas1D, center:Canvas2D, color:Color):
        # todo
        return
    def add_line(self, start:Canvas2D, end:Canvas2D, color:Color):
        # todo
        return
    def add_text(self, text:str, center:Canvas2D, color:Color, size:int):
        #todo
        return
    def clean(self):
        # TODO
        return
    def draw(self):
        #todo
        return
    def should_close(self) -> bool:
        return rl.WindowShouldClose()
    def close(self):
        rl.CloseWindow()
    
#raylib实现：add_*只记录绘制命令，draw时统一绘制一帧，clean清空命令。
class CanvasRaylib(Canvas):
    def __init__(self, width:int, height:int, resizable:bool=True, title:str="foxel", background:Color=Color(0, 0, 0, 255)):
        if resizable:
            rl.SetConfigFlags(rl.FLAG_WINDOW_RESIZABLE)
        rl.InitWindow(width, height, title.encode("utf-8"))
        self.background = background
        self.commands = []

    @staticmethod
    def _color(color:Color):
        return (color.r, color.g, color.b, color.a)

    def add_square(self, side:Canvas1D, center:Canvas2D, color:Color):
        half = side.x / 2
        pos = (center.x - half, center.y - half)
        size = (side.x, side.x)
        c = self._color(color)
        self.commands.append(lambda: rl.DrawRectangleV(pos, size, c))

    def add_line(self, start:Canvas2D, end:Canvas2D, color:Color):
        s, e = (start.x, start.y), (end.x, end.y)
        c = self._color(color)
        self.commands.append(lambda: rl.DrawLineV(s, e, c))

    def add_text(self, text:str, center:Canvas2D, color:Color, size:int):
        t = text.encode("utf-8")
        width = rl.MeasureText(t, size)
        x = int(center.x - width / 2)
        y = int(center.y - size / 2)
        c = self._color(color)
        self.commands.append(lambda: rl.DrawText(t, x, y, size, c))

    def clean(self):
        self.commands.clear()

    def draw(self):
        rl.BeginDrawing()
        rl.ClearBackground(self._color(self.background))
        for cmd in self.commands:
            cmd()
        rl.EndDrawing()

    def should_close(self) -> bool:
        return rl.WindowShouldClose()

    def close(self):
        rl.CloseWindow()