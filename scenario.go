package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type ScenarioUpdate struct {
	DelayMs int          `json:"delay_ms"`
	Bids    [][2]float64 `json:"bids"`
	Asks    [][2]float64 `json:"asks"`
}

type InitialState struct {
	Bids [][2]float64 `json:"bids"`
	Asks [][2]float64 `json:"asks"`
}

type Scenario struct {
	Initial InitialState     `json:"initial"`
	Updates []ScenarioUpdate `json:"updates"`
}

func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read scenario file %q: %w", path, err)
	}

	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("cannot parse scenario file %q: %w", path, err)
	}

	return &s, nil
}

func toLevels(s [][2]float64) []Level {
	if len(s) == 0 {
		return []Level{}
	}
	levels := make([]Level, len(s))
	for i, v := range s {
		levels[i] = Level{Price: v[0], Amount: v[1]}
	}
	return levels
}
