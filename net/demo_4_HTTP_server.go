package main

import (
	"fmt"
	"io"
	"io/ioutil"
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


/* 通过 struct 携带更多的信息 */
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

/* chaining handler */
// auditLog() ---> authCheck() ---> chainFunctionHandler()
// 返回的匿名函数通常都不会里面执行的，就像原本注入的函数一样
// 因为这个横向函数仅仅是增加了一些横向的东西，所以参数、返回值都是一样的。
// 怎么进来就怎么出去，不发生丝毫的变化，原本的函数没有任何感知

// Usage case 1: function 包裹
func auditLog(function http.HandlerFunc) http.HandlerFunc {
	// 横向观测点不能写在这里，这只会被运行一次的！
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("demo a audit Log. ") // 横向动作
		function(w, r)                          // 直接转发去原本的函数里面
	}
}

func authCheck(function http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("after auth check\n")
		function(w, r)
	}
}

func chainFunctionHandler(w http.ResponseWriter, r *http.Request) {
	// 看，log 跟 auth 的逻辑可以相互隔离，避免依赖
	fmt.Printf("[%s]==> chainFunctionHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	io.WriteString(w, "finish chaining function call\n")
}

// Usage case 2: object 包裹
// 因为现在是一个 object 要穿越这些横向拓展，所以参数跟返回值变通一下就是了
func objAuditLog(obj http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		fmt.Printf("demo a obejct audit Log. ") // 横向动作
		obj.ServeHTTP(w, r)
	})
}

func objAuthCheck(obj http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		fmt.Printf("after object auth check\n")
		obj.ServeHTTP(w, r)
	})
}

type chainObj struct {
	name string
}

func (obj *chainObj) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> object[%s] chain visit %s\n", r.RemoteAddr, obj.name, r.URL.Path)
	io.WriteString(w, r.URL.Path+" finish object chain call\n")
}

/* request demo */
func requestDemoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> requestDemoHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	// dump request Header
	for key, valueSlice := range r.Header {
		for _, value := range valueSlice {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	// dump request body
	buf := make([]byte, 50)
	r.Body.Read(buf)
	fmt.Printf("body:[%s]\n", string(buf))
	fmt.Println("====== all done ======")
}

/* HTTP URL encoded demo */
//  Issue by: curl 'http://localhost:8080/http-url-encoded?param-1=value-1&param-2=123' -X POST -d 'param-1=value-2&param-3=456'
//
//	POST /http-web-form?param-1=val-1&param-2=123 HTTP/1.1
//	Host: localhost:8080
//	User-Agent: curl/7.83.1
//	Accept: */*
//	Content-Length: 27
//	Content-Type: application/x-www-form-urlencoded
//	\r\n
//	param-1=value-2&param-3=456
//
func httpUrlEncodedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> httpUrlEncodedHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	// parse request body data to Form
	if err := r.ParseForm(); err != nil {
		fmt.Println("request parse Form error, ", err)
		return
	}

	// dump http.Request.Form
	// 无论你的参数是放在 URL 上，还是 body 里面，都能够被解析进来这里
	// 在 HTML Form 的 value 总是在 URL 的 value 前面
	// 即使是 %20 也会帮你变回空格
	fmt.Println("==== dump r.Form ====")
	for key, valueSlice := range r.Form {
		for _, value := range valueSlice {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	// dump http value from request body(HTML form)
	// 只支持 x-www-form-urlencoded 类型
	fmt.Println("==== dump r.PostForm ====")
	for key, valueSlice := range r.PostForm {
		for _, value := range valueSlice {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	fmt.Println("====== all done ======")
}

/* HTTP form-data demo */
//  Issue by: curl 'http://localhost:8080/http-form-data' -F 'Para-1=Value-1' -F 'Para-2=Value-2' -F 'Para-1=Value-3'
func httpFormDataHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> httpFormDataHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	// parse request body data to Form
	if err := r.ParseMultipartForm(1024); err != nil {
		fmt.Println("request parse MultipartForm error, ", err)
		return
	}

	// dump http.Request.Form
	// 只包含在 body 里面的参数
	fmt.Println("==== dump r.Form ====")
	for key, valueSlice := range r.MultipartForm.Value {
		for _, value := range valueSlice {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	fmt.Println("====== all done ======")
}

/* HTTP upload file demo */
// Issue by: curl http://localhost:8080/http-file-upload -X POST -F 'fileName=@/PATH/TO/file.txt'
func httpFileUpoladHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> httpFileUpoladHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	var data []byte
	if false {
		// 解析方式 1: 手动解析
		r.ParseMultipartForm(1024)
		fileHeader := r.MultipartForm.File["fileName"][0]
		file, err := fileHeader.Open()
		if err != nil {
			fmt.Println("request parse fileHeader.Open() error, ", err)
			return
		}
		data, err = ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("request read file error, ", err)
			return
		}

	} else {
		// 解析方式 2: 直接调用
		file, _, err := r.FormFile("fileName")
		if err != nil {
			fmt.Println("request parse FormFile() error, ", err)
			return
		}
		data, err = ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("request read file error, ", err)
			return
		}
	}
	fmt.Printf("data:[%s]\n", string(data))

	fmt.Println("====== all done ======")
}

/* server dispatch cookie */
func dispatchCookieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> dispatchCookieHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	cookie1 := http.Cookie{
		Name: "cookie_one",
		Value: "cookie-value-one",
		HttpOnly: true,
	}
	cookie2 := http.Cookie{
		Name: "cookie_two",
		Value: "cookie-value-two",
		HttpOnly: true,
	}

	w.Header().Set("Set-Cookie", cookie1.String())
	http.SetCookie(w, &cookie2)
}

/* client upload cookie */
func uploadCookieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> uploadCookieHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	// cookie 本质上就是一个 HTTP Header，所以直接通过 r.Header 也获取也没毛病
	{
		cookies := r.Header["Cookie"]
		fmt.Println("via r.Header map:", cookies)
	}

	// 获得指定 cookie
	{
		cookie, err := r.Cookie("key-1")
		if err != nil {
			fmt.Println("can not find cookie with key-1", err)
			return
		}
		fmt.Println("via r.Cookie(), key-1 =", cookie)
	}

	// 直接拿全部 cookie
	{
		cookies := r.Cookies()
		fmt.Println("via r.Cookies()", cookies)
	}
}

