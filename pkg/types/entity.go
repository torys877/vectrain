package types

type Entity struct {
	ID      [16]byte
	UUID    string
	Text    string
	Payload map[string]string
	Vector  []float32
	Err     error
}

type ErrorEntity struct {
	Entity
	Err error
}
