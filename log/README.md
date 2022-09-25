## log

> base on: go1.17.7 linux/amd64
>
> https://pkg.go.dev/log

- 支持指定文件输出位置，而且是采用 `io.writer` 作为接口的，所以是能够将日志通过网络的方式传输出去的（`func SetOutput(w io.Writer)`）



### quick demo

**demo 1**

```go
package main

import (
	"log"
	"os"
)

func main() {
	// log to file
	logFile, err := os.Create("./log.log")
	defer logFile.Close()
	if err != nil {
		log.Fatalln("create file log.log failed")
	}
	logger := log.New(logFile, "[Debug] ", log.Lshortfile)
	logger.Print("call Print: line1")
	logger.Println("call Println: line2")

	// change configuration for log package
	logger.SetPrefix("[Info] ")
	logger.SetFlags(log.Ldate)
	logger.SetOutput(os.Stdout)
	logger.Print("Info check stdout")
}
```

```bash
[root@LC demo]# go run main.go 
[Info] 2022/09/24 Info check stdout
[root@LC demo]# cat log.log 
[Debug] main.go:17: call Print: line1
[Debug] main.go:18: call Println: line2
[root@LC demo]# 
```





### `Logger` struct

```go
// A Logger represents an active logging object that generates lines of
// output to an io.Writer. Each logging operation makes a single call to
// the Writer's Write method. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
// Logger 本身并不需要是一个 io.Writer, 因为 Logger 具体往哪里 log 是看配置的，所以 Logger has-a io.Writer 也是可以的
type Logger struct {
	// ensures atomic writes; protects the following fields, 并发安全
	mu     sync.Mutex
	// prefix on each line to identify the logger (but see Lmsgprefix)，perfix 是配置后固定的，不需要每次都生成
	prefix string     
	flag   int        // properties
	// destination for output, 通过 io.Writer 进行解耦，让 Logger 既能够向 socket 输出，也能够向文件输出
	out    io.Writer  
	buf    []byte     // for accumulating text to write，格式化时使用
}
```



### 输出日志的过程

**step 1:**

- 首先是我们调用不同的 print log 函数
- 然后利用 `fmt.Sprint` 来完成字符串的格式化（完成用户输出内容的处理）

- 基本支持三种输出格式：`Print`, `Printf`, `Println`

```go
// step 1: 然后利用 fmt.Sprint 来完成字符串的格式化（完成用户输出内容的处理）
// step 2: 在 Logger 这一层完成 log 的格式化
// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) { l.Output(2, fmt.Sprint(v...)) }

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(v ...interface{}) { l.Output(2, fmt.Sprintln(v...)) }
```



**step 2:**

- 在 Logger 这一层完成 log 的格式化，使得 `Output()` 拿到的即使完成了 fmt 的 string

- 但是最后依然是调用底层的 `func (l *Logger) Output(calldepth int, s string)`

```go
// 完成前缀、时间、追加换行、实际输出等工作；不需要理会 fmt 的事情
func (l *Logger) Output(calldepth int, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int

	// 实际 write 是加锁的，所以不会有并发问题
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.flag&(Lshortfile|Llongfile) != 0 {
		// 临时释放锁，因为获取文件信息实在是太耗时了
		// 这里基本都是操作函数内的临时变量，所以不锁也是可以的
		// Release lock while getting caller info - it's expensive.
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}
	l.buf = l.buf[:0] // reset

	/* Header */
	l.formatHeader(&l.buf, now, file, line) // 直接让底层操作当前的 slice

	/* 实际内容 */
	l.buf = append(l.buf, s...) // 避免生成新的 slice 对象

	/* 追加换行 */
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf) // 实际输出
	return err
}


// 格式化 log 前缀，这个函数的调用是被锁保护起来的，因为每个 Logger 的 buf 就一个
func (l *Logger) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	/* 加入前缀 */
	if l.flag&Lmsgprefix == 0 {
		*buf = append(*buf, l.prefix...)
	}

	/* 加入时间 */
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		.....
	    year, month, day := t.Date()
	    itoa(buf, year, 4)
	    *buf = append(*buf, '/')
	    ......
	}

	/* 文件名、行号 */
	if l.flag&(Lshortfile|Llongfile) != 0 {
		........
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
	if l.flag&Lmsgprefix != 0 {
		*buf = append(*buf, l.prefix...)
	}
}

```

