package store

import "github.com/helpful-engineering/cth/model"

// Store defines the interface for persisting and retrieving CTH inventories.
// json.go implements this for the Crawl phase; muninn.go implements it for Walk+.
type Store interface {
	// Load retrieves an inventory by programme ID (e.g., "QBP").
	Load(id string) (model.Inventory, error)

	// Save persists an inventory, overwriting any existing record with the same
	// programme + version key.
	Save(inv model.Inventory) error

	// Query returns inventories matching the filter.
	Query(filter QueryFilter) ([]model.Inventory, error)
}

// QueryFilter constrains which inventories are returned by Store.Query.
type QueryFilter struct {
	// Programme restricts results to a single programme (exact match, empty = all).
	Programme string

	// MinVersion returns only inventories at or above this semver string.
	MinVersion string

	// Tags returns only inventories whose ProgrammeMeta carries all listed tags.
	Tags []string
}
