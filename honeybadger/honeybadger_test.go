package honeybadger

import (
	"testing"
)

func TestFaultsUrl(t *testing.T) {
	hbUrl := NewURL(HB_API_ENDPOINT).SetApiKey("abc").SetPage(1).ProjectFaults(123)
	expected := HB_API_ENDPOINT + "/123/faults?auth_token=abc&page=1"
	if url := hbUrl; url != expected {
		t.Errorf(`Error during parsing: expected %q but got %q`, expected, url)
	}
}

func TestNoticesUrl(t *testing.T) {
	hbUrl := NewURL(HB_API_ENDPOINT).SetApiKey("abc").SetPage(1).FaultNotices(123, 456)
	expected := HB_API_ENDPOINT + "/123/faults/456/notices?auth_token=abc&page=1"
	if url := hbUrl; url != expected {
		t.Errorf(`Error during parsing: expected %q but got %q`, expected, url)
	}
}
