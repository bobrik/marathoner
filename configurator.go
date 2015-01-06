package marathoner

// ConfiguratorImplementation is something that updates config with a new state
type ConfiguratorImplementation interface {
	Update(State, *bool) error
}

// Configurator can update config of a specific implementation.
// It is only needed to keep name static with different implementations.
type Configurator struct {
	impl ConfiguratorImplementation
}

// Updates updates configuration on implementation.
func (c *Configurator) Update(s State, r *bool) error {
	return c.impl.Update(s, r)
}