// test case:
//   curl -i http://localhost:8080/
//   curl -i http://localhost:8080/hello
//   curl -i http://localhost:8080/function-example
//   curl -i http://localhost:8080/object-example
//   curl -i http://localhost:8080/bad-example
//   curl -i http://localhost:8080/header-control
//   curl -i http://localhost:8080/chaining-function
//   curl -i http://localhost:8080/chaining-object
//   curl -i -d "Name=name&Age=10" http://localhost:8080/request-demo
//   curl 'http://localhost:8080/http-url-encoded?param-1=value-1&param-2=123' -X POST -d 'param-1=value-2&param-3=456'
//   curl 'http://localhost:8080/http-form-data' -F 'Para-1=Value-1' -F 'Para-2=Value-2' -F 'Para-1=Value-3'
//   curl http://localhost:8080/http-file-upload -X POST -F 'fileName=@/PATH/TO/file.txt'
//   curl -i http://localhost:8080/dispatch-cookie
//   curl -v http://localhost:8080/upload-cookie --cookie 'key-1=value-1' --cookie 'key-2=value-2' --cookie 'key-1=value-3'
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

	// chaining example, pipeline
	/* 这种串联方式，实际上是一种横向拓展的方式。
	 * 优点:
	 * 不会让没有关联的代码相互依赖：http 的处理逻辑，不用加上 log 的依赖
	 * 增加横向观察，钩子的时候，不需要改动框架的代码
	 *
	 * 缺点:
	 * 作为横向观察，无法传入、传出额外的参数
	 *
	 * 注意，通过返回匿名函数的方式，达到的是从外向内的执行效果。
	 * 而 C 语言传递回调函数的方式，只能达到从内向外的执行效果
	 */
	// chaining-function
	mux.HandleFunc("/chaining-function", auditLog(authCheck(chainFunctionHandler)))
	mux.Handle("/chaining-object", objAuditLog(objAuthCheck(&chainObj{"chaining"})))

	// HTTP Form demo
	mux.HandleFunc("/request-demo", requestDemoHandler)
	mux.HandleFunc("/http-url-encoded", httpUrlEncodedHandler)
	mux.HandleFunc("/http-form-data", httpFormDataHandler)
	mux.HandleFunc("/http-file-upload", httpFileUpoladHandler)

	// cookie demo
	mux.HandleFunc("/dispatch-cookie", dispatchCookieHandler)
	mux.HandleFunc("/upload-cookie", uploadCookieHandler)

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