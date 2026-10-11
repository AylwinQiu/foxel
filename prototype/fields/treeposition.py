import math
from ..world import *
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

def tree_to_phy(tree:TreePosition) -> PhysicalPosition2D:
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

def phy_to_tree(phy:PhysicalPosition2D, depth:int) -> TreePosition:
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


class TreePositionField(Field):
    @staticmethod
    def get(world:World, id:CellID):
        #TODO
        pass
