package assrt

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

var assrtInstance *Supplier

func defInstance() {

	pkg.ReadCustomAuthFile(log_helper.GetLogger4Tester())

	authKey := random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}

	assrtInstance = NewSupplier(file_downloader.NewFileDownloader(
		cache_center.NewCacheCenter("test", log_helper.GetLogger4Tester()), authKey))
}

func TestSupplier_getSubListFromFile(t *testing.T) {
	t.Skip("integration test depends on local media files and assrt availability")

	//videoFPath := "X:\\电影\\失控玩家 (2021)\\失控玩家 (2021).mp4"
	//isMovie := true
	defInstance()
	//videoFPath := "X:\\连续剧\\杀死伊芙 (2018)\\Season 4\\Killing Eve - S04E08 - Hello, Losers WEBDL-1080p.mkv"
	//videoFPath := "X:\\连续剧\\Why Didn’t They Ask Evans!\\Season 1\\Why Didn’t They Ask Evans! - S01E01 - Episode 1 WEBRip-1080p.mp4"
	videoFPath := "X:\\连续剧\\Pantheon\\Season 1\\Pantheon - S01E03 - Reign of Winter WEBDL-1080p.mkv"
	//videoFPath := "X:\\连续剧\\风骚律师 (2015)\\Season 6\\Better Call Saul - S06E05 - Black and Blue WEBDL-1080p.mkv"
	isMovie := false

	got, err := assrtInstance.getSubListFromFile(videoFPath, isMovie)
	if err != nil {
		t.Error(err)
	}
	for i, info := range got {
		println(i, info.Name, info.FileUrl)
	}
}

func TestSupplier_CheckAlive(t *testing.T) {
	t.Skip("integration test depends on assrt availability")

	defInstance()
	bok, speed := assrtInstance.CheckAlive()
	println(bok, speed)

}

func TestAssrtSearchKeywordOrder(t *testing.T) {
	want := []string{"cn", "en", "org", "file"}
	if !reflect.DeepEqual(assrtSearchKeywordOrder, want) {
		t.Fatalf("assrtSearchKeywordOrder = %#v; want %#v", assrtSearchKeywordOrder, want)
	}
}

func TestSortAssrtSearchSubsPrefersMatchingMetadata(t *testing.T) {
	subs := []SearchSubItem{
		{
			Id:         1,
			VoteScore:  10,
			Videoname:  "My Show S01E03 720p HDTV-OTHER",
			NativeName: "My Show S01E03 720p HDTV-OTHER",
			Revision:   1,
		},
		{
			Id:         2,
			VoteScore:  8,
			Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
			Revision:   1,
		},
	}

	sortAssrtSearchSubs(subs, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	if subs[0].Id != 2 {
		t.Fatalf("expected metadata-matching subtitle first, got %#v", subs[0])
	}
}

func TestSortAssrtSearchSubsBreaksVoteTieWithEpisodeMetadata(t *testing.T) {
	subs := []SearchSubItem{
		{
			Id:         10,
			VoteScore:  8,
			Videoname:  "My Show S01E04 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E04 1080p WEB-DL-GROUP",
			Revision:   1,
		},
		{
			Id:         11,
			VoteScore:  8,
			Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
			Revision:   1,
		},
	}

	sortAssrtSearchSubs(subs, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	if subs[0].Id != 11 {
		t.Fatalf("expected exact episode match first, got %#v", subs[0])
	}
}

func TestScoreAssrtSearchSubIncludesAuthorityScore(t *testing.T) {
	matcher := ranking.NewTargetMatcher(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	sub := SearchSubItem{
		VoteScore:  8,
		Revision:   2,
		Subtype:    "bilingual subtitle",
		Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
		NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
	}

	got := scoreAssrtSearchSub(sub, matcher)
	want := ranking.BaseScore(matcher, ranking.BaseScoreOptions{
		AuthorityScore:      84,
		Subtype:             sub.Subtype,
		ReleaseNames:        []string{sub.Videoname, sub.NativeName},
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
	})
	if got != want {
		t.Fatalf("scoreAssrtSearchSub = %d, want %d", got, want)
	}
}

func TestAssrtCandidateMetadata(t *testing.T) {
	sub := SearchSubItem{
		VoteScore:  8,
		Revision:   2,
		Subtype:    "bilingual subtitle",
		Videoname:  "Video.Name",
		NativeName: "Native.Name",
	}

	metadata := assrtCandidateMetadata(sub)
	if metadata.AuthorityScore != 84 || metadata.Subtype != sub.Subtype {
		t.Fatalf("unexpected metadata %#v", metadata)
	}
	if len(metadata.ReleaseNames) != 2 || metadata.ReleaseNames[0] != sub.Videoname || metadata.ReleaseNames[1] != sub.NativeName {
		t.Fatalf("unexpected release names %#v", metadata.ReleaseNames)
	}
}

func TestGetSubByKeyWordAllowsArrayLanglist(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldRootURL := cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl
	oldToken := cfg.SubtitleSources.AssrtSettings.Token
	t.Cleanup(func() {
		cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = oldRootURL
		cfg.SubtitleSources.AssrtSettings.Token = oldToken
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":[{"id":"7","vote_score":"8","revision":"2","lang":{"langlist":[]}}],"result":"succeed","keyword":"bad"},"status":0}`))
	}))
	defer server.Close()

	cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = server.URL
	cfg.SubtitleSources.AssrtSettings.Token = "test-token"

	supplier := NewSupplier(&file_downloader.FileDownloader{Log: log_helper.GetLogger4Tester()})
	supplier.theSearchInterval = 0

	got, err := supplier.getSubByKeyWord("bad")
	if err != nil {
		t.Fatalf("expected array langlist payload to be accepted, got error: %v", err)
	}
	if got == nil || len(got.Sub.Subs) != 1 {
		t.Fatalf("unexpected result: %#v", got)
	}
	if got.Sub.Subs[0].Id != 7 || got.Sub.Subs[0].VoteScore != 8 || got.Sub.Subs[0].Revision != 2 {
		t.Fatalf("unexpected numeric fields: %#v", got.Sub.Subs[0])
	}
}
