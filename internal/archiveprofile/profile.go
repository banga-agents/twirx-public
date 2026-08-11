// Package archiveprofile performs a narrow deterministic extraction from an
// already verified historical representation. It has no network, mapping, or
// canon-admission authority.
package archiveprofile

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	Format  = "tw.archive-native-profile/0.1"
	Locator = "html/head/title[1]/text()"
	MaxBody = 5 << 20
)

type Spec struct {
	Format      string `json:"format"`
	ID          string `json:"id"`
	OriginID    string `json:"origin_id"`
	OperationID string `json:"operation_id"`
	Subject     string `json:"subject"`
	Predicate   string `json:"predicate"`
	Locator     string `json:"locator"`
	Kind        string `json:"kind"`
	MediaType   string `json:"media_type"`
}

type ExtractionPlan struct {
	Format     string   `json:"format"`
	ProfileID  string   `json:"profile_id"`
	Predicate  string   `json:"native_predicate"`
	Locator    string   `json:"native_locator"`
	Transforms []string `json:"transformations"`
	Mappings   []string `json:"mappings"`
}

type Statement struct {
	NativeLexical string
	Locator       string
}

func ParseSpec(data []byte) (Spec, error) {
	var spec Spec
	policy := jsonbounded.Policy{MaxBytes: 16 << 10, MaxDepth: 4, MaxScalarBytes: 4096, MaxContainerEntries: 32, MaxTokens: 256}
	if err := jsonbounded.Decode(data, &spec, policy, true); err != nil {
		return spec, err
	}
	if spec.Format != Format || !identifier(spec.ID) || !identifier(spec.OriginID) || !identifier(spec.OperationID) || !identifier(spec.Subject) || !identifier(spec.Predicate) || spec.Locator != Locator || spec.Kind != "state" || spec.MediaType != "text/html" {
		return spec, errors.New("archiveprofile: invalid native profile")
	}
	return spec, nil
}

func PlanBytes(spec Spec) ([]byte, error) {
	if _, err := ParseSpec(mustMarshal(spec)); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(ExtractionPlan{Format: "tw.archive-extraction-plan/0.1", ProfileID: spec.ID, Predicate: spec.Predicate, Locator: spec.Locator, Transforms: []string{}, Mappings: []string{}}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ExtractTitle returns the exact UTF-8 bytes between the first title tags in
// the head. It deliberately does not decode entities, collapse whitespace, or
// apply a semantic mapping.
func ExtractTitle(body []byte) (Statement, error) {
	if len(body) == 0 || len(body) > MaxBody || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return Statement{}, errors.New("archiveprofile: representation is outside bounds")
	}
	lower := bytes.ToLower(body)
	_, headOpenEnd, err := findOpeningTag(lower, "head", 0, len(lower))
	if err != nil || headOpenEnd < 0 {
		return Statement{}, errors.New("archiveprofile: head element absent")
	}
	_, titleOpenEnd, err := findOpeningTagBeforeClose(lower, "title", "head", headOpenEnd, len(lower))
	if err != nil || titleOpenEnd < 0 {
		return Statement{}, errors.New("archiveprofile: title element absent from head")
	}
	titleClose, _, err := findClosingTag(lower, "title", titleOpenEnd, len(lower))
	if err != nil || titleClose < 0 {
		return Statement{}, errors.New("archiveprofile: closing title element absent")
	}
	lexical := body[titleOpenEnd:titleClose]
	if len(lexical) == 0 || len(lexical) > 64<<10 || bytes.IndexByte(lexical, '<') >= 0 {
		return Statement{}, errors.New("archiveprofile: title lexical value is unresolved")
	}
	return Statement{NativeLexical: string(lexical), Locator: Locator}, nil
}

func findOpeningTagBeforeClose(lower []byte, wanted, boundary string, offset, limit int) (int, int, error) {
	for offset < limit {
		start, end, name, closing, err := nextTag(lower, offset, limit)
		if err != nil || start < 0 {
			return -1, -1, err
		}
		if closing && name == boundary {
			return -1, -1, nil
		}
		if !closing && name == wanted {
			return start, end, nil
		}
		if !closing && (name == "script" || name == "style") {
			_, rawEnd, rawErr := findClosingTag(lower, name, end, limit)
			if rawErr != nil || rawEnd < 0 {
				return -1, -1, errors.New("archiveprofile: unterminated raw-text element")
			}
			offset = rawEnd
			continue
		}
		offset = end
	}
	return -1, -1, nil
}

func findOpeningTag(lower []byte, wanted string, offset, limit int) (int, int, error) {
	for offset < limit {
		start, end, name, closing, err := nextTag(lower, offset, limit)
		if err != nil || start < 0 {
			return -1, -1, err
		}
		if !closing && name == wanted {
			return start, end, nil
		}
		if !closing && wanted != name && (name == "script" || name == "style") {
			_, rawEnd, rawErr := findClosingTag(lower, name, end, limit)
			if rawErr != nil || rawEnd < 0 {
				return -1, -1, errors.New("archiveprofile: unterminated raw-text element")
			}
			offset = rawEnd
			continue
		}
		offset = end
	}
	return -1, -1, nil
}

func findClosingTag(lower []byte, wanted string, offset, limit int) (int, int, error) {
	for offset < limit {
		start, end, name, closing, err := nextTag(lower, offset, limit)
		if err != nil || start < 0 {
			return -1, -1, err
		}
		if closing && name == wanted {
			return start, end, nil
		}
		if !closing && (name == "script" || name == "style") {
			_, rawEnd, rawErr := findClosingTag(lower, name, end, limit)
			if rawErr != nil || rawEnd < 0 {
				return -1, -1, errors.New("archiveprofile: unterminated raw-text element")
			}
			offset = rawEnd
			continue
		}
		offset = end
	}
	return -1, -1, nil
}

func nextTag(lower []byte, offset, limit int) (int, int, string, bool, error) {
	for offset < limit {
		relative := bytes.IndexByte(lower[offset:limit], '<')
		if relative < 0 {
			return -1, -1, "", false, nil
		}
		start := offset + relative
		if bytes.HasPrefix(lower[start:limit], []byte("<!--")) {
			close := bytes.Index(lower[start+4:limit], []byte("-->"))
			if close < 0 {
				return -1, -1, "", false, errors.New("archiveprofile: unterminated comment")
			}
			offset = start + 4 + close + 3
			continue
		}
		cursor := start + 1
		closing := false
		if cursor < limit && lower[cursor] == '/' {
			closing = true
			cursor++
		}
		nameStart := cursor
		for cursor < limit && tagNameByte(lower[cursor]) {
			cursor++
		}
		if cursor == nameStart {
			offset = start + 1
			continue
		}
		name := string(lower[nameStart:cursor])
		quote := byte(0)
		for cursor < limit {
			character := lower[cursor]
			if quote != 0 {
				if character == quote {
					quote = 0
				}
			} else if character == '\'' || character == '"' {
				quote = character
			} else if character == '>' {
				return start, cursor + 1, name, closing, nil
			}
			cursor++
		}
		return -1, -1, "", false, errors.New("archiveprofile: unterminated tag")
	}
	return -1, -1, "", false, nil
}

func tagNameByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || value == ':' || value == '-'
}

func identifier(value string) bool {
	if value == "" || len(value) > 1024 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if character < 0x21 || character == 0x7f {
			return false
		}
	}
	return true
}

func mustMarshal(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}
