package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// curl -i http://127.0.0.1:80/template/string
func templateStringHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateStringHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	tmpl, err := template.New("tmpl-string").Parse(`<h1>title line</h1>`)
	//tmpl, err := template.New("tmpl-string").Parse("string without HTML tag") // also Parse("string without HTML tag")
	if err != nil {
		fmt.Println("template.New() error with:", err)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/insert-string
func templateInsertHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateInsertHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	tmpl, err := template.New("tmpl-insert").Parse(`<h1>title {{.}}</h1>`) // {{.}} as a mark
	if err != nil {
		fmt.Println("template.New() error with:", err)
		return
	}

	err = tmpl.Execute(w, "insert-string")
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/insert-object-field
func templateInsertObjectHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateInsertObjectHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	tmpl, err := template.New("tmpl-insert").Parse(`<h1>title {{.Name}} with {{.Age}}</h1>`)
	if err != nil {
		fmt.Println("template.New() error with:", err)
		return
	}

	type Obj struct {
		Name string
		Age  int
	}
	err = tmpl.Execute(w, Obj{Name: "object-name-string", Age: 10}) // {{.Name}} map to Obj.Name field
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/embedded-loop
func templateEmbeddedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateEmbeddedHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	tmpl := template.New("tmpl-embedded")
	// Person.Name, Person.Email, Friend.Name
	tmpl, err := tmpl.Parse(`<h1>{{.Name}} Information</h1>
		<h2>Email List:</h2>
		{{range .Email}}
			Email: {{.}} <br />
		{{end}}
		<h2>Friends:</h2>
		{{with .Friends}}
			{{range .}}
				Friend name: {{.Name}} <br />
			{{end}}
		{{end}}`)

	if err != nil {
		fmt.Println("template.Parse() error with:", err)
		return
	}

	type Friend struct {
		Name string
	}

	type Person struct {
		Name   string
		Email  []string
		Friends []Friend
	}

	jone := Person{
		Name: "jone",
		Email: []string{"jone@gmail.com", "jone@qq.com"},
		Friends: []Friend{
			{Name: "jack"},
			{Name: "amy"},
		},
	}

	err = tmpl.Execute(w, jone)
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/if-else
func templateIfElseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateIfElseHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	tmpl := template.New("tmpl-if-else")
	// Person.Name, Person.Email, Friend.Name
	tmpl, err := tmpl.Parse(`<h1>{{.Name}} Information</h1>
		{{if .PrintEmail}}
			<h2>Email List:</h2>
			{{range .Email}}
				Email: {{.}} <br />
			{{end}}
		{{end}}
		{{if .PrintFriend}}
			<h2>Friends:</h2>
			{{with .Friends}}
				{{range .}}
					Friend name: {{.}} <br />
				{{end}}
			{{end}}
		{{else}}
			<h2>Friends: is empty</h2>
		{{end}}`)
	if err != nil {
		fmt.Println("template.Parse() error with:", err)
		return
	}


	type Person struct {
		Name   string
		PrintEmail bool
		Email  []string
		PrintFriend bool
		Friends []string
	}

	jone := Person{
		Name: "jone",
		PrintEmail: true,
		Email: []string{"jone@gmail.com", "jone@qq.com"},
		PrintFriend: false,
		Friends: []string{"amy", "jack"},
	}

	err = tmpl.Execute(w, jone)
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/pipeline
// pass data over template buildin function by pipeline, '|'
func templatePipelineHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templatePipelineHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	tmpl := template.New("tmpl-pipeline")
	tmpl, err := tmpl.Parse(`<h1>Pipeline Demo</h1>
		{{if . | not }}
			<h2>if branch</h2>
		{{else}}
			<h2>else branch</h2>
		{{end}}`)
	// try below two case:
	// {{if . | not }}
	// {{if . | not | not }}

	if err != nil {
		fmt.Println("template.Parse() error with:", err)
		return
	}

	branch := false
	err = tmpl.Execute(w, branch)
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/variable
// pass data over template function by pipeline, '|'
func templateVariableHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateVariableHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	tmpl := template.New("tmpl-variable")
	tmpl, err := tmpl.Parse(`<h1>Variable Demo</h1>
		{{$TmplName := .Name}}
		{{$tmp := "tmp-data"}}
		{{with .Info}}
			[template-Name:{{$TmplName}}] <br />
			[variable-in-tmplate:{{$tmp}}] <br />
			[Info:{{.}}] <br />
		{{end}}`)

	if err != nil {
		fmt.Println("template.Parse() error with:", err)
		return
	}

	type templateDemo struct {
		Name string
		Info string
	}

	err = tmpl.Execute(w, templateDemo{
		Name: "template-demo-Name",
		Info: "information data string",
	})
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/function
func Upper(args ...interface{}) string {
	if len(args) != 1 {
		return "function do nothing"
	}

	str, ok := args[0].(string)
	if !ok {
		return "args type not string"
	}

	return strings.ToUpper(str)
}

func templateFunctionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateFunctionHandler visit %s\n", r.RemoteAddr, r.URL.Path)

	tmpl := template.New("tmpl-function")
	tmpl.Funcs(template.FuncMap{"upperFunc": Upper})
	tmpl, err := tmpl.Parse(`<h1>Function Demo</h1>
		[before:{{.}}] <br />	
		[after:{{. | upperFunc}}] <br />`)

	if err != nil {
		fmt.Println("template.Parse() error with:", err)
		return
	}

	err = tmpl.Execute(w, "string-demo")
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/file
func templateFileHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	tmpl, err := template.ParseFiles("example.html")
	//tmpl, err := template.ParseFiles("demo.html")
	// tmpl := template.Must(template.ParseFiles("example.html")) // this is better
	if err != nil {
		fmt.Println("template.ParseFiles() error with:", err)
		return
	}

	err = tmpl.Execute(w,nil)
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}

