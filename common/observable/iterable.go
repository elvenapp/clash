//go:build foss

package observable

type Iterable[T any] <-chan T
