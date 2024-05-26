package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	// ApiUrl is the official Kraken API Endpoint
	ApiUrl = "https://api.kraken.com"

	ApiVersion = "0"

	// ApiUserAgent identifies this library with the Kraken API
	ApiUserAgent = "Kraken GO API Agent (https://github.com/jenyasd/ltp-server)"
)

var (
	ErrNoPairs         = errors.New("no pairs provided")
	ErrCannotCreateReq = errors.New("cannot create request")
	ErrRequest         = errors.New("cannot make request")
	ErrReadBody        = errors.New("cannot read request body")
	ErrGetContentType  = errors.New("cannot get Content-Type")
	ErrBadContentType  = errors.New("wrong Content-Type")
	ErrParseJsonBody   = errors.New("cannot parse json")
	ErrKraken          = errors.New("kraken API error")
	ErrFloatPars       = errors.New("cannot parse float")
)

type PairTickerInfo struct {
	Close []string `json:"c"` // 'c' stands for the last trade closed. Other info is not required
}

type TickerResponse map[string]*PairTickerInfo

type KrakenResponse struct {
	Error  []string    `json:"error"`
	Result interface{} `json:"result"`
}

func DefaultPriceRequester() *PriceRequester {
	client := &http.Client{}
	return NewPriceRequester(ApiUrl, ApiVersion, client)
}

func NewPriceRequester(url, version string, client *http.Client) *PriceRequester {
	return &PriceRequester{
		apiUrl:     url,
		apiVersion: version,
		client:     client,
	}
}

type PriceRequester struct {
	apiUrl     string
	apiVersion string
	client     *http.Client
}

func (pr *PriceRequester) GetLtp(pairs ...string) ([]PriceInfo, error) {
	if len(pairs) == 0 {
		return nil, ErrNoPairs
	}

	tickerResp := TickerResponse{}
	params := url.Values{}
	params.Add("pair", strings.Join(pairs, ","))

	err := pr.queryGet("Ticker", params, &tickerResp)
	if err != nil {
		return nil, err
	}

	priceInfos := make([]PriceInfo, 0, len(pairs))
	for _, curr := range pairs {
		tickerData, ok := tickerResp[curr]
		if !ok {
			fmt.Printf("No data in response for %s\n", curr)
			continue
		}

		if len(tickerData.Close) == 0 {
			fmt.Printf("No last traded price for %s\n", curr)
			continue
		}

		lastTradedPrice, err := strconv.ParseFloat(tickerData.Close[0], 64)
		if err != nil {
			return nil, errors.Join(ErrFloatPars, err)
		}

		priceInfos = append(priceInfos, PriceInfo{
			Pair:   curr,
			Amount: lastTradedPrice,
		})
	}

	return priceInfos, nil
}

func (pr *PriceRequester) queryGet(reqPath string, values url.Values, typ interface{}) error {
	reqUrl := fmt.Sprintf("%s/%s/public/%s", pr.apiUrl, ApiVersion, reqPath)

	encodedValues := values.Encode()
	fullURL := reqUrl + "?" + encodedValues

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return errors.Join(ErrCannotCreateReq, err)
	}

	return pr.doApiRequest(req, nil, typ)
}

func (pr *PriceRequester) doApiRequest(req *http.Request, headers map[string]string, typ interface{}) error {
	req.Header.Add("User-Agent", ApiUserAgent)
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Execute request
	resp, err := pr.client.Do(req)
	if err != nil {
		return errors.Join(ErrRequest, err)
	}
	defer resp.Body.Close()

	// Read request
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Join(ErrReadBody, err)
	}

	// Check mime type of response
	mimeType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return errors.Join(ErrGetContentType, err)
	}

	if mimeType != "application/json" {
		err := errors.New(fmt.Sprintf("Response Content-Type is '%s', but should be 'application/json'.", mimeType))
		return errors.Join(ErrBadContentType, err)
	}

	var krakenResponse KrakenResponse
	if typ != nil {
		krakenResponse.Result = typ
	}

	err = json.Unmarshal(body, &krakenResponse)
	if err != nil {
		return errors.Join(ErrParseJsonBody, err)
	}

	if len(krakenResponse.Error) > 0 {
		return errors.Join(ErrKraken, err)
	}

	return nil
}
