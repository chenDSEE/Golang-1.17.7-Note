package main

import "testing"

/* result(when always return at first): (fast)  fastPath > slowPath   (slow)
[root@LC benchmark]# go test *.go -v -bench='Benchmark_.*Path' -benchmem -cpu=1
goos: linux
goarch: amd64
cpu: Intel(R) Xeon(R) Platinum 8269CY CPU @ 2.50GHz
Benchmark_slowPath
Benchmark_slowPath 	643822761	         1.847 ns/op	       0 B/op	       0 allocs/op
Benchmark_fastPath
Benchmark_fastPath 	1000000000	         0.4116 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	command-line-arguments	1.836s
[root@LC benchmark]#
*/

// go test *.go -v -bench='Benchmark_slowPath' -benchmem
func Benchmark_slowPath(b *testing.B) {
	data := []int{10, 20, 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slowPath(data) // can not inline
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_fastPath' -benchmem
func Benchmark_fastPath(b *testing.B) {
	data := []int{10, 20, 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fastPath(data)
	}
	b.StopTimer()
}
