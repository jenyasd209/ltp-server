package src

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockPriceGetter struct {
	Prices []PriceInfo
	Err    error
}

func (mpg *MockPriceGetter) GetLtp(pairs ...string) ([]PriceInfo, error) {
	return mpg.Prices, mpg.Err
}

func TestServer_HandlePriceRequest_ValidGetRequest(t *testing.T) {
	mockPrices := []PriceInfo{
		{Pair: BtcUsdPair, Amount: 50000.0},
		{Pair: BtcEurPair, Amount: 45000.0},
		{Pair: BtcChfPair, Amount: 47000.0},
	}
	mockGetter := &MockPriceGetter{Prices: mockPrices}
	server := NewServer(":8080", mockGetter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ltp", nil)
	w := httptest.NewRecorder()

	server.handlePriceRequest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var ltpResponse LtpResponse
	err := json.NewDecoder(resp.Body).Decode(&ltpResponse)
	require.NoError(t, err, err)
	assert.Equal(t, len(ltpResponse.Ltp), len(mockPrices))
}

func TestServer_HandlePriceRequest_Caching(t *testing.T) {
	cachedVal := 50000.0
	mockPrices := []PriceInfo{
		{Pair: BtcUsdPair, Amount: cachedVal},
	}
	mockGetter := &MockPriceGetter{Prices: mockPrices}
	server := NewServer(":8080", mockGetter)
	server.cachedTime = 2 * time.Second

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ltp", nil)
	w := httptest.NewRecorder()

	// First request to cache
	server.handlePriceRequest(w, req)

	// Change price to check if price cached
	newVal := 60000.0
	mockGetter.Prices = []PriceInfo{
		{Pair: BtcUsdPair, Amount: newVal},
	}

	// Should return cached value
	w = httptest.NewRecorder()
	server.handlePriceRequest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var ltpResponse LtpResponse
	err := json.NewDecoder(resp.Body).Decode(&ltpResponse)
	require.NoError(t, err, err)
	assert.Equal(t, ltpResponse.Ltp[0].Amount, cachedVal)

	time.Sleep(server.cachedTime)

	// Should return new value
	w = httptest.NewRecorder()
	server.handlePriceRequest(w, req)

	respNew := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, respNew.StatusCode)

	ltpResponse = LtpResponse{}
	err = json.NewDecoder(respNew.Body).Decode(&ltpResponse)
	require.NoError(t, err, err)
	assert.Equal(t, ltpResponse.Ltp[0].Amount, newVal)
}

func TestServer_HandlePriceRequest_FaultCases(t *testing.T) {

	tests := []struct {
		name        string
		mockGetter  *MockPriceGetter
		reqMethod   string
		expRespCode int
	}{
		{
			name:        "InvalidMethod",
			mockGetter:  &MockPriceGetter{},
			reqMethod:   http.MethodPost,
			expRespCode: http.StatusBadRequest,
		},
		{
			name:        "GetLtpError",
			mockGetter:  &MockPriceGetter{Err: fmt.Errorf("some error")},
			reqMethod:   http.MethodGet,
			expRespCode: http.StatusInternalServerError,
		},
		{
			name:        "JsonMarshalError",
			mockGetter:  &MockPriceGetter{Prices: []PriceInfo{{Pair: BtcUsdPair, Amount: math.NaN()}}},
			reqMethod:   http.MethodGet,
			expRespCode: http.StatusInternalServerError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := NewServer(":8080", test.mockGetter)

			req := httptest.NewRequest(test.reqMethod, "/api/v1/ltp", nil)
			w := httptest.NewRecorder()

			server.handlePriceRequest(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.expRespCode, resp.StatusCode)
		})
	}
}
