# slice, array, map 的随机 access 速度对比（size = 16）

从快到慢排列：array ≈ switch-case > slice >> map (基于 go1.17.7)
- map 比其他的慢了一个数量级
- 不嫌麻烦的话，直接用 switch case 是比较好的，因为还可以处理异常输入
  - 显然，array 的方式代码是最简单的，而且直接增加枚举值就好了，但是没有异常输入的处理能力，异常输入直接靠 panic
```bash
[root@LC benchmark]# go version
go version go1.17.7 linux/amd64
[root@LC benchmark]# go test -v -bench='Benchmark_Access.*' -cpu=1 -benchtime=5s
goos: linux
goarch: amd64
pkg: benchmark
cpu: Intel(R) Xeon(R) Platinum 8269CY CPU @ 2.50GHz
Benchmark_AccessStackArray_SmallSize
Benchmark_AccessStackArray_SmallSize 	1000000000	         0.3325 ns/op
Benchmark_AccessArray_SmallSize
Benchmark_AccessArray_SmallSize      	1000000000	         0.3209 ns/op
Benchmark_AccessSlice_SmallSize
Benchmark_AccessSlice_SmallSize      	1000000000	         0.5421 ns/op
Benchmark_AccessMap_SmallSize
Benchmark_AccessMap_SmallSize        	719862216	         8.234 ns/op
Benchmark_AccessSwitchCase_SmallSize
Benchmark_AccessSwitchCase_SmallSize 	1000000000	         0.3337 ns/op
PASS
ok  	benchmark	8.457s
[root@LC benchmark]# 
```
