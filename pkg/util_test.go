package pkg

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestValidateDownloadedPayloadRejectsErrorPage(t *testing.T) {
	err := ValidateDownloadedPayload("sample.zip", []byte("<html>not found</html>"))
	if err == nil {
		t.Fatal("expected error page to be rejected")
	}
}

func TestValidateDownloadedPayloadAcceptsZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("sample.srt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := ValidateDownloadedPayload("sample.zip", buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}
