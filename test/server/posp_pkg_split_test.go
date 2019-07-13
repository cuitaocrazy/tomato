package server

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/cuitaocrazy/tomato/pkg/server"
)

func TestPospPkgSplit(t *testing.T) {
	comm(t)
	empty(t)
	pkgErr(t)
}

func comm(t *testing.T) {
	data, _ := hex.DecodeString("000000013100000002313200000003313233")
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(server.PospPkgSplit)
	as := make([]string, 0)
	for scanner.Scan() {
		as = append(as, scanner.Text())
	}

	if scanner.Err() != nil {
		t.Error(scanner.Err())
	}

	if len(as) != 3 {
		t.Error("期望3， 实际" + string(len(as)))
	}

	if as[0] != "1" {
		t.Error("期望1， 实际" + as[0])
	}

	if as[1] != "12" {
		t.Error("期望12， 实际" + as[1])
	}

	if as[2] != "123" {
		t.Error("期望123， 实际" + as[2])
	}
}

func empty(t *testing.T) {
	data := []byte{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(server.PospPkgSplit)

	if scanner.Scan() {
		t.Error("期望false")
	}

	if scanner.Err() != nil {
		t.Error("期望无错误")
	}
}

func pkgErr(t *testing.T) {
	data, _ := hex.DecodeString("0000000231")
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(server.PospPkgSplit)

	scanner.Scan()

	if scanner.Err() != server.ErrPkg {
		t.Error("期望错误是Package error")
	}
}
