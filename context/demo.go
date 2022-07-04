package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"context"
	"time"
	"math/rand"
	"sync"
)

type User struct {
	id   int
	name string
	info string
}

// 即使被 cancel 之后，query 的结果还是要处理的，只不过可以直接不理会而已
func queryName(nameCH chan string, uid int) {
	time.Sleep(50 * time.Millisecond)
	nameCH <- fmt.Sprintf("name-%d", uid) // never block
}

func queryInfo(infoCH chan string, uid int) {
	time.Sleep(50 * time.Millisecond)
	infoCH <- fmt.Sprintf("info-%d", uid) // never block
}

// 也可以并发去查不同的字段，这时候 checkUserInfo 是负责监听上层 ctx.Done()，以及 merge 其他几个 channel
func checkUserInfo(ctx context.Context, reslutCH chan User, uid int) bool {
	if rand.Int() % 2 == 0 {
		return false
	}
	
	// buffer 1 to avoid goroutine leak
	nameCH := make(chan string, 1)
	infoCH := make(chan string, 1)
	go queryName(nameCH, uid)
	go queryInfo(infoCH, uid)

	// 可以不启用单独的 goroutine 作为 merge + 监听
	// 因为 checkUserInfo() 已经是运行在一个独立的 goroutine 里了，
	// 所以这里就不再单独启动一个 goroutine 进行监听了
	// merge up all result
	var user User
	user.id = uid
	for i := 0; i < 2; i++ {
		select {
		case user.name = <- nameCH:
		case user.info = <- infoCH:
		case <- ctx.Done():
			// all channel will be GC, no goroutine will be hang forever
			fmt.Printf("User[%d] had been cancel\n", uid)
			return true
		}
	}

	if user.id != -1 {
		reslutCH <- user
	}
	return true
}

const queryNum = 10
const errorLimit = 3
// main 作为监控者，设定了 timeout，超时全部 checkUserInfo 都取消；出错次数监控，出错次数太多，全部 checkUserInfo 都取消；
// checkUserInfo 作为被监控者，需要监听 main 发过来的监控信号，
// 同时还要进行实际的查询动作，并发查询结果放到另一个 channel 里，
// 才能跟 ctx.Done() 进行多路复用的监听
func main() {

	fmt.Printf("====== request come for %d users information ======\n", queryNum)

	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
	defer cancel()

	resultCH := make(chan User, queryNum)

	/* check queryNum users information(fan-in) */
	var wg sync.WaitGroup
	wg.Add(queryNum)
	cnt := int32(0)
	for uid := 0; uid < queryNum; uid++ {
		go func(uid int) {
			if(checkUserInfo(ctx, resultCH, uid) == false) {
				atomic.AddInt32(&cnt, 1)
				fmt.Printf("checkUserInfo() fail for user[%d]\n", uid)
				if atomic.LoadInt32(&cnt) >= errorLimit {
					fmt.Printf("###!!! too many error, cancel all !!!###\n")
					cancel()
				}
			} else {
				fmt.Printf("checkUserInfo() success for user[%d]\n", uid)
			}
			wg.Done()
		}(uid)
	}

	/* print queryNum users information as a response */
	wg.Wait()
	close(resultCH)
	if atomic.LoadInt32(&cnt) < errorLimit {
		for user := range resultCH {
			fmt.Printf("User[id:%d][name:%s][info:%s]\n", user.id, user.name, user.info)
		}
	}

	time.Sleep(5 * time.Second) // wait all sub-goroutine be canceled and exit
	fmt.Printf("====== end with goroutine[%d] ======\n", runtime.NumGoroutine())
}


func init() {
	rand.Seed(time.Now().Unix())
}