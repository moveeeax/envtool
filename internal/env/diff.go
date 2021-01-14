package env

import "sort"

// ChangeKind classifies how a key differs between two documents.
type ChangeKind int

const (
	// Added marks a key present in the right document but not the left.
	Added ChangeKind = iota
	// Removed marks a key present in the left document but not the right.
	Removed
	// Changed marks a key present in both with differing values.
	Changed
)

func (k ChangeKind) String() string {
	switch k {
	case Added:
		return "added"
	case Removed:
		return "removed"
	case Changed:
		return "changed"
	default:
		return "unknown"
	}
}

// Change is a single difference between two documents.
type Change struct {
	Key   string
	Kind  ChangeKind
	Left  string
	Right string
}

// Diff compares left and right and returns the changes needed to turn left into
// right. Results are sorted by key for stable output.
func Diff(left, right *Doc) []Change {
	var changes []Change
	for _, e := range left.entries {
		rv, ok := right.Get(e.Key)
		if !ok {
			changes = append(changes, Change{Key: e.Key, Kind: Removed, Left: e.Value})
			continue
		}
		if rv != e.Value {
			changes = append(changes, Change{Key: e.Key, Kind: Changed, Left: e.Value, Right: rv})
		}
	}
	for _, e := range right.entries {
		if !left.Has(e.Key) {
			changes = append(changes, Change{Key: e.Key, Kind: Added, Right: e.Value})
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Key < changes[j].Key
	})
	return changes
}
