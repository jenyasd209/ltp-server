package src_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jenyasd209/ltp-server/src"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceRequester_GetLtp(t *testing.T) {
	expectedRes := []src.PriceInfo{
		{Pair: src.BtcChfPair, Amount: 1000.1},
		{Pair: src.BtcEurPair, Amount: 2000.2},
		{Pair: src.BtcUsdPair, Amount: 3000.3},
	}

	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expPath := "/0/public/Ticker"
		assert.Equal(t, expPath, r.URL.Path)

		params := url.Values{}
		params.Add("pair", strings.Join([]string{src.BtcChfPair, src.BtcEurPair, src.BtcUsdPair}, ","))
		expParams := params.Encode()
		assert.Equal(t, expParams, r.URL.RawQuery)

		// Mock response
		tr := src.TickerResponse{}
		for _, p := range expectedRes {
			tr[p.Pair] = &src.PairTickerInfo{Close: []string{fmt.Sprintf("%f", p.Amount)}}
		}

		responseData := src.KrakenResponse{
			Error:  make([]string, 0),
			Result: tr,
		}
		bData, err := json.Marshal(responseData)
		require.NoError(t, err, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bData)
		require.NoError(t, err, err)
	}))
	defer server.Close()

	cl := &http.Client{}
	priceRequester := src.NewPriceRequester(server.URL, src.ApiVersion, cl)

	priceInfos, err := priceRequester.GetLtp(src.BtcChfPair, src.BtcEurPair, src.BtcUsdPair)
	assert.NoError(t, err, err)
	assert.Equal(t, expectedRes, priceInfos)
}

func TestPriceRequester_GetLtp_NoPairs(t *testing.T) {
	priceRequester := src.DefaultPriceRequester()

	_, err := priceRequester.GetLtp()
	assert.EqualError(t, err, src.ErrNoPairs.Error())
}
