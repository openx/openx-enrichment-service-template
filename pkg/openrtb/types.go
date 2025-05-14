package openrtb

import (
	openrtb20 "github.com/prebid/openrtb/v20/openrtb2"
)

// EnrichmentRequest is an alias for the standard OpenRTB 2.5 BidRequest
type EnrichmentRequest = openrtb20.BidRequest

// EnrichmentResponse is an alias for the standard OpenRTB 2.5 BidRequest
// This allows us to return a fragment of a BidRequest that will be merged back into the original
type EnrichmentResponse = openrtb20.BidRequest
