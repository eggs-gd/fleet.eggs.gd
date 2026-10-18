package mdfile

// ListRecords reads key as a block of records, each starting with a "  - "
// line and continuing over more-indented sibling-field lines, e.g.
//
//	milestones:
//	  - id: v1-launch
//	    title: v1 launch
//	    state: active
//	  - id: smart-project-setup
//	    title: Smart project setup
//	    state: parked
//
// Field values come back unquoted. Returns nil, nil when key is absent.
func ListRecords(text, key string) ([]map[string]string, error) {
	d, err := ParseDoc(text)
	if err != nil {
		return nil, err
	}
	return d.Records(key), nil
}

// SetRecords replaces key with a block list of records, each written in the
// given field order — map iteration order is not stable in Go, so callers
// pass fields explicitly. A field absent from a record is omitted from its
// block. Values are quoted as needed, so any string round-trips through
// ListRecords.
func SetRecords(text, key string, fields []string, records []map[string]string) (string, error) {
	d, err := ParseDoc(text)
	if err != nil {
		return "", err
	}
	d.SetRecords(key, fields, records)
	return d.String(), nil
}
