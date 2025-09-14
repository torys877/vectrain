package types

type Typer interface {
	Type() string
}

type TypedConfig struct {
	TypeName string      `yaml:"type" validate:"required"`
	Config   interface{} `yaml:"config" validate:"required"`
}

func (k *TypedConfig) Type() string { return k.TypeName }
