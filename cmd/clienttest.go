package main

import (
	"fmt"
	"net"
)

func main2() {
	c, err := net.Dial("tcp", "localhost:4444")
	if err != nil {
		fmt.Println(err)
		return
	}
	n, err := c.Write([]byte("hello"))
	buf := make([]byte, 1024)
	n, err = c.Read(buf)
	fmt.Println(string(buf[:n]))
	c.Close()
}
