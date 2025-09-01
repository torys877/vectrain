package types

type Entity struct {
	ID      int64
	Text    string
	Payload map[string]string
}

type ErrorEntity struct {
	Err string
	Entity
}
