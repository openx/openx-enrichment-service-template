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
