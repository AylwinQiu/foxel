def a(xx:Int):
    print(xx)

def main():
    var kk:List[Int] =[0 for i in range(10000000)]
    for i in range(1000000000000000):
        var j:Int = i
        var zz:Int = 0
        while j>10:
            if j%2==0:j=j//2
            else:j=j*3+1
            kk[zz] = j
            zz+=1
        if i%1000000==0:
            print(i, kk[0:20])