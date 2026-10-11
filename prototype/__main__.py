import numpy as np
from .canvas import CanvasRaylib, Canvas1D, Canvas2D, Color
print("Prototype v0.0.1")
print(__file__)
input()
ray = CanvasRaylib(800, 600);
while not ray.should_close():
    ray.add_square(Canvas1D(90), Canvas2D(400, 300), Color(200, 200, 200, 254))
    ray.draw()
if False:
    for i in range(10000000):
        j=i#np.int64(i)
        while j>10:
            if j%2==0:
                j=j//2
            else:
                j=j*3+1
        if i%10000==0:
            print(i)
        