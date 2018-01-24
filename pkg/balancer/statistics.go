package balancer

import (
	"math"
	"sync"
	"time"
)

// Statistics used to determain balancing
type Statistics struct {
	*sync.RWMutex
	UUID              string    `json:"uuid"`
	ClientsConnected  int64     `json:"clientsconnected"`
	ClientsConnects   int64     `json:"clientsconnects"`
	RX                int64     `json:"rx"`
	TX                int64     `json:"tx"`
	Preference        int       `json:"preference"`
	Topology          []string  `json:"topology"`
	TimeCounter       chan bool `json:"-"`         // counts the elements
	TimeTimer         int       `json:"timetimer"` // time to keep elements
	ResponseTimeValue []float64 `json:"responsetimevalue"`
}

// NewStatistics returns new statistics
func NewStatistics(UUID string, counterSize int) *Statistics {
	return &Statistics{
		RWMutex:           new(sync.RWMutex),
		TimeCounter:       make(chan bool, counterSize),
		TimeTimer:         30, // how long to keep timed histroy (applicable to last connected clients, and response time values)
		ResponseTimeValue: make([]float64, 100),
		UUID:              UUID,
	}
}

// Reset zero's the statistic values
func (s *Statistics) Reset() {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnects = 0
	s.ClientsConnected = 0
	s.RX = 0
	s.TX = 0
	s.ResponseTimeValue = []float64{}
	// TODO: how to reset TimeCounter ? and do we need to since it expires in 30 seconds anyway
}

// ClientsConnectedAdd adds a client to the counter
func (s *Statistics) ClientsConnectedAdd(i int64) {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnected += i
}

// ClientsConnectedSub adds a client to the counter
func (s *Statistics) ClientsConnectedSub(i int64) {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnected -= i
}

// ClientsConnectedSet adds a client to the counter
func (s *Statistics) ClientsConnectedSet(count int64) {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnected = count
}

// ClientsConnectsAdd adds a client to the counter
func (s *Statistics) ClientsConnectsAdd(i int64) {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnects += i
}

// ClientsConnectsSub adds a client to the counter
func (s *Statistics) ClientsConnectsSub(i int64) {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnects -= i
}

// ClientsConnectsSet adds a client to the counter
func (s *Statistics) ClientsConnectsSet(count int64) {
	s.Lock()
	defer s.Unlock()
	s.ClientsConnects = count
}

// RXAdd adds a client to the counter
func (s *Statistics) RXAdd(rx int64) {
	s.Lock()
	defer s.Unlock()
	s.RX += rx
}

// TXAdd adds a client to the counter
func (s *Statistics) TXAdd(tx int64) {
	s.Lock()
	defer s.Unlock()
	s.TX += tx
}

// TimeCounterAdd counter for entriest per X seconds
func (s *Statistics) TimeCounterAdd() {
	go func() {
		for {
			select {
			case s.TimeCounter <- true:
				ticker := time.NewTicker(time.Duration(s.TimeTimer) * time.Second)
				<-ticker.C
				ticker.Stop()
				<-s.TimeCounter
				return
			default:
			}
			return
		}
	}()
}

// TimeCounterGet gets current count of the timer
func (s *Statistics) TimeCounterGet() int {
	return len(s.TimeCounter)
}

// ResponseTimeAdd adds a response time to the array, and cycles through them if full
func (s *Statistics) ResponseTimeAdd(f float64) {
	s.Lock()
	defer s.Unlock()
	if len(s.ResponseTimeValue) <= cap(s.ResponseTimeValue) {
		s.ResponseTimeValue = append(s.ResponseTimeValue, f)
		go s.responseTimeRemoveFirstDelayed()
	}
}

// ResponseTimeValueMerge merges 2 response time arrays
func (s *Statistics) ResponseTimeValueMerge(f []float64) {
	s.Lock()
	defer s.Unlock()
	s.ResponseTimeValue = append(s.ResponseTimeValue, f...)
}

func (s *Statistics) responseTimeRemoveFirstDelayed() {
	ticker := time.NewTicker(time.Duration(s.TimeTimer) * time.Second)
	for {
		select {
		case _ = <-ticker.C:
			s.responseTimeRemoveFirst()
			ticker.Stop()
			return
		}
	}
}

func (s *Statistics) responseTimeRemoveFirst() {
	s.Lock()
	defer s.Unlock()
	if len(s.ResponseTimeValue) < 1 {
		return
	}

	_, s.ResponseTimeValue = s.ResponseTimeValue[0], s.ResponseTimeValue[1:]
}

// ResponseTimeGet gets current averate response time
func (s *Statistics) ResponseTimeGet() float64 {
	if s.ResponseTimeValue == nil || len(s.ResponseTimeValue) <= 5 { // require 5 data points before using the values
		return 0
	}

	s.RLock()
	defer s.RUnlock()
	var r float64
	r = 0
	for _, v := range s.ResponseTimeValue {
		r = r + v
	}

	return toFixed(r/float64(len(s.ResponseTimeValue)), 4)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

// UUIDGet returns the UUID
func (s *Statistics) UUIDGet() string {
	s.RLock()
	defer s.RUnlock()
	return s.UUID
}

// ClientsConnectedGet returns the clients connected
func (s *Statistics) ClientsConnectedGet() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.ClientsConnected
}

// ClientsConnectsGet returns the clients requests
func (s *Statistics) ClientsConnectsGet() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.ClientsConnects
}

// RXGet returns received traffic
func (s *Statistics) RXGet() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.RX
}

// TXGet returns sent traffic
func (s *Statistics) TXGet() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.TX
}

// ResponseTimeValueGet returns the responsetime values
func (s *Statistics) ResponseTimeValueGet() []float64 {
	s.RLock()
	defer s.RUnlock()
	return s.ResponseTimeValue
}
