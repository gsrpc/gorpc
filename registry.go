package gorpc

import (
	"errors"
	"sync"

	"github.com/gsdocker/gserrors"
)

// Errors
var (
	ErrRegistryNotFound = errors.New("search found nothing")
)

// Registry the service registry
type Registry interface {
	// Update update the registry
	Update(items map[string]uint16)
	// ServiceName get service name by service id
	ServiceName(id uint16) string
	// ServiceID get service id by service name
	ServiceID(name string) uint16
}

type _Registry struct {
	sync.RWMutex                   // mixin read/write mutex
	name         map[string]uint16 // the service registry name indexer
	id           map[uint16]string // the service registry id indexer
}

// NewRegistry create new registry table
func NewRegistry() Registry {
	return &_Registry{
		name: make(map[string]uint16),
		id:   make(map[uint16]string),
	}
}

func (registry *_Registry) Update(items map[string]uint16) {
	registry.Lock()
	defer registry.Unlock()

	registry.name = items

	registry.id = make(map[uint16]string)

	for k, v := range registry.name {
		registry.id[v] = k
	}
}

func (registry *_Registry) ServiceName(id uint16) string {
	registry.RLock()
	defer registry.RUnlock()

	if val, ok := registry.id[id]; ok {
		return val
	}

	gserrors.Panicf(ErrRegistryNotFound, "can't found service name for %s", id)

	return ""
}

func (registry *_Registry) ServiceID(name string) uint16 {
	registry.RLock()
	defer registry.RUnlock()

	if val, ok := registry.name[name]; ok {
		return val
	}

	gserrors.Panicf(ErrRegistryNotFound, "can't found service id for %s", name)

	return 0
}

var registry = NewRegistry()

// RegistryUpdate update default registry
func RegistryUpdate(items map[string]uint16) {
	registry.Update(items)
}

// ServiceName get service name by service id
func ServiceName(id uint16) string {
	return registry.ServiceName(id)
}

// ServiceID get service id by service name
func ServiceID(name string) uint16 {
	return registry.ServiceID(name)
}
