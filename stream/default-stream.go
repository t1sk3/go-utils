package stream

import "utils/optional"

type defaultStream[T any] struct {
	Stream[T]
	data []T
}

func FromSlice[T any](slice []T) Stream[T] {
	return &defaultStream[T]{data: slice}
}

func FromArray[T any](array []T) Stream[T] {
	return &defaultStream[T]{data: array[:]}
}

func (ds *defaultStream[T]) Filter(predicate func(T) bool) Stream[T] {
	if predicate == nil {
		panic("Filter requires a non-nil function")
	}
	var filtered []T
	for _, v := range ds.data {
		if predicate(v) {
			filtered = append(filtered, v)
		}
	}
	return &defaultStream[T]{data: filtered}
}

func (ds *defaultStream[T]) Map(mapper func(T) any) Stream[any] {
	if mapper == nil {
		panic("Map requires a non-nil function")
	}
	var mapped []any
	for _, v := range ds.data {
		mapped = append(mapped, mapper(v))
	}
	return &defaultStream[any]{data: mapped}
}

func (ds *defaultStream[T]) ForEach(consumer func(T)) {
	if consumer == nil {
		panic("ForEach requires a non-nil function")
	}
	for _, v := range ds.data {
		consumer(v)
	}
}

func (ds *defaultStream[T]) FindFirst() optional.Optional[T] {
	if len(ds.data) == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(ds.data[0])
}

func (ds *defaultStream[T]) FindAny() optional.Optional[T] {
	if len(ds.data) == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(ds.data[0])
}

func (ds *defaultStream[T]) AllMatch(predicate func(T) bool) bool {
	if predicate == nil {
		panic("AllMatch requires a non-nil function")
	}
	for _, v := range ds.data {
		if !predicate(v) {
			return false
		}
	}
	return true
}

func (ds *defaultStream[T]) AnyMatch(predicate func(T) bool) bool {
	if predicate == nil {
		panic("AnyMatch requires a non-nil function")
	}
	for _, v := range ds.data {
		if predicate(v) {
			return true
		}
	}
	return false
}

func (ds *defaultStream[T]) NoneMatch(predicate func(T) bool) bool {
	if predicate == nil {
		panic("NoneMatch requires a non-nil function")
	}
	return !ds.AnyMatch(predicate)
}

func (ds *defaultStream[T]) Distinct() Stream[T] {
	seen := make(map[interface{}]struct{})
	var distinct []T
	for _, v := range ds.data {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			distinct = append(distinct, v)
		}
	}
	return &defaultStream[T]{data: distinct}
}

func (ds *defaultStream[T]) Sorted(comparator func(T, T) int) Stream[T] {
	if comparator == nil {
		panic("Sorted2 requires a non-nil function")
	}
	sortedData := make([]T, len(ds.data))
	copy(sortedData, ds.data)
	for i := 0; i < len(sortedData); i++ {
		for j := 0; j < len(sortedData)-i-1; j++ {
			if comparator(sortedData[j], sortedData[j+1]) > 0 {
				sortedData[j], sortedData[j+1] = sortedData[j+1], sortedData[j]
			}
		}
	}
	return &defaultStream[T]{data: sortedData}
}

func (ds *defaultStream[T]) Count() int64 {
	return int64(len(ds.data))
}

func (ds *defaultStream[T]) Min(comparator func(T, T) int) optional.Optional[T] {
	if len(ds.data) == 0 {
		return optional.Empty[T]()
	}
	if comparator == nil {
		panic("Min requires a non-nil function")
	}
	min := ds.data[0]
	for _, v := range ds.data[1:] {
		if comparator(v, min) < 0 {
			min = v
		}
	}
	return optional.Of(min)
}

func (ds *defaultStream[T]) Max(comparator func(T, T) int) optional.Optional[T] {
	if len(ds.data) == 0 {
		return optional.Empty[T]()
	}
	if comparator == nil {
		panic("Max requires a non-nil function")
	}
	max := ds.data[0]
	for _, v := range ds.data[1:] {
		if comparator(v, max) > 0 {
			max = v
		}
	}
	return optional.Of(max)
}

func (ds *defaultStream[T]) Limit(n int64) Stream[T] {
	if n < 0 {
		panic("Limit requires a non-negative integer")
	}
	if n >= ds.Count() {
		return ds
	}
	return &defaultStream[T]{data: ds.data[:n]}
}

func (ds *defaultStream[T]) Skip(n int64) Stream[T] {
	if n < 0 {
		panic("Skip requires a non-negative integer")
	}
	if n >= ds.Count() {
		return &defaultStream[T]{data: []T{}}
	}
	return &defaultStream[T]{data: ds.data[n:]}
}

func (ds *defaultStream[T]) ToSlice() []T {
	return ds.data
}

func (ds *defaultStream[T]) Reduce(binaryOperator func(T, T) T) optional.Optional[T] {
	if binaryOperator == nil {
		panic("Reduce requires a non-nil function")
	}
	if len(ds.data) == 0 {
		return optional.Empty[T]()
	}
	result := ds.data[0]
	for _, v := range ds.data[1:] {
		result = binaryOperator(result, v)
	}
	return optional.Of(result)
}
