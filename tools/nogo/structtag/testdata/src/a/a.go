package a

type AlsoBad struct {
	Name  string `json:"name"`
	Other string `json:"name"` // want `struct field Other repeats json tag "name" also at a.go:4`
}

type MalformedTag struct {
	Field string `json:"field" xml` // want `struct field tag .* not compatible with reflect.StructTag.Get`
}

type Good struct {
	Field string `json:"field"` // OK
}
