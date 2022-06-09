****

![golang_net_interface.jpg](bVbGTux.jpeg)





# TODO

> https://colobu.com/2019/02/23/1m-go-tcp-connection/





# net package

## Reference

> https://colobu.com/2019/02/23/1m-go-tcp-connection/
>
> https://tonybai.com/2015/11/17/tcp-programming-in-golang/
>
> https://tonybai.com/2021/07/28/classic-blocking-network-tcp-stream-protocol-parsing-practice-in-go/



## demo

### TCP echo server

```go
package main

import (
	"log"
	"fmt"
	"bufio"
	"net"
)

const proto = "tcp"
const ipAddr = "localhost:25000"

const buffSize = 256

func main() {
	/* Listen TCP in localhost:2000 */
	proto := "tcp"

	l, err := net.Listen(proto, ipAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	fmt.Printf("====== Server Listen %s on %s ======\n", proto, ipAddr)

	/* do echo and close */
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		info := fmt.Sprintf("[%s --> %s]", conn.LocalAddr().String(), conn.RemoteAddr().String())
		fmt.Printf("new connection: %s\n", info)
		// go handleTcpEcho(conn)
		go handleTcpClient(conn)
	}
}

// perform only one Echo action for per TCP connection
func handleTcpEchoOnce(c net.Conn) {
	info := fmt.Sprintf("[%s --> %s]", c.LocalAddr().String(), c.RemoteAddr().String())

	/* read data */
	buff := make([]byte, buffSize)
	nr, err := c.Read(buff)
	if err != nil {
		fmt.Println("come across error:", err)
	}

	/* write back */
	nw, _ := c.Write(buff[:nr]) // do echo
	fmt.Printf("%s: %s", info, string(buff))

	// Shut down the connection.
	c.Close()
	fmt.Printf("%s: read %d bytes, write %d bytes. close and exit\n", info, nr, nw)
}

// perform echo until TCP connect close
func handleTcpClient(c net.Conn) {
	info := fmt.Sprintf("[%s --> %s]", c.LocalAddr().String(), c.RemoteAddr().String())
	reader := bufio.NewReader(c)
	totalRead, totalWrite := 0, 0
	for {
		/* read data */
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("come across error:", err)
			break
		}
		totalRead += len(line)

		/* echo back */
		nw, _ := c.Write(line) // do echo
		fmt.Printf("%s: %s", info, string(line))
		totalWrite += nw
	}

	c.Close()
	fmt.Printf("%s: read %d bytes, write %d bytes. close and exit\n", info, totalRead, totalWrite)
}
```



```go
/* demo for TCP client */
telnet localhost 25000
```







## Reference

> https://dev.to/hgsgtk/how-go-handles-network-and-system-calls-when-tcp-server-1nbd



## key point







## interface

### net.Conn

- 就 TCP 而言，net package 不应该为 `net.TCPConn` 裹上一层 buf。因为 TCP 本身就是 stream 式的协议，应用怎么解决粘包问题，是调用者的事。所以标准库在不知道分割标准的前提下，直接为 `net.TCPConn` 裹上一层 buf，实在是不合适
  - 显然，HTTP 不一样。HTTP 已经有了标准的应用层包分割的标准，所以直接在标准库内部裹上 buf，分包也是可以的。甚至是对使用者更友好的

