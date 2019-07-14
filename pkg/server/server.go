package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrServerClosed 服务已关闭
var ErrServerClosed = errors.New("Server closed")

var bufPool = sync.Pool{
	New: func() interface{} {
		// 目前换成长度最大4K，以后写别的服务再说
		return make([]byte, 4*1024)
	},
}

var pkgBufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func pkgSplit(data []byte) (advence int, token []byte, err error) {
	if len(data) >= 4 {
		pkgLen := int(binary.BigEndian.Uint32(data[:4]))
		if len(data) >= 4+pkgLen {
			return 4 + pkgLen, data[4 : 4+pkgLen], nil
		}
	}

	return 0, nil, nil
}

func connServe(ctx context.Context, conn net.Conn, done func()) {
	buf := bufPool.Get().([]byte)
	log := log.WithFields(logrus.Fields{
		"LocalAddr":  conn.LocalAddr(),
		"RemoteAddr": conn.RemoteAddr(),
	})

	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
		log.Debug("exist serve")
		defer bufPool.Put(buf)
		defer done()
	}()

	t := buf
	needRead := true
	readableCount := 0

	for {
		if needRead {
			log.Debug("begin recive length field package")
			select {
			case <-ctx.Done():
				log.Debug("context done")
				return
			default:
			}
			conn.SetReadDeadline(time.Now().Add(1 * 1e9))
			n, err := conn.Read(t[readableCount:])
			if err != nil {
				if ne, ok := err.(net.Error); ok {
					if ne.Timeout() {
						continue
					}
				}

				if err == io.EOF {
					return
				}
				log.Error(err)
				return
			}
			readableCount += n
		}

		advence, pkg, err := pkgSplit(t[:readableCount])

		if err != nil {
			return
		}

		if pkg != nil {
			log.Debugf("length field package recived, data:%X", pkg)
			t = t[advence:]
			readableCount -= advence
			// todo
			println(hex.EncodeToString(pkg))
			b, _ := hex.DecodeString("00000003313233")
			conn.Write(b)
			needRead = false
			continue
		}

		if readableCount != 0 {
			copy(buf, t[:readableCount])
		}
		t = buf
		needRead = true
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

// Serve TODO
func (s *Server) Serve(l *net.TCPListener) error {
	log.Debugf("begin accepting, addr `%v`", l.Addr())
	var tempDelay time.Duration
	s.waitGroup.Add(1)
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		l.Close()
		cancel()
		log.Debugf("end accepting, addr `%v`", l.Addr())
		s.waitGroup.Done()
	}()

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

		log.Debugf("client `%v` connected", rwc.RemoteAddr())
		tempDelay = 0
		s.waitGroup.Add(1)
		go connServe(ctx, rwc, func() {
			s.waitGroup.Done()
			rwc.Close()
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
	log.Infof("listen and serve, addr `%v`", laddr)
	return s.Serve(listener)
}

// Shutdown 关闭服务器
func (s *Server) Shutdown() {
	if s.closeChan != nil {
		close(s.closeChan)
	}
	log.Info("begin shutdown")
	s.waitGroup.Wait()
}

// ListenAndServe TODO
func ListenAndServe(addr string) error {
	server := &Server{}
	return server.ListenAndServe(addr)
}
