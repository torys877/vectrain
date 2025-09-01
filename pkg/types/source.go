package types

type Source interface {
	Fetch() any
	Name() string
}