```go
// Conn is a generic stream-oriented network connection.

//
// Multiple goroutines may invoke methods on a Conn simultaneously.
type Conn interface {
	// Read reads data from the connection.
	// Read can be made to time out and return an error after a fixed
	// time limit; see SetDeadline and SetReadDeadline.
	Read(b []byte) (n int, err error)

	// Write writes data to the connection.
	// Write can be made to time out and return an error after a fixed
	// time limit; see SetDeadline and SetWriteDeadline.
	Write(b []byte) (n int, err error)

	// Close closes the connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
	Close() error

	// LocalAddr returns the local network address.
	LocalAddr() Addr

	// RemoteAddr returns the remote network address.
	RemoteAddr() Addr

	// SetDeadline sets the read and write deadlines associated
	// with the connection. It is equivalent to calling both
	// SetReadDeadline and SetWriteDeadline.
	//
	// A deadline is an absolute time after which I/O operations
	// fail instead of blocking. The deadline applies to all future
	// and pending I/O, not just the immediately following call to
	// Read or Write. After a deadline has been exceeded, the
	// connection can be refreshed by setting a deadline in the future.
	//
	// If the deadline is exceeded a call to Read or Write or to other
	// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
	// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
	// The error's Timeout method will return true, but note that there
	// are other possible errors for which the Timeout method will
	// return true even if the deadline has not been exceeded.
	//
	// An idle timeout can be implemented by repeatedly extending
	// the deadline after successful Read or Write calls.
	//
	// A zero value for t means I/O operations will not time out.
	SetDeadline(t time.Time) error

	// SetReadDeadline sets the deadline for future Read calls
	// and any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the deadline for future Write calls
	// and any currently-blocked Write call.
	// Even if write times out, it may return n > 0, indicating that
	// some of the data was successfully written.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error
}

```



#### net.TCPConn

```go
// TCPConn is an implementation of the Conn interface for TCP network
// connections.
type TCPConn struct {
    // embeded 的方式直接继承 net.conn 的 Read() + Write()
	conn
}

type conn struct {
	fd *netFD
}

// 将 netFD 封装为 TCPConn
func newTCPConn(fd *netFD) *TCPConn {
	c := &TCPConn{conn{fd}}
	setNoDelay(c.fd, true) // 默认立马发包出去
	return c
}
```



**读写能力**

- 显然 `net.TCPConn` 是具有 socket read\write 能力的，==是比较原始的裸 IO，不包含 buffer==



`io.Reader`, `io.Writer`

- 直接继承 `net.conn` 的 `Write()` 跟 `Read()`

```bash
> net.(*TCPConn).Write() <autogenerated>:1 (hits goroutine(23):3 total:3) (PC: 0x4c76a6)
Warning: debugging optimized function
(dlv) s
> net.(*conn).Write() /usr/local/go/src/net/net.go:191 (PC: 0x4bb4ea)
Warning: debugging optimized function
   186:		}
   187:		return n, err
   188:	}
   189:	
   190:	// Write implements the Conn Write method.
// 继承的是 conn 的 Write()
=> 191:	func (c *conn) Write(b []byte) (int, error) {
   192:		if !c.ok() {
   193:			return 0, syscall.EINVAL
   194:		}
   195:		n, err := c.fd.Write(b)
   196:		if err != nil {
(dlv) bp
Breakpoint runtime-fatal-throw (enabled) at 0x433040 for runtime.throw() /usr/local/go/src/runtime/panic.go:1188 (0)
Breakpoint unrecovered-panic (enabled) at 0x4333a0 for runtime.fatalpanic() /usr/local/go/src/runtime/panic.go:1271 (0)
	print runtime.curg._panic.arg
Breakpoint 1 (enabled) at 0x4c76a6 for net.(*TCPConn).Write() <autogenerated>:1 (3)
(dlv) bt
0  0x00000000004bb4ea in net.(*conn).Write
   at /usr/local/go/src/net/net.go:191
1  0x00000000004c76ce in net.(*TCPConn).Write
   at <autogenerated>:1
2  0x00000000004c9e18 in main.handleTcpClient
   at ./main.go:75
3  0x00000000004c9814 in main.main·dwrap·2
   at ./main.go:36
4  0x0000000000460c41 in runtime.goexit
   at /usr/local/go/src/runtime/asm_amd64.s:1581
```



