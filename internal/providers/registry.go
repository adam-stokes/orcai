package providers

import (
	"fmt"
	"os/exec"
)

// Registry holds the merged set of bundled and user-defined provider profiles
// and caches binary availability so callers can filter to only usable providers.
type Registry struct {
	all       []Profile
	available []Profile
	byName    map[string]*Profile
}

// NewRegistry builds a Registry by loading bundled profiles, then user profiles
// from userDir. User profiles win on name collision (they override the bundled
// profile with the same name). Binary detection (exec.LookPath) is run once at
// construction time.
//
// If userDir does not exist the call succeeds and only bundled profiles are loaded.
func NewRegistry(userDir string) (*Registry, error) {
	bundled, err := LoadBundled()
	if err != nil {
		return nil, fmt.Errorf("registry: load bundled: %w", err)
	}

	user, err := LoadUser(userDir)
	if err != nil {
		return nil, fmt.Errorf("registry: load user: %w", err)
	}

	// Merge: start with bundled, then let user profiles overwrite by name.
	merged := make(map[string]Profile, len(bundled)+len(user))
	// Preserve insertion order for a stable All() result.
	order := make([]string, 0, len(bundled)+len(user))

	for _, p := range bundled {
		if _, seen := merged[p.Name]; !seen {
			order = append(order, p.Name)
		}
		merged[p.Name] = p
	}
	for _, p := range user {
		if _, seen := merged[p.Name]; !seen {
			order = append(order, p.Name)
		}
		merged[p.Name] = p
	}

	all := make([]Profile, 0, len(order))
	for _, name := range order {
		all = append(all, merged[name])
	}

	// Build lookup map and detect available binaries.
	byName := make(map[string]*Profile, len(all))
	var available []Profile
	for i := range all {
		p := &all[i]
		byName[p.Name] = p
		if p.Binary != "" {
			if _, err := exec.LookPath(p.Binary); err == nil {
				available = append(available, *p)
			}
		}
	}

	return &Registry{
		all:       all,
		available: available,
		byName:    byName,
	}, nil
}

// All returns every profile loaded into the registry, regardless of whether
// the provider binary is present on the host.
func (r *Registry) All() []Profile {
	return r.all
}

// Available returns only the profiles whose binary was found in PATH at
// registry construction time.
func (r *Registry) Available() []Profile {
	return r.available
}

// Get returns the profile with the given name. The second return value reports
// whether a matching profile was found.
func (r *Registry) Get(name string) (*Profile, bool) {
	p, ok := r.byName[name]
	return p, ok
}
