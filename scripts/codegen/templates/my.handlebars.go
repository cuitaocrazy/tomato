package codegen

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

//ErrLenNotMeet  长度错误
// var ErrLenNotMeet = errors.New("Length does not meet the definition")

//ErrFieldUnpackFuncNotExist 字段解包函数不存在
var ErrFieldUnpackFuncNotExist = errors.New("field unpack function not exist")
var ErrFieldPackFuncNotExist = errors.New("field pack function not exist")

//ErrBCD bcd错误
var ErrBCD = errors.New("BCD error")

// ErrValueTooLong 值的长度过长
var ErrValueTooLong = errors.New("value too long")

// ErrValueTooSmall 值的长度过短
var ErrValueTooSmall = errors.New("value too small")

// ErrFieldNotFound 没有找到域定义
var ErrFieldNotFound = errors.New("field not found")

// ErrPackDataTooSmall 报数据过短
var ErrPackDataTooSmall = errors.New("package data too small")

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

func getBytesSlice(length int, data *bytes.Buffer) ([]byte, error) {
	if data.Len() < length {
		return nil, ErrPackDataTooSmall
	}

	return append([]byte{}, data.Next(length)...), nil
}

func unpackVarLenFiled(lenByteCount int, maxLen int, data *bytes.Buffer) ([]byte, error) {
	if data.Len() < lenByteCount {
		return nil, ErrPackDataTooSmall
	}

	rawLen := data.Next(lenByteCount)

	fieldLength, err := getBytesBCD(rawLen)

	if err != nil {
		return nil, err
	}

	if fieldLength > maxLen {
		return nil, ErrValueTooLong
	}

	return getBytesSlice(fieldLength, data)
}

// #endregion