```go
>> net/net.go:
// Read implements the Conn Read method.
func (c *conn) Read(b []byte) (int, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	n, err := c.fd.Read(b)
	if err != nil && err != io.EOF {
		err = &OpError{Op: "read", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}

>> net/fd_posix.go:
func (fd *netFD) Read(p []byte) (n int, err error) {
	n, err = fd.pfd.Read(p)
	runtime.KeepAlive(fd)
	return n, wrapSyscallError(readSyscallName, err)
}

>> src/internal/poll/fd_unix.go:
// Read implements io.Reader.
func (fd *FD) Read(p []byte) (int, error) {
        ....
        for {
         	   // 实际上就是 syscall.Read(fd.Sysfd, p)
                n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
                if err != nil {
                        n = 0
                        if err == syscall.EAGAIN && fd.pd.pollable() {
                        // 实际上设置的是非阻塞 IO，但是对于 goroutine 表现为阻塞 IO
                                if err = fd.pd.waitRead(fd.isFile); err == nil {
                                        // 本 goroutine 正式开始读取数据
                                        continue
                                }
                        }
                }
                // 成功读取数据，返回
                err = fd.eofError(n, err)
                return n, err
        }
}

>>  src/internal/poll/fd_poll_runtime.go:
// block 住这个 goroutine
func (pd *pollDesc) wait(mode int, isFile bool) error {
        if pd.runtimeCtx == 0 {
                return errors.New("waiting for unsupported file type")
        }
        res := runtime_pollWait(pd.runtimeCtx, mode)
        return convertErr(res, isFile)
}

>> src/runtime/netpoll.go:
// poll_runtime_pollWait, which is internal/poll.runtime_pollWait,
// waits for a descriptor to be ready for reading or writing,
// according to mode, which is 'r' or 'w'.
// This returns an error code; the codes are defined above.
//go:linkname poll_runtime_pollWait internal/poll.runtime_pollWait
func poll_runtime_pollWait(pd *pollDesc, mode int) int {
        errcode := netpollcheckerr(pd, int32(mode))
        if errcode != pollNoError {
                return errcode
        }
        // As for now only Solaris, illumos, and AIX use level-triggered IO.
        if GOOS == "solaris" || GOOS == "illumos" || GOOS == "aix" {
                netpollarm(pd, mode)
        }
        for !netpollblock(pd, int32(mode), false) {
                errcode = netpollcheckerr(pd, int32(mode))
                if errcode != pollNoError {
                        return errcode
                }
                // Can happen if timeout has fired and unblocked us,
                // but before we had a chance to run, timeout has been reset.
                // Pretend it has not happened and retry.
        }
        return pollNoError
}

>> src/runtime/netpoll.go:
// returns true if IO is ready, or false if timedout or closed
// waitio - wait only for completed IO, ignore errors
// Concurrent calls to netpollblock in the same mode are forbidden, as pollDesc
// can hold only a single waiting goroutine for each mode.
func netpollblock(pd *pollDesc, mode int32, waitio bool) bool {
		.......
        // need to recheck error states after setting gpp to pdWait
        // this is necessary because runtime_pollUnblock/runtime_pollSetDeadline/deadlineimpl
        // do the opposite: store to closing/rd/wd, membarrier, load of rg/wg
        if waitio || netpollcheckerr(pd, mode) == 0 {
            	// 看，就在这里，把 goroutine park 掉
                gopark(netpollblockcommit, unsafe.Pointer(gpp), waitReasonIOWait, traceEvGoBlockNet, 5)
        }
		......
}

//===========================================================
>> net/net.go:
// Write implements the Conn Write method.
func (c *conn) Write(b []byte) (int, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	n, err := c.fd.Write(b)
	if err != nil {
		err = &OpError{Op: "write", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}

>> net/fd_posix.go:
func (fd *netFD) Write(p []byte) (nn int, err error) {
    // 跟 Read 基本同理，读不到的话，就 park 掉 goroutine
	nn, err = fd.pfd.Write(p)
	runtime.KeepAlive(fd)
	return nn, wrapSyscallError(writeSyscallName, err)
}
```





`io.ReaderFrom`

```go
// ReadFrom implements the io.ReaderFrom ReadFrom method.
func (c *TCPConn) ReadFrom(r io.Reader) (int64, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	n, err := c.readFrom(r)
	if err != nil && err != io.EOF {
		err = &OpError{Op: "readfrom", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}
```







