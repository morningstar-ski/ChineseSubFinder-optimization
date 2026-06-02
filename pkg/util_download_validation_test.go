package pkg

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestValidateSubtitleDownloadPayload(t *testing.T) {
	log := logrus.New()
	validSRT := []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n")
	validZip := buildZipPayload(t, "sample.srt", validSRT)
	inspector := func(body []byte, ext string) (bool, error) {
		trimmed := bytes.TrimSpace(body)
		return len(trimmed) > 0 && bytes.Contains(trimmed, []byte("-->")), nil
	}

	tests := []struct {
		name        string
		fileName    string
		contentType string
		statusCode  int
		body        []byte
		wantErr     bool
	}{
		{
			name:        "valid srt",
			fileName:    "sample.srt",
			contentType: "text/plain",
			statusCode:  200,
			body:        validSRT,
		},
		{
			name:        "html error page",
			fileName:    "sample.srt",
			contentType: "text/html",
			statusCode:  200,
			body:        []byte("<html><body>error</body></html>"),
			wantErr:     true,
		},
		{
			name:        "valid zip",
			fileName:    "sample.zip",
			contentType: "application/zip",
			statusCode:  200,
			body:        validZip,
		},
		{
			name:        "invalid zip",
			fileName:    "sample.zip",
			contentType: "application/zip",
			statusCode:  200,
			body:        []byte("not a zip"),
			wantErr:     true,
		},
		{
			name:        "bad status",
			fileName:    "sample.srt",
			contentType: "text/plain",
			statusCode:  404,
			body:        validSRT,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubtitleDownloadPayload(log, inspector, "https://example.com/file", tt.fileName, tt.contentType, tt.statusCode, tt.body)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func buildZipPayload(t *testing.T, name string, payload []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buf.Bytes()
}
