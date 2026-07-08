package xui

import (
	"errors"
	"testing"
)

func TestIsAlternatePanelEndpointErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("POST https://host/panel/xray/update: unexpected end of JSON input; body="), true},
		{errors.New("login: empty response (status 502)"), true},
		{errors.New("POST https://host/panel/api/setting/update: invalid character 'p' after top-level value; body=404 page not found"), true},
		{errors.New("POST https://host/panel/api/setting/update: invalid settings"), false},
	}

	for _, tc := range cases {
		if got := isAlternatePanelEndpointErr(tc.err); got != tc.want {
			t.Fatalf("isAlternatePanelEndpointErr(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
