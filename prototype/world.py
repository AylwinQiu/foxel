from math import ceil
from typing import Callable
import ctypes
import ctypes as C
import math


# cell的最大深度
MAX_CELL_DEPTH = 64
# 在此深度cell的边长对应物理距离的1.0
IDENTITY_DEPTH = 32
# 约定：
#   根节点(深度0)的中心在物理原点，边长为 2**IDENTITY_DEPTH（深度为 IDENTITY_DEPTH 时边长为1.0）。
#   象限按数学惯例：1=(+x,+y) 2=(-x,+y) 3=(-x,-y) 4=(+x,-y)。
#   落在分界线上的点归入正方向（x>=cx 视为右，y>=cy 视为上）。
# 象限 -> (x方向符号, y方向符号)
_QUADRANT_SIGN = {1: (1, 1), 2: (-1, 1), 3: (-1, -1), 4: (1, -1)}

class PhysicalPosition2D(C.Structure):
    _fields_ = [
        ("x", C.c_double),
        ("y", C.c_double),
    ]

class PhysicalPosition1D(C.Structure):
    _fields_ = [
        ('x', C.c_double),
    ]

type Rotation = float

class Field[T]:
    '场的定义'
    def get(self, world:World, id:CellID) -> T|None:
        ...
    def rev(self, world:World, t:T) -> CellID|None:
        ...
    def have_rev(self, world:World) -> bool:
        '返回该场是否可逆'
        return False
    def delete(self, world:World, id:CellID):
        '删除一个cell'
        ...


#全局唯一的cellid生成
__next_cell_id_status:int = 0
# 方块的id
class CellID:
    def __init__(self):
        global __next_cell_id_status
        self.id = __next_cell_id_status
        __next_cell_id_status += 1



class Cell:
    '定义了方块节点，我们通过这个四叉树可以拿到CellID.'
    def __init__(self, child:list[Cell]|None):
        self._child = child
        self._id:CellID = CellID()
    def get_child(self, x:int):
        # x should in [0, 1,2,3]
        if self._child!=None:
            if x not in (0,1,2,3,):
                raise Exception("x should in (0,1,2,3)")
            return self._child[x]


class TreePosition:
    def __init__(self, tree:list[int]):
        'tree：这个里的int可以取0，1，2，3，4；1-4代表的是四个象限。0表示深度到头。遇到的第一个0就是序列的结尾'
        #检查tree的长度
        if (l:=len(tree))!=MAX_CELL_DEPTH:
            raise Exception(f"Input list should have lenth {MAX_CELL_DEPTH} but got {l}")
        self._tree = tree
    def tree(self):
        return self._tree
    def __hash__(self):
        return hash(str(self._tree))
        pass
    def depth(self):
        '获得深度（从第一个向后数直到遇到第一个0时有多少非零数字？）'
        for i in range(len(self._tree)):
            if self._tree[i]==0:
                return i
        return len(self._tree)

class Offset:
    def __init__(self, x:float, y:float):
        if x>=0.6 or x<=-0.6 or y>=0.6 or y<=-0.6:
            raise Exception("x and y should in range (-0.6, 0.6)")
        self._x = x
        self._y = y
    def get(self) -> tuple[float, float]:
        return self._x, self._y

class OffsetField(Field[Offset]):
    """定义中心偏移场（为了让每个方块可以平滑移动，我们需要让方块有一个offset场。）  
    offset的单位是当前的cell的边长，大小是(-0.6, 0,6)这样可以保证不会在一半周期的时候来回震荡。
    """
    def __init__(self):
        self.offset:dict[CellID, Offset] = {}
    def get(self, world:World, id:CellID) -> Offset|None:
        return self.offset.get(id)
    def rev(self, world:World, t:Offset) -> CellID|None:
        return None
    def have_rev(self, world:World) -> bool:
        '返回该场是否可逆'
        return False
    def delete(self, world:World, id:CellID):
        '删除一个cell'
        self.offset.pop(id, None)

class World:
    def __init__(self):
        # 先生成一个空的根cell，处在第0层
        self.cell_tree:list[Cell] = [Cell(None)]
        # 这个保存着各种不同的场，储存方法是场名和对应的场。
        self.fields:dict[str, Field] = {}
    def delete_cell(self, cellid:CellID) :
        '删除一个cell。调用所有field的delete属性'
        for field in self.fields.values():
            field.delete(self, cellid)
    def add_cell(self, cell_status:dict):
        '添加一个cell。cell_status是cell的所有信息。'
        # TODO
        pass
    @staticmethod
    def tree_to_phy_no_offset(tree:TreePosition) -> PhysicalPosition2D:
        '''该函数把树坐标转换成对应物理坐标，不考虑offset。返回的是该cell中心的物理坐标'''
        x, y = 0.0, 0.0
        for k, q in enumerate(tree.tree()):
            if q == 0:
                break
            sx, sy = _QUADRANT_SIGN[q]
            # 深度k的cell边长为 2**(IDENTITY_DEPTH-k)，子cell中心相对父cell中心偏移其边长的1/4
            offset = math.ldexp(1.0, IDENTITY_DEPTH - k - 2)
            x += sx * offset
            y += sy * offset
        return PhysicalPosition2D(x, y)
    @staticmethod
    def phy_to_tree_no_offset(phy:PhysicalPosition2D, depth:int) -> TreePosition:
        '''该函数把物理坐标转化成对应深度的树坐标， 不考虑offset'''
        if not 0 <= depth <= MAX_CELL_DEPTH:
            raise ValueError(f"depth should in [0, {MAX_CELL_DEPTH}] but got {depth}")
        half_root = math.ldexp(1.0, IDENTITY_DEPTH - 1)
        if not (-half_root <= phy.x < half_root and -half_root <= phy.y < half_root):
            raise ValueError(f"({phy.x}, {phy.y}) is outside the root cell [-{half_root}, {half_root})")
        tree = [0] * MAX_CELL_DEPTH
        cx, cy = 0.0, 0.0
        for k in range(depth):
            right = phy.x >= cx
            up = phy.y >= cy
            if up:
                q = 1 if right else 2
            else:
                q = 4 if right else 3
            tree[k] = q
            sx, sy = _QUADRANT_SIGN[q]
            offset = math.ldexp(1.0, IDENTITY_DEPTH - k - 2)
            cx += sx * offset
            cy += sy * offset
        return TreePosition(tree)
    @staticmethod
    def tree_to_phy(treepos:TreePosition, offset:Offset) -> PhysicalPosition2D:
        """考虑offset后，将数坐标和偏移转化成物理坐标"""
        center = World.tree_to_phy_no_offset(treepos)
        # offset的单位是当前cell的边长
        side = math.ldexp(1.0, IDENTITY_DEPTH - treepos.depth())
        ox, oy = offset.get()
        return PhysicalPosition2D(center.x + ox * side, center.y + oy * side)
    @staticmethod
    def phy_to_tree(phypos:PhysicalPosition2D, depth:int) -> tuple[TreePosition, Offset]:
        """将一个物理坐标转化成对应深度的数坐标和对应的偏移量"""
        treepos = World.phy_to_tree_no_offset(phypos, depth)
        center = World.tree_to_phy_no_offset(treepos)
        side = math.ldexp(1.0, IDENTITY_DEPTH - depth)
        # 点总落在所选cell内，因此偏移在[-0.5, 0.5)之间
        return treepos, Offset((phypos.x - center.x) / side, (phypos.y - center.y) / side)
    
    