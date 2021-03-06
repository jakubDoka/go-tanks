// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package game

import "github.com/jakubDoka/mlok/logic/memory/gen"

// TankCapsule is something like an optional type, it holds boolean about whether
// it contains value though it does not hold pointer
type TankCapsule struct {
	occupied bool
	value    Tank
}

// TankStorage generates IDs witch makes no need to use hashing,
// only drawback is that you cannot choose the id, it will be assigned
// like a pointer, but without putting presure no gc, brilliant TankStorage for
// components. Its highly unlikely you will run out of ids as they are reused
type TankStorage struct {
	vec      []TankCapsule
	freeIDs  gen.IntVec
	occupied []int
	count    int
	outdated bool
}

// Blanc allocates blanc space adds
func (s *TankStorage) Blanc() {
	s.freeIDs = append(s.freeIDs, len(s.vec))
	s.vec = append(s.vec, TankCapsule{})
}

// Allocate id allocates if it is free, else it returns nil
func (s *TankStorage) AllocateID(id int) *Tank {
	if int(id) >= len(s.vec) || s.vec[id].occupied {
		return nil
	}

	idx, _ := s.freeIDs.BiSearch(id, gen.IntBiComp)
	s.freeIDs.Remove(idx)

	return &s.vec[id].value
}

// Allocate allocates an value and returns id and pointer to it. Note that
// allocate does not always allocate at all and just reuses freed space,
// returned pointer also does not point to zero value and you have to overwrite all
// properties to get expected behavior
func (s *TankStorage) Allocate() (*Tank, int) {
	s.count++
	s.outdated = true

	l := len(s.freeIDs)
	if l != 0 {
		id := s.freeIDs[l-1]
		s.freeIDs = s.freeIDs[:l-1]
		s.vec[id].occupied = true
		return &s.vec[id].value, id
	}

	id := len(s.vec)
	s.vec = append(s.vec, TankCapsule{})

	s.vec[id].occupied = true
	return &s.vec[id].value, id
}

// Remove removes a value and frees memory for something else
//
// panic if there is nothing to free
func (s *TankStorage) Remove(id int) {
	if !s.vec[id].occupied {
		panic("removeing already removed value")
	}

	s.count--
	s.outdated = true

	s.freeIDs.BiInsert(id, gen.IntBiComp)
	s.vec[id].occupied = false
}

// Item returns pointer to value under the "id", accessing random id can result in
// random value that can be considered unoccupied
//
// method panics if id is not occupied
func (s *TankStorage) Item(id int) *Tank {
	if !s.vec[id].occupied {
		panic("accessing non occupied id")
	}

	return &s.vec[id].value
}

// Used returns whether id is used
func (s *TankStorage) Used(id int) bool {
	return s.vec[id].occupied
}

// Len returns size of TankStorage
func (s *TankStorage) Len() int {
	return len(s.vec)
}

// Count returns amount of values stored
func (s *TankStorage) Count() int {
	return s.count
}

// update updates state of occupied slice, every time you remove or add
// element, TankStorage gets outdated, this makes it up to date
func (s *TankStorage) update() {
	s.outdated = false
	s.occupied = s.occupied[:0]
	l := len(s.vec)
	for i := 0; i < l; i++ {
		if s.vec[i].occupied {
			s.occupied = append(s.occupied, i)
		}
	}
}

// Occupied return all occupied ids in TankStorage, this method panics if TankStorage is outdated
// See Update method.
func (s *TankStorage) Occupied() []int {
	if s.outdated {
		s.update()
	}

	return s.occupied
}

// Clear clears TankStorage, but keeps allocated space
func (s *TankStorage) Clear() {
	s.vec = s.vec[:0]
	s.occupied = s.occupied[:0]
	s.freeIDs = s.freeIDs[:0]
	s.count = 0
}

// SlowClear clears the the TankStorage slowly with is tradeoff for having faster allocating speed
func (s *TankStorage) SlowClear() {
	for i := range s.vec {
		if s.vec[i].occupied {
			s.freeIDs = append(s.freeIDs, i)
			s.vec[i].occupied = false
		}
	}

	s.occupied = s.occupied[:0]
	s.count = 0
}

