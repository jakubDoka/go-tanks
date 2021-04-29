cf = open("cube.obj")

vertices = []
faces = []

def valid(a):
    return a.strip() != ""

def spl(a, fn):
    return list(map(fn, filter(valid ,a.split(" "))))
    

for l in cf:
    if l[0] == "v":
        vertices.append(spl(l[2:], float))
    if l[0] == "f":
        faces.append(spl(l[2:], int))

graph = {}

for f in faces:
    for i in f:
        if not i in graph:
            graph[i] = set()
    print(f)
    graph[f[0]] |= set(f[1:])
    graph[f[1]] |= set([f[0],f[2]])
    graph[f[2]] |= set(f[:2])

for g in graph:
    print(g, ":", graph[g])
