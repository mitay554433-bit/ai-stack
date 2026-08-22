package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/pkg/fieldapi"
)

func cString(s string) *C.char {
	return C.CString(s)
}

//export FieldStatusJSON
func FieldStatusJSON(stateRoot *C.char) *C.char {
	if stateRoot == nil {
		return cString(`{"error":"state root required"}`)
	}

	rt, err := fieldapi.Open(
		C.GoString(stateRoot),
		reason.GemmaFromEnv(),
	)
	if err != nil {
		return cString(`{"error":"runtime open failed"}`)
	}

	wire, err := rt.StatusJSON()
	if err != nil {
		return cString(`{"error":"status failed"}`)
	}

	return cString(wire)
}

//export FieldFree
func FieldFree(p unsafe.Pointer) {
	C.free(p)
}

func main() {}
