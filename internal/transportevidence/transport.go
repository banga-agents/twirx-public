// Package transportevidence encodes the immutable E2 redirect chain and a
// strict allowlist of representation-relevant response headers. It augments,
// but does not mutate, the E1 observation envelope.
package transportevidence

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const (
	Version             = "tw.transport/0.2"
	MaxBytes            = 128 << 10
	MaxURLBytes         = 8192
	MaxHeaders          = 32
	MaxHeaderValueBytes = 8192
)

var allowedHeaderNames = map[string]struct{}{
	"content-encoding": {}, "content-language": {}, "content-location": {}, "content-type": {},
}

type Evidence struct {
	Version, RequestURL, FinalURL, PolicyID string
	Redirects                               []safefetch.Redirect
	Headers                                 []safefetch.Header
}

func FromFetch(result *safefetch.Result, policyID string) (*Evidence, error) {
	if result == nil {
		return nil, errors.New("transportevidence: nil fetch result")
	}
	evidence := &Evidence{Version: Version, RequestURL: result.RequestURL, FinalURL: result.FinalURL, PolicyID: policyID, Redirects: append([]safefetch.Redirect(nil), result.Redirects...), Headers: append([]safefetch.Header(nil), result.Headers...)}
	if err := evidence.Validate(); err != nil {
		return nil, err
	}
	return evidence, nil
}

func (e *Evidence) Validate() error {
	if e == nil || e.Version != Version {
		return errors.New("transportevidence: unsupported version")
	}
	for name, value := range map[string]string{"request URL": e.RequestURL, "final URL": e.FinalURL, "policy ID": e.PolicyID} {
		if err := bounded(name, value, MaxURLBytes); err != nil {
			return err
		}
	}
	if len(e.Redirects) > 20 {
		return errors.New("transportevidence: too many redirects")
	}
	for i, redirect := range e.Redirects {
		if err := bounded("redirect from", redirect.FromURL, MaxURLBytes); err != nil {
			return err
		}
		if err := bounded("redirect to", redirect.ToURL, MaxURLBytes); err != nil {
			return err
		}
		if redirect.Status < 300 || redirect.Status > 399 {
			return fmt.Errorf("transportevidence: redirect %d has non-redirect status", i)
		}
		if i == 0 && redirect.FromURL != e.RequestURL {
			return errors.New("transportevidence: redirect chain does not start at request URL")
		}
		if i > 0 && redirect.FromURL != e.Redirects[i-1].ToURL {
			return errors.New("transportevidence: redirect chain is discontinuous")
		}
	}
	if len(e.Redirects) > 0 && e.Redirects[len(e.Redirects)-1].ToURL != e.FinalURL {
		return errors.New("transportevidence: redirect chain does not end at final URL")
	}
	if len(e.Redirects) == 0 && e.RequestURL != e.FinalURL {
		return errors.New("transportevidence: changed final URL requires redirect evidence")
	}
	if len(e.Headers) > MaxHeaders {
		return errors.New("transportevidence: too many headers")
	}
	previousName, previousValue := "", ""
	for _, header := range e.Headers {
		if _, ok := allowedHeaderNames[header.Name]; !ok {
			return fmt.Errorf("transportevidence: header %q is not allowed", header.Name)
		}
		if header.Name < previousName || (header.Name == previousName && header.Value < previousValue) {
			return errors.New("transportevidence: headers are not sorted")
		}
		if err := bounded("header value", header.Value, MaxHeaderValueBytes); err != nil {
			return err
		}
		previousName, previousValue = header.Name, header.Value
	}
	return nil
}

func (e *Evidence) MarshalCBOR() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(7)
	enc.Text(e.Version)
	enc.Text(e.RequestURL)
	enc.Text(e.FinalURL)
	enc.Text(e.PolicyID)
	enc.Array(uint64(len(e.Redirects)))
	for _, redirect := range e.Redirects {
		enc.Array(3)
		enc.Text(redirect.FromURL)
		enc.Uint(uint64(redirect.Status))
		enc.Text(redirect.ToURL)
	}
	enc.Array(uint64(len(e.Headers)))
	for _, header := range e.Headers {
		enc.Array(2)
		enc.Text(header.Name)
		enc.Text(header.Value)
	}
	enc.Array(0)
	encoded := enc.Bytes()
	if len(encoded) > MaxBytes {
		return nil, errors.New("transportevidence: encoded artifact exceeds bound")
	}
	return encoded, nil
}

func UnmarshalCBOR(data []byte) (*Evidence, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return nil, errors.New("transportevidence: byte length outside bounds")
	}
	dec := cborlite.NewDecoder(data)
	n, err := dec.Array()
	if err != nil || n != 7 {
		return nil, errors.New("transportevidence: invalid top-level array")
	}
	e := &Evidence{}
	if e.Version, err = dec.Text(128); err != nil {
		return nil, err
	}
	if e.RequestURL, err = dec.Text(MaxURLBytes); err != nil {
		return nil, err
	}
	if e.FinalURL, err = dec.Text(MaxURLBytes); err != nil {
		return nil, err
	}
	if e.PolicyID, err = dec.Text(256); err != nil {
		return nil, err
	}
	redirectCount, err := dec.Array()
	if err != nil || redirectCount > 20 {
		return nil, errors.New("transportevidence: invalid redirect array")
	}
	for i := uint64(0); i < redirectCount; i++ {
		entry, entryErr := dec.Array()
		if entryErr != nil || entry != 3 {
			return nil, errors.New("transportevidence: invalid redirect entry")
		}
		var redirect safefetch.Redirect
		if redirect.FromURL, err = dec.Text(MaxURLBytes); err != nil {
			return nil, err
		}
		status, statusErr := dec.Uint()
		if statusErr != nil || status > 599 {
			return nil, errors.New("transportevidence: invalid redirect status")
		}
		redirect.Status = int(status)
		if redirect.ToURL, err = dec.Text(MaxURLBytes); err != nil {
			return nil, err
		}
		e.Redirects = append(e.Redirects, redirect)
	}
	headerCount, err := dec.Array()
	if err != nil || headerCount > MaxHeaders {
		return nil, errors.New("transportevidence: invalid header array")
	}
	for i := uint64(0); i < headerCount; i++ {
		entry, entryErr := dec.Array()
		if entryErr != nil || entry != 2 {
			return nil, errors.New("transportevidence: invalid header entry")
		}
		var header safefetch.Header
		if header.Name, err = dec.Text(64); err != nil {
			return nil, err
		}
		if header.Value, err = dec.Text(MaxHeaderValueBytes); err != nil {
			return nil, err
		}
		e.Headers = append(e.Headers, header)
	}
	extensions, err := dec.Array()
	if err != nil || extensions != 0 {
		return nil, errors.New("transportevidence: extensions must be empty")
	}
	if dec.Remaining() != 0 {
		return nil, errors.New("transportevidence: trailing bytes")
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return e, nil
}

func bounded(name, value string, max int) error {
	if value == "" || len(value) > max || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("transportevidence: %s is invalid or outside bounds", name)
	}
	return nil
}

// SortHeaders is exported for controlled fixture producers. Network fetches
// already emit canonical order.
func SortHeaders(headers []safefetch.Header) {
	sort.Slice(headers, func(i, j int) bool {
		if headers[i].Name == headers[j].Name {
			return headers[i].Value < headers[j].Value
		}
		return headers[i].Name < headers[j].Name
	})
}
