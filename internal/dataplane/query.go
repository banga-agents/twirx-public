package dataplane

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
)

const (
	QueryVersion           = "tw.semantic-query/0.1"
	SubscriptionVersion    = "tw.semantic-subscription/0.1"
	QueryResultVersion     = "tw.semantic-query-result/0.1"
	MaterializationVersion = "tw.materialization-manifest/0.1"
	EconomicEventVersion   = "tw.economic-event/0.1"
)

type QuerySubject struct {
	Concept OptionalText
	IDs     []string
}

type QueryDimension struct {
	Key      string
	Relation string
	Values   []TypedValue
}

type QueryTime struct {
	Mode  string
	From  OptionalText
	Until OptionalText
}

type QueryOntology struct {
	MaximumDepth              uint64
	MaximumPathCostMillionths uint64
	AllowedEdgeStatuses       []string
}

type QuerySources struct {
	AllowedOriginIDs        []string
	MinimumDistinctOrigins  uint64
	AllowedAuthorityClasses []string
}

type QueryTrust struct {
	AllowedLanes           []string
	AllowedMappingStatuses []string
}

type QueryFreshness struct {
	MaximumAgeSeconds *uint64
	StaleBehavior     string
}

type QueryEconomics struct {
	MaximumPrice          OptionalText
	Currency              OptionalText
	AllowedFundingClasses []string
}

type QueryExecution struct {
	AllowMaterializedState bool
	AllowLiveRefresh       bool
	MaximumLiveOrigins     uint64
	DeadlineMilliseconds   uint64
}

type QueryProof struct {
	Level         string
	IncludePlan   bool
	IncludeNative bool
}

type QueryLimits struct {
	MaximumResults    uint64
	MaximumPackets    uint64
	MaximumProofBytes uint64
}

type Query struct {
	Version    string
	Select     []string
	Subject    QuerySubject
	Dimensions []QueryDimension
	Time       QueryTime
	Ontology   QueryOntology
	Sources    QuerySources
	Trust      QueryTrust
	Freshness  QueryFreshness
	Economics  QueryEconomics
	Conflicts  string
	Execution  QueryExecution
	Proof      QueryProof
	Preference string
	Limits     QueryLimits
}

