// Sample Go file with cgo and unsafe imports for boundaries tests.
package fixture

// #include <stdio.h>
import "C"

import (
	"reflect"
	"unsafe"
)

func Ptr() {
	var x int = 10
	_ = unsafe.Pointer(&x)
}

func Header() {
	var s []byte
	h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	_ = h
}
