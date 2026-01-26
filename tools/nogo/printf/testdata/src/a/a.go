package a

import "fmt"

func bad() {
	fmt.Printf("%d", "hello") // want "fmt.Printf format %d has arg \"hello\" of wrong type string"
}

func alsobad() {
	fmt.Printf("%s %s", "one") // want "fmt.Printf format %s reads arg #2, but call has 1 arg"
}

func good() {
	fmt.Printf("%s", "hello") // OK
	fmt.Printf("%d", 42)      // OK
}
