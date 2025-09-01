package types

type Storage interface {
	Store()
	Name() string
}
