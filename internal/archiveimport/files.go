package archiveimport

import "errors"

// LoadIndexResponse reads and parses a bounded regular file produced by an
// explicitly authorized acquisition step. It never performs network access.
func LoadIndexResponse(path string, order WorkOrder, collectionID, route string) ([]Capture, error) {
	if err := order.Validate(); err != nil {
		return nil, err
	}
	data, err := readRegular(path, int64(order.MaxIndexResponseBytes))
	if err != nil {
		return nil, errors.New("archiveimport: cannot read bounded index response")
	}
	return ParseIndexResponse(data, order, collectionID, route)
}

// PublishCaptureFile reads a bounded regular WARC-range file and publishes an
// immutable evidence spool. Acquisition remains outside this offline package.
func PublishCaptureFile(output string, loaded *LoadedWorkOrder, capture Capture, rangeStatus int, contentRange, warcPath string) (*CaptureEvidence, error) {
	if loaded == nil || loaded.Order.Validate() != nil || capture.validateIdentity() != nil {
		return nil, errors.New("archiveimport: invalid capture-file authority")
	}
	compressed, err := readRegular(warcPath, int64(loaded.Order.MaxCompressedRecordBytes))
	if err != nil {
		return nil, errors.New("archiveimport: cannot read bounded WARC range")
	}
	return PublishCapture(output, loaded, capture, rangeStatus, contentRange, compressed)
}
