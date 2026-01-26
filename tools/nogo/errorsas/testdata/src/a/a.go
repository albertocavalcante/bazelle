package a

import (
	"errors"
	"os"
)

func bad() {
	var err error
	var pathErr os.PathError
	errors.As(err, pathErr) // want "second argument to errors.As must be a non-nil pointer to either a type that implements error, or to any interface type"
}

func good() {
	var err error
	var pathErr *os.PathError
	errors.As(err, &pathErr) // OK - pointer to pointer
}