func (q Query) Validate() error {
	if q.Version != QueryVersion {
		return fmt.Errorf("%w: query version", ErrInvalid)
	}
	if err := validateSortedUniqueText("selected concepts", q.Select, 32); err != nil || len(q.Select) == 0 {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: at least one selected concept is required", ErrInvalid)
	}
	if err := validateOptionalIdentifier("subject concept", q.Subject.Concept); err != nil {
		return err
	}
	if err := validateSortedUniqueText("subject IDs", q.Subject.IDs, 256); err != nil {
		return err
	}
	if !q.Subject.Concept.Present && len(q.Subject.IDs) == 0 {
		return fmt.Errorf("%w: query subject is empty", ErrInvalid)
	}
	if len(q.Dimensions) > 32 {
		return fmt.Errorf("%w: too many query dimensions", ErrInvalid)
	}
	for i, dimension := range q.Dimensions {
		if err := validateIdentifier("dimension key", dimension.Key); err != nil {
			return err
		}
		if i > 0 && strings.Compare(q.Dimensions[i-1].Key, dimension.Key) >= 0 {
			return fmt.Errorf("%w: query dimensions must be strictly sorted", ErrInvalid)
		}
		if err := validateEnum("dimension relation", dimension.Relation, "eq", "in", "lt", "lte", "gt", "gte"); err != nil {
			return err
		}
		if len(dimension.Values) == 0 || len(dimension.Values) > 64 {
			return fmt.Errorf("%w: dimension values outside 1..64", ErrInvalid)
		}
		var prior []byte
		for _, value := range dimension.Values {
			if err := value.Validate(); err != nil {
				return err
			}
			var enc cborlite.Encoder
			encodeTypedValue(&enc, value)
			current := enc.Bytes()
			if prior != nil && bytes.Compare(prior, current) >= 0 {
				return fmt.Errorf("%w: dimension values must be canonically sorted", ErrInvalid)
			}
			prior = current
		}
	}
	if err := validateEnum("time mode", q.Time.Mode, "current", "as_of", "between", "history"); err != nil {
		return err
	}
	if err := validateOptionalUTCSecond("time from", q.Time.From); err != nil {
		return err
	}
	if err := validateOptionalUTCSecond("time until", q.Time.Until); err != nil {
		return err
	}
	switch q.Time.Mode {
	case "current":
		if q.Time.From.Present || q.Time.Until.Present {
			return fmt.Errorf("%w: current mode has no bounds", ErrInvalid)
		}
	case "as_of":
		if !q.Time.Until.Present || q.Time.From.Present {
			return fmt.Errorf("%w: as_of requires until only", ErrInvalid)
		}
	case "between":
		if !q.Time.From.Present || !q.Time.Until.Present || q.Time.From.Value > q.Time.Until.Value {
			return fmt.Errorf("%w: invalid between bounds", ErrInvalid)
		}
	case "history":
		if q.Time.From.Present && q.Time.Until.Present && q.Time.From.Value > q.Time.Until.Value {
			return fmt.Errorf("%w: invalid history bounds", ErrInvalid)
		}
	}
	if q.Ontology.MaximumDepth > 16 || q.Ontology.MaximumPathCostMillionths > 16000000 {
		return fmt.Errorf("%w: ontology bounds exceeded", ErrInvalid)
	}
	if err := validateEnumTextSet("edge statuses", q.Ontology.AllowedEdgeStatuses, 3, 1, "candidate", "reviewed", "disputed"); err != nil {
		return err
	}
	if err := validateSortedUniqueText("allowed origin IDs", q.Sources.AllowedOriginIDs, 256); err != nil {
		return err
	}
	if q.Sources.MinimumDistinctOrigins == 0 || q.Sources.MinimumDistinctOrigins > 32 {
		return fmt.Errorf("%w: minimum origins outside 1..32", ErrInvalid)
	}
	if len(q.Sources.AllowedOriginIDs) > 0 && q.Sources.MinimumDistinctOrigins > uint64(len(q.Sources.AllowedOriginIDs)) {
		return fmt.Errorf("%w: minimum origins exceeds allowlist", ErrInvalid)
	}
	if err := validateSortedUniqueText("authority classes", q.Sources.AllowedAuthorityClasses, 32); err != nil {
		return err
	}
	if err := validateEnumTextSet("trust lanes", q.Trust.AllowedLanes, 3, 1, "observed_native", "provisional_semantic", "attested_semantic"); err != nil {
		return err
	}
	if err := validateEnumTextSet("mapping statuses", q.Trust.AllowedMappingStatuses, 4, 1, "none", "candidate", "reviewed", "disputed"); err != nil {
		return err
	}
	if q.Freshness.MaximumAgeSeconds != nil && *q.Freshness.MaximumAgeSeconds > 315576000 {
		return fmt.Errorf("%w: maximum age exceeds bound", ErrInvalid)
	}
	if err := validateEnum("stale behavior", q.Freshness.StaleBehavior, "exclude", "return_explicit_stale", "request_refresh"); err != nil {
		return err
	}
	if q.Freshness.StaleBehavior == "request_refresh" && !q.Execution.AllowLiveRefresh {
		return fmt.Errorf("%w: refresh behavior requires live-refresh permission", ErrInvalid)
	}
	if q.Economics.MaximumPrice.Present {
		if !decimalPattern.MatchString(q.Economics.MaximumPrice.Value) && !integerPattern.MatchString(q.Economics.MaximumPrice.Value) {
			return fmt.Errorf("%w: maximum price is not canonical decimal text", ErrInvalid)
		}
		if !q.Economics.Currency.Present {
			return fmt.Errorf("%w: maximum price requires currency", ErrInvalid)
		}
	} else if q.Economics.MaximumPrice.Value != "" || q.Economics.Currency.Present {
		return fmt.Errorf("%w: currency requires maximum price", ErrInvalid)
	}
	if err := validateCurrency(q.Economics.Currency); err != nil {
		return err
	}
	if err := validateSortedUniqueText("funding classes", q.Economics.AllowedFundingClasses, 16); err != nil {
		return err
	}
	if err := validateEnum("conflict mode", q.Conflicts, "preserve_sources", "group_equivalent", "reject_conflict"); err != nil {
		return err
	}
	if q.Execution.MaximumLiveOrigins > 8 || q.Execution.DeadlineMilliseconds == 0 || q.Execution.DeadlineMilliseconds > 30000 {
		return fmt.Errorf("%w: execution bounds exceeded", ErrInvalid)
	}
	if !q.Execution.AllowLiveRefresh && q.Execution.MaximumLiveOrigins != 0 {
		return fmt.Errorf("%w: live-origin budget without live refresh", ErrInvalid)
	}
	if !q.Execution.AllowMaterializedState && !q.Execution.AllowLiveRefresh {
		return fmt.Errorf("%w: query has no admitted execution mode", ErrInvalid)
	}
	if err := validateEnum("proof level", q.Proof.Level, "packet", "field", "bundle"); err != nil {
		return err
	}
	if !q.Proof.IncludeNative {
		return fmt.Errorf("%w: native proof is mandatory", ErrInvalid)
	}
	if err := validatePreference(q.Preference); err != nil {
		return err
	}
	if q.Limits.MaximumResults == 0 || q.Limits.MaximumResults > 1000 || q.Limits.MaximumPackets == 0 || q.Limits.MaximumPackets > 10000 || q.Limits.MaximumProofBytes < 1024 || q.Limits.MaximumProofBytes > 16777216 {
		return fmt.Errorf("%w: query limits outside bounds", ErrInvalid)
	}
	return nil
}

func validatePreference(value string) error {
	return validateEnum("preference", value, "most_authoritative", "freshest", "fastest", "least_expensive", "highest_proof", "balanced")
}

func validateEnumTextSet(name string, values []string, maximum int, minimum int, allowed ...string) error {
	if len(values) < minimum || len(values) > maximum {
		return fmt.Errorf("%w: %s count outside bounds", ErrInvalid, name)
	}
	if err := validateSortedUniqueText(name, values, maximum); err != nil {
		return err
	}
	for _, value := range values {
		if err := validateEnum(name, value, allowed...); err != nil {
			return err
		}
	}
	return nil
}

