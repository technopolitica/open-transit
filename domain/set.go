package domain

import (
	"encoding/json"
	"fmt"
	"sort"

	"golang.org/x/exp/constraints"
)

type Set[T constraints.Ordered] []T

func NewSet[T constraints.Ordered](items ...T) Set[T] {
	seen := make(map[T]bool, len(items))
	elements := make([]T, 0, len(items))
	for _, pt := range items {
		if seen[pt] {
			continue
		}
		seen[pt] = true
		elements = append(elements, pt)
	}
	sort.Slice(elements, func(i, j int) bool {
		return elements[i] < elements[j]
	})
	return elements
}

func (pts *Set[T]) UnmarshalJSON(data []byte) (err error) {
	var elements []T
	err = json.Unmarshal(data, &elements)
	if err != nil {
		return
	}
	*pts = NewSet(elements...)
	return
}

func Stringify[T fmt.Stringer](items []T) []string {
	strs := make([]string, 0, len(items))
	for _, item := range items {
		strs = append(strs, item.String())
	}
	return strs
}
