package a

import "sync"

func bad(m sync.Mutex) { // want "passes lock by value: sync.Mutex"
	use(m) // want "call of use copies lock value: sync.Mutex"
}

func use(m sync.Mutex) { // want "use passes lock by value: sync.Mutex"
}

func alsobad() {
	var m sync.Mutex
	m2 := m  // want "assignment copies lock value to m2: sync.Mutex"
	use(m2) // want "call of use copies lock value: sync.Mutex"
}

func good(m *sync.Mutex) { // OK - passed by pointer
	m.Lock()
	defer m.Unlock()
}
