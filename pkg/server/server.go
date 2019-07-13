package server

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// ErrServerClosed 服务已关闭
var ErrServerClosed = errors.New("Server closed")

var bufPool = sync.Pool{
	New: func() interface{} {
		// 目前换成长度最大4K，以后写别的服务再说
		return make([]byte, 4*1024)
	},
}

type conn struct {
	rwc net.Conn
}

func (c *conn) serve(ctx context.Context, done func()) {
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)
	for {
		select {
		case <-ctx.Done():
			done()
			return
		default:
		}
		n, err := c.rwc.Read(buf)

		if err != nil {
			c.rwc.Close()
			return
		}

		n, err = c.rwc.Write(buf[:n])

		if err != nil {
			c.rwc.Close()
			return
		}
	}
}

// Server TODO
type Server struct {
	waitGroup sync.WaitGroup
	closeChan chan struct{}
}

func (s *Server) getCloseChan() <-chan struct{} {
	if s.closeChan == nil {
		s.closeChan = make(chan struct{})
	}
	return s.closeChan
}

func (s *Server) newConn(rwc net.Conn) *conn {
	c := &conn{rwc}
	return c
}

// Serve TODO
func (s *Server) Serve(l *net.TCPListener) error {
	defer l.Close()
	s.waitGroup.Add(1)
	defer s.waitGroup.Done()
	var tempDelay time.Duration
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-s.getCloseChan():
			return ErrServerClosed
		default:
		}
		l.SetDeadline(time.Now().Add(1 * 1e9))
		rwc, err := l.Accept()

		// 错误处理，主要处理临时错误，抄自http/server.go
		if err != nil {
			if ne, ok := err.(net.Error); ok {
				if ne.Timeout() {
					continue
				}
				if ne.Temporary() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}
					time.Sleep(tempDelay)
					continue
				}
			}
			return err
		}
		// end 错误处理

		tempDelay = 0
		c := s.newConn(rwc)
		s.waitGroup.Add(1)
		go c.serve(ctx, func() {
			s.waitGroup.Done()
		})
	}
}

// ListenAndServe TODO
func (s *Server) ListenAndServe(addr string) error {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}
	return s.Serve(listener)
}

// Shutdown 关闭服务器
func (s *Server) Shutdown() {
	if s.closeChan != nil {
		close(s.closeChan)
	}
	s.waitGroup.Wait()
}

// ListenAndServe TODO
func ListenAndServe(addr string) error {
	server := &Server{}
	return server.ListenAndServe(addr)
}
