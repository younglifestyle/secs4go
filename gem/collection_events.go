package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// CollectionEvent represents a GEM collection event definition (CEID).
type CollectionEvent struct {
	info idInfo

	Name string
}

// NewCollectionEvent constructs a collection event with the supplied identifier and name.
func NewCollectionEvent(id interface{}, name string) (*CollectionEvent, error) {
	info, err := newIDInfo(id)
	if err != nil {
		return nil, err
	}
	return &CollectionEvent{info: info, Name: name}, nil
}

func (ce *CollectionEvent) ID() interface{} {
	return ce.info.raw
}

func (ce *CollectionEvent) idKey() string {
	return ce.info.key
}

func (ce *CollectionEvent) idNode() ast.ItemNode {
	return ce.info.node
}

// ReportDefinition captures a RPTID and the list of VID identifiers attached to it.
type ReportDefinition struct {
	info    idInfo
	vidKeys []string
}

func newReportDefinition(id interface{}, vids []idInfo) (*ReportDefinition, error) {
	info, err := newIDInfo(id)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(vids))
	for _, vid := range vids {
		keys = append(keys, vid.key)
	}
	return &ReportDefinition{info: info, vidKeys: keys}, nil
}

func (rd *ReportDefinition) ID() interface{} {
	return rd.info.raw
}

func (rd *ReportDefinition) idKey() string {
	return rd.info.key
}

func (rd *ReportDefinition) idNode() ast.ItemNode {
	return rd.info.node
}

// collectionEventLink ties a CEID to the list of report identifiers and enable flag.
type collectionEventLink struct {
	reports []string
	enabled bool
}

func newCollectionEventLink(reports []string) *collectionEventLink {
	copyReports := make([]string, len(reports))
	copy(copyReports, reports)
	return &collectionEventLink{reports: copyReports, enabled: true}
}

func (link *collectionEventLink) removeReport(key string) {
	for idx, rpt := range link.reports {
		if rpt == key {
			link.reports = append(link.reports[:idx], link.reports[idx+1:]...)
			return
		}
	}
}

func ensureIDInfoSlice(values []interface{}) ([]idInfo, error) {
	result := make([]idInfo, 0, len(values))
	for _, v := range values {
		info, err := newIDInfo(v)
		if err != nil {
			return nil, fmt.Errorf("invalid identifier %v: %w", v, err)
		}
		result = append(result, info)
	}
	return result, nil
}
