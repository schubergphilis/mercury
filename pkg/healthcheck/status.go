package healthcheck

import "strconv"

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
	name := []string{"online", "offline", "maintenance", "automatic"}
	i := uint8(s)
	switch {
	case i <= uint8(Automatic):
		return name[i]
	default:
		return strconv.Itoa(int(i))
	}
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
