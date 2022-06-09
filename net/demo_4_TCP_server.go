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
