package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/helpful-engineering/cth/model"
)

// LoadInventory reads a CTH inventory from a JSON file.
func LoadInventory(path string) (model.Inventory, error) {
	var inv model.Inventory

	data, err := os.ReadFile(path)
	if err != nil {
		return inv, fmt.Errorf("failed to read inventory file: %w", err)
	}

	if err := json.Unmarshal(data, &inv); err != nil {
		return inv, fmt.Errorf("failed to unmarshal inventory: %w", err)
	}

	// Structural validation
	if err := inv.Validate(); err != nil {
		return inv, fmt.Errorf("invalid inventory structure: %w", err)
	}

	return inv, nil
}

// SaveInventory writes a CTH inventory to a JSON file with indentation.
func SaveInventory(inv model.Inventory, path string) error {
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write inventory file: %w", err)
	}

	return nil
}

// LoadMultiple reads multiple inventory files in parallel (for merge workflows).
func LoadMultiple(paths []string) ([]model.Inventory, error) {
	results := make([]model.Inventory, len(paths))

	for i, path := range paths {
		inv, err := LoadInventory(path)
		if err != nil {
			return nil, fmt.Errorf("path[%d] (%s): %w", i, path, err)
		}
		results[i] = inv
	}

	return results, nil
}
