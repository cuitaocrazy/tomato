package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cuitaocrazy/tomato/pkg/server"
)

func main() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	server := &server.Server{}
	go func() {
		sig := <-sigs
		fmt.Println(sig)
		server.Shutdown()
	}()

	err := server.ListenAndServe("localhost:4444")

	if err != nil {
		fmt.Println(err)
	}
}
