package main

import "testing"

/* result: (fast)
           1. inline function/method = inline interface type assert method
           2. noinline function/method = noinline interface type assert method
           3. interface method
           (slow)
[root@LC benchmark]# go test *.go -v -bench='Benchmark_.*line.*' -benchmem
goos: linux
goarch: amd64
cpu: Intel(R) Xeon(R) Platinum 8269CY CPU @ 2.50GHz
Benchmark_InlineFunc
Benchmark_InlineFunc-2                           	1000000000	         0.3164 ns/op	       0 B/op	       0 allocs/op
Benchmark_NoinlineFunc
Benchmark_NoinlineFunc-2                         	706127727	         1.620 ns/op	       0 B/op	       0 allocs/op
Benchmark_InlineMethod
Benchmark_InlineMethod-2                         	1000000000	         0.3782 ns/op	       0 B/op	       0 allocs/op
Benchmark_NoinlineMethod
Benchmark_NoinlineMethod-2                       	792969295	         1.404 ns/op	       0 B/op	       0 allocs/op
Benchmark_InterfaceInlineMethod
Benchmark_InterfaceInlineMethod-2                	486609109	         2.163 ns/op	       0 B/op	       0 allocs/op
Benchmark_InterfaceNoinlineMethod
Benchmark_InterfaceNoinlineMethod-2              	323111224	         3.773 ns/op	       0 B/op	       0 allocs/op
Benchmark_InterfaceInlineMethod_TypeAssert
Benchmark_InterfaceInlineMethod_TypeAssert-2     	1000000000	         0.3225 ns/op	       0 B/op	       0 allocs/op
Benchmark_InterfaceNoinlineMethod_TypeAssert
Benchmark_InterfaceNoinlineMethod_TypeAssert-2   	750869515	         1.595 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	command-line-arguments	7.971s
[root@LC benchmark]#
*/

// go test *.go -v -bench='Benchmark_InlineFunc' -benchmem
func Benchmark_InlineFunc(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inlineFunc(i)
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_NoinlineFunc' -benchmem
func Benchmark_NoinlineFunc(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		noinlineFunc(i)
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_InlineMethod' -benchmem
func Benchmark_InlineMethod(b *testing.B) {
	v := inlineVar(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.inlineMethod()
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_NoinlineMethod' -benchmem
func Benchmark_NoinlineMethod(b *testing.B) {
	v := inlineVar(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.noinlineMethod()
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_InterfaceInlineMethod' -benchmem
func Benchmark_InterfaceInlineMethod(b *testing.B) {
	iv := inlineVar(10)
	var intf inlineInterface
	intf = iv

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		intf.inlineMethod()
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_InterfaceNoinlineMethod' -benchmem
func Benchmark_InterfaceNoinlineMethod(b *testing.B) {
	iv := inlineVar(10)
	var intf inlineInterface
	intf = iv

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		intf.noinlineMethod()
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_InterfaceInlineMethod_TypeAssert' -benchmem
func Benchmark_InterfaceInlineMethod_TypeAssert(b *testing.B) {
	iv := inlineVar(10)
	var intf inlineInterface
	intf = iv

	typeAssert := intf.(inlineVar)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		typeAssert.inlineMethod()
	}
	b.StopTimer()
}

// go test *.go -v -bench='Benchmark_InterfaceNoinlineMethod_TypeAssert' -benchmem
func Benchmark_InterfaceNoinlineMethod_TypeAssert(b *testing.B) {
	iv := inlineVar(10)
	var intf inlineInterface
	intf = iv

	typeAssert := intf.(inlineVar)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		typeAssert.noinlineMethod()
	}
	b.StopTimer()
}
