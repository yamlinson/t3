// Package event stores common events used by other packages
package event

// SwitchToModel instructs Bubble Tea to display a given model
type SwitchToModel struct {
	Data map[string]any
}

// ShutdownMsg instructs a model to clean up prior to shutdown
type ShutdownMsg struct{}