// curl -i http://127.0.0.1:80/template/embedded-files
func templateEmbeddedFilesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s]==> templateEmbeddedFilesHandler visit %s\n", r.RemoteAddr, r.URL.Path)
	tmpl, err := template.ParseFiles("title.html", "example.html", "string.html")
	if err != nil {
		fmt.Println("template.ParseFiles() error with:", err)
		return
	}

	//err = tmpl.ExecuteTemplate(w, "demo", nil)
	err = tmpl.ExecuteTemplate(w, "example", nil)
	//err = tmpl.ExecuteTemplate(w, "string", nil)
	if err != nil {
		fmt.Println("template.Execute() error with:", err)
		return
	}
}


// Reference:
//  https://colobu.com/2019/11/05/Golang-Templates-Cheatsheet/
//  https://learnku.com/docs/build-web-application-with-golang/074-template-processing/3198
//  https://www.cnblogs.com/f-ck-need-u/p/10053124.html
func main() {
	fmt.Println("====== HTTP Server Start ======")

	// setp 1: 建立路由表，并注入相应 URL 的 handler
	mux := http.NewServeMux()

	// HTML from string
	mux.HandleFunc("/template/string", templateStringHandler)
	mux.HandleFunc("/template/insert-string", templateInsertHandler)
	mux.HandleFunc("/template/insert-object-field", templateInsertObjectHandler)
	mux.HandleFunc("/template/embedded-loop", templateEmbeddedHandler)
	mux.HandleFunc("/template/if-else", templateIfElseHandler)
	mux.HandleFunc("/template/pipeline", templatePipelineHandler)
	mux.HandleFunc("/template/variable", templateVariableHandler)
	mux.HandleFunc("/template/function", templateFunctionHandler)

	// HTML from file
	mux.HandleFunc("/template/file", templateFileHandler)
	mux.HandleFunc("/template/embedded-files", templateEmbeddedFilesHandler)

	// step 2: 启动 http 的监听，并且把这个路由表传递给相应的 TCP server
	server := &http.Server{Addr: "127.0.0.1:80", Handler: mux}
	_ = server.ListenAndServe()
}
