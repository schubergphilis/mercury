package healthcheck

import (
	"fmt"
	"strconv"
)

// Status holds the status of a check
type Status uint8

// Status values of a check
const (
	Automatic = Status(iota)
	Online
	Offline
	Maintenance
)

func (s Status) String() string {
	if _, ok := StatusTypeToString[s]; !ok {
		return strconv.Itoa(int(s))
	}
	return StatusTypeToString[s]
}

func (s *Status) UnmarshalText(text []byte) error {
	var err error
	if _, ok := StringToStatusType[string(text)]; !ok {
		return fmt.Errorf("unknown status type: %s", text)
	}
	t := StringToStatusType[string(text)]
	s = &t
	return err
}

// StatusTypeToString converts status to string
var StatusTypeToString = map[Status]string{
	Automatic:   "automatic",
	Online:      "online",
	Offline:     "offline",
	Maintenance: "maintance",
}

// StringToStatusType converts string to status
var StringToStatusType = map[string]Status{
	"automatic":   Automatic,
	"online":      Online,
	"offline":     Offline,
	"maintenance": Maintenance,
}
