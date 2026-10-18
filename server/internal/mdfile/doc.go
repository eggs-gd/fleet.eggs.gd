package mdfile

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

// ErrNoFrontmatter is returned when the text has no complete frontmatter block.
var ErrNoFrontmatter = errors.New("no complete frontmatter block")

// splitEnvelope splits `---\n<block>\n---<rest>` text. block is the lines
// between the markers without the final newline; rest starts at the closing
// marker's leading newline. It is the one place that knows the envelope.
func splitEnvelope(text string) (block, rest string, ok bool) {
	if !strings.HasPrefix(text, "---\n") {
		return "", "", false
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return "", "", false
	}
	end += 4
	return text[4:end], text[end:], true
}

// Doc is a Markdown file's frontmatter as raw lines, plus everything from the
// closing marker on. It is the single frontmatter editor: an edit touches only
// the key it names and leaves every other byte as it was. Callers that need
// serialization against concurrent writers hold EditMu (or their own lock)
// around parse-edit-write.
type Doc struct {
	lines []string
	rest  string
}

// ParseDoc reads the leading frontmatter block of text.
func ParseDoc(text string) (*Doc, error) {
	block, rest, ok := splitEnvelope(text)
	if !ok {
		return nil, ErrNoFrontmatter
	}
	return &Doc{lines: strings.Split(block, "\n"), rest: rest}, nil
}

// String returns the document with the edits applied.
func (d *Doc) String() string {
	return "---\n" + strings.Join(d.lines, "\n") + d.rest
}

// Body returns the text after the closing marker, without leading blank space.
func (d *Doc) Body() string {
	return strings.TrimLeft(strings.TrimPrefix(d.rest, "\n---"), "\n\r\t ")
}

// Render writes the document as a normalized card: the edited frontmatter, a
// blank line, then body.
func (d *Doc) Render(body string) string {
	return "---\n" + strings.Join(d.lines, "\n") + "\n---\n\n" + body
}

// keyRange finds the lines a key occupies: its own line plus any indented
// continuation lines, or block-list dashes written flush left. It returns -1
// when the key is absent.
func (d *Doc) keyRange(key string) (start, end int) {
	for i, l := range d.lines {
		if !strings.HasPrefix(l, key+":") {
			continue
		}
		end = i
		for end+1 < len(d.lines) && isContinuation(d.lines[end+1]) {
			end++
		}
		return i, end
	}
	return -1, -1
}

func isContinuation(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "- ")
}

func (d *Doc) replace(key string, block []string) {
	if start, end := d.keyRange(key); start >= 0 {
		d.lines = append(d.lines[:start], append(block, d.lines[end+1:]...)...)
		return
	}
	d.lines = append(d.lines, block...)
}

// SetRaw replaces key with "key: value" as given, or appends it before the
// closing marker.
func (d *Doc) SetRaw(key, value string) { d.replace(key, []string{key + ": " + value}) }

// SetString is SetRaw with the value trimmed and quoted when a plain scalar
// could not carry it.
func (d *Doc) SetString(key, value string) {
	d.SetRaw(key, Quote(strings.TrimSpace(value)))
}

// Remove drops key and its continuation lines. A missing key is a no-op.
func (d *Doc) Remove(key string) {
	if start, end := d.keyRange(key); start >= 0 {
		d.lines = append(d.lines[:start], d.lines[end+1:]...)
	}
}

// Scalar reads a top-level key's value, unquoted. ok is false when absent.
func (d *Doc) Scalar(key string) (value string, ok bool) {
	start, _ := d.keyRange(key)
	if start < 0 {
		return "", false
	}
	return unquote(strings.TrimPrefix(d.lines[start], key+":")), true
}

// Items reads key as a block list or an inline [a, b] list, unquoted.
func (d *Doc) Items(key string) []string {
	start, end := d.keyRange(key)
	if start < 0 {
		return nil
	}
	var items []string
	if inline := strings.TrimSpace(strings.TrimPrefix(d.lines[start], key+":")); inline != "" {
		for _, part := range strings.Split(strings.Trim(inline, "[]"), ",") {
			if part = unquote(part); part != "" {
				items = append(items, part)
			}
		}
		return items
	}
	for _, l := range d.lines[start+1 : end+1] {
		if item, ok := strings.CutPrefix(strings.TrimSpace(l), "-"); ok {
			if item = unquote(item); item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

// SetItems replaces key with a block list written as given; no items leaves a
// bare "key:".
func (d *Doc) SetItems(key string, items []string) {
	block := []string{key + ":"}
	for _, item := range items {
		block = append(block, "  - "+item)
	}
	d.replace(key, block)
}

// SetStrings replaces key with a block list of quoted strings, or "key: []"
// when there are none.
func (d *Doc) SetStrings(key string, values []string) {
	if len(values) == 0 {
		d.SetRaw(key, "[]")
		return
	}
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = Quote(strings.TrimSpace(value))
	}
	d.SetItems(key, quoted)
}

// Records reads key as a block of records; see ListRecords.
func (d *Doc) Records(key string) []map[string]string {
	start, end := d.keyRange(key)
	if start < 0 {
		return nil
	}
	var records []map[string]string
	var current map[string]string
	for _, l := range d.lines[start+1 : end+1] {
		if item, ok := strings.CutPrefix(l, "  - "); ok {
			current = map[string]string{}
			records = append(records, current)
			setRecordField(current, item)
			continue
		}
		if current == nil {
			continue
		}
		setRecordField(current, strings.TrimSpace(l))
	}
	return records
}

// SetRecords replaces key with a block of records in the given field order.
// Values are quoted as needed, so any string round-trips through Records.
func (d *Doc) SetRecords(key string, fields []string, records []map[string]string) {
	block := []string{key + ":"}
	for _, record := range records {
		prefix := "  - "
		for _, field := range fields {
			value, ok := record[field]
			if !ok {
				continue
			}
			block = append(block, prefix+field+": "+Quote(value))
			prefix = "    "
		}
	}
	d.replace(key, block)
}

// SetFirstParagraph replaces the first non-heading, non-empty paragraph after
// the frontmatter block, leaving any heading before it and anything after the
// following blank line untouched. It appends the paragraph when the body has
// none yet.
func (d *Doc) SetFirstParagraph(paragraph string) error {
	_, body, ok := strings.Cut(d.rest, "\n---")
	if !ok {
		return ErrNoFrontmatter
	}
	d.rest = "\n---" + replaceFirstParagraph(body, paragraph)
	return nil
}

// Quote double-quotes a value that a plain YAML scalar could not carry safely
// — a colon, '#', quotes, a backslash, flow or indicator characters, edge
// whitespace, or any control character (a newline would otherwise end the
// frontmatter early). Other values come back unchanged.
func Quote(value string) string {
	needs := strings.ContainsAny(value, `:#[]{}"'\`) ||
		strings.ContainsFunc(value, unicode.IsControl) ||
		value != strings.TrimSpace(value)
	if !needs && value != "" {
		needs = strings.ContainsRune("-?&*!|>%@`", rune(value[0]))
	}
	if !needs {
		return value
	}
	return strconv.Quote(value)
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

// unquoteValue reverses Quote: a double-quoted value is unescaped with strconv
// so quotes, colons and newlines round-trip; anything else falls back to
// unquote.
func unquoteValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	return unquote(value)
}

func setRecordField(record map[string]string, line string) {
	field, value, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	record[strings.TrimSpace(field)] = unquoteValue(value)
}
