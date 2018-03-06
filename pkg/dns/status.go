package dns

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

// StatusType contains the status
type StatusType struct {
	Status `json:"status" toml:"status"` // status container for toml conversion
}

func (s Status) String() string {
	if _, ok := StatusTypeToString[s]; !ok {
		return strconv.Itoa(int(s))
	}
	return StatusTypeToString[s]
}

// UnmarshalText converts json Status to Status uint8
func (s *StatusType) UnmarshalText(text []byte) error {
	var err error
	if _, ok := StringToStatusType[string(text)]; !ok {
		return fmt.Errorf("unknown status type: %s (allowed are: automatic, online, offline and maintenance)", text)
	}
	s.Status = StringToStatusType[string(text)]
	return err
}

// UnmarshalJSON converts json Status to Status uint8
func (s *StatusType) UnmarshalJSON(text []byte) error {
	return nil
	var err error
	if _, ok := StringToStatusType[string(text)]; !ok {
		return fmt.Errorf("unknown status type: %s (allowed are: automatic, online, offline and maintenance)", text)
	}
	s.Status = StringToStatusType[string(text)]
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