### net.Addr

- 这是一个很好的抽象，统合了 tcp、udp、unix 的多种情况
- 因为一个网络地址，其实就是 `socket 种类 + 三层协议种类 + 三层地址选择 + 四层协议种类 + 四层端口选择`

```go
// Addr represents a network end point address.
//
// The two methods Network and String conventionally return strings
// that can be passed as the arguments to Dial, but the exact form
// and meaning of the strings is up to the implementation.
type Addr interface {
	Network() string // name of the network (for example, "tcp", "udp")
	String() string  // string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
}
```



#### net.IP

```go
// An IP is a single IP address, a slice of bytes.
// Functions in this package accept either 4-byte (IPv4)
// or 16-byte (IPv6) slices as input.
//
// Note that in this documentation, referring to an
// IP address as an IPv4 address or an IPv6 address
// is a semantic property of the address, not just the
// length of the byte slice: a 16-byte slice can still
// be an IPv4 address.
// 并不属于 net.Addr interface, 因为 net.Addr 还包括的四层协议种类的信息
// 单纯的 IP 并不包括四层信息
// 但是 net.IP 是 net.TCPAddr, net.UDPAddr 的成员之一
type IP []byte
```





#### net.TCPAddr

```go
// TCPAddr represents the address of a TCP end point.
type TCPAddr struct {
	IP   IP
	Port int
	Zone string // IPv6 scoped addressing zone
}

// Network returns the address's network name, "tcp".
func (a *TCPAddr) Network() string { return "tcp" }

func (a *TCPAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	ip := ipEmptyString(a.IP)
	if a.Zone != "" {
		return JoinHostPort(ip+"%"+a.Zone, itoa.Itoa(a.Port))
	}
	return JoinHostPort(ip, itoa.Itoa(a.Port))
}
```



#### net.UDPAddr

```go
// UDPAddr represents the address of a UDP end point.
type UDPAddr struct {
	IP   IP
	Port int
	Zone string // IPv6 scoped addressing zone
}

// Network returns the address's network name, "udp".
func (a *UDPAddr) Network() string { return "udp" }

func (a *UDPAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	ip := ipEmptyString(a.IP)
	if a.Zone != "" {
		return JoinHostPort(ip+"%"+a.Zone, itoa.Itoa(a.Port))
	}
	return JoinHostPort(ip, itoa.Itoa(a.Port))
}
```









### net.Listener

- 作为一个 `net.Listener` 核心的功能是：知道自己监听什么端口，能够 `close()`, `accept()`, 所以也就有了下面的抽象。至于怎么实例化、配置这个具体的 `net.Listener` 实例，那是 struct 要考虑的事情，而不是 interface

```go
// A Listener is a generic network listener for stream-oriented protocols.
//
// Multiple goroutines may invoke methods on a Listener simultaneously.
// call flow:
// 1. 创建一个实现了 net.Listener interface 的实例（比如：net.sysListener struct）
// 2. 自己完成：创建 socket、bind、listen 的工作
// 3. 开始调用 net.Listener.Accept() 等待连接进来
/**
 * 为什么不把 bind、listen 也抽象出来呢？
 * 首先 net.Listener 是 a generic network listener for stream-oriented protocols
 * bind 不 bind 嘛，看情况而定。但是 Accept() 是必然的。Golang 追求的是小 interface
 * 所以你是可以在 net.Listener interface 的基础上，在套一层含有 bind、listen 的 interface
 * 但是你这个 wrapper 就已经很具体了，没啥通用性，那还不如扔进 struct 里面，而不是 interface
 */
type Listener interface {
	// Accept waits for and returns the next connection to the listener.
	Accept() (Conn, error)

	// Close closes the listener.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the listener's network address.
	Addr() Addr
}
```



#### net.TCPListener