func boolUint(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func decodeBool(dec *cborlite.Decoder) (bool, error) {
	value, err := dec.Uint()
	if err != nil || value > 1 {
		return false, fmt.Errorf("%w: boolean must be 0 or 1", ErrInvalid)
	}
	return value == 1, nil
}

func MarshalQuery(q Query) ([]byte, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(16)
	enc.Text(q.Version)
	encodeTextSet(&enc, q.Select)
	enc.Array(2)
	encodeOptionalText(&enc, q.Subject.Concept)
	encodeTextSet(&enc, q.Subject.IDs)
	enc.Array(uint64(len(q.Dimensions)))
	for _, dimension := range q.Dimensions {
		enc.Array(3)
		enc.Text(dimension.Key)
		enc.Text(dimension.Relation)
		enc.Array(uint64(len(dimension.Values)))
		for _, value := range dimension.Values {
			encodeTypedValue(&enc, value)
		}
	}
	enc.Array(3)
	enc.Text(q.Time.Mode)
	encodeOptionalText(&enc, q.Time.From)
	encodeOptionalText(&enc, q.Time.Until)
	enc.Array(3)
	enc.Uint(q.Ontology.MaximumDepth)
	enc.Uint(q.Ontology.MaximumPathCostMillionths)
	encodeTextSet(&enc, q.Ontology.AllowedEdgeStatuses)
	enc.Array(3)
	encodeTextSet(&enc, q.Sources.AllowedOriginIDs)
	enc.Uint(q.Sources.MinimumDistinctOrigins)
	encodeTextSet(&enc, q.Sources.AllowedAuthorityClasses)
	enc.Array(2)
	encodeTextSet(&enc, q.Trust.AllowedLanes)
	encodeTextSet(&enc, q.Trust.AllowedMappingStatuses)
	enc.Array(2)
	if q.Freshness.MaximumAgeSeconds == nil {
		enc.Nil()
	} else {
		enc.Uint(*q.Freshness.MaximumAgeSeconds)
	}
	enc.Text(q.Freshness.StaleBehavior)
	enc.Array(3)
	encodeOptionalText(&enc, q.Economics.MaximumPrice)
	encodeOptionalText(&enc, q.Economics.Currency)
	encodeTextSet(&enc, q.Economics.AllowedFundingClasses)
	enc.Array(1)
	enc.Text(q.Conflicts)
	enc.Array(4)
	enc.Uint(boolUint(q.Execution.AllowMaterializedState))
	enc.Uint(boolUint(q.Execution.AllowLiveRefresh))
	enc.Uint(q.Execution.MaximumLiveOrigins)
	enc.Uint(q.Execution.DeadlineMilliseconds)
	enc.Array(3)
	enc.Text(q.Proof.Level)
	enc.Uint(boolUint(q.Proof.IncludePlan))
	enc.Uint(boolUint(q.Proof.IncludeNative))
	enc.Text(q.Preference)
	enc.Array(3)
	enc.Uint(q.Limits.MaximumResults)
	enc.Uint(q.Limits.MaximumPackets)
	enc.Uint(q.Limits.MaximumProofBytes)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalQuery(data []byte) (Query, error) {
	var q Query
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 16 {
		return q, fmt.Errorf("%w: query array", ErrInvalid)
	}
	if q.Version, err = dec.Text(MaxIdentifier); err != nil {
		return q, err
	}
	if q.Select, err = decodeTextSet(dec, 32, 1); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 2 {
		return q, fmt.Errorf("%w: query subject array", ErrInvalid)
	}
	if q.Subject.Concept, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return q, err
	}
	if q.Subject.IDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return q, err
	}
	dimensionCount, countErr := dec.Array()
	if countErr != nil || dimensionCount > 32 {
		return q, fmt.Errorf("%w: query dimension count", ErrInvalid)
	}
	for i := uint64(0); i < dimensionCount; i++ {
		if n, e := dec.Array(); e != nil || n != 3 {
			return q, fmt.Errorf("%w: query dimension array", ErrInvalid)
		}
		var dimension QueryDimension
		if dimension.Key, err = dec.Text(MaxIdentifier); err != nil {
			return q, err
		}
		if dimension.Relation, err = dec.Text(MaxIdentifier); err != nil {
			return q, err
		}
		valueCount, valueErr := dec.Array()
		if valueErr != nil || valueCount == 0 || valueCount > 64 {
			return q, fmt.Errorf("%w: query dimension value count", ErrInvalid)
		}
		for j := uint64(0); j < valueCount; j++ {
			value, decodeErr := decodeTypedValue(dec)
			if decodeErr != nil {
				return q, decodeErr
			}
			dimension.Values = append(dimension.Values, value)
		}
		q.Dimensions = append(q.Dimensions, dimension)
	}
	if n, e := dec.Array(); e != nil || n != 3 {
		return q, fmt.Errorf("%w: query time array", ErrInvalid)
	}
	if q.Time.Mode, err = dec.Text(MaxIdentifier); err != nil {
		return q, err
	}
	if q.Time.From, err = decodeOptionalText(dec, 20); err != nil {
		return q, err
	}
	if q.Time.Until, err = decodeOptionalText(dec, 20); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 3 {
		return q, fmt.Errorf("%w: query ontology array", ErrInvalid)
	}
	if q.Ontology.MaximumDepth, err = dec.Uint(); err != nil {
		return q, err
	}
	if q.Ontology.MaximumPathCostMillionths, err = dec.Uint(); err != nil {
		return q, err
	}
	if q.Ontology.AllowedEdgeStatuses, err = decodeTextSet(dec, 3, 1); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 3 {
		return q, fmt.Errorf("%w: query sources array", ErrInvalid)
	}
	if q.Sources.AllowedOriginIDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return q, err
	}
	if q.Sources.MinimumDistinctOrigins, err = dec.Uint(); err != nil {
		return q, err
	}
	if q.Sources.AllowedAuthorityClasses, err = decodeTextSet(dec, 32, 0); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 2 {
		return q, fmt.Errorf("%w: query trust array", ErrInvalid)
	}
	if q.Trust.AllowedLanes, err = decodeTextSet(dec, 3, 1); err != nil {
		return q, err
	}
	if q.Trust.AllowedMappingStatuses, err = decodeTextSet(dec, 4, 1); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 2 {
		return q, fmt.Errorf("%w: freshness array", ErrInvalid)
	}
	if nilValue, nilErr := dec.TryNil(); nilErr != nil {
		return q, nilErr
	} else if !nilValue {
		value, valueErr := dec.Uint()
		if valueErr != nil {
			return q, valueErr
		}
		q.Freshness.MaximumAgeSeconds = &value
	}
	if q.Freshness.StaleBehavior, err = dec.Text(MaxIdentifier); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 3 {
		return q, fmt.Errorf("%w: economics array", ErrInvalid)
	}
	if q.Economics.MaximumPrice, err = decodeOptionalText(dec, 128); err != nil {
		return q, err
	}
	if q.Economics.Currency, err = decodeOptionalText(dec, 3); err != nil {
		return q, err
	}
	if q.Economics.AllowedFundingClasses, err = decodeTextSet(dec, 16, 0); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 1 {
		return q, fmt.Errorf("%w: conflicts array", ErrInvalid)
	}
	if q.Conflicts, err = dec.Text(MaxIdentifier); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 4 {
		return q, fmt.Errorf("%w: execution array", ErrInvalid)
	}
	if q.Execution.AllowMaterializedState, err = decodeBool(dec); err != nil {
		return q, err
	}
	if q.Execution.AllowLiveRefresh, err = decodeBool(dec); err != nil {
		return q, err
	}
	if q.Execution.MaximumLiveOrigins, err = dec.Uint(); err != nil {
		return q, err
	}
	if q.Execution.DeadlineMilliseconds, err = dec.Uint(); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 3 {
		return q, fmt.Errorf("%w: proof array", ErrInvalid)
	}
	if q.Proof.Level, err = dec.Text(MaxIdentifier); err != nil {
		return q, err
	}
	if q.Proof.IncludePlan, err = decodeBool(dec); err != nil {
		return q, err
	}
	if q.Proof.IncludeNative, err = decodeBool(dec); err != nil {
		return q, err
	}
	if q.Preference, err = dec.Text(MaxIdentifier); err != nil {
		return q, err
	}
	if n, e := dec.Array(); e != nil || n != 3 {
		return q, fmt.Errorf("%w: limits array", ErrInvalid)
	}
	if q.Limits.MaximumResults, err = dec.Uint(); err != nil {
		return q, err
	}
	if q.Limits.MaximumPackets, err = dec.Uint(); err != nil {
		return q, err
	}
	if q.Limits.MaximumProofBytes, err = dec.Uint(); err != nil {
		return q, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return q, err
	}
	if err = finish(dec); err != nil {
		return q, err
	}
	return q, q.Validate()
}

