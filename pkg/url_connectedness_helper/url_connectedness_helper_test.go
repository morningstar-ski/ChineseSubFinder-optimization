package url_connectedness_helper

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUrlConnectednessTest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notFoundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundServer.Close()

	type args struct {
		testUrl   string
		proxyAddr string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "no proxy success", args: args{
			testUrl:   server.URL,
			proxyAddr: "",
		}, want: true, wantErr: false},
		{name: "no proxy bad status", args: args{
			testUrl:   notFoundServer.URL,
			proxyAddr: "",
		}, want: false, wantErr: false},
		{name: "unsupported proxy scheme", args: args{
			testUrl:   server.URL,
			proxyAddr: "socks5://127.0.0.1:1080",
		}, want: false, wantErr: true},
		{name: "invalid proxy format", args: args{
			testUrl:   server.URL,
			proxyAddr: "://bad-proxy",
		}, want: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := UrlConnectednessTest(tt.args.testUrl, tt.args.proxyAddr)
			if (err != nil) != tt.wantErr {
				t.Errorf("UrlConnectednessTest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UrlConnectednessTest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
