// Package robotstxt parses and evaluates the Robots Exclusion Protocol.
// It performs no network access. Retrieval policy remains a separate trust
// boundary.
package robotstxt

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxBytes is the E3 parsing ceiling. RFC 9309 requires a parser limit of
	// at least 500 KiB; E3 uses exactly that bound and fails closed above it.
	MaxBytes       = 500 * 1024
	MaxTargetBytes = 16 << 10
)

type Rule struct {
	Allow       bool
	Pattern     string
	normalized  string
	anchorEnd   bool
	specificity int
}

type Group struct {
	Agents []string
	Rules  []Rule
}

type Document struct {
	Groups      []Group
	ParseErrors int
}

type Decision struct {
	Allowed     bool
	Matched     bool
	Pattern     string
	Specificity int
}

type FetchResult string

const (
	FetchSuccessful    FetchResult = "successful"
	FetchUnavailable   FetchResult = "unavailable"
	FetchUnreachable   FetchResult = "unreachable"
	FetchRedirectLimit FetchResult = "redirect_limit"
)

// ClassifyFetch records RFC 9309 retrieval outcomes without turning them into
// Atlas admission. E3 treats unreachable and uncertain outcomes as closed;
// a 4xx unavailable result still requires an explicit human access decision.
func ClassifyFetch(status, redirects int, transportFailed bool) FetchResult {
	if transportFailed {
		return FetchUnreachable
	}
	if redirects > 5 {
		return FetchRedirectLimit
	}
	switch {
	case status >= 200 && status <= 299:
		return FetchSuccessful
	case status >= 400 && status <= 499:
		return FetchUnavailable
	default:
		return FetchUnreachable
	}
}

func Parse(data []byte) (*Document, error) {
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("robotstxt: document exceeds %d-byte limit", MaxBytes)
	}
	if !utf8.Valid(data) {
		return nil, errors.New("robotstxt: document is not valid UTF-8")
	}
	document := &Document{Groups: []Group{}}
	var current *Group
	hadRule := false
	for _, rawLine := range splitLines(data) {
		line := trimProtocolSpace(rawLine)
		if index := strings.IndexByte(line, '#'); index >= 0 {
			line = trimProtocolSpace(line[:index])
		}
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			document.ParseErrors++
			continue
		}
		key := strings.ToLower(trimProtocolSpace(line[:colon]))
		value := trimProtocolSpace(line[colon+1:])
		switch key {
		case "user-agent":
			if !validProductToken(value, true) {
				document.ParseErrors++
				continue
			}
			if current == nil || hadRule {
				if current != nil {
					document.Groups = append(document.Groups, *current)
				}
				current = &Group{Agents: []string{}, Rules: []Rule{}}
				hadRule = false
			}
			current.Agents = append(current.Agents, value)
		case "allow", "disallow":
			if current == nil {
				continue
			}
			hadRule = true
			if value == "" {
				continue
			}
			rule, err := parseRule(key == "allow", value)
			if err != nil {
				document.ParseErrors++
				continue
			}
			current.Rules = append(current.Rules, rule)
		default:
			// Other records, including Sitemap, cannot terminate a group.
		}
	}
	if current != nil {
		document.Groups = append(document.Groups, *current)
	}
	return document, nil
}

func (d *Document) Evaluate(productToken, pathQuery string) (Decision, error) {
	if d == nil {
		return Decision{}, errors.New("robotstxt: document is nil")
	}
	if len(productToken) > 128 || !validProductToken(productToken, false) {
		return Decision{}, errors.New("robotstxt: invalid product token")
	}
	if len(pathQuery) > MaxTargetBytes {
		return Decision{}, errors.New("robotstxt: target exceeds byte limit")
	}
	if pathQuery == "/robots.txt" || strings.HasPrefix(pathQuery, "/robots.txt?") {
		return Decision{Allowed: true}, nil
	}
	normalized, err := normalizePath(pathQuery, false)
	if err != nil {
		return Decision{}, fmt.Errorf("robotstxt: target: %w", err)
	}
	rules := d.rulesFor(productToken)
	decision := Decision{Allowed: true}
	for _, rule := range rules {
		if !globMatches(rule.normalized, normalized, rule.anchorEnd) {
			continue
		}
		if !decision.Matched || rule.specificity > decision.Specificity || rule.specificity == decision.Specificity && rule.Allow && !decision.Allowed {
			decision = Decision{Allowed: rule.Allow, Matched: true, Pattern: rule.Pattern, Specificity: rule.specificity}
		}
	}
	return decision, nil
}

