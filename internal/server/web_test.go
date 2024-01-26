package server

import (
	"net/url"
	"testing"
)

func TestSreekiMapper(t *testing.T) {
	for _, tc := range []struct {
		inpUrl string
		outUrl string
	}{
		{
			inpUrl: "https://en.sreekipedia.org",
			outUrl: "https://en.wikipedia.org",
		},
		{
			inpUrl: "https://en.sreekipedia.org/wiki/Hello",
			outUrl: "https://en.wikipedia.org",
		},
		{
			inpUrl: "https://en.m.sreekipedia.org",
			outUrl: "https://en.m.wikipedia.org",
		},
		{
			inpUrl: "https://zh.sreekipedia.org",
			outUrl: "https://zh.wikipedia.org",
		},
	} {
		u, _ := url.Parse(tc.inpUrl)
		u2, ok := sreekiMapper(u)
		if !ok {
			t.Errorf("sreekiMapper(%s) returned false", tc.inpUrl)
		}
		if u2.String() != tc.outUrl {
			t.Errorf("sreekiMapper(%s) = %s, want %s", tc.inpUrl, u2.String(), tc.outUrl)
		}
	}
}
