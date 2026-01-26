package a

import (
	"errors"
	"fmt"
)

func bad() {
	errors.New("error") // want "result of errors.New call not used"
	fmt.Errorf("error") // want "result of fmt.Errorf call not used"
}

func good() {
	err := errors.New("error")
	_ = err

	_ = fmt.Errorf("error")
}
