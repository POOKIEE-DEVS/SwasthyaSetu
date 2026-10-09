package store

import (
	"testing"
	"time"
)

func TestParseTimeReadsEveryStoredForm(t *testing.T) {
	want := time.Date(2026, 10, 4, 9, 30, 15, 123456000, time.UTC)
	for _, src := range []any{
		want,
		want.In(time.FixedZone("NPT", 5*3600+45*60)),
		"2026-10-04 09:30:15.123456",       // older rows: no zone
		"2026-10-04 09:30:15.123456+00:00", // written by this one
		[]byte("2026-10-04T15:15:15.123456+05:45"),
	} {
		got, err := parseTime(src)
		if err != nil || got == nil || !got.Equal(want) || got.Location() != time.UTC {
			t.Errorf("%v: got %v, %v", src, got, err)
		}
	}
	if got, err := parseTime(nil); got != nil || err != nil {
		t.Errorf("NULL: %v, %v", got, err)
	}
	if _, err := parseTime("yesterday"); err == nil {
		t.Error("garbage should be an error")
	}
}
