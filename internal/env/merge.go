package env

// Merge overlays docs left to right onto a fresh document. Later documents win
// on key conflicts, while the resulting key order follows first appearance
// across all inputs. The inputs are not modified.
func Merge(docs ...*Doc) *Doc {
	out := New()
	for _, d := range docs {
		if d == nil {
			continue
		}
		for _, e := range d.entries {
			out.Set(e.Key, e.Value)
		}
	}
	return out
}
