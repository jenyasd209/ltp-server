package src

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	BtcUsdPair = "BTC/USD"
	BtcEurPair = "BTC/EUR"
	BtcChfPair = "BTC/CHF"
)

var (
	ErrBadLtpMethod   = "bad method to get ltp"
	ErrGetLtp         = "error getting ltp"
	ErrMarshalJsonLtp = "error marshal json"
	ErrWriteResponse  = "error write response"
)

type PriceGetter interface {
	GetLtp(...string) ([]PriceInfo, error)
}

type PriceInfo struct {
	Pair   string  `json:"pair"`
	Amount float64 `json:"amount"`
}

type LtpResponse struct {
	Ltp []PriceInfo `json:"ltp"`
}

func NewServer(addr string, getter PriceGetter) *Server {
	return &Server{
		Addr:           addr,
		PriceGetter:    getter,
		cachedTime:     time.Minute,
		lastCached:     atomic.Pointer[time.Time]{},
		cachedResponse: atomic.Pointer[[]byte]{},
	}
}

type Server struct {
	Addr string

	PriceGetter PriceGetter

	cachedTime     time.Duration
	lastCached     atomic.Pointer[time.Time]
	cachedResponse atomic.Pointer[[]byte]
}

func (s *Server) Listen() error {
	http.HandleFunc("/api/v1/ltp", s.handlePriceRequest)

	fmt.Printf("Start listenning on %s...\n", s.Addr)
	return http.ListenAndServe(s.Addr, nil)
}

func (s *Server) handlePriceRequest(w http.ResponseWriter, r *http.Request) {
	// check method
	fmt.Printf("New %s request on %s from %s\n", r.Method, r.URL, r.RemoteAddr)
	if r.Method != http.MethodGet {
		http.Error(w, ErrBadLtpMethod, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// if the last cached resp in not aged return it
	if s.lastCached.Load() != nil && time.Now().Sub(*s.lastCached.Load()) < s.cachedTime {
		// Load and write the cached response
		_, err := w.Write(*s.cachedResponse.Load())
		if err != nil {
			http.Error(w, ErrWriteResponse, http.StatusInternalServerError)
			return
		}

		return
	}

	// retrieve prices from Kraken
	res, err := s.PriceGetter.GetLtp(BtcChfPair, BtcUsdPair, BtcEurPair)
	if err != nil {
		http.Error(w, ErrGetLtp, http.StatusInternalServerError)
		return
	}

	// Create a response
	resp := LtpResponse{Ltp: make([]PriceInfo, 0, len(res))}
	for _, re := range res {
		resp.Ltp = append(resp.Ltp, re)
	}

	// Marshal the response
	bResp, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, ErrMarshalJsonLtp, http.StatusInternalServerError)
		return
	}
	// Cache the response
	s.cachedResponse.Store(&bResp)
	now := time.Now()
	s.lastCached.Store(&now)

	// Write the response
	_, err = w.Write(bResp)
	if err != nil {
		http.Error(w, ErrWriteResponse, http.StatusInternalServerError)
		return
	}
}
