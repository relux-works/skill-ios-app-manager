package entitlements

// ListedEntitlement is one entitlement entry emitted by List.
type ListedEntitlement struct {
	Key   string
	Value Value
}

// List returns all entries from an entitlement plist.
func List(entitlementsPath string) ([]ListedEntitlement, error) {
	doc, err := LoadPlistFile(entitlementsPath)
	if err != nil {
		return nil, err
	}

	keys := doc.Keys()
	entries := make([]ListedEntitlement, 0, len(keys))
	for _, key := range keys {
		value, ok := doc.Get(key)
		if !ok {
			continue
		}
		entries = append(entries, ListedEntitlement{
			Key:   key,
			Value: value,
		})
	}

	return entries, nil
}
