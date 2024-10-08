package set

import "github.com/google/uuid"

// UUIDSet is a Set to store UUIDs.
type UUIDSet struct {
	elements map[uuid.UUID]struct{}
	slice    []*uuid.UUID
}

// Add adds a new UUID
func (u *UUIDSet) Add(element uuid.UUID) {
	if !u.Has(element) {
		u.elements[element] = struct{}{}
		u.slice = append(u.slice, &element)
	}
}

// Has checks if previously added UUID
// exists or not.
func (u *UUIDSet) Has(element uuid.UUID) bool {
	_, ok := u.elements[element]
	return ok
}

// Size returns the size of the set.
func (u *UUIDSet) Size() int {
	return len(u.elements)
}

// Slice returns a slice of stored UUIDs
// from the set in insertion order.
func (u *UUIDSet) Slice() []uuid.UUID {
	var ids []uuid.UUID
	for k := range u.slice {
		if u.Has(*u.slice[k]) {
			ids = append(ids, *u.slice[k])
		}
	}
	return ids
}

// NewUUIDSet creates a new UUIDSet and returns
// a pointer to it.
func NewUUIDSet() *UUIDSet {
	return &UUIDSet{elements: make(map[uuid.UUID]struct{})}
}
