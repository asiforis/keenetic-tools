package sys

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestEntwareZone(t *testing.T) {
	tzif, err := os.ReadFile("/usr/share/zoneinfo/Europe/Moscow")
	if err != nil {
		t.Skip("no /usr/share/zoneinfo")
	}
	read := func(p string) ([]byte, error) {
		if p != LocalTime {
			t.Fatalf("read %s", p)
		}
		return tzif, nil
	}
	z := entwareZone(false, read)
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	if z == nil || at.In(z).Format("15:04") != "12:00" {
		t.Fatalf("zone = %v", z)
	}
	if entwareZone(true, read) != nil {
		t.Fatal("TZ set: Go's choice must stay")
	}
	missing := func(string) ([]byte, error) { return nil, errors.New("no file") }
	if entwareZone(false, missing) != nil {
		t.Fatal("no /opt/etc/localtime: Go's choice must stay")
	}
}