type Subscription struct {
	Version                string
	QueryDigest            Digest
	DeltaClasses           []string
	DeltaKinds             []string
	Delivery               string
	ResumeAfterSequence    uint64
	MaximumEventsPerMinute uint64
	ProofLevel             string
	ExpiresAt              OptionalText
}

func (s Subscription) Validate() error {
	if s.Version != SubscriptionVersion {
		return fmt.Errorf("%w: subscription version", ErrInvalid)
	}
	if err := validateEnumTextSet("delta classes", s.DeltaClasses, 3, 1, "origin", "semantic", "canon"); err != nil {
		return err
	}
	if err := validateEnumTextSet("delta kinds", s.DeltaKinds, 16, 1, "added", "modified", "withdrawn", "restored", "source_retracted", "mapped", "remapped", "narrowed", "broadened", "disputed", "attested", "de_attested", "module_added", "module_superseded", "mapping_superseded", "closure_changed"); err != nil {
		return err
	}
	if err := validateEnum("delivery", s.Delivery, "sse", "poll"); err != nil {
		return err
	}
	if s.MaximumEventsPerMinute == 0 || s.MaximumEventsPerMinute > 600 {
		return fmt.Errorf("%w: subscription rate outside bounds", ErrInvalid)
	}
	if err := validateEnum("subscription proof", s.ProofLevel, "packet", "bundle"); err != nil {
		return err
	}
	return validateOptionalUTCSecond("subscription expiry", s.ExpiresAt)
}

