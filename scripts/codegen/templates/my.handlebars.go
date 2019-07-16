package codegen

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
)

//ErrLenNotMeet  长度错误
var ErrLenNotMeet = errors.New("Length does not meet the definition")

//ErrFieldUnpackFuncNotExist 字段解包函数不存在
var ErrFieldUnpackFuncNotExist = errors.New("field unpack function not exist")

//ErrBCD bcd错误
var ErrBCD = errors.New("BCD error")

var ErrValueTooLong = errors.New("value too long")
var ErrValueTooSmall = errors.New("value too small")

// #region unpack
func getBCD(b byte) (int, error) {
	h := int(b >> 4)
	l := int(b & 0xf)
	if h > 9 || l > 9 {
		return 0, ErrBCD
	}
	return h*10 + l, nil
}

func getBytesBCD(data []byte) (int, error) {
	byteLen := len(data)
	if byteLen > 4 {
		return 0, ErrBCD
	}
	if byteLen == 0 {
		return 0, nil
	}

	var bcd = 0
	for i, b := range data {
		ret, err := getBCD(b)
		if err != nil {
			return 0, err
		}
		e := int(math.Pow10((byteLen - i - 1) * 2))
		bcd += e * ret
	}
	return bcd, nil
}

func getBytesSlice(length int, data []byte) ([]byte, []byte, error) {
	if len(data) < length {
		return nil, nil, ErrLenNotMeet
	}
	return data[:length], data[length:], nil
}

func unpackVarLenFiled(lenType int, data []byte) ([]byte, []byte, error) {
	rawLen, newData, err := getBytesSlice(lenType/2+lenType%2, data)
	fieldLength, err := getBytesBCD(rawLen)
	if err != nil {
		return nil, nil, err
	}
	return getBytesSlice(fieldLength, newData)
}

// #endregion

// #region pack
func packBCD(byteLen int, data []byte, buf bytes.Buffer) error {
	if len(data) > byteLen {
		return ErrValueTooLong
	}

	if len(data) == byteLen {
		buf.Write(data)
		return nil
	}

	buf.Write(make([]byte, byteLen-len(data)))
	buf.Write(data)
	return nil
}

func packBytes(byteLen int, data []byte, buf bytes.Buffer) error {
	if len(data) > byteLen {
		return ErrValueTooLong
	}

	if len(data) < byteLen {
		return ErrValueTooSmall
	}

	buf.Write(data)
	return nil
}

func packString(byteLen int, data []byte, buf bytes.Buffer) error {
	if len(data) > byteLen {
		return ErrValueTooLong
	}

	if len(data) == byteLen {
		buf.Write(data)
		return nil
	}

	buf.Write(data)
	buf.Write(bytes.Repeat([]byte{0x20}, byteLen-len(data)))
	return nil
}

func packVarLenField(lenSize int, maxLen int, data []byte, buf bytes.Buffer) error {
	if len(data) > maxLen {
		return ErrValueTooLong
	}
	lenBufSize := lenSize/2 + lenSize%2
	lenBuf := make([]byte, lenBufSize)
	dataLen := len(data)
	for i := lenBufSize - 1; i >= 0; i-- {
		var t int
		dataLen, t = dataLen/100, dataLen%100
		lenBuf[i] = byte(((t / 10) << 4) | (t % 10))
	}

	buf.Write(lenBuf)
	buf.Write(data)
	return nil
}

// #endregion

func getBitMapArray(bitmap []byte) (fieldIndex []int) {
	for i, b := range bitmap {
		for j := byte(0); j < 8; j++ {
			if (0x80>>j)&b != 0 {
				fieldIndex = append(fieldIndex, i*8+int(j)+1)
			}
		}
	}
	return
}

type fieldUnpackFunc func([]byte) ([]byte, []byte, error)

type setFieldValueFunc func([]byte) error
type FieldValueMap map[int][]byte

var setFieldValueFuncMap map[int]setFieldValueFunc
var ErrFieldNotFound = errors.New("field not found")

func (fvm FieldValueMap) Exist(fieldId int) bool {
	_, ok := fvm[fieldId]
	return ok
}

func (fvm FieldValueMap) GetBCD(fieldId int) (string, error) {
	if v, ok := fvm[fieldId]; ok {
		buf := bytes.Buffer{}
		for _, b := range v {
			bcd, err := getBCD(b)
			if err != nil {
				return "", err
			}
			_, err = buf.WriteString(fmt.Sprintf("%02d", bcd))
			if err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}

	return "", ErrFieldNotFound
}

func (fvm FieldValueMap) GetBCDAndTrimPadding(fieldId int) (string, error) {
	str, err := fvm.GetBCD(fieldId)

	if err != nil {
		return "", err
	}

	str = strings.TrimLeft(str, "0")

	if len(str) == 0 {
		return "0", nil
	}
	return str, nil
}

func (fvm FieldValueMap) GetBytes(fieldId int) ([]byte, error) {
	if v, ok := fvm[fieldId]; ok {
		return v, nil
	}
	return nil, ErrFieldNotFound
}

func (fvm FieldValueMap) GetString(fieldId int) (string, error) {
	bs, err := fvm.GetBytes(fieldId)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func (fvm FieldValueMap) SetBCD(fieldId int, value string) error {
	if len(value)%2 != 0 {
		value = "0" + value
	}
	bs, err := hex.DecodeString(value)
	if err != nil {
		return err
	}

	fvm[fieldId] = bs
	return nil
}

func (fvm FieldValueMap) SetBytes(fieldId int, value []byte) error {
	fvm[fieldId] = value
	return nil
}

func (fvm FieldValueMap) SetString(fieldId int, value string) error {
	fvm[fieldId] = []byte(value)
	return nil
}

var fieldUnpackFuncMap = make(map[int]fieldUnpackFunc)

func init() {
	// {{#fields}}
	// {{#if isFix}}
	// 	fieldUnpackFuncMap[{{id}}] = func(data []byte) ([]byte, []byte, error) {
	// 		return getBytesSlice({{length}}, data)
	// 	}
	// {{else}}
	// 	fieldUnpackFuncMap[{{id}}] = func(data []byte) ([]byte, []byte, error) {
	// 		return unpackVarLenFiled({{varLenByteCount}}, data)
	// 	}
	// {{/if}}
	// {{/fields}}
}

func unpack(data []byte) (FieldValueMap, error) {
	bitmap, data, err := getBytesSlice(8, data)
	if err != nil {
		return nil, err
	}
	fieldIndexs := getBitMapArray(bitmap)
	if len(fieldIndexs) > 0 && fieldIndexs[0] == 1 {
		bitmap, data, err = getBytesSlice(8, data)
		if err != nil {
			return nil, err
		}
		fieldIndexs = fieldIndexs[1:]
		for _, index := range getBitMapArray(bitmap) {
			fieldIndexs = append(fieldIndexs, 64+index)
		}
	}

	fvmap := map[int][]byte{}

	for _, fi := range fieldIndexs {
		if fuf, ok := fieldUnpackFuncMap[fi]; ok {
			var fv []byte
			fv, data, err = fuf(data)
			if err != nil {
				return nil, err
			}
			fvmap[fi] = fv
		} else {
			return nil, ErrFieldUnpackFuncNotExist
		}
	}

	return fvmap, nil
}
