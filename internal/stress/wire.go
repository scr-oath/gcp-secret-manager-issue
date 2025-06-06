//go:build wireinject
// +build wireinject

package stress

import "github.com/google/wire"

var DefaultSet = wire.NewSet(
	wire.Struct(new(StressorImpl), "*"),
	wire.Bind(new(Stressor), new(*StressorImpl)),
)
var buildSet = DefaultSet

func NewStressor() Stressor {
	panic(wire.Build(buildSet))
}
