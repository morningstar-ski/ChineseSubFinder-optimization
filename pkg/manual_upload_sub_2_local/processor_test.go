package manual_upload_sub_2_local

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/save_sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/normal"
)

func TestNewManualUploadSub2Local(t *testing.T) {

	log := log_helper.GetLogger4Tester()
	saveSubHelper := save_sub_helper.NewSaveSubHelper(log, normal.NewFormatter(log), nil)
	got := NewManualUploadSub2Local(log, saveSubHelper, nil)
	if got == nil {
		t.Fatal("NewManualUploadSub2Local() returned nil")
	}
	if got.subParserHub == nil {
		t.Fatal("NewManualUploadSub2Local() did not initialize subParserHub")
	}
}