```go
// TCPListener is a TCP network listener. Clients should typically
// use variables of type Listener instead of assuming TCP.
type TCPListener struct {
	fd *netFD
	lc ListenConfig
}

// Accept implements the Accept method in the Listener interface; it
// waits for the next call and returns a generic Conn.
// 实际上底层依然是调用 net.TCPListener.fd.accept()
func (l *TCPListener) Accept() (Conn, error) {
	if !l.ok() {
		return nil, syscall.EINVAL
	}
	c, err := l.accept()
	if err != nil {
		return nil, &OpError{Op: "accept", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return c, nil
}

func (ln *TCPListener) accept() (*TCPConn, error) {
	fd, err := ln.fd.accept()
	if err != nil {
		return nil, err
	}
	tc := newTCPConn(fd)
	if ln.lc.KeepAlive >= 0 {
		setKeepAlive(fd, true)
		ka := ln.lc.KeepAlive
		if ln.lc.KeepAlive == 0 {
			ka = defaultTCPKeepAlive
		}
		setKeepAlivePeriod(fd, ka)
	}
	return tc, nil
}
```









## function

















## net 与底层 socket

### 初始化监听 socket

> 调用 `tcpListener := net.Listen("tcp", "localhost:8888")`
>
> 核心目的：
>
> 1. 设置监听的协议类型
> 2. 设置监听的端口
> 3. 设置监听的地址
> 4. 根据配置，创建监听的 socket

![img](mulkk8vrj9j8v1zv6my6.png)

<center>IP:Port 解析过程</center>

![A diagram describing the structure inside of the net package](5tid0ud2obcjgv57rchi.png)

<center>TCP server 过程</center>

