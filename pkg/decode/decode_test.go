package decode

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestGetImdbAndYearMovieXml(t *testing.T) {

	rootDir := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"movies", "Army of the Dead (2021)"}, 4, false)

	wantid := "tt0993840"
	wantyear := "2021"
	dirPth := filepath.Join(rootDir, "movie.xml")
	imdbInfo, err := getVideoNfoInfoFromMovieXml(dirPth)
	if err != nil {
		t.Error(err)
	}
	if imdbInfo.ImdbId != wantid {
		t.Errorf("got = %v, want %v", imdbInfo.ImdbId, wantid)
	}
	if imdbInfo.Year != wantyear {
		t.Errorf("got = %v, want %v", imdbInfo.Year, wantyear)
	}
}

func Test_getImdbAndYearNfo(t *testing.T) {
	movieRoot := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"movies", "Army of the Dead (2021)"}, 4, false)
	tvShowRoot := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"nfo_files", "tvshow"}, 4, false)
	tvShowRootAlt := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"nfo_files", "tvshow"}, 3, false)
	type args struct {
		nfoFilePath string
		rootKey     string
	}
	tests := []struct {
		name    string
		args    args
		want    types.VideoNfoInfo
		wantErr bool
	}{
		{
			name: "Army of the Dead (2021) WEBDL-1080p.nfo", args: args{
				nfoFilePath: filepath.Join(movieRoot, "Army of the Dead (2021) WEBDL-1080p.nfo"),
				rootKey:     "movie",
			},
			want: types.VideoNfoInfo{
				ImdbId:      "tt0993840",
				Title:       "活死人军团",
				Year:        "2021",
				ReleaseDate: "2021-05-13",
			},
			wantErr: false,
		},
		{
			name: "tvshow_00 (2).nfo", args: args{
				nfoFilePath: filepath.Join(tvShowRoot, "tvshow_00 (2).nfo"),
				rootKey:     "tvshow",
			},
			want: types.VideoNfoInfo{
				ImdbId:      "tt0346314",
				Title:       "Ghost in the Shell: Stand Alone Complex",
				ReleaseDate: "2002-10-01",
			},
			wantErr: false,
		},
		{
			name: "tvshow_00 (3).nfo", args: args{
				nfoFilePath: filepath.Join(tvShowRoot, "tvshow_00 (3).nfo"),
				rootKey:     "tvshow",
			},
			want: types.VideoNfoInfo{
				ImdbId:      "tt1856010",
				Title:       "House of Cards (US)",
				ReleaseDate: "2013-02-01",
			},
			wantErr: false,
		},
		{
			name: "tvshow_00 (63).nfo", args: args{
				nfoFilePath: filepath.Join(tvShowRootAlt, "tvshow_00 (63).nfo"),
				rootKey:     "tvshow",
			},
			want: types.VideoNfoInfo{
				ImdbId:      "tt3581920",
				TVdbId:      "392256",
				Title:       "The Last of Us",
				ReleaseDate: "2023-01-15",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVideoNfoInfo(tt.args.nfoFilePath, tt.args.rootKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVideoNfoInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getVideoNfoInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSeasonAndEpisodeFromFileName(t *testing.T) {
	str := `杀死伊芙 第二季(-简繁英双语字幕-FIX字幕侠)Killing.Eve.S02.Do.You.Know.How.to.Dispose.of.a.Body.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb.rar`
	b, s, e, err := GetSeasonAndEpisodeFromSubFileName(str)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("\n\n%t\t S%dE%d\n", b, s, e)
}

func TestGetSeasonAndEpisodeFromSubFileName1xPattern(t *testing.T) {
	isFullSeason, season, episode, err := GetSeasonAndEpisodeFromSubFileName("Game of Thrones - 1x01 - Winter is Coming.720p HDTV.cn.srt")
	if err != nil {
		t.Fatalf("GetSeasonAndEpisodeFromSubFileName() error = %v", err)
	}
	if isFullSeason != false || season != 1 || episode != 1 {
		t.Fatalf("GetSeasonAndEpisodeFromSubFileName() = (%t, %d, %d), want (false, 1, 1)", isFullSeason, season, episode)
	}
}

func TestGetVideoInfoFromFileFullPathFallsBackToFileNameWhenEpisodeNfoBroken(t *testing.T) {
	rootDir := t.TempDir()
	videoPath := filepath.Join(rootDir, "Rick.and.Morty.S01E06.1080p.WEB-DL.mkv")
	nfoPath := filepath.Join(rootDir, "Rick.and.Morty.S01E06.1080p.WEB-DL.nfo")

	if err := os.WriteFile(videoPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write video file: %v", err)
	}
	if err := os.WriteFile(nfoPath, []byte("<episodedetails><title>broken"), 0o644); err != nil {
		t.Fatalf("write nfo file: %v", err)
	}

	got, _, err := GetVideoInfoFromFileFullPath(videoPath, false)
	if err != nil {
		t.Fatalf("GetVideoInfoFromFileFullPath() error = %v", err)
	}
	if got.Season != 1 || got.Episode != 6 {
		t.Fatalf("GetVideoInfoFromFileFullPath() = S%dE%d, want S1E6", got.Season, got.Episode)
	}
	if got.Title != "Rick and Morty" {
		t.Fatalf("GetVideoInfoFromFileFullPath() title = %q, want %q", got.Title, "Rick and Morty")
	}
}

func TestGetNumber2Float(t *testing.T) {
	testString := "asd&^%1998.2jh aweo "
	outNumber, err := GetNumber2Float(testString)
	if err != nil {
		t.Error(err)
	}
	if outNumber != 1998.2 {
		t.Error("not the same")
	}
}

func TestGetNumber2int(t *testing.T) {

	testString := "asd&^%1998jh aweo "
	outNumber, err := GetNumber2int(testString)
	if err != nil {
		t.Error(err)
	}
	if outNumber != 1998 {
		t.Error("not the same")
	}
}

func TestGetImdbInfo4SeriesDir(t *testing.T) {
	unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"series"}, 4, false)

	type args struct {
		seriesDir string
	}
	tests := []struct {
		name    string
		args    args
		want    types.VideoNfoInfo
		wantErr bool
	}{
		{
			name: "Loki",
			args: args{seriesDir: unit_test_helper.GetTestDataResourceRootPath([]string{"series", "Loki"}, 4, false)},
			want: types.VideoNfoInfo{
				ImdbId:      "tt9140554",
				Title:       "Loki",
				ReleaseDate: "2021-06-09",
			},
			wantErr: false,
		},
		{
			name: "辛普森一家",
			args: args{seriesDir: unit_test_helper.GetTestDataResourceRootPath([]string{"series", "辛普森一家"}, 4, false)},
			want: types.VideoNfoInfo{
				ImdbId:      "tt0096697",
				Title:       "The Simpsons",
				ReleaseDate: "1989-12-17",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetVideoNfoInfo4SeriesDir(tt.args.seriesDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVideoNfoInfo4SeriesDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVideoNfoInfo4SeriesDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFakeBDMVWorked(t *testing.T) {
	unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"movies", "澶辨帶鐜╁ (2021)"}, 4, false)

	rootDir := unit_test_helper.GetTestDataResourceRootPath([]string{"movies", "失控玩家 (2021)"}, 4, false)

	bok, dbmvFPath, _ := IsFakeBDMVWorked(filepath.Join(rootDir, "失控玩家 (2021).mp4"))
	if bok == false {
		t.Fatal("IsFakeBDMVWorked error")
	}
	println(dbmvFPath)
}

func TestGetImdbInfo4Movie(t *testing.T) {
	//type args struct {
	//	movieFileFullPath string
	//}
	//tests := []struct {
	//	name    string
	//	args    args
	//	want    types.VideoNfoInfo
	//	wantErr bool
	//}{
	//	{name: "00", args: args{
	//		movieFileFullPath: "X:\\电影\\Death on the Nile (2022)\\Death on the Nile (2022) Bluray-1080p.mkv",
	//	}},
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		got, err := GetVideoNfoInfo4Movie(tt.args.movieFileFullPath)
	//		if (err != nil) != tt.wantErr {
	//			t.Errorf("GetVideoNfoInfo4Movie() error = %v, wantErr %v", err, tt.wantErr)
	//			return
	//		}
	//		if !reflect.DeepEqual(got, tt.want) {
	//			t.Errorf("GetVideoNfoInfo4Movie() got = %v, want %v", got, tt.want)
	//		}
	//	})
	//}
}

func TestGetVideoNfoInfoReadsOriginalTitle(t *testing.T) {
	rootDir := t.TempDir()
	nfoPath := filepath.Join(rootDir, "tvshow.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<tvshow>
  <title>拥挤的房间</title>
  <originaltitle>The Crowded Room</originaltitle>
</tvshow>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write nfo file: %v", err)
	}

	got, err := getVideoNfoInfo(nfoPath, "tvshow")
	if err != nil {
		t.Fatalf("getVideoNfoInfo() error = %v", err)
	}
	if got.Title != "拥挤的房间" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.OriginalTitle != "The Crowded Room" {
		t.Fatalf("original title = %q", got.OriginalTitle)
	}
}
