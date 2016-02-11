package hashring

import (
	"hash/crc32"
	"sort"
	"sync"

	"github.com/gsdocker/gserrors"
)

type _HashRing []uint32

func (c _HashRing) Len() int {
	return len(c)
}

func (c _HashRing) Less(i, j int) bool {
	return c[i] < c[j]
}

func (c _HashRing) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Hash .
type Hash func(date []byte) uint32

// HashRing .
type HashRing struct {
	sync.RWMutex                        // mixin rw locker
	circle       _HashRing              // consistent hash ring
	hash         Hash                   // hash function
	values       map[uint32]interface{} // real value storage map
}

// New .
func New() *HashRing {
	return &HashRing{
		hash:   crc32.ChecksumIEEE,
		values: make(map[uint32]interface{}),
	}
}

// Put insert data into hash ring
func (ring *HashRing) Put(key string, value interface{}) interface{} {

	ring.Lock()
	defer ring.Unlock()

	id := ring.hash([]byte(key))

	if old, ok := ring.values[id]; ok {
		return old
	}

	ring.circle = append(ring.circle, id)

	ring.values[id] = value

	sort.Sort(ring.circle)

	return nil
}

// Get get value by key
func (ring *HashRing) Get(key string) (interface{}, bool) {

	id := ring.hash([]byte(key))

	ring.RLock()
	defer ring.RUnlock()

	if len(ring.circle) == 0 {
		return nil, false
	}

	i := sort.Search(len(ring.circle), func(x int) bool {
		return ring.circle[x] >= id
	})

	if i == len(ring.circle) {
		i = 0
	}

	val, ok := ring.values[ring.circle[i]]

	gserrors.Assert(ok, "values must exists")

	return val, true
}

// Remove remove value from key
func (ring *HashRing) Remove(key string) interface{} {

	id := ring.hash([]byte(key))

	ring.Lock()
	defer ring.Unlock()

	val, ok := ring.values[id]

	var circle _HashRing

	if ok {
		for _, i := range ring.circle {

			if i == id {
				continue
			}

			circle = append(circle, i)

		}

		ring.circle = circle

		sort.Sort(ring.circle)
	}

	delete(ring.values, id)

	return val
}
