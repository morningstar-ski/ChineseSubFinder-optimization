package subtitle_best_api

import (
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
)

func TestSubtitleBestApi_AuthRequired(t *testing.T) {

	bapi := NewSubtitleBestApi(log_helper.GetLogger4Tester(), random_auth_key.AuthKey{
		BaseKey:  random_auth_key.BaseKey,
		AESKey16: random_auth_key.AESKey16,
		AESIv16:  random_auth_key.AESIv16,
	})

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "CheckAlive", run: bapi.CheckAlive},
		{name: "GetCode", run: func() error {
			_, err := bapi.GetCode()
			return err
		}},
		{name: "GetMediaInfo", run: func() error {
			_, err := bapi.GetMediaInfo("tt7278862", "imdb", "series")
			return err
		}},
		{name: "ConvertId", run: func() error {
			_, err := bapi.ConvertId("438148", "tmdb", "movie")
			return err
		}},
		{name: "FeedBack", run: func() error {
			_, err := bapi.FeedBack("feedback-id", "1.0.0", "None", true, true)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatal("expected auth validation error")
			}
			if !strings.Contains(err.Error(), "auth key is not set") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
