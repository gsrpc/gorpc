package gorpc

import (
	"bufio"
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"sync"

	"github.com/gsdocker/gserrors"
)

// Errors
var (
	ErrRegistryNotFound = errors.New("search found nothing")
)

var registryRegex = regexp.MustCompile(`(?P<name>[A-Za-z0-9_]+(\.[A-Za-z0-9_]+)*)=(?P<id>[0-9]+)`)

// Errors
var (
	ErrRegistry = errors.New("load registry file error")
)

// Registry the service registry
type Registry interface {
	// Update update the registry
	Update(items map[string]uint16)
	// ServiceName get service name by service id
	ServiceName(id uint16) string
	// ServiceID get service id by service name
	ServiceID(name string) uint16
	// Load load registry from io.Reader
	Load(r io.Reader, name string)
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

func (registry *_Registry) Load(origin io.Reader, name string) {
	reader := bufio.NewReader(origin)

	lines := 0

	items := make(map[string]uint16)

	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				break
			}

			gserrors.Panicf(err, "read registry file error :%s", name)
		}

		tokens := registryRegex.FindStringSubmatch(line)

		if tokens == nil {
			gserrors.Panicf(ErrRegistry, "load registry file error: invalid format\n\t%s(%d)", name, lines)
		}

		serviceName := ""

		serviceID := uint16(0)

		for i, name := range registryRegex.SubexpNames() {
			if name == "name" {
				serviceName = tokens[i]
			}

			if name == "id" {
				val, err := strconv.ParseInt(tokens[i], 0, 32)

				if err != nil {
					gserrors.Panicf(err, "load registry file error: invalid format\n\t%s(%d)", name, lines)
				}

				if val > math.MaxUint16 {
					gserrors.Panicf(ErrRegistry, "load registry file error: id out of range\n\t%s(%d)", name, lines)
				}

				serviceID = uint16(val)
			}
		}

		items[serviceName] = serviceID

		lines++

	}

	registry.Update(items)
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

// RegistryLoad load registry table from io.Reader
func RegistryLoad(r io.Reader, name string) {
	registry.Load(r, name)
}

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
