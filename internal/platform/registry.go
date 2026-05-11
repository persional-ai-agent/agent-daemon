package platform

import "sync"

var (
	regMu   sync.RWMutex
	adapters = map[string]Adapter{}
)

func Register(a Adapter) {
	if a == nil {
		return
	}
	name := a.Name()
	if name == "" {
		return
	}
	regMu.Lock()
	defer regMu.Unlock()
	adapters[name] = a
}

func Unregister(name string) {
	regMu.Lock()
	defer regMu.Unlock()
	delete(adapters, name)
}

func Get(name string) (Adapter, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
a, ok := adapters[name]
	return a, ok
}

func Names() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(adapters))
	for k := range adapters {
		out = append(out, k)
	}
	return out
}

