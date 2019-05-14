package healthcheck

import (
	"testing"
	"time"
)

func TestDataParsing(t *testing.T) {

	checks := map[string]string{
		"###aewf###": "###aewf###",
		"aewf":       "aewf",
		"###DATE+10m2006-01-02T15:04:05###.000Z":    "2012-12-12T12:22:12.000Z",
		"###DATE-2h2006-01-02T15:04:05###":          "2012-12-12T10:12:12",
		"###DATE-2h2006-01-02T15:04:05|UTC###":      "2012-12-12T08:12:12",
		"###DATE-2h2006-01-02T15:04:05.000Z|UTC###": "2012-12-12T08:12:12.000Z",
		"###DATE-9qhello###":                        "date parse error:time: unknown unit q in duration 9q",
	}

	// Set static time to test functions
	tm, _ := time.Parse(time.RFC3339, "2012-12-12T12:12:12+02:00")

	for in, out := range checks {
		parsed := postDataParser(tm, in)
		if parsed != out {
			t.Errorf("Date parsing check for:%s returned:%s expected:%s", in, parsed, out)
		}
	}

	invalidTime, _ := time.Parse(time.RFC3339, "")
	parsed := postDataParser(invalidTime, "###DATE###")
	if parsed != "INVALID TIME" {
		t.Errorf("Date parsing check for:zerodate returned:%s expected:INVALID TIME", parsed)
	}

}
