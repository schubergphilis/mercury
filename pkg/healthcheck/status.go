package healthcheck

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	t := strings.Replace(string(text), "\"", "", -1)
	if _, ok := StringToStatusType[t]; !ok {
		return fmt.Errorf("unknown status type1: %s (allowed are: automatic, online, offline and maintenance)", text)
	}
	s.Status = StringToStatusType[t]
	return err
}

// UnmarshalJSON converts json Status to Status uint8
func (s *StatusType) UnmarshalJSON(text []byte) error {
	var err error
	var tmp struct {
		Status int
	}
	err = json.Unmarshal(text, &tmp)
	if err != nil {
		fmt.Printf("Err: %s", err)
	}
	s.Status = IntToStatusType[tmp.Status]
	return nil
}

// MarshalJSON converts json Status to []byte
/*
func (s *StatusType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", s.String())), nil
}
*/

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

// IntToStatusType converts int to status
var IntToStatusType = map[int]Status{
	0: Automatic,
	1: Online,
	2: Offline,
	3: Maintenance,
}
