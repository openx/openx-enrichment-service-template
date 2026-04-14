package openrtb

import (
	openrtb20 "github.com/prebid/openrtb/v20/openrtb2"
)

// EnrichmentRequest is an alias for the standard OpenRTB 2.5 BidRequest
type EnrichmentRequest = openrtb20.BidRequest

// EnrichmentResponse represents the allowed response fields for enrichment
type EnrichmentResponse struct {
	ID   string          `json:"id"`
	User *EnrichmentUser `json:"user,omitempty"`
	Imp  []EnrichmentImp `json:"imp,omitempty"`
}

// EnrichmentImp represents the allowed impression fields for enrichment
type EnrichmentImp struct {
	ID  string         `json:"id"`
	PMP *EnrichmentPMP `json:"pmp,omitempty"`
}

// EnrichmentPMP represents the allowed PMP fields for enrichment
type EnrichmentPMP struct {
	Deals []openrtb20.Deal `json:"deals,omitempty"`
}

// EnrichmentUser represents the allowed user fields for enrichment
type EnrichmentUser struct {
	Data []openrtb20.Data `json:"data,omitempty"`
	Ext  *EnrichmentExt   `json:"ext,omitempty"`
}

// EnrichmentExt represents the allowed extension fields
type EnrichmentExt struct {
	EIDs []openrtb20.EID `json:"eids,omitempty"`
}
