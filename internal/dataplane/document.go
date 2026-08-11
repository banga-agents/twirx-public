package dataplane

import "fmt"

const (
	KindPacket          = "packet"
	KindBatch           = "batch"
	KindDelta           = "delta"
	KindQuery           = "query"
	KindSubscription    = "subscription"
	KindQueryResult     = "query-result"
	KindMaterialization = "materialization"
	KindEconomicEvent   = "economic-event"
	KindSnapshot        = "snapshot"
)

var DocumentKinds = []string{
	KindBatch,
	KindDelta,
	KindEconomicEvent,
	KindMaterialization,
	KindPacket,
	KindQuery,
	KindQueryResult,
	KindSnapshot,
	KindSubscription,
}

// ValidateDocument dispatches to one exact bounded parser. A caller cannot
// select an embedded version or a generic CBOR mode through untrusted bytes.
func ValidateDocument(kind string, data []byte) error {
	switch kind {
	case KindPacket:
		_, err := UnmarshalPacket(data)
		return err
	case KindBatch:
		_, err := UnmarshalBatchManifest(data)
		return err
	case KindDelta:
		_, err := UnmarshalDelta(data)
		return err
	case KindQuery:
		_, err := UnmarshalQuery(data)
		return err
	case KindSubscription:
		_, err := UnmarshalSubscription(data)
		return err
	case KindQueryResult:
		_, err := UnmarshalQueryResult(data)
		return err
	case KindMaterialization:
		_, err := UnmarshalMaterializationManifest(data)
		return err
	case KindEconomicEvent:
		_, err := UnmarshalEconomicEvent(data)
		return err
	case KindSnapshot:
		_, err := UnmarshalSnapshotManifest(data)
		return err
	default:
		return fmt.Errorf("%w: unsupported document kind %q", ErrInvalid, kind)
	}
}
