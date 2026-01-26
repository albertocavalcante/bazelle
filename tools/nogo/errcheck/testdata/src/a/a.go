package a

import "os"

func bad() {
	os.Open("file") // want "unchecked error"
}

func good() {
	_, _ = os.Open("file") // OK - explicitly ignored
}

func alsoGood() error {
	f, err := os.Open("file")
	if err != nil {
		return err
	}
	_ = f
	return nil
}
