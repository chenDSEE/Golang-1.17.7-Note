package main

import (
	"fmt"
	"io"
	"net/http"
)

// http 本身就是要求无状态，所以不暴露太多信息给每一个 handler，合情合理
// 直接注入 function 还是比较像面向过程的方案
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> helloHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	io.WriteString(w, "Hello, world!\n")
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> echoHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	io.WriteString(w, r.URL.Path + "\n")
}

func helperHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> helperHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	io.WriteString(w, r.URL.Path + "\n")
}

func headerControlHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> headerControlHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	
	/* set HTTP Header */
	w.Header().Set("Allow", http.MethodPost) // Header.Set() 只会设置一个 Allow Header
	
	w.Header().Add("Cache-Control", "public") // 设置多个 Cache-Control Header
	w.Header().Add("Cache-Control", "max-age=31536000")
	
	/* set HTTP status code */
	// 一定要在 w.Write() 之前设置 status code，不然会有默认值的
	w.WriteHeader(http.StatusAccepted)

	/* set HTTP body */
	io.WriteString(w, r.URL.Path + "\n")
	io.WriteString(w, "With status code 202 and Cache-Control Header" + "\n")
}

// 通过 struct 携带更多的信息
/* 因为 object 本身就已经可以很好的命名
 * 所有 object 处理 http 的 method 名字就是 ServeHTTP
 * 不需要额外命名。这是一个面向对象的方案
 */
type OBJ struct {
	data string
}

func (o *OBJ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> object[%s] visit %s\n", r.RemoteAddr, o.data, r.URL.Path)
	io.WriteString(w, r.URL.Path+"\n")
}

/* route version */
// test case:
//   curl -i http://localhost:8080/
//   curl -i http://localhost:8080/hello
//   curl -i http://localhost:8080/function-example
//   curl -i http://localhost:8080/object-example
//   curl -i http://localhost:8080/bad-example
//   curl -i http://localhost:8080/header-control
func main() {
	fmt.Println("====== HTTP Server Start ======")

	/* http server 能够正常工作的前提是：
	 * 1. 能够与客户端建立网络连接（根据 RFC，底层必须使用 TCP 协议）
	 * 2. http server 能够正常解析浏览器发来的 http 报文
	 * 3. http server 能够根据 URL 来定位到浏览器访问的资源
	 *
	 * 很显然，http server 可以被分解为三个动作：路由表的建立 + 启动 http 监听 + 完成不同 URL 下的 handler 代码
	 * 至于怎么处理 HTTP 协议，转化为 app 的数据结构来进行使用，其实是可以由框架一手包办的。
	 * 只有业务是没办法被标准库完成的
	 */
	// setp 1: 建立路由表，并注入相应 URL 的 handler
	// 直接使用 http.HandleFunc() 其实跟自己创建一个 mux 没什么区别
	mux := http.NewServeMux()

	// 使用方案 1：
	// 注入函数、匿名函数使用 http.ServeMux.HandleFunc()
	// 看函数签名就知道：是能够接收函数注入的
	// func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	mux.HandleFunc("/", echoHandler)
	mux.HandleFunc("/function-example", helloHandler)
	mux.HandleFunc("/header-control", headerControlHandler)

	// 使用方案 2：
	// 注入 object 使用 http.ServeMux.Handle()
	// 注入 object，可以携带更多的数据信息
	// func (mux *ServeMux) Handle(pattern string, handler Handler)
	obj := &OBJ{data: "obj-data-string"}
	mux.Handle("/object-example", obj)

	// 使用方案 3：（！！不推荐！！）
	// 通过强制转换的方式，注入函数
	// 这种时候，用 http.ServeMux.HandleFunc() 更适合（内置强制转换）
	// 用 http.ServeMux.Handle() 还得自己手动强制转换
	mux.Handle("/bad-example", http.HandlerFunc(helperHandler))

	// step 2: 启动 http 的监听，并且把这个路由表传递给相应的 TCP server
	server := &http.Server{Addr: "localhost:8080", Handler: mux}
	server.ListenAndServe()

	/* 看，实际上路由表 ServeMux 是一个很独立的东西，
	   甚至同一个路由表是可以让两个不同的 TCP server 端口一起使用的
	server1 := &http.Server{Addr: "localhost:8080", Handler: mux}
	server2 := &http.Server{Addr: "localhost:80", Handler: mux}
	go server1.ListenAndServe()
	server2.ListenAndServe()
	*/
}