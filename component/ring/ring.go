//go:build foss

package ring

type Ring[T any] struct {
	values []T

	position int
	limit    int
}

func NewRing[T any](capacity int) *Ring[T] {
	return &Ring[T]{
		values: make([]T, capacity),
	}
}

func (r *Ring[T]) Position() int {
	return r.position
}

func (r *Ring[T]) Limit() int {
	return r.limit
}

func (r *Ring[T]) Get(index int, out []T) (int, int, bool) {
	if index < r.position {
		index = r.position
	}

	available := r.limit - index
	if available < 0 {
		return -1, index, false
	}

	if len(out) > available {
		out = out[:available]
	}

	base := index % len(r.values)
	copied := copy(out, r.values[base:])
	return copied + copy(out[copied:], r.values[:base]), index, true
}

func (r *Ring[T]) Append(toAppend []T) {
	appendLength := len(toAppend)
	valuesCapacity := len(r.values)
	if appendLength > valuesCapacity {
		toAppend = toAppend[appendLength-valuesCapacity:]
	}

	base := r.position % valuesCapacity
	copied := copy(r.values[base:], toAppend)
	copy(r.values, toAppend[copied:])

	r.limit += appendLength
	if r.position < r.limit-valuesCapacity {
		r.position = r.limit - valuesCapacity
	}
}
