package main

import (
	"encoding/hex"
	"fmt"
	"net"
)

func main() {
	c, err := net.Dial("tcp", "localhost:4444")
	if err != nil {
		fmt.Println(err)
		return
	}
	b, err := hex.DecodeString("00000003303031")
	n, err := c.Write(b)
	buf := make([]byte, 1024)
	n, err = c.Read(buf)
	fmt.Println(string(buf[:n]))
	c.Close()
}