func MarshalSubscription(s Subscription) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(10)
	enc.Text(s.Version)
	encodeDigest(&enc, s.QueryDigest)
	encodeTextSet(&enc, s.DeltaClasses)
	encodeTextSet(&enc, s.DeltaKinds)
	enc.Text(s.Delivery)
	enc.Uint(s.ResumeAfterSequence)
	enc.Uint(s.MaximumEventsPerMinute)
	enc.Text(s.ProofLevel)
	encodeOptionalText(&enc, s.ExpiresAt)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalSubscription(data []byte) (Subscription, error) {
	var s Subscription
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return s, err
	}
	if n, e := dec.Array(); e != nil || n != 10 {
		return s, fmt.Errorf("%w: subscription array", ErrInvalid)
	}
	if s.Version, err = dec.Text(MaxIdentifier); err != nil {
		return s, err
	}
	if s.QueryDigest, err = decodeDigest(dec); err != nil {
		return s, err
	}
	if s.DeltaClasses, err = decodeTextSet(dec, 3, 1); err != nil {
		return s, err
	}
	if s.DeltaKinds, err = decodeTextSet(dec, 16, 1); err != nil {
		return s, err
	}
	if s.Delivery, err = dec.Text(MaxIdentifier); err != nil {
		return s, err
	}
	if s.ResumeAfterSequence, err = dec.Uint(); err != nil {
		return s, err
	}
	if s.MaximumEventsPerMinute, err = dec.Uint(); err != nil {
		return s, err
	}
	if s.ProofLevel, err = dec.Text(MaxIdentifier); err != nil {
		return s, err
	}
	if s.ExpiresAt, err = decodeOptionalText(dec, 20); err != nil {
		return s, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return s, err
	}
	if err = finish(dec); err != nil {
		return s, err
	}
	return s, s.Validate()
}

type QueryResultRow struct {
	SubjectID         string
	PredicateID       string
	Status            string
	NativeTerm        string
	NativeLocator     string
	NativeLexical     string
	SemanticTerm      OptionalText
	Typed             *TypedValue
	OriginID          string
	PacketDigest      Digest
	ObservationDigest Digest
	Lane              string
	ObservedAt        string
}

type ConflictGroup struct {
	SemanticKeyDigest Digest
	Kind              string
	PacketDigests     []Digest
	Resolution        string
}

type QueryResult struct {
	Version             string
	QueryDigest         Digest
	PlanDigest          Digest
	Preference          string
	SnapshotSequence    uint64
	Status              string
	Rows                []QueryResultRow
	Conflicts           []ConflictGroup
	ProofArtifacts      []DigestSizeEntry
	EconomicEventDigest Digest
	GeneratedAt         string
}

func (r QueryResult) Validate() error {
	if r.Version != QueryResultVersion {
		return fmt.Errorf("%w: query result version", ErrInvalid)
	}
	if err := validatePreference(r.Preference); err != nil {
		return err
	}
	if err := validateEnum("result status", r.Status, "resolved", "partial", "unresolved"); err != nil {
		return err
	}
	if len(r.Rows) > 1000 || len(r.Conflicts) > 256 {
		return fmt.Errorf("%w: result collections exceed bounds", ErrInvalid)
	}
	for i, row := range r.Rows {
		if err := row.Validate(); err != nil {
			return fmt.Errorf("%w: row %d: %v", ErrInvalid, i, err)
		}
		if i > 0 && compareRows(r.Rows[i-1], row) >= 0 {
			return fmt.Errorf("%w: rows must be strictly sorted", ErrInvalid)
		}
	}
	for i, conflict := range r.Conflicts {
		if err := conflict.Validate(); err != nil {
			return fmt.Errorf("%w: conflict %d: %v", ErrInvalid, i, err)
		}
		if i > 0 && bytes.Compare(r.Conflicts[i-1].SemanticKeyDigest[:], conflict.SemanticKeyDigest[:]) >= 0 {
			return fmt.Errorf("%w: conflicts must be strictly sorted", ErrInvalid)
		}
	}
	if len(r.ProofArtifacts) == 0 || len(r.ProofArtifacts) > 10000 {
		return fmt.Errorf("%w: proof artifact count outside bounds", ErrInvalid)
	}
	if err := validateDigestEntries("proof artifacts", r.ProofArtifacts, 10000); err != nil {
		return err
	}
	if err := validateUTCSecond("generated at", r.GeneratedAt); err != nil {
		return err
	}
	if r.Status == "resolved" && len(r.Rows) == 0 {
		return fmt.Errorf("%w: resolved result requires rows", ErrInvalid)
	}
	if r.Status == "unresolved" && len(r.Rows) != 0 {
		return fmt.Errorf("%w: unresolved result cannot carry rows", ErrInvalid)
	}
	return nil
}

func (r QueryResultRow) Validate() error {
	identifiers := []struct{ name, value string }{
		{"subject", r.SubjectID},
		{"predicate", r.PredicateID},
		{"native term", r.NativeTerm},
		{"native locator", r.NativeLocator},
		{"origin", r.OriginID},
	}
	for _, field := range identifiers {
		maximum := MaxIdentifier
		if field.name == "native locator" {
			maximum = MaxLocator
		}
		if err := validateText(field.name, field.value, maximum, false); err != nil {
			return err
		}
	}
	if err := validateEnum("row status", r.Status, "resolved", "unknown", "not_observed", "not_provided", "not_applicable", "withheld", "redacted", "unresolved", "contradictory", "invalid", "confirmed_absent"); err != nil {
		return err
	}
	if r.Status == "resolved" {
		if err := validateText("native lexical", r.NativeLexical, MaxLexical, true); err != nil {
			return err
		}
	} else if r.NativeLexical != "" || r.Typed != nil {
		return fmt.Errorf("%w: unresolved row must omit lexical and typed values", ErrInvalid)
	}
	if err := validateOptionalIdentifier("semantic term", r.SemanticTerm); err != nil {
		return err
	}
	if r.Typed != nil {
		if r.Status != "resolved" {
			return fmt.Errorf("%w: typed row requires resolved status", ErrInvalid)
		}
		if err := r.Typed.Validate(); err != nil {
			return err
		}
	}
	if err := validateEnum("row lane", r.Lane, "observed_native", "provisional_semantic", "attested_semantic"); err != nil {
		return err
	}
	if r.Lane == "observed_native" && r.SemanticTerm.Present {
		return fmt.Errorf("%w: observed-native row cannot claim a semantic term", ErrInvalid)
	}
	if r.Lane != "observed_native" && !r.SemanticTerm.Present {
		return fmt.Errorf("%w: semantic row requires semantic term", ErrInvalid)
	}
	return validateUTCSecond("row observed at", r.ObservedAt)
}

