package plugin

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Plugin interface defines the contract for framework plugins
type Plugin interface {
	// Plugin information
	Name() string
	Version() string
	Description() string
	
	// Lifecycle management
	Init(ctx context.Context, framework interface{}) error
	Start() error
	Stop() error
	
	// Dependency management
	Dependencies() []string
	
	// Configuration
	Configure(config map[string]interface{}) error
}

// BasePlugin provides a default implementation of the Plugin interface
type BasePlugin struct {
	name        string
	version     string
	description string
	dependencies []string
	config      map[string]interface{}
	framework   interface{}
	ctx         context.Context
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version, description string) *BasePlugin {
	return &BasePlugin{
		name:        name,
		version:     version,
		description: description,
		dependencies: []string{},
		config:      make(map[string]interface{}),
	}
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BasePlugin) Version() string {
	return p.version
}

// Description returns the plugin description
func (p *BasePlugin) Description() string {
	return p.description
}

// Init initializes the plugin
func (p *BasePlugin) Init(ctx context.Context, framework interface{}) error {
	p.ctx = ctx
	p.framework = framework
	return nil
}

// Start starts the plugin
func (p *BasePlugin) Start() error {
	return nil
}

// Stop stops the plugin
func (p *BasePlugin) Stop() error {
	return nil
}

// Dependencies returns the plugin dependencies
func (p *BasePlugin) Dependencies() []string {
	return p.dependencies
}

// Configure configures the plugin
func (p *BasePlugin) Configure(config map[string]interface{}) error {
	p.config = config
	return nil
}

// Manager manages plugins lifecycle
type Manager struct {
	plugins      map[string]Plugin
	pluginsMutex sync.RWMutex
	started      map[string]bool
	logger       *log.Logger
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		started: make(map[string]bool),
		logger:  log.Default(),
	}
}

// SetLogger sets the logger for the plugin manager
func (m *Manager) SetLogger(logger *log.Logger) {
	m.logger = logger
}

// Register registers a plugin
func (m *Manager) Register(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	
	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	
	m.pluginsMutex.Lock()
	defer m.pluginsMutex.Unlock()
	
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	// Check dependencies
	for _, dep := range plugin.Dependencies() {
		if _, exists := m.plugins[dep]; !exists {
			return fmt.Errorf("dependency %s not found for plugin %s", dep, name)
		}
	}
	
	m.plugins[name] = plugin
	m.started[name] = false
	
	m.logger.Printf("Registered plugin: %s v%s", name, plugin.Version())
	return nil
}

// Unregister unregisters a plugin
func (m *Manager) Unregister(name string) error {
	m.pluginsMutex.Lock()
	defer m.pluginsMutex.Unlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	// Check if plugin is running
	if m.started[name] {
		if err := plugin.Stop(); err != nil {
			m.logger.Printf("Error stopping plugin %s: %v", name, err)
		}
	}
	
	// Check if other plugins depend on this one
	for pName, p := range m.plugins {
		if pName == name {
			continue
		}
		for _, dep := range p.Dependencies() {
			if dep == name {
				return fmt.Errorf("cannot unregister %s: plugin %s depends on it", name, pName)
			}
		}
	}
	
	delete(m.plugins, name)
	delete(m.started, name)
	
	m.logger.Printf("Unregistered plugin: %s", name)
	return nil
}

// Get gets a plugin by name
func (m *Manager) Get(name string) (Plugin, error) {
	m.pluginsMutex.RLock()
	defer m.pluginsMutex.RUnlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	
	return plugin, nil
}

// List returns all registered plugins
func (m *Manager) List() []Plugin {
	m.pluginsMutex.RLock()
	defer m.pluginsMutex.RUnlock()
	
	list := make([]Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		list = append(list, plugin)
	}
	
	return list
}

// InitAll initializes all plugins
func (m *Manager) InitAll(ctx context.Context, framework interface{}) error {
	m.pluginsMutex.RLock()
	defer m.pluginsMutex.RUnlock()
	
	// Initialize plugins in dependency order
	initialized := make(map[string]bool)
	
	for len(initialized) < len(m.plugins) {
		progress := false
		
		for name, plugin := range m.plugins {
			if initialized[name] {
				continue
			}
			
			// Check if all dependencies are initialized
			canInit := true
			for _, dep := range plugin.Dependencies() {
				if !initialized[dep] {
					canInit = false
					break
				}
			}
			
			if canInit {
				m.logger.Printf("Initializing plugin: %s", name)
				if err := plugin.Init(ctx, framework); err != nil {
					return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
				}
				initialized[name] = true
				progress = true
			}
		}
		
		if !progress {
			return fmt.Errorf("circular dependency detected in plugins")
		}
	}
	
	return nil
}

// StartAll starts all plugins
func (m *Manager) StartAll() error {
	m.pluginsMutex.Lock()
	defer m.pluginsMutex.Unlock()
	
	// Start plugins in dependency order
	started := make(map[string]bool)
	
	for len(started) < len(m.plugins) {
		progress := false
		
		for name, plugin := range m.plugins {
			if started[name] {
				continue
			}
			
			// Check if all dependencies are started
			canStart := true
			for _, dep := range plugin.Dependencies() {
				if !started[dep] {
					canStart = false
					break
				}
			}
			
			if canStart {
				m.logger.Printf("Starting plugin: %s", name)
				if err := plugin.Start(); err != nil {
					return fmt.Errorf("failed to start plugin %s: %w", name, err)
				}
				m.started[name] = true
				started[name] = true
				progress = true
			}
		}
		
		if !progress {
			return fmt.Errorf("circular dependency detected in plugins")
		}
	}
	
	return nil
}

// StopAll stops all plugins
func (m *Manager) StopAll() error {
	m.pluginsMutex.Lock()
	defer m.pluginsMutex.Unlock()
	
	// Stop plugins in reverse dependency order
	stopped := make(map[string]bool)
	var errors []error
	
	for len(stopped) < len(m.plugins) {
		progress := false
		
		for name, plugin := range m.plugins {
			if stopped[name] || !m.started[name] {
				continue
			}
			
			// Check if any plugin depends on this one
			canStop := true
			for pName, p := range m.plugins {
				if stopped[pName] || pName == name {
					continue
				}
				for _, dep := range p.Dependencies() {
					if dep == name {
						canStop = false
						break
					}
				}
				if !canStop {
					break
				}
			}
			
			if canStop {
				m.logger.Printf("Stopping plugin: %s", name)
				if err := plugin.Stop(); err != nil {
					errors = append(errors, fmt.Errorf("failed to stop plugin %s: %w", name, err))
				}
				m.started[name] = false
				stopped[name] = true
				progress = true
			}
		}
		
		if !progress {
			// Force stop remaining plugins
			for name, plugin := range m.plugins {
				if !stopped[name] && m.started[name] {
					m.logger.Printf("Force stopping plugin: %s", name)
					if err := plugin.Stop(); err != nil {
						errors = append(errors, fmt.Errorf("failed to stop plugin %s: %w", name, err))
					}
					m.started[name] = false
					stopped[name] = true
				}
			}
			break
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors stopping plugins: %v", errors)
	}
	
	return nil
}

// IsStarted checks if a plugin is started
func (m *Manager) IsStarted(name string) bool {
	m.pluginsMutex.RLock()
	defer m.pluginsMutex.RUnlock()
	return m.started[name]
}