package main

import "testing"

/* result: (fast)    array = switch-case > slice > map     (slow)
[root@LC benchmark]# go test *.go -v -bench='Benchmark_Access.*' -cpu=1 -benchtime=5s
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
*/

const smallSize = 16

// go test *.go -v -bench='Benchmark_AccessStackArray_SmallSize' -cpu=1 -benchtime=5s
func Benchmark_AccessStackArray_SmallSize(b *testing.B) {
	var tmp string
	var array [smallSize]string
	for i := 0; i < smallSize; i++ {
		array[i] = "demo-string"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmp = array[i&0xf]
	}
	b.StopTimer()

	_ = tmp
}

// go test *.go -v -bench='Benchmark_AccessArray_SmallSize' -cpu=1 -benchtime=5s
func Benchmark_AccessArray_SmallSize(b *testing.B) {
	var tmp string
	var array *[smallSize]string = new([smallSize]string) // slice and map also in heap
	for i := 0; i < smallSize; i++ {
		array[i] = "demo-string"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmp = array[i&0xf]
	}
	b.StopTimer()

	_ = tmp
}

// go test *.go -v -bench='Benchmark_AccessSlice_SmallSize' -cpu=1 -benchtime=5s
func Benchmark_AccessSlice_SmallSize(b *testing.B) {
	var tmp string
	slice := make([]string, 0, 10)
	for i := 0; i < smallSize; i++ {
		slice = append(slice, "demo-string")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmp = slice[i&0xf]
	}
	b.StopTimer()

	_ = tmp
}

// go test *.go -v -bench='Benchmark_AccessMap_SmallSize' -cpu=1 -benchtime=5s
func Benchmark_AccessMap_SmallSize(b *testing.B) {
	var tmp string
	Map := make(map[int]string, smallSize)
	//Map := make(map[int]string)
	for i := 0; i < smallSize; i++ {
		Map[i] = "demo-string"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmp = Map[i&0xf]
	}
	b.StopTimer()

	_ = tmp
}

// go test *.go -v -bench='Benchmark_AccessSwitchCase_SmallSize' -cpu=1 -benchtime=5s
func Benchmark_AccessSwitchCase_SmallSize(b *testing.B) {
	var tmp string

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i & 0xf {
		case 0:
			tmp = "demo-string"
		case 1:
			tmp = "demo-string"
		case 2:
			tmp = "demo-string"
		case 3:
			tmp = "demo-string"
		case 4:
			tmp = "demo-string"
		case 5:
			tmp = "demo-string"
		case 6:
			tmp = "demo-string"
		case 7:
			tmp = "demo-string"
		case 8:
			tmp = "demo-string"
		case 9:
			tmp = "demo-string"
		case 10:
			tmp = "demo-string"
		case 11:
			tmp = "demo-string"
		case 12:
			tmp = "demo-string"
		case 13:
			tmp = "demo-string"
		case 14:
			tmp = "demo-string"
		case 15:
			tmp = "demo-string"
		}
	}
	b.StopTimer()

	_ = tmp
}
