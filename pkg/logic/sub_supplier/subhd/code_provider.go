package subhd

import (
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
)

type SubHDCodeProvider interface {
	GetCode() (string, string, error)
}

type SubtitleBestCodeProvider struct {
	fileDownloader *file_downloader.FileDownloader
}

func NewSubtitleBestCodeProvider(fileDownloader *file_downloader.FileDownloader) *SubtitleBestCodeProvider {
	return &SubtitleBestCodeProvider{fileDownloader: fileDownloader}
}

func (p *SubtitleBestCodeProvider) GetCode() (string, string, error) {
	code, err := p.fileDownloader.MediaInfoDealers.SubtitleBestApi.GetCode()
	if err != nil {
		return "", "", err
	}
	return time.Now().Format("2006-01-02"), code, nil
}

type ProviderError struct {
	Reason string
	Err    error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Reason
	}
	return e.Reason + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapReason(reason string, err error) error {
	if err == nil {
		return &ProviderError{Reason: reason}
	}
	return &ProviderError{Reason: reason, Err: err}
}

func reasonOf(err error) string {
	if err == nil {
		return ""
	}
	if providerErr, ok := err.(*ProviderError); ok {
		return providerErr.Reason
	}
	return ReasonProbeFailed
}
