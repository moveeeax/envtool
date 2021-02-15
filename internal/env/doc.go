package env

import "sort"

// Entry is a single key/value pair from a dotenv document. Order of entries is
// preserved so that formatting round-trips stay close to the original file.
type Entry struct {
	Key   string
	Value string
}

// Doc is an ordered collection of environment entries. Keys are unique; setting
// an existing key updates it in place rather than appending a duplicate.
type Doc struct {
	entries []Entry
	index   map[string]int
}

// New returns an empty document.
func New() *Doc {
	return &Doc{index: make(map[string]int)}
}

// Len reports the number of entries in the document.
func (d *Doc) Len() int {
	return len(d.entries)
}

// Has reports whether key is present.
func (d *Doc) Has(key string) bool {
	_, ok := d.index[key]
	return ok
}

// Get returns the value for key and whether it was present.
func (d *Doc) Get(key string) (string, bool) {
	i, ok := d.index[key]
	if !ok {
		return "", false
	}
	return d.entries[i].Value, true
}

// Set inserts or updates key with value, preserving first-seen order.
func (d *Doc) Set(key, value string) {
	if d.index == nil {
		d.index = make(map[string]int)
	}
	if i, ok := d.index[key]; ok {
		d.entries[i].Value = value
		return
	}
	d.index[key] = len(d.entries)
	d.entries = append(d.entries, Entry{Key: key, Value: value})
}

// Delete removes key if present and reports whether anything was removed.
func (d *Doc) Delete(key string) bool {
	i, ok := d.index[key]
	if !ok {
		return false
	}
	d.entries = append(d.entries[:i], d.entries[i+1:]...)
	delete(d.index, key)
	for k, j := range d.index {
		if j > i {
			d.index[k] = j - 1
		}
	}
	return true
}

// Keys returns the keys in document order.
func (d *Doc) Keys() []string {
	keys := make([]string, len(d.entries))
	for i, e := range d.entries {
		keys[i] = e.Key
	}
	return keys
}

// Entries returns a copy of the entries in document order.
func (d *Doc) Entries() []Entry {
	out := make([]Entry, len(d.entries))
	copy(out, d.entries)
	return out
}

// SortedCopy returns a new document with the same entries ordered by key.
func (d *Doc) SortedCopy() *Doc {
	out := New()
	keys := d.Keys()
	sort.Strings(keys)
	for _, k := range keys {
		v, _ := d.Get(k)
		out.Set(k, v)
	}
	return out
}
