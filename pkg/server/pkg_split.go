package server

import (
	"encoding/binary"
	"errors"
	"io"
)

// ErrPkg TODO
var ErrPkg = errors.New("Package error")

// PospPkgSplit POSP8583包分割器
// 由于Scanner不支持net.Conn的读取超时处理，现在弃用
func PospPkgSplit(data []byte, atEOF bool) (advence int, token []byte, err error) {
	if len(data) > 4 {
		pkgLen := int(binary.BigEndian.Uint32(data[:4]))

		if pkgLen > 2*2048 {
			return 0, nil, ErrPkg
		}

		if len(data) >= 4+pkgLen {
			return 4 + pkgLen, data[4 : 4+pkgLen], nil
		}

		if atEOF {
			return 0, nil, ErrPkg
		}

		return 0, nil, nil
	}

	if len(data) > 0 && atEOF {
		return 0, nil, ErrPkg
	}

	if atEOF {
		return 0, nil, io.EOF
	}

	return 0, nil, nil
}