func (c ConflictGroup) Validate() error {
	if err := validateEnum("conflict kind", c.Kind, "value", "time", "identity", "mapping", "authority"); err != nil {
		return err
	}
	if err := validateSortedUniqueDigests("conflict packets", c.PacketDigests, 32, 2); err != nil {
		return err
	}
	return validateEnum("conflict resolution", c.Resolution, "preserved", "caller_policy_selected", "unresolved")
}

func compareRows(a, b QueryResultRow) int {
	for _, pair := range [][2]string{{a.SubjectID, b.SubjectID}, {a.PredicateID, b.PredicateID}, {a.OriginID, b.OriginID}, {a.ObservedAt, b.ObservedAt}} {
		if cmp := strings.Compare(pair[0], pair[1]); cmp != 0 {
			return cmp
		}
	}
	return bytes.Compare(a.PacketDigest[:], b.PacketDigest[:])
}

func MarshalQueryResult(r QueryResult) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(12)
	enc.Text(r.Version)
	encodeDigest(&enc, r.QueryDigest)
	encodeDigest(&enc, r.PlanDigest)
	enc.Text(r.Preference)
	enc.Uint(r.SnapshotSequence)
	enc.Text(r.Status)
	enc.Array(uint64(len(r.Rows)))
	for _, row := range r.Rows {
		encodeQueryResultRow(&enc, row)
	}
	enc.Array(uint64(len(r.Conflicts)))
	for _, conflict := range r.Conflicts {
		enc.Array(4)
		encodeDigest(&enc, conflict.SemanticKeyDigest)
		enc.Text(conflict.Kind)
		encodeDigestSet(&enc, conflict.PacketDigests)
		enc.Text(conflict.Resolution)
	}
	encodeDigestEntries(&enc, r.ProofArtifacts)
	encodeDigest(&enc, r.EconomicEventDigest)
	enc.Text(r.GeneratedAt)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func encodeQueryResultRow(enc *cborlite.Encoder, row QueryResultRow) {
	enc.Array(13)
	enc.Text(row.SubjectID)
	enc.Text(row.PredicateID)
	enc.Text(row.Status)
	enc.Text(row.NativeTerm)
	enc.Text(row.NativeLocator)
	enc.Text(row.NativeLexical)
	encodeOptionalText(enc, row.SemanticTerm)
	if row.Typed == nil {
		enc.Nil()
	} else {
		encodeTypedValue(enc, *row.Typed)
	}
	enc.Text(row.OriginID)
	encodeDigest(enc, row.PacketDigest)
	encodeDigest(enc, row.ObservationDigest)
	enc.Text(row.Lane)
	enc.Text(row.ObservedAt)
}

func UnmarshalQueryResult(data []byte) (QueryResult, error) {
	var r QueryResult
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return r, err
	}
	if n, e := dec.Array(); e != nil || n != 12 {
		return r, fmt.Errorf("%w: query result array", ErrInvalid)
	}
	if r.Version, err = dec.Text(MaxIdentifier); err != nil {
		return r, err
	}
	if r.QueryDigest, err = decodeDigest(dec); err != nil {
		return r, err
	}
	if r.PlanDigest, err = decodeDigest(dec); err != nil {
		return r, err
	}
	if r.Preference, err = dec.Text(MaxIdentifier); err != nil {
		return r, err
	}
	if r.SnapshotSequence, err = dec.Uint(); err != nil {
		return r, err
	}
	if r.Status, err = dec.Text(MaxIdentifier); err != nil {
		return r, err
	}
	rowCount, e := dec.Array()
	if e != nil || rowCount > 1000 {
		return r, fmt.Errorf("%w: row count", ErrInvalid)
	}
	for i := uint64(0); i < rowCount; i++ {
		row, rowErr := decodeQueryResultRow(dec)
		if rowErr != nil {
			return r, rowErr
		}
		r.Rows = append(r.Rows, row)
	}
	conflictCount, e := dec.Array()
	if e != nil || conflictCount > 256 {
		return r, fmt.Errorf("%w: conflict count", ErrInvalid)
	}
	for i := uint64(0); i < conflictCount; i++ {
		if n, arrayErr := dec.Array(); arrayErr != nil || n != 4 {
			return r, fmt.Errorf("%w: conflict array", ErrInvalid)
		}
		var conflict ConflictGroup
		if conflict.SemanticKeyDigest, err = decodeDigest(dec); err != nil {
			return r, err
		}
		if conflict.Kind, err = dec.Text(MaxIdentifier); err != nil {
			return r, err
		}
		if conflict.PacketDigests, err = decodeDigestSet(dec, 32, 2); err != nil {
			return r, err
		}
		if conflict.Resolution, err = dec.Text(MaxIdentifier); err != nil {
			return r, err
		}
		r.Conflicts = append(r.Conflicts, conflict)
	}
	if r.ProofArtifacts, err = decodeDigestEntries(dec, 10000); err != nil {
		return r, err
	}
	if r.EconomicEventDigest, err = decodeDigest(dec); err != nil {
		return r, err
	}
	if r.GeneratedAt, err = dec.Text(20); err != nil {
		return r, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return r, err
	}
	if err = finish(dec); err != nil {
		return r, err
	}
	return r, r.Validate()
}

