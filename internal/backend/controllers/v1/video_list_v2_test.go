package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/video_list_helper"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestMoviePosterReturnsEmptyPayloadWhenPathMapMisses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	movieDir := filepath.Join(t.TempDir(), "电影")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(movieDir, "示例电影.mkv")
	if err := os.WriteFile(videoPath, []byte(strings.Repeat("v", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "poster.jpg"), []byte(strings.Repeat("p", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}

	cb := &ControllerBase{
		log:             logrus.New(),
		pathUrlMap:      map[string]string{filepath.Join(t.TempDir(), "坏路径"): "/movie_dir_0"},
		videoListHelper: video_list_helper.NewVideoListHelper(logrus.New()),
	}

	body, err := json.Marshal(backend2.MovieInfoV2{
		Name:             "示例电影",
		MainRootDirFPath: movieDir,
		VideoFPath:       videoPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/list/movie_poster", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	cb.MoviePoster(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var posterInfo backend2.PosterInfo
	if err := json.Unmarshal(recorder.Body.Bytes(), &posterInfo); err != nil {
		t.Fatal(err)
	}
	if posterInfo.Url != "" {
		t.Fatalf("expected empty poster url, got %q", posterInfo.Url)
	}
}

func TestOneMovieSubsReturnsFileListWhenPathMapMisses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	movieDir := filepath.Join(t.TempDir(), "电影")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(movieDir, "示例电影.mkv")
	subPath := filepath.Join(movieDir, "示例电影.srt")
	if err := os.WriteFile(videoPath, []byte(strings.Repeat("v", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subPath, []byte(strings.Repeat("字幕测试内容\n", 128)), 0o644); err != nil {
		t.Fatal(err)
	}

	cb := &ControllerBase{
		log:             logrus.New(),
		pathUrlMap:      map[string]string{filepath.Join(t.TempDir(), "坏路径"): "/movie_dir_0"},
		videoListHelper: video_list_helper.NewVideoListHelper(logrus.New()),
	}

	body, err := json.Marshal(backend2.MovieInfoV2{
		Name:             "示例电影",
		MainRootDirFPath: movieDir,
		VideoFPath:       videoPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/list/one_movie_subs", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	cb.OneMovieSubs(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var movieSubsInfo backend2.MovieSubsInfo
	if err := json.Unmarshal(recorder.Body.Bytes(), &movieSubsInfo); err != nil {
		t.Fatal(err)
	}
	if len(movieSubsInfo.SubFPathList) != 1 || movieSubsInfo.SubFPathList[0] != subPath {
		t.Fatalf("unexpected subtitle file list: %#v", movieSubsInfo.SubFPathList)
	}
	if len(movieSubsInfo.SubUrlList) != 0 {
		t.Fatalf("expected empty subtitle url list, got %#v", movieSubsInfo.SubUrlList)
	}
}
