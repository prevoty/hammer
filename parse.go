package hammer

import (
	"errors"
	"reflect"
	"unsafe"

	"github.com/prevoty/hammer/ast"
)

/*
	#cgo CFLAGS: -Ihammer/src
	#cgo LDFLAGS: hammer/build/opt/src/libhammer.a
	#include <hammer.h>
	#include <stddef.h>

	int HParsedTokenUnionOffset();

	int HParsedTokenUnionOffset() {
		return offsetof(HParsedToken, sint);
	}
*/
import "C"

var parseFailed = errors.New("parse failed")

func Parse(parser HParser, input []byte) (token ast.Token, err error) {
	res := CParse(parser, input)
	defer res.Free()

	if res.r == nil {
		return token, parseFailed
	}

	return convertToken(res.r.ast), nil
}

func convertToken(ctoken HParsedToken) ast.Token {
	if ctoken == nil {
		return ast.Token{}
	}

	token := ast.Token{
		ByteOffset: int64(ctoken.index),
		BitOffset:  int8(ctoken.bit_offset),
	}

	switch ctoken.token_type {
	case C.TT_NONE:
		token.Value = ast.None
	case C.TT_BYTES:
		token.Value = convertHBytes(ctoken)
	case C.TT_SINT:
		token.Value = *(*int64)(unionPointer(ctoken))
	case C.TT_UINT:
		token.Value = *(*uint64)(unionPointer(ctoken))
	case C.TT_SEQUENCE:
		token.Value = convertHCountedArray(ctoken)
	}

	return token
}

var unionOffset = uintptr(C.HParsedTokenUnionOffset())

func unionPointer(ctoken HParsedToken) unsafe.Pointer {
	// Conversion here is to ensure ctoken is in fact a pointer
	ptr := (*C.HParsedToken)(ctoken)
	return unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + unionOffset)
}

func convertHBytes(ctoken HParsedToken) []byte {
	hbytes := *(*C.HBytes)(unionPointer(ctoken))
	return C.GoBytes(unsafe.Pointer(hbytes.token), C.int(hbytes.len))
}

func convertHCountedArray(ctoken HParsedToken) []ast.Token {
	hca := *(**C.HCountedArray)(unionPointer(ctoken))

	// elems is a []*C.HParsedToken using the hca.elements as a backing array
	shdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(hca.elements)),
		Len:  int(hca.used),
		Cap:  int(hca.used),
	}
	elems := *(*[]*C.HParsedToken)(unsafe.Pointer(&shdr))

	ret := make([]ast.Token, hca.used)
	for i, token := range elems {
		ret[i] = convertToken(token)
	}

	return ret
}