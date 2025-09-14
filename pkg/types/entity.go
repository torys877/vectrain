package types

type Entity struct {
	//ID      [16]byte
	ID      string
	UUID    string
	Text    string
	Payload map[string]string
	Vector  []float32
	Err     error
}