// BulletCapsule is something like an optional type, it holds boolean about whether
// it contains value though it does not hold pointer
type BulletCapsule struct {
	occupied bool
	value    Bullet
}

// BulletStorage generates IDs witch makes no need to use hashing,
// only drawback is that you cannot choose the id, it will be assigned
// like a pointer, but without putting presure no gc, brilliant BulletStorage for
// components. Its highly unlikely you will run out of ids as they are reused
type BulletStorage struct {
	vec      []BulletCapsule
	freeIDs  gen.IntVec
	occupied []int
	count    int
	outdated bool
}

// Blanc allocates blanc space adds
func (s *BulletStorage) Blanc() {
	s.freeIDs = append(s.freeIDs, len(s.vec))
	s.vec = append(s.vec, BulletCapsule{})
}

// Allocate id allocates if it is free, else it returns nil
func (s *BulletStorage) AllocateID(id int) *Bullet {
	if int(id) >= len(s.vec) || s.vec[id].occupied {
		return nil
	}

	idx, _ := s.freeIDs.BiSearch(id, gen.IntBiComp)
	s.freeIDs.Remove(idx)

	return &s.vec[id].value
}

// Allocate allocates an value and returns id and pointer to it. Note that
// allocate does not always allocate at all and just reuses freed space,
// returned pointer also does not point to zero value and you have to overwrite all
// properties to get expected behavior
func (s *BulletStorage) Allocate() (*Bullet, int) {
	s.count++
	s.outdated = true

	l := len(s.freeIDs)
	if l != 0 {
		id := s.freeIDs[l-1]
		s.freeIDs = s.freeIDs[:l-1]
		s.vec[id].occupied = true
		return &s.vec[id].value, id
	}

	id := len(s.vec)
	s.vec = append(s.vec, BulletCapsule{})

	s.vec[id].occupied = true
	return &s.vec[id].value, id
}

// Remove removes a value and frees memory for something else
//
// panic if there is nothing to free
func (s *BulletStorage) Remove(id int) {
	if !s.vec[id].occupied {
		panic("removeing already removed value")
	}

	s.count--
	s.outdated = true

	s.freeIDs.BiInsert(id, gen.IntBiComp)
	s.vec[id].occupied = false
}

// Item returns pointer to value under the "id", accessing random id can result in
// random value that can be considered unoccupied
//
// method panics if id is not occupied
func (s *BulletStorage) Item(id int) *Bullet {
	if !s.vec[id].occupied {
		panic("accessing non occupied id")
	}

	return &s.vec[id].value
}

// Used returns whether id is used
func (s *BulletStorage) Used(id int) bool {
	return s.vec[id].occupied
}

// Len returns size of BulletStorage
func (s *BulletStorage) Len() int {
	return len(s.vec)
}

// Count returns amount of values stored
func (s *BulletStorage) Count() int {
	return s.count
}

// update updates state of occupied slice, every time you remove or add
// element, BulletStorage gets outdated, this makes it up to date
func (s *BulletStorage) update() {
	s.outdated = false
	s.occupied = s.occupied[:0]
	l := len(s.vec)
	for i := 0; i < l; i++ {
		if s.vec[i].occupied {
			s.occupied = append(s.occupied, i)
		}
	}
}

// Occupied return all occupied ids in BulletStorage, this method panics if BulletStorage is outdated
// See Update method.
func (s *BulletStorage) Occupied() []int {
	if s.outdated {
		s.update()
	}

	return s.occupied
}

// Clear clears BulletStorage, but keeps allocated space
func (s *BulletStorage) Clear() {
	s.vec = s.vec[:0]
	s.occupied = s.occupied[:0]
	s.freeIDs = s.freeIDs[:0]
	s.count = 0
}

// SlowClear clears the the BulletStorage slowly with is tradeoff for having faster allocating speed
func (s *BulletStorage) SlowClear() {
	for i := range s.vec {
		if s.vec[i].occupied {
			s.freeIDs = append(s.freeIDs, i)
			s.vec[i].occupied = false
		}
	}

	s.occupied = s.occupied[:0]
	s.count = 0
}
