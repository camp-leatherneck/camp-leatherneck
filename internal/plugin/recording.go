package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunResult represents the outcome of a plugin execution.
type RunResult string

const (
	ResultSuccess RunResult = "success"
	ResultFailure RunResult = "failure"
	ResultSkipped RunResult = "skipped"
)

// PluginRunRecord represents data for recording a plugin run.
type PluginRunRecord struct {
	PluginName string
	RigName    string
	Result     RunResult
	Body       string
}

// PluginRunBead represents a recorded plugin run (kept for CLI history display).
type PluginRunBead struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Labels    []string  `json:"labels"`
	Result    RunResult `json:"-"` // Populated from labels or direct assignment
}

// pluginState is the on-disk format for plugin run history (state.json).
type pluginState struct {
	LastDispatch time.Time        `json:"last_dispatch"`
	History      []*PluginRunBead `json:"history,omitempty"`
}

const maxHistoryEntries = 50

// Recorder handles plugin run recording and cooldown gate queries.
// State is persisted to <townRoot>/plugins/<name>/state.json — no Dolt dependency.
type Recorder struct {
	townRoot string
}

// NewRecorder creates a new plugin run recorder.
func NewRecorder(townRoot string) *Recorder {
	return &Recorder{townRoot: townRoot}
}

func (r *Recorder) statePath(pluginName string) string {
	return filepath.Join(r.townRoot, "plugins", pluginName, "state.json")
}

func (r *Recorder) readState(pluginName string) (*pluginState, error) {
	data, err := os.ReadFile(r.statePath(pluginName)) //nolint:gosec // G304: path is constructed from trusted internal values
	if os.IsNotExist(err) {
		return &pluginState{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading plugin state: %w", err)
	}
	var state pluginState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupt state — treat as no history rather than erroring.
		return &pluginState{}, nil
	}
	return &state, nil
}

func (r *Recorder) writeState(pluginName string, state *pluginState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling plugin state: %w", err)
	}
	path := r.statePath(pluginName)
	if err := os.WriteFile(path, data, 0644); err != nil { //nolint:gosec // G306: 0644 is intentional
		return fmt.Errorf("writing plugin state: %w", err)
	}
	return nil
}

// RecordRun records a plugin dispatch to state.json.
// Returns a synthetic run ID for display purposes.
func (r *Recorder) RecordRun(record PluginRunRecord) (string, error) {
	state, err := r.readState(record.PluginName)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	id := fmt.Sprintf("run-%d", now.UnixNano())

	labels := []string{
		"type:plugin-run",
		fmt.Sprintf("plugin:%s", record.PluginName),
		fmt.Sprintf("result:%s", record.Result),
	}
	if record.RigName != "" {
		labels = append(labels, fmt.Sprintf("rig:%s", record.RigName))
	}

	entry := &PluginRunBead{
		ID:        id,
		Title:     fmt.Sprintf("Plugin run: %s", record.PluginName),
		CreatedAt: now,
		Labels:    labels,
		Result:    record.Result,
	}

	state.LastDispatch = now
	state.History = append([]*PluginRunBead{entry}, state.History...)
	if len(state.History) > maxHistoryEntries {
		state.History = state.History[:maxHistoryEntries]
	}

	if err := r.writeState(record.PluginName, state); err != nil {
		return "", err
	}
	return id, nil
}

// GetLastRun returns the most recent run for a plugin.
// Returns nil if no runs found.
func (r *Recorder) GetLastRun(pluginName string) (*PluginRunBead, error) {
	state, err := r.readState(pluginName)
	if err != nil {
		return nil, err
	}
	if len(state.History) == 0 {
		return nil, nil
	}
	return state.History[0], nil
}

// GetRunsSince returns all runs for a plugin since the given duration.
// Duration format: "1h", "24h", "30m", etc. Empty string returns all runs.
func (r *Recorder) GetRunsSince(pluginName string, since string) ([]*PluginRunBead, error) {
	state, err := r.readState(pluginName)
	if err != nil {
		return nil, err
	}

	if since == "" {
		return state.History, nil
	}

	d, err := time.ParseDuration(since)
	if err != nil {
		return nil, fmt.Errorf("parsing duration %q: %w", since, err)
	}
	cutoff := time.Now().Add(-d)

	var runs []*PluginRunBead
	for _, run := range state.History {
		if run.CreatedAt.After(cutoff) {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

// CountRunsSince returns the count of runs for a plugin since the given duration.
// Used by the cooldown gate in dispatchPlugins.
func (r *Recorder) CountRunsSince(pluginName string, since string) (int, error) {
	runs, err := r.GetRunsSince(pluginName, since)
	if err != nil {
		return 0, err
	}
	return len(runs), nil
}