```go
net/dial.go:
func Listen(network, address string) (Listener, error) {
        var lc ListenConfig
        // 使用默认的 ListenConfig, 内置 DNS 解析器
        return lc.Listen(context.Background(), network, address)
}

func (lc *ListenConfig) Listen(ctx context.Context, network, address string) (Listener, error) {
		// 解析 DNS、协议种类、端口
        addrs, err := DefaultResolver.resolveAddrList(ctx, "listen", network, address, nil)

        sl := &sysListener{
                ListenConfig: *lc,
                network:      network,
                address:      address,
        }
        var l Listener
        la := addrs.first(isIPv4) // 执行 isIPv4() 的检查，找出第一个满足条件的地址
        switch la := la.(type) {
        case *TCPAddr:
			   // 绑定、监听响应的地址
                // 创建一个 TCP 的 socket，并 bind、listen
                l, err = sl.listenTCP(ctx, la)
		......
        }

        return l, nil
}

net/tcpsock_posix.go:
func (sl *sysListener) listenTCP(ctx context.Context, laddr *TCPAddr) (*TCPListener, error) {
		// 创建 SOCK_STREAM 类型的 socket
        fd, err := internetSocket(ctx, sl.network, laddr, nil, syscall.SOCK_STREAM, 0, "listen", sl.ListenConfig.Control)
        if err != nil {
                return nil, err
        }

		// wrap listen socket fd as a TCPListener
        return &TCPListener{fd: fd, lc: sl.ListenConfig}, nil
}

net/ipsock_posix.go:
func internetSocket(ctx context.Context, net string, laddr, raddr sockaddr, sotype, proto int, mode string, ctrlFn func(string, string, syscall.RawConn) error) (fd *netFD, err error) {
        if (runtime.GOOS == "aix" || runtime.GOOS == "windows" || runtime.GOOS == "openbsd") && mode == "dial" && raddr.isWildcard() {
                raddr = raddr.toLocal(net)
        }
        family, ipv6only := favoriteAddrFamily(net, laddr, raddr, mode)
        return socket(ctx, net, family, sotype, proto, ipv6only, laddr, raddr, ctrlFn)
}

net/sock_posix.go:
// socket returns a network file descriptor that is ready for
// asynchronous I/O using the network poller.
func socket(ctx context.Context, net string, family, sotype, proto int, ipv6only bool, laddr, raddr sockaddr, ctrlFn func(string, string, syscall.RawConn) error) (fd *netFD, err error) {
        // 1. centos 7 的话，会走 src/net/sock_cloexec.go:sysSocket()
        // 2. 再走 hook_unix.go:syscall.Socket()
        // TCP 的话，family = 2 (syscall.AF_INET), proto = 0, sotype = 1 (syscall.SOCK_STREAM)
        s, err := sysSocket(family, sotype, proto)
        if err != nil {
                return nil, err
        }
	    .........
        // 封装原始 fd
        if fd, err = newFD(s, family, sotype, net); err != nil {
                poll.CloseFunc(s)
                return nil, err
        }

        // This function makes a network file descriptor for the
        // following applications:
        //
        // - An endpoint holder that opens a passive stream
        //   connection, known as a stream listener
        if laddr != nil && raddr == nil {
                switch sotype {
                case syscall.SOCK_STREAM, syscall.SOCK_SEQPACKET:
                        if err := fd.listenStream(laddr, listenerBacklog(), ctrlFn); err != nil {
                                fd.Close()
                                return nil, err
                        }
                        return fd, nil
                case syscall.SOCK_DGRAM:
					..........
                }
        }
    	.....
        // dailer 相关的
        return fd, nil
}

net/sock_cloexec.go:
func sysSocket(family, sotype, proto int) (int, error) {
		// SOCK_NONBLOCK: 看，虽然 read、write 的时候，是 block，但实际设置的确实非阻塞
        /* SOCK_CLOEXEC(close on exec() call): 
         * Note that the use of this flag is essential in some multithreaded programs.
         * 这个 fd 我在 fork 子进程后执行 exec() 时就关闭，避免父进程重启后，无法再次监听这个端口（因为子进程占用着）
         * 这就意味着，这个 socket fd 是不能被子进程继承的。
         * 既可以是是避免 fd leak，也可以是权限控制，避免高权限进程 open 的 fd，被低权限的子进程使用
         */
        s, err := socketFunc(family, sotype|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, proto)

        switch err {
        case nil:
                // 通常这里就会正常返回了
                return s, nil
        default:
                return -1, os.NewSyscallError("socket", err)
        case syscall.EPROTONOSUPPORT, syscall.EINVAL:
            // 兼容不支持 SOCK_NONBLOCK、SOCK_CLOEXEC 旧版本系统
            // If we get an EINVAL error on Linux
            // or EPROTONOSUPPORT error on FreeBSD, fall back to using
            // socket without them.
        }
	    .........
}

// net fd 相关
net/fd_posix.go:
// Network file descriptor.
type netFD struct {
	pfd poll.FD

	// immutable until Close
	family      int
	sotype      int
	isConnected bool // handshake completed or use of association with peer
	net         string
	laddr       Addr
	raddr       Addr
}

net/fd_unix.go:
func newFD(sysfd, family, sotype int, net string) (*netFD, error) {
	ret := &netFD{
		pfd: poll.FD{
			Sysfd:         sysfd,
			IsStream:      sotype == syscall.SOCK_STREAM,
			ZeroReadIsEOF: sotype != syscall.SOCK_DGRAM && sotype != syscall.SOCK_RAW,
		},
		family: family,
		sotype: sotype,
		net:    net,
	}
	return ret, nil
}

// 完成 bind + listen
net/sock_posix.go:
func (fd *netFD) listenStream(laddr sockaddr, backlog int, ctrlFn func(string, string, syscall.RawConn) error) error {
	var err error
	if err = setDefaultListenerSockopts(fd.pfd.Sysfd); err != nil {
		return err
	}
	var lsa syscall.Sockaddr
	if lsa, err = laddr.sockaddr(fd.family); err != nil {
		return err
	}
	if ctrlFn != nil {
		c, err := newRawConn(fd)
		if err != nil {
			return err
		}
		if err := ctrlFn(fd.ctrlNetwork(), laddr.String(), c); err != nil {
			return err
		}
	}
	if err = syscall.Bind(fd.pfd.Sysfd, lsa); err != nil {
		return os.NewSyscallError("bind", err)
	}
	if err = listenFunc(fd.pfd.Sysfd, backlog); err != nil {
		return os.NewSyscallError("listen", err)
	}
	if err = fd.init(); err != nil {
		return err
	}
	lsa, _ = syscall.Getsockname(fd.pfd.Sysfd)
	fd.setAddr(fd.addrFunc()(lsa), nil)
	return nil
}
```