func decodeQueryResultRow(dec *cborlite.Decoder) (QueryResultRow, error) {
	var row QueryResultRow
	n, err := dec.Array()
	if err != nil || n != 13 {
		return row, fmt.Errorf("%w: result row array", ErrInvalid)
	}
	if row.SubjectID, err = dec.Text(MaxIdentifier); err != nil {
		return row, err
	}
	if row.PredicateID, err = dec.Text(MaxIdentifier); err != nil {
		return row, err
	}
	if row.Status, err = dec.Text(MaxIdentifier); err != nil {
		return row, err
	}
	if row.NativeTerm, err = dec.Text(MaxIdentifier); err != nil {
		return row, err
	}
	if row.NativeLocator, err = dec.Text(MaxLocator); err != nil {
		return row, err
	}
	if row.NativeLexical, err = dec.Text(MaxLexical); err != nil {
		return row, err
	}
	if row.SemanticTerm, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return row, err
	}
	if nilValue, nilErr := dec.TryNil(); nilErr != nil {
		return row, nilErr
	} else if !nilValue {
		value, valueErr := decodeTypedValue(dec)
		if valueErr != nil {
			return row, valueErr
		}
		row.Typed = &value
	}
	if row.OriginID, err = dec.Text(MaxIdentifier); err != nil {
		return row, err
	}
	if row.PacketDigest, err = decodeDigest(dec); err != nil {
		return row, err
	}
	if row.ObservationDigest, err = decodeDigest(dec); err != nil {
		return row, err
	}
	if row.Lane, err = dec.Text(MaxIdentifier); err != nil {
		return row, err
	}
	if row.ObservedAt, err = dec.Text(20); err != nil {
		return row, err
	}
	return row, nil
}

type MaterializationManifest struct {
	Version              string
	MaterializationID    string
	DefinitionDigest     Digest
	CanonVersion         string
	ThroughSequence      uint64
	PacketDigests        []Digest
	ResultArtifactDigest Digest
	RowCount             uint64
	BuiltAt              string
}

func (m MaterializationManifest) Validate() error {
	if m.Version != MaterializationVersion {
		return fmt.Errorf("%w: materialization version", ErrInvalid)
	}
	if err := validateIdentifier("materialization ID", m.MaterializationID); err != nil {
		return err
	}
	if err := validateIdentifier("canon version", m.CanonVersion); err != nil {
		return err
	}
	if err := validateSortedUniqueDigests("materialization packets", m.PacketDigests, 100000, 0); err != nil {
		return err
	}
	if m.RowCount > 10000000 {
		return fmt.Errorf("%w: materialization row count", ErrInvalid)
	}
	return validateUTCSecond("built at", m.BuiltAt)
}

func MarshalMaterializationManifest(m MaterializationManifest) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(10)
	enc.Text(m.Version)
	enc.Text(m.MaterializationID)
	encodeDigest(&enc, m.DefinitionDigest)
	enc.Text(m.CanonVersion)
	enc.Uint(m.ThroughSequence)
	encodeDigestSet(&enc, m.PacketDigests)
	encodeDigest(&enc, m.ResultArtifactDigest)
	enc.Uint(m.RowCount)
	enc.Text(m.BuiltAt)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalMaterializationManifest(data []byte) (MaterializationManifest, error) {
	var m MaterializationManifest
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return m, err
	}
	if n, e := dec.Array(); e != nil || n != 10 {
		return m, fmt.Errorf("%w: materialization array", ErrInvalid)
	}
	if m.Version, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.MaterializationID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.DefinitionDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.CanonVersion, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.ThroughSequence, err = dec.Uint(); err != nil {
		return m, err
	}
	if m.PacketDigests, err = decodeDigestSet(dec, 100000, 0); err != nil {
		return m, err
	}
	if m.ResultArtifactDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.RowCount, err = dec.Uint(); err != nil {
		return m, err
	}
	if m.BuiltAt, err = dec.Text(20); err != nil {
		return m, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return m, err
	}
	if err = finish(dec); err != nil {
		return m, err
	}
	return m, m.Validate()
}

type EconomicResources struct{ Requests, TransferredBytes, CPUMilliseconds, PeakMemoryBytes, EvidenceBytesWritten, ProofBytesReturned, HumanReviewSeconds uint64 }
type EconomicMoney struct{ Currency, Amount, Class string }
type EconomicEvent struct {
	Version, EventID, OccurredAt string
	OriginID                     OptionalText
	WorkType                     string
	QueryDigest, BatchID         OptionalDigest
	Resources                    EconomicResources
	FundingClass                 string
	SponsorID                    OptionalText
	Cost, Revenue                EconomicMoney
	MeasurementMethod            string
}

