package openrtb

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestEnrichmentRequest(t *testing.T) {
	// Test that EnrichmentRequest is compatible with OpenRTB 2.5 BidRequest
	request := EnrichmentRequest{
		ID: "test-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "imp1",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	// Test conversion to OpenRTB BidRequest
	openrtbRequest := openrtb2.BidRequest(request)
	assert.Equal(t, request.ID, openrtbRequest.ID)
	assert.Len(t, openrtbRequest.Imp, 1)
	assert.Equal(t, request.Imp[0].ID, openrtbRequest.Imp[0].ID)

	// Test conversion from OpenRTB BidRequest
	convertedRequest := EnrichmentRequest(openrtbRequest)
	assert.Equal(t, request.ID, convertedRequest.ID)
	assert.Len(t, convertedRequest.Imp, 1)
	assert.Equal(t, request.Imp[0].ID, convertedRequest.Imp[0].ID)

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(request)
	assert.NoError(t, err)

	var unmarshaled EnrichmentRequest
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, request.ID, unmarshaled.ID)
	assert.Len(t, unmarshaled.Imp, 1)
	assert.Equal(t, request.Imp[0].ID, unmarshaled.Imp[0].ID)
}

func TestEnrichmentResponse(t *testing.T) {
	response := EnrichmentResponse{
		ID: "test-id",
		User: &EnrichmentUser{
			Data: []openrtb2.Data{
				{
					Name: "segment-provider.com",
					Segment: []openrtb2.Segment{
						{ID: "123"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaled EnrichmentResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, response.ID, unmarshaled.ID)
	assert.NotNil(t, unmarshaled.User)
	assert.Len(t, unmarshaled.User.Data, 1)
	assert.Equal(t, response.User.Data[0].Name, unmarshaled.User.Data[0].Name)
	assert.Len(t, unmarshaled.User.Data[0].Segment, 1)
	assert.Equal(t, response.User.Data[0].Segment[0].ID, unmarshaled.User.Data[0].Segment[0].ID)
}

func TestEnrichmentResponseWithDealEnrichment(t *testing.T) {
	response := EnrichmentResponse{
		ID: "test-id",
		Imp: []EnrichmentImp{
			{
				ID: "imp1",
				PMP: &EnrichmentPMP{
					Deals: []openrtb2.Deal{
						{
							ID:          "deal-123",
							BidFloor:    2.50,
							BidFloorCur: "USD",
						},
						{
							ID:          "deal-456",
							BidFloor:    3.00,
							BidFloorCur: "EUR",
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaled EnrichmentResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, response.ID, unmarshaled.ID)
	assert.NotNil(t, unmarshaled.Imp)
	assert.Len(t, unmarshaled.Imp, 1)
	assert.Equal(t, response.Imp[0].ID, unmarshaled.Imp[0].ID)
	assert.NotNil(t, unmarshaled.Imp[0].PMP)
	assert.Len(t, unmarshaled.Imp[0].PMP.Deals, 2)
	assert.Equal(t, response.Imp[0].PMP.Deals[0].ID, unmarshaled.Imp[0].PMP.Deals[0].ID)
	assert.Equal(t, response.Imp[0].PMP.Deals[0].BidFloor, unmarshaled.Imp[0].PMP.Deals[0].BidFloor)
	assert.Equal(t, response.Imp[0].PMP.Deals[0].BidFloorCur, unmarshaled.Imp[0].PMP.Deals[0].BidFloorCur)
	assert.Equal(t, response.Imp[0].PMP.Deals[1].ID, unmarshaled.Imp[0].PMP.Deals[1].ID)
	assert.Equal(t, response.Imp[0].PMP.Deals[1].BidFloor, unmarshaled.Imp[0].PMP.Deals[1].BidFloor)
	assert.Equal(t, response.Imp[0].PMP.Deals[1].BidFloorCur, unmarshaled.Imp[0].PMP.Deals[1].BidFloorCur)
}

func TestEnrichmentResponseWithMultipleEnrichments(t *testing.T) {
	// Test response with segments, EIDs, and deal enrichment
	response := EnrichmentResponse{
		ID: "test-id",
		User: &EnrichmentUser{
			Data: []openrtb2.Data{
				{
					Name: "segment-provider.com",
					Segment: []openrtb2.Segment{
						{ID: "123"},
					},
				},
			},
			Ext: &EnrichmentExt{
				EIDs: []openrtb2.EID{
					{
						Source: "id-provider.com",
						UIDs: []openrtb2.UID{
							{ID: "abc"},
						},
					},
				},
			},
		},
		Imp: []EnrichmentImp{
			{
				ID: "imp1",
				PMP: &EnrichmentPMP{
					Deals: []openrtb2.Deal{
						{
							ID:          "deal-123",
							BidFloor:    2.50,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaled EnrichmentResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, response.ID, unmarshaled.ID)

	// Verify user enrichment
	assert.NotNil(t, unmarshaled.User)
	assert.Len(t, unmarshaled.User.Data, 1)
	assert.Len(t, unmarshaled.User.Data[0].Segment, 1)
	assert.Equal(t, "123", unmarshaled.User.Data[0].Segment[0].ID)
	assert.NotNil(t, unmarshaled.User.Ext)
	assert.Len(t, unmarshaled.User.Ext.EIDs, 1)
	assert.Equal(t, "id-provider.com", unmarshaled.User.Ext.EIDs[0].Source)

	// Verify deal enrichment
	assert.NotNil(t, unmarshaled.Imp)
	assert.Len(t, unmarshaled.Imp, 1)
	assert.NotNil(t, unmarshaled.Imp[0].PMP)
	assert.Len(t, unmarshaled.Imp[0].PMP.Deals, 1)
	assert.Equal(t, "deal-123", unmarshaled.Imp[0].PMP.Deals[0].ID)
	assert.Equal(t, 2.50, unmarshaled.Imp[0].PMP.Deals[0].BidFloor)
	assert.Equal(t, "USD", unmarshaled.Imp[0].PMP.Deals[0].BidFloorCur)
}