### 监听 socket 开始 accept TCP 链接

> 调用 `conn, err := net.Listener.Accept()`
>
> 目的：block 等待 TCP 连接的到来

![A diagram describing the structure inside of the net package](gjt8ly04e2bghr5ngfgh.png)

<center>Accept 过程</center>

- 总体来说没什么特别，就是对原始的系统调用封装而已，返回一个 `net.Conn` interface, 底层是 `net.TCPConn` struct

```go
应为 listener 是个 interface，在 TCP 的 case 中，那就是 *TCPListener struct

net/tcpsock.go:
type TCPListener struct {
	fd *netFD
	lc ListenConfig
}

// Accept implements the Accept method in the Listener interface; it
// waits for the next call and returns a generic Conn.
func (l *TCPListener) Accept() (Conn, error) {
	if !l.ok() {
		return nil, syscall.EINVAL
	}
	c, err := l.accept()
	if err != nil {
		return nil, &OpError{Op: "accept", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return c, nil
}

net/tcpsock_posix.go:
func (ln *TCPListener) accept() (*TCPConn, error) {
	fd, err := ln.fd.accept()
	if err != nil {
		return nil, err
	}
	tc := newTCPConn(fd)
	if ln.lc.KeepAlive >= 0 {
		setKeepAlive(fd, true)
		ka := ln.lc.KeepAlive
		if ln.lc.KeepAlive == 0 {
			ka = defaultTCPKeepAlive
		}
		setKeepAlivePeriod(fd, ka)
	}
	return tc, nil
}

net/fd_unix.go:
func (fd *netFD) accept() (netfd *netFD, err error) {
	d, rsa, errcall, err := fd.pfd.Accept()
	if err != nil {
		if errcall != "" {
			err = wrapSyscallError(errcall, err)
		}
		return nil, err
	}

    // 裹上 netFD 的封装
	if netfd, err = newFD(d, fd.family, fd.sotype, fd.net); err != nil {
		poll.CloseFunc(d)
		return nil, err
	}
	if err = netfd.init(); err != nil {
		netfd.Close()
		return nil, err
	}
    
    // 填充底层 socket 信息
	lsa, _ := syscall.Getsockname(netfd.pfd.Sysfd)
	netfd.setAddr(netfd.addrFunc()(lsa), netfd.addrFunc()(rsa))
	return netfd, nil
}

// fd 相关
internal/poll/fd_unix.go:
// Accept wraps the accept network call.
func (fd *FD) Accept() (int, syscall.Sockaddr, string, error) {
	if err := fd.readLock(); err != nil {
		return -1, nil, "", err
	}
	defer fd.readUnlock()

	if err := fd.pd.prepareRead(fd.isFile); err != nil {
		return -1, nil, "", err
	}
	for {
		s, rsa, errcall, err := accept(fd.Sysfd)
		if err == nil {
			return s, rsa, "", err
		}
		switch err {
		case syscall.EINTR:
			continue
		case syscall.EAGAIN:
			if fd.pd.pollable() {
				if err = fd.pd.waitRead(fd.isFile); err == nil {
					continue
				}
			}
		case syscall.ECONNABORTED:
			// This means that a socket on the listen
			// queue was closed before we Accept()ed it;
			// it's a silly error, so try again.
			continue
		}
		return -1, nil, errcall, err
	}
}

```









### 读 socket

参考上面 `net.TCPConn` 的分析章节



### 写 socket

参考上面 `net.TCPConn` 的分析章节





























































# net/http package























