func (m EconomicMoney) Validate() error {
	if err := validateCurrency(OptionalText{Present: true, Value: m.Currency}); err != nil {
		return err
	}
	if !decimalPattern.MatchString(m.Amount) && !integerPattern.MatchString(m.Amount) {
		return fmt.Errorf("%w: money amount is not canonical", ErrInvalid)
	}
	return validateIdentifier("money class", m.Class)
}
func (e EconomicEvent) Validate() error {
	if e.Version != EconomicEventVersion {
		return fmt.Errorf("%w: economic event version", ErrInvalid)
	}
	if err := validateIdentifier("event ID", e.EventID); err != nil {
		return err
	}
	if err := validateUTCSecond("occurred at", e.OccurredAt); err != nil {
		return err
	}
	if err := validateOptionalIdentifier("origin ID", e.OriginID); err != nil {
		return err
	}
	if err := validateIdentifier("work type", e.WorkType); err != nil {
		return err
	}
	if err := validateOptionalDigest("query digest", e.QueryDigest); err != nil {
		return err
	}
	if err := validateOptionalDigest("batch ID", e.BatchID); err != nil {
		return err
	}
	if e.Resources.Requests > 1000000 || e.Resources.TransferredBytes > 1099511627776 || e.Resources.CPUMilliseconds > 4294967295 || e.Resources.PeakMemoryBytes > 1099511627776 || e.Resources.EvidenceBytesWritten > 1099511627776 || e.Resources.ProofBytesReturned > 1099511627776 || e.Resources.HumanReviewSeconds > 4294967295 {
		return fmt.Errorf("%w: economic resource bound", ErrInvalid)
	}
	if err := validateIdentifier("funding class", e.FundingClass); err != nil {
		return err
	}
	if err := validateOptionalIdentifier("sponsor ID", e.SponsorID); err != nil {
		return err
	}
	if err := e.Cost.Validate(); err != nil {
		return err
	}
	if err := e.Revenue.Validate(); err != nil {
		return err
	}
	return validateIdentifier("measurement method", e.MeasurementMethod)
}

func encodeMoney(enc *cborlite.Encoder, m EconomicMoney) {
	enc.Array(3)
	enc.Text(m.Currency)
	enc.Text(m.Amount)
	enc.Text(m.Class)
}
func decodeMoney(dec *cborlite.Decoder) (EconomicMoney, error) {
	var m EconomicMoney
	n, err := dec.Array()
	if err != nil || n != 3 {
		return m, fmt.Errorf("%w: money array", ErrInvalid)
	}
	if m.Currency, err = dec.Text(3); err != nil {
		return m, err
	}
	if m.Amount, err = dec.Text(128); err != nil {
		return m, err
	}
	if m.Class, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	return m, nil
}
func MarshalEconomicEvent(e EconomicEvent) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(13)
	enc.Text(e.Version)
	enc.Text(e.EventID)
	enc.Text(e.OccurredAt)
	encodeOptionalText(&enc, e.OriginID)
	enc.Text(e.WorkType)
	encodeOptionalDigest(&enc, e.QueryDigest)
	encodeOptionalDigest(&enc, e.BatchID)
	enc.Array(7)
	enc.Uint(e.Resources.Requests)
	enc.Uint(e.Resources.TransferredBytes)
	enc.Uint(e.Resources.CPUMilliseconds)
	enc.Uint(e.Resources.PeakMemoryBytes)
	enc.Uint(e.Resources.EvidenceBytesWritten)
	enc.Uint(e.Resources.ProofBytesReturned)
	enc.Uint(e.Resources.HumanReviewSeconds)
	enc.Text(e.FundingClass)
	encodeOptionalText(&enc, e.SponsorID)
	encodeMoney(&enc, e.Cost)
	encodeMoney(&enc, e.Revenue)
	enc.Text(e.MeasurementMethod)
	enc.Array(0)
	return boundedEncoding(&enc)
}
func UnmarshalEconomicEvent(data []byte) (EconomicEvent, error) {
	var e EconomicEvent
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return e, err
	}
	if n, x := dec.Array(); x != nil || n != 13 {
		return e, fmt.Errorf("%w: economic event array", ErrInvalid)
	}
	if e.Version, err = dec.Text(MaxIdentifier); err != nil {
		return e, err
	}
	if e.EventID, err = dec.Text(MaxIdentifier); err != nil {
		return e, err
	}
	if e.OccurredAt, err = dec.Text(20); err != nil {
		return e, err
	}
	if e.OriginID, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return e, err
	}
	if e.WorkType, err = dec.Text(MaxIdentifier); err != nil {
		return e, err
	}
	if e.QueryDigest, err = decodeOptionalDigest(dec); err != nil {
		return e, err
	}
	if e.BatchID, err = decodeOptionalDigest(dec); err != nil {
		return e, err
	}
	if n, x := dec.Array(); x != nil || n != 7 {
		return e, fmt.Errorf("%w: economic resources array", ErrInvalid)
	}
	values := []*uint64{&e.Resources.Requests, &e.Resources.TransferredBytes, &e.Resources.CPUMilliseconds, &e.Resources.PeakMemoryBytes, &e.Resources.EvidenceBytesWritten, &e.Resources.ProofBytesReturned, &e.Resources.HumanReviewSeconds}
	for _, target := range values {
		if *target, err = dec.Uint(); err != nil {
			return e, err
		}
	}
	if e.FundingClass, err = dec.Text(MaxIdentifier); err != nil {
		return e, err
	}
	if e.SponsorID, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return e, err
	}
	if e.Cost, err = decodeMoney(dec); err != nil {
		return e, err
	}
	if e.Revenue, err = decodeMoney(dec); err != nil {
		return e, err
	}
	if e.MeasurementMethod, err = dec.Text(MaxIdentifier); err != nil {
		return e, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return e, err
	}
	if err = finish(dec); err != nil {
		return e, err
	}
	return e, e.Validate()
}