func (d *Document) rulesFor(productToken string) []Rule {
	exact := make([]Rule, 0)
	wildcard := make([]Rule, 0)
	foundExact := false
	for _, group := range d.Groups {
		matchesExact := false
		matchesWildcard := false
		for _, agent := range group.Agents {
			if strings.EqualFold(agent, productToken) {
				matchesExact = true
			}
			if agent == "*" {
				matchesWildcard = true
			}
		}
		if matchesExact {
			foundExact = true
			exact = append(exact, group.Rules...)
		} else if matchesWildcard {
			wildcard = append(wildcard, group.Rules...)
		}
	}
	if foundExact {
		return exact
	}
	return wildcard
}

func parseRule(allow bool, pattern string) (Rule, error) {
	if pattern[0] != '/' && pattern[0] != '*' {
		return Rule{}, errors.New("rule must begin with slash or wildcard")
	}
	anchor := strings.HasSuffix(pattern, "$")
	matchPattern := pattern
	if anchor {
		matchPattern = strings.TrimSuffix(matchPattern, "$")
	}
	normalized, err := normalizePath(matchPattern, true)
	if err != nil {
		return Rule{}, err
	}
	specificity := 0
	for index := 0; index < len(normalized); index++ {
		if normalized[index] != '*' {
			specificity++
		}
	}
	return Rule{Allow: allow, Pattern: pattern, normalized: normalized, anchorEnd: anchor, specificity: specificity}, nil
}

func normalizePath(value string, pattern bool) (string, error) {
	if value == "" || !utf8.ValidString(value) {
		return "", errors.New("path is empty or invalid UTF-8")
	}
	if !pattern && value[0] != '/' {
		return "", errors.New("target must begin with slash")
	}
	var result strings.Builder
	result.Grow(len(value))
	for index := 0; index < len(value); {
		current := value[index]
		if current == '%' {
			if index+2 >= len(value) {
				return "", errors.New("incomplete percent encoding")
			}
			high, okHigh := hexValue(value[index+1])
			low, okLow := hexValue(value[index+2])
			if !okHigh || !okLow {
				return "", errors.New("invalid percent encoding")
			}
			decoded := high<<4 | low
			if isUnreserved(decoded) {
				result.WriteByte(decoded)
			} else {
				result.WriteByte('%')
				result.WriteByte(upperHex(decoded >> 4))
				result.WriteByte(upperHex(decoded & 0x0f))
			}
			index += 3
			continue
		}
		if current < utf8.RuneSelf {
			if current < 0x20 || current == 0x7f {
				return "", errors.New("control character in path")
			}
			result.WriteByte(current)
			index++
			continue
		}
		runeValue, width := utf8.DecodeRuneInString(value[index:])
		encoded := make([]byte, utf8.RuneLen(runeValue))
		utf8.EncodeRune(encoded, runeValue)
		for _, octet := range encoded[:width] {
			result.WriteByte('%')
			result.WriteByte(upperHex(octet >> 4))
			result.WriteByte(upperHex(octet & 0x0f))
		}
		index += width
	}
	return result.String(), nil
}

func globMatches(pattern, target string, anchorEnd bool) bool {
	patternIndex, targetIndex := 0, 0
	starIndex, retryIndex := -1, 0
	for targetIndex < len(target) {
		if patternIndex == len(pattern) {
			return !anchorEnd
		}
		if pattern[patternIndex] == '*' {
			starIndex = patternIndex
			patternIndex++
			retryIndex = targetIndex
			continue
		}
		if pattern[patternIndex] == target[targetIndex] {
			patternIndex++
			targetIndex++
			continue
		}
		if starIndex >= 0 {
			retryIndex++
			targetIndex = retryIndex
			patternIndex = starIndex + 1
			continue
		}
		return false
	}
	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}

func validProductToken(value string, wildcard bool) bool {
	if wildcard && value == "*" {
		return true
	}
	if value == "" {
		return false
	}
	for index := range value {
		char := value[index]
		if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func splitLines(data []byte) []string {
	data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
	data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
	parts := bytes.Split(data, []byte{'\n'})
	lines := make([]string, len(parts))
	for index := range parts {
		lines[index] = string(parts[index])
	}
	return lines
}

func trimProtocolSpace(value string) string {
	return strings.Trim(value, " \t")
}

func hexValue(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func isUnreserved(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || strings.ContainsRune("-._~", rune(value))
}

func upperHex(value byte) byte {
	return "0123456789ABCDEF"[value&0x0f]
}