// #region pack
func packBCD(byteLen int, data []byte, buf *bytes.Buffer) error {
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

func packBytes(byteLen int, data []byte, buf *bytes.Buffer) error {
	if len(data) > byteLen {
		return ErrValueTooLong
	}

	if len(data) < byteLen {
		return ErrValueTooSmall
	}

	buf.Write(data)
	return nil
}

func packString(byteLen int, data []byte, buf *bytes.Buffer) error {
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

func packVarLenField(lenByteCount int, maxLen int, data []byte, buf *bytes.Buffer) error {
	if len(data) > maxLen {
		return ErrValueTooLong
	}
	lenBuf := make([]byte, lenByteCount)
	dataLen := len(data)
	for i := lenByteCount - 1; i >= 0; i-- {
		var t int
		dataLen, t = dataLen/100, dataLen%100
		lenBuf[i] = byte(((t / 10) << 4) | (t % 10))
	}

	buf.Write(lenBuf)
	buf.Write(data)
	return nil
}

// #endregion

func getBitmapIndex(bitmap []byte) (fieldIndex []int) {
	for i, b := range bitmap {
		for j := byte(0); j < 8; j++ {
			if (0x80>>j)&b != 0 {
				fieldIndex = append(fieldIndex, i*8+int(j)+1)
			}
		}
	}
	return
}

func getBitmap(fieldIndex []int) []byte {
	ret := make([]byte, 8)
	for _, i := range fieldIndex {
		bi, bo := (i-1)/8, (i-1)%8
		ret[bi] = ret[bi] | (0x80 >> byte(bo))
	}
	return ret
}

// 解包函数
type fieldUnpackFunc func(*bytes.Buffer) ([]byte, error)

// 打包函数
type fieldPackFunc func([]byte, *bytes.Buffer) error

// #region FieldValueMap
// FieldValueMap 基础的域值映射字典
type FieldValueMap map[int][]byte

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

func (fvm FieldValueMap) GetStringAndTrim(fieldId int) (string, error) {
	str, err := fvm.GetString(fieldId)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(str, " "), nil
}

func (fvm FieldValueMap) SetBCD(fieldId int, value string) error {
	if len(value)%2 != 0 {
		value = "0" + value
	}
	matched, _ := regexp.Match(`^[0-9]+$`, []byte(value))
	if matched {
		return ErrBCD
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

// #endregion

var fieldUnpackFuncMap = map[int]fieldUnpackFunc{}
var fieldPackFuncMap = map[int]fieldPackFunc{}

// 无奈的工具
func tRetErr(fn func([]byte) (int, error), bs ...[]byte) error {
	for _, b := range bs {
		_, err := fn(b)
		if err != nil {
			return err
		}
	}
	return nil
}
func tWrite(buf *bytes.Buffer, bs ...[]byte) error {
	return tRetErr(buf.Write, bs...)
}

func tRead(buf *bytes.Buffer, bs ...[]byte) error {
	return tRetErr(buf.Read, bs...)
}

type TPDU struct {
	ID   byte
	Src  [2]byte
	Desc [2]byte
}

func (tpdu *TPDU) fill(buf *bytes.Buffer) error {
	id, err := buf.ReadByte()
	if err != nil {
		return err
	}
	tpdu.ID = id
	return tRead(buf, tpdu.Src[:], tpdu.Desc[:])
}

func (tpdu *TPDU) to(buf *bytes.Buffer) error {
	err := buf.WriteByte(tpdu.ID)
	if err != nil {
		return err
	}
	return tWrite(buf, tpdu.Src[:], tpdu.Desc[:])
}

var lriFlag = []byte{0x4c, 0x52, 0x49, 0x00, 0x1c}

type LRI struct {
	Flag [5]byte
	ANI  [8]byte
	DNIS [8]byte
	LRI  [12]byte
}

func (lri *LRI) fill(buf *bytes.Buffer) error {
	return tRead(buf, lri.Flag[:], lri.ANI[:], lri.DNIS[:], lri.LRI[:])
}

func (lri *LRI) to(buf *bytes.Buffer) error {
	return tWrite(buf, lri.Flag[:], lri.ANI[:], lri.DNIS[:], lri.LRI[:])
}

type AppHead struct {
	Version [2]byte
	MTI     [2]byte
}

func (ah *AppHead) fill(buf *bytes.Buffer) error {
	return tRead(buf, ah.Version[:], ah.MTI[:])
}

func (ah *AppHead) to(buf *bytes.Buffer) error {
	return tWrite(buf, ah.Version[:], ah.MTI[:])
}

type Head struct {
	TPDU    *TPDU
	LRI     *LRI
	AppHead *AppHead
}

func init() {
{{#fields}}
{{#if isFix}}
	fieldUnpackFuncMap[{{id}}] = func(data *bytes.Buffer) ([]byte, error) {
		return getBytesSlice({{length}}, data)
	}
{{#if isNum}}
	fieldPackFuncMap[{{id}}] = func(data []byte, buf *bytes.Buffer) error {
		return packBCD({{length}}, data, buf)
	}
{{/if}}
{{#if isByte}}
	fieldPackFuncMap[{{id}}] = func(data []byte, buf *bytes.Buffer) error {
		return packBytes({{length}}, data, buf)
	}
{{/if}}
{{#if isChar}}
	fieldPackFuncMap[{{id}}] = func(data []byte, buf *bytes.Buffer) error {
		return packString({{length}}, data, buf)
	}
{{/if}}
{{else}}
	fieldUnpackFuncMap[{{id}}] = func(data *bytes.Buffer) ([]byte, error) {
		return unpackVarLenFiled({{varLenByteCount}}, {{length}}, data)
	}
	fieldPackFuncMap[{{id}}] = func(data []byte, buf *bytes.Buffer) error {
		return packVarLenField({{varLenByteCount}}, {{length}}, data, buf)
	}
{{/if}}
{{/fields}}
}

func unpack(data *bytes.Buffer) (*Head, FieldValueMap, error) {
	// 无奈
	retErr := func(err error) (*Head, FieldValueMap, error) {
		return nil, nil, err
	}

	if data.Len() < 20 {
		return retErr(ErrPackDataTooSmall)
	}

	// #region 读取头
	head := &Head{}
	var tpdu TPDU
	err := tpdu.fill(data)
	if err != nil {
		return retErr(err)
	}
	head.TPDU = &tpdu

	if bytes.Equal(data.Bytes()[:5], lriFlag) {
		var lri LRI
		err = lri.fill(data)
		if err != nil {
			return retErr(err)
		}
		head.LRI = &lri
	}

	var appHead AppHead
	err = appHead.fill(data)
	if err != nil {
		return retErr(err)
	}
	head.AppHead = &appHead
	// #endregion

	bitmap, err := getBytesSlice(8, data)
	if err != nil {
		return retErr(err)
	}
	fieldIndexs := getBitmapIndex(bitmap)
	if len(fieldIndexs) > 0 && fieldIndexs[0] == 1 {
		bitmap, err = getBytesSlice(8, data)
		if err != nil {
			return retErr(err)
		}
		fieldIndexs = fieldIndexs[1:]
		for _, index := range getBitmapIndex(bitmap) {
			fieldIndexs = append(fieldIndexs, 64+index)
		}
	}

	fvmap := FieldValueMap{}

	for _, fi := range fieldIndexs {
		if fuf, ok := fieldUnpackFuncMap[fi]; ok {
			var fv []byte
			fv, err = fuf(data)
			if err != nil {
				return retErr(err)
			}
			fvmap[fi] = fv
		} else {
			return retErr(ErrFieldUnpackFuncNotExist)
		}
	}

	return head, fvmap, nil
}

func pack(head *Head, fvm FieldValueMap, buf *bytes.Buffer) error {
	err := head.TPDU.to(buf)
	if err != nil {
		return err
	}
	err = head.AppHead.to(buf)
	if err != nil {
		return err
	}
	var bmf1, bmf2, keys []int

	for k := range fvm {
		keys = append(keys, k)
		if k > 64 {
			bmf2 = append(bmf2, k-64)
		} else {
			bmf1 = append(bmf1, k)
		}
	}
	sort.Ints(keys)

	_, err = buf.Write(getBitmap(bmf1))
	if err != nil {
		return err
	}

	if len(bmf2) > 0 {
		_, err = buf.Write(getBitmap(bmf2))
		if err != nil {
			return err
		}
	}

	for _, k := range keys {
		if fpf, ok := fieldPackFuncMap[k]; ok {
			err = fpf(fvm[k], buf)
			if err != nil {
				return err
			}
		} else {
			return ErrFieldPackFuncNotExist
		}
	}
	return nil
}
