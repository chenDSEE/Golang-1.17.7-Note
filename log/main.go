package main

import (
	"fmt"
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
	logger := log.New(logFile, "[Debug] ", log.Lshortfile) // 记录文件、行号
	logger.Print("call Print: line1") // 会自动换行
	logger.Println("call Println: line2")

	// change configuration for log package
	logger.SetPrefix("[Info] ")
	logger.SetFlags(log.Ldate)   // 开启日期记录
	logger.SetOutput(os.Stdout)	 // 将日志重定向输出到屏幕
	logger.Print("Info check stdout")
}