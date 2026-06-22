package pkg

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/regex_things"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	browser "github.com/allanpk716/fake-useragent"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	subtitleDownloadTimeout    = common.HTMLTimeOut + 30*time.Second
	subtitleDownloadRetryCount = 2
	publicIPRequestTimeout     = 3 * time.Second
	publicIPCacheTTL           = 10 * time.Minute
)

var (
	publicIPCacheMu      sync.RWMutex
	publicIPCacheValue   string
	publicIPCacheExpires time.Time
)

// NewHttpClient 新建一个 resty 的对象
func NewHttpClient(referer ...string) (*resty.Client, error) {
	//const defUserAgent = "Mozilla/5.0 (Macintosh; U; Intel Mac OS X 10_6_8; en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50"
	//const defUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36 Edg/91.0.864.41"

	var UserAgent, Referer string
	// ------------------------------------------------
	// 随机的 Browser
	UserAgent = browser.Random()
	// ------------------------------------------------
	httpClient := resty.New().SetTransport(&http.Transport{
		DisableKeepAlives:   true,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
	})
	httpClient.SetTimeout(common.HTMLTimeOut)
	httpClient.SetRetryCount(1)
	// ------------------------------------------------
	// 设置 Referer
	if len(referer) > 0 {
		Referer = referer[0]
		if len(Referer) > 0 {
			httpClient.SetHeader("Referer", Referer)
		}
	}
	// ------------------------------------------------
	// 设置 Header
	httpClient.SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   UserAgent,
	})
	// ------------------------------------------------
	// 不要求安全链接
	httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	// ------------------------------------------------
	// http 代理
	HttpProxyAddress := local_http_proxy_server.GetProxyUrl()
	if HttpProxyAddress != "" {
		httpClient.SetProxy(HttpProxyAddress)
	} else {
		httpClient.RemoveProxy()
	}

	return httpClient, nil
}

func newSubtitleDownloadHTTPClient() (*resty.Client, error) {
	httpClient, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	httpClient.SetTimeout(subtitleDownloadTimeout)
	httpClient.SetRetryCount(subtitleDownloadRetryCount)
	return httpClient, nil
}

func newPublicIPHTTPClient() (*resty.Client, error) {
	httpClient, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	httpClient.SetTimeout(publicIPRequestTimeout)
	httpClient.SetRetryCount(0)
	return httpClient, nil
}

func getPublicIP(inputSite string) string {

	var client *resty.Client
	client, err := newPublicIPHTTPClient()
	if err != nil {
		return ""
	}
	response, err := client.R().Get(inputSite)
	if err != nil {
		return ""
	}
	return response.String()
}

func GetPublicIP(log *logrus.Logger, queue *settings.TaskQueue) string {
	if cached := getCachedPublicIP(); cached != "" {
		return cached
	}

	defPublicIPSites := []string{
		"https://myip.biturl.top/",
		"https://ip4.seeip.org/",
		"https://ipecho.net/plain",
		"https://api-ipv4.ip.sb/ip",
		"https://api.ipify.org/",
		"http://myexternalip.com/raw",
	}

	customPublicIPSites := make([]string, 0)
	if queue.CheckPublicIPTargetSite != "" {
		// 自定义了公网IP查询网站
		tSites := strings.Split(queue.CheckPublicIPTargetSite, ";")
		if tSites != nil && len(tSites) > 0 {
			customPublicIPSites = append(customPublicIPSites, tSites...)
		}
	} else {
		customPublicIPSites = append(customPublicIPSites, defPublicIPSites...)
	}

	publicIP := resolvePublicIPFromSites(log, customPublicIPSites, getPublicIP)
	if publicIP != "" {
		cachePublicIP(publicIP)
	}
	return publicIP
}

func resolvePublicIPFromSites(log *logrus.Logger, publicIPSites []string, fetch func(string) string) string {
	for i, publicIPSite := range publicIPSites {
		log.Debugln("[GetPublicIP]", i, publicIPSite)
		publicIP := strings.TrimSpace(fetch(publicIPSite))
		if publicIP == "" {
			continue
		}

		parsedIP := extractIPFromText(publicIP)
		if parsedIP == "" {
			continue
		}

		log.Infoln("[GetPublicIP]", parsedIP)
		return parsedIP
	}

	return ""
}

func extractIPFromText(raw string) string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch {
		case r >= '0' && r <= '9':
			return false
		case r >= 'a' && r <= 'f':
			return false
		case r >= 'A' && r <= 'F':
			return false
		case r == '.' || r == ':':
			return false
		default:
			return true
		}
	})

	for _, field := range fields {
		if strings.Contains(field, ".") == false && strings.Contains(field, ":") == false {
			continue
		}
		parsedIP := net.ParseIP(field)
		if parsedIP == nil {
			continue
		}
		return parsedIP.String()
	}

	return ""
}

func getCachedPublicIP() string {
	publicIPCacheMu.RLock()
	defer publicIPCacheMu.RUnlock()

	if publicIPCacheValue == "" || time.Now().After(publicIPCacheExpires) {
		return ""
	}

	return publicIPCacheValue
}

func cachePublicIP(ip string) {
	publicIPCacheMu.Lock()
	defer publicIPCacheMu.Unlock()

	publicIPCacheValue = strings.TrimSpace(ip)
	if publicIPCacheValue == "" {
		publicIPCacheExpires = time.Time{}
		return
	}
	publicIPCacheExpires = time.Now().Add(publicIPCacheTTL)
}

func clearPublicIPCache() {
	publicIPCacheMu.Lock()
	defer publicIPCacheMu.Unlock()

	publicIPCacheValue = ""
	publicIPCacheExpires = time.Time{}
}

// DownFile 从指定的 url 下载文件
type downloadResult struct {
	body        []byte
	filename    string
	statusCode  int
	contentType string
}

type SubtitleContentInspector func(body []byte, ext string) (bool, error)

func downloadFileRaw(l *logrus.Logger, urlStr string) (*downloadResult, error) {
	httpClient, err := newSubtitleDownloadHTTPClient()
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.R().Get(urlStr)
	if err != nil {
		return nil, err
	}

	filename := GetFileName(l, resp.RawResponse)
	if filename == "" {
		l.Warningln("DownFile.GetFileName is string.empty", urlStr)
	}

	result := &downloadResult{
		body:     resp.Body(),
		filename: filename,
	}
	if resp.RawResponse != nil {
		result.statusCode = resp.RawResponse.StatusCode
		result.contentType = resp.RawResponse.Header.Get("Content-Type")
	}

	return result, nil
}

func DownFile(l *logrus.Logger, urlStr string) ([]byte, string, error) {
	result, err := downloadFileRaw(l, urlStr)
	if err != nil {
		return nil, "", err
	}
	return result.body, result.filename, nil
}

func DownSubtitleFile(l *logrus.Logger, inspector SubtitleContentInspector, urlStr string) ([]byte, string, error) {
	result, err := downloadFileRaw(l, urlStr)
	if err != nil {
		return nil, "", err
	}

	if err := ValidateSubtitleDownloadPayload(l, inspector, urlStr, result.filename, result.contentType, result.statusCode, result.body); err != nil {
		return nil, "", err
	}

	return result.body, result.filename, nil
}

// GetFileName 获取下载文件的文件名
func GetFileName(l *logrus.Logger, resp *http.Response) string {
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return ""
	}
	contentDisposition := resp.Header.Get("Content-Disposition")
	if len(contentDisposition) == 0 {
		return fallbackFileNameFromURL(l, resp.Request.URL.String())
	}
	if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
		if fileName := decodeRFC5987FileName(params["filename*"]); fileName != "" {
			return fileName
		}
		if fileName := strings.TrimSpace(params["filename"]); fileName != "" {
			return fileName
		}
	}
	re := regexp.MustCompile(`filename=["]*([^";]+)["]*`)
	matched := re.FindStringSubmatch(contentDisposition)
	if matched != nil && len(matched) > 1 && strings.TrimSpace(matched[1]) != "" {
		return strings.TrimSpace(matched[1])
	}
	if fileName := decodeRFC5987FileName(extractRFC5987Value(contentDisposition)); fileName != "" {
		return fileName
	}
	l.Errorln("GetFileName.Content-Disposition", contentDisposition)
	return fallbackFileNameFromURL(l, resp.Request.URL.String())
}

func fallbackFileNameFromURL(l *logrus.Logger, rawURL string) string {
	m := regexp.MustCompile(`^(.*/)?(?:$|(.+?)(?:(\.[^.]*$)|$))`).FindStringSubmatch(rawURL)
	if m == nil || len(m) < 4 {
		if l != nil {
			l.Warningln("GetFileName.regexp.MustCompile.FindStringSubmatch", rawURL)
		}
		return ""
	}
	return m[2] + m[3]
}

func extractRFC5987Value(contentDisposition string) string {
	idx := strings.Index(strings.ToLower(contentDisposition), "filename*=")
	if idx < 0 {
		return ""
	}
	value := strings.TrimSpace(contentDisposition[idx+len("filename*="):])
	if semi := strings.Index(value, ";"); semi >= 0 {
		value = value[:semi]
	}
	return strings.Trim(value, "\"")
}

func decodeRFC5987FileName(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "\""))
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, "'", 3)
	encoded := value
	if len(parts) == 3 {
		encoded = parts[2]
	}
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(decoded)
}

// ValidateSubtitleDownloadPayload 校验下载内容是否像一个可用的字幕文件或压缩包
func ValidateSubtitleDownloadPayload(l *logrus.Logger, inspector SubtitleContentInspector, urlStr, fileName, contentType string, statusCode int, body []byte) error {
	if statusCode != 0 && (statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices) {
		return fmt.Errorf("unexpected http status %d for %s", statusCode, urlStr)
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return fmt.Errorf("empty download body for %s", urlStr)
	}

	normalizedContentType := strings.ToLower(contentType)
	if strings.Contains(normalizedContentType, "html") ||
		strings.Contains(normalizedContentType, "xml") ||
		strings.Contains(normalizedContentType, "json") {
		return fmt.Errorf("unexpected content-type %q for %s", contentType, urlStr)
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(urlStr))
	}

	if archiveExt := detectArchiveExt(ext, body); archiveExt != "" {
		if err := validateArchivePayload(archiveExt, body); err != nil {
			return fmt.Errorf("invalid archive payload for %s: %w", urlStr, err)
		}
		return nil
	}

	if inspector == nil {
		return fmt.Errorf("subtitle content inspector is nil for %s", urlStr)
	}

	found, err := inspector(body, ext)
	if err != nil {
		return fmt.Errorf("parse subtitle payload for %s: %w", urlStr, err)
	}
	if found == false {
		return fmt.Errorf("download payload is not a subtitle file for %s", urlStr)
	}

	return nil
}

func detectArchiveExt(ext string, body []byte) string {
	switch strings.ToLower(ext) {
	case ".zip", ".tar", ".rar", ".7z":
		return strings.ToLower(ext)
	}

	if len(body) >= 4 && bytes.HasPrefix(body, []byte("PK\x03\x04")) {
		return ".zip"
	}
	if len(body) >= 6 && bytes.HasPrefix(body, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
		return ".7z"
	}
	if len(body) >= 7 && (bytes.HasPrefix(body, []byte("Rar!\x1A\x07\x00")) || bytes.HasPrefix(body, []byte("Rar!\x1A\x07\x01\x00"))) {
		return ".rar"
	}
	if len(body) >= 512 {
		tr := tar.NewReader(bytes.NewReader(body))
		if _, err := tr.Next(); err == nil {
			return ".tar"
		}
	}

	return ""
}

func validateArchivePayload(ext string, body []byte) error {
	switch strings.ToLower(ext) {
	case ".zip":
		_, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		return err
	case ".tar":
		tr := tar.NewReader(bytes.NewReader(body))
		_, err := tr.Next()
		return err
	case ".rar":
		if bytes.HasPrefix(body, []byte("Rar!\x1A\x07\x00")) || bytes.HasPrefix(body, []byte("Rar!\x1A\x07\x01\x00")) {
			return nil
		}
		return fmt.Errorf("invalid rar signature")
	case ".7z":
		if bytes.HasPrefix(body, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
			return nil
		}
		return fmt.Errorf("invalid 7z signature")
	default:
		return fmt.Errorf("unsupported archive ext %s", ext)
	}
}

func AddBaseUrl(baseUrl, url string) string {
	if strings.Contains(url, "://") {
		return url
	}
	return fmt.Sprintf("%s%s", baseUrl, url)
}

// IsDir 存在且是文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// IsFile 存在且是文件
func IsFile(filePath string) bool {
	s, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

// FileNameIsBDMV 是否是 BDMV 蓝光目录，符合返回 true，以及 fakseVideoFPath
func FileNameIsBDMV(id_bdmv_fileFPath string) (bool, string) {
	/*
		这类蓝光视频比较特殊，它没有具体的一个后缀名的视频文件而是由两个文件夹来存储视频数据
		* BDMV
		* CERTIFICATE
		但是不管如何，都需要使用一个文件作为锚点，就选定 CERTIFICATE 中的 id.bdmv 文件
		后续的下载逻辑也需要单独为这个文件进行处理，比如，从这个文件向上一层获取 nfo 文件，
		以及再上一层得到视频文件夹名称等
	*/

	if strings.ToLower(filepath.Base(id_bdmv_fileFPath)) == common.FileBDMV {

		// 这个文件是确认了，那么就需要查看这个文件父级目录是不是 CERTIFICATE 文件夹
		// 且 CERTIFICATE 需要和 BDMV 文件夹都存在
		CERDir := filepath.Dir(id_bdmv_fileFPath)
		BDMVDir := filepath.Join(filepath.Dir(CERDir), "BDMV")
		if IsDir(CERDir) == true && IsDir(BDMVDir) == true {
			return true, filepath.Join(filepath.Dir(CERDir), filepath.Base(filepath.Dir(CERDir))+common.VideoExtMp4)
		}
	}

	return false, ""
}

// ResetWantedVideoExt 重置视频后缀名
func ResetWantedVideoExt() {
	_wantedExtMap = make(map[string]string)
	_defExtMap = make(map[string]string)
}

// IsWantedVideoExtDef 后缀名是否符合规则
func IsWantedVideoExtDef(fileName string) bool {

	if len(_wantedExtMap) < 1 {
		_defExtMap[common.VideoExtMp4] = common.VideoExtMp4
		_defExtMap[common.VideoExtMkv] = common.VideoExtMkv
		_defExtMap[common.VideoExtRmvb] = common.VideoExtRmvb
		_defExtMap[common.VideoExtIso] = common.VideoExtIso
		_defExtMap[common.VideoExtM2ts] = common.VideoExtM2ts

		_wantedExtMap[common.VideoExtMp4] = common.VideoExtMp4
		_wantedExtMap[common.VideoExtMkv] = common.VideoExtMkv
		_wantedExtMap[common.VideoExtRmvb] = common.VideoExtRmvb
		_wantedExtMap[common.VideoExtIso] = common.VideoExtIso
		_wantedExtMap[common.VideoExtM2ts] = common.VideoExtM2ts

		for _, videoExt := range settings.Get().AdvancedSettings.CustomVideoExts {
			_wantedExtMap[videoExt] = videoExt
		}
	}
	fileExt := strings.ToLower(filepath.Ext(fileName))
	_, bFound := _wantedExtMap[fileExt]
	return bFound
}

func GetEpisodeKeyName(season, eps int, zerofill ...bool) string {

	if len(zerofill) < 1 || zerofill[0] == false {
		return "S" + strconv.Itoa(season) + "E" + strconv.Itoa(eps)
	} else {
		return fmt.Sprintf("S%02dE%02d", season, eps)
	}
}

// CopyFile copies a single file from src to dst
func CopyFile(src, dst string) error {
	var err error
	var srcFd *os.File
	var dstFd *os.File
	var srcInfo os.FileInfo

	if srcFd, err = os.Open(src); err != nil {
		return err
	}
	defer func() {
		_ = srcFd.Close()
	}()

	if dstFd, err = os.Create(dst); err != nil {
		return err
	}
	defer func() {
		_ = dstFd.Close()
	}()

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return err
	}
	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// CopyDir copies a whole directory recursively
func CopyDir(src string, dst string) error {
	var err error
	var fds []os.DirEntry
	var srcInfo os.FileInfo

	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if fds, err = os.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := filepath.Join(src, fd.Name())
		dstfp := filepath.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = CopyDir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = CopyFile(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

// CloseChrome 强行结束没有关闭的 Chrome 进程
func CloseChrome(l *logrus.Logger) {

	defer func() {
		l.Infoln("CloseChrome End")
	}()

	l.Infoln("CloseChrome Start...")

	cmdString := ""
	var command *exec.Cmd
	sysType := runtime.GOOS
	if sysType == "linux" {
		// LINUX系统
		cmdString = "pkill -x chrome || pkill -x chromium"
		command = exec.Command("/bin/sh", "-c", cmdString)
	}
	if sysType == "windows" {
		killedCount, err := closeOwnedChromeWindows(l)
		if err != nil {
			l.Warningln("CloseChrome", err)
			return
		}
		if killedCount == 0 {
			l.Debugln("CloseChrome no owned browser process")
			return
		}
		l.Infoln("CloseChrome killed owned browser count:", killedCount)
		return
	}
	if sysType == "darwin" {
		// macOS
		// https://stackoverflow.com/questions/57079120/using-exec-command-in-golang-how-do-i-open-a-new-terminal-and-execute-a-command
		cmdString = `tell application "/Applications/Google Chrome.app" to quit`
		command = exec.Command("osascript", "-s", "h", "-e", cmdString)
	}
	if cmdString == "" || command == nil {
		l.Errorln("CloseChrome OS:", sysType)
		return
	}
	err := command.Run()
	if err != nil {
		if isCloseChromeNoProcessErr(sysType, err) {
			l.Debugln("CloseChrome no running browser process")
			return
		}
		l.Warningln("CloseChrome", err)
	}
}

type windowsChromeProcessInfo struct {
	ProcessId       int    `json:"ProcessId"`
	ParentProcessId int    `json:"ParentProcessId"`
	CommandLine     string `json:"CommandLine"`
}

func closeOwnedChromeWindows(l *logrus.Logger) (int, error) {
	rodRoot := filepath.Clean(DefRodTmpRootFolder())
	if rodRoot == "" {
		return 0, fmt.Errorf("empty rod root folder")
	}

	script := `$ErrorActionPreference = 'Stop'
$procs = @(Get-CimInstance Win32_Process -Filter "Name='chrome.exe'" | Select-Object ProcessId,ParentProcessId,CommandLine)
$procs | ConvertTo-Json -Depth 3 -Compress`
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return 0, err
	}

	processes, err := parseWindowsChromeProcessList(output)
	if err != nil {
		return 0, err
	}

	targetPids := make([]int, 0)
	for _, proc := range processes {
		if shouldCloseOwnedChromeProcessWindows(proc.CommandLine, rodRoot) {
			targetPids = append(targetPids, proc.ProcessId)
		}
	}
	if len(targetPids) == 0 {
		return 0, nil
	}

	killed := 0
	for _, pid := range targetPids {
		cmd := exec.Command("taskkill.exe", "/F", "/T", "/PID", strconv.Itoa(pid))
		if err := cmd.Run(); err != nil {
			if isTaskKillNoProcessErr(err) {
				continue
			}
			return killed, err
		}
		killed++
		if l != nil {
			l.Debugln("CloseChrome killed owned browser pid:", pid)
		}
	}

	return killed, nil
}

func parseWindowsChromeProcessList(raw []byte) ([]windowsChromeProcessInfo, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var many []windowsChromeProcessInfo
	if err := json.Unmarshal([]byte(trimmed), &many); err == nil {
		return many, nil
	}

	var one windowsChromeProcessInfo
	if err := json.Unmarshal([]byte(trimmed), &one); err != nil {
		return nil, err
	}
	return []windowsChromeProcessInfo{one}, nil
}

func shouldCloseOwnedChromeProcessWindows(commandLine string, rodRoot string) bool {
	cmd := strings.TrimSpace(strings.ToLower(commandLine))
	if cmd == "" {
		return false
	}

	userDataDir := extractChromeUserDataDir(commandLine)
	if userDataDir == "" {
		return false
	}

	normalizedUserDataDir := strings.ToLower(filepath.Clean(userDataDir))
	normalizedRodRoot := strings.ToLower(filepath.Clean(rodRoot))
	if normalizedUserDataDir == normalizedRodRoot {
		return true
	}

	prefix := normalizedRodRoot
	if strings.HasSuffix(prefix, string(filepath.Separator)) == false {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(normalizedUserDataDir, prefix)
}

func extractChromeUserDataDir(commandLine string) string {
	const flagName = "--user-data-dir="
	lower := strings.ToLower(commandLine)
	start := strings.Index(lower, flagName)
	if start < 0 {
		return ""
	}

	value := commandLine[start+len(flagName):]
	if value == "" {
		return ""
	}

	if value[0] == '"' {
		value = value[1:]
		end := strings.Index(value, `"`)
		if end < 0 {
			return strings.TrimSpace(value)
		}
		return strings.TrimSpace(value[:end])
	}

	end := strings.IndexAny(value, " \t")
	if end < 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(value[:end])
}

func isTaskKillNoProcessErr(err error) bool {
	if err == nil {
		return false
	}
	exitErr, ok := err.(*exec.ExitError)
	if ok && exitErr.ExitCode() == 128 {
		return true
	}
	return strings.TrimSpace(err.Error()) == "exit status 128"
}

func isCloseChromeNoProcessErr(sysType string, err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	errText := strings.TrimSpace(err.Error())
	if ok {
		switch sysType {
		case "linux", "darwin":
			if exitErr.ExitCode() == 1 {
				return true
			}
		case "windows":
			if exitErr.ExitCode() == 128 {
				return true
			}
		}
	}

	switch sysType {
	case "linux", "darwin":
		return errText == "exit status 1"
	case "windows":
		return errText == "exit status 128"
	default:
		return false
	}
}

// OSCheck 强制的系统支持检查
func OSCheck() bool {
	sysType := runtime.GOOS
	if sysType == "linux" {
		return true
	}
	if sysType == "windows" {
		return true
	}
	if sysType == "darwin" {
		return true
	}

	return false
}

// FixWindowPathBackSlash 修复 Windows 反斜杠的梗
func FixWindowPathBackSlash(path string) string {
	return strings.Replace(path, string(filepath.Separator), "/", -1)
}

func WriteStrings2File(desfilePath string, strings []string) error {
	dstFile, err := os.Create(desfilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = dstFile.Close()
	}()
	allString := ""
	for _, s := range strings {
		allString += s + "\r\n"
	}
	_, err = dstFile.WriteString(allString)
	if err != nil {
		return err
	}
	return nil
}

func TimeNumber2Time(inputTimeNumber float64) time.Time {
	newTime := time.Time{}.Add(time.Duration(inputTimeNumber * math.Pow10(9)))
	return newTime
}

func Time2SecondNumber(inTime time.Time) float64 {
	outSecond := 0.0
	outSecond += float64(inTime.Hour() * 60 * 60)
	outSecond += float64(inTime.Minute() * 60)
	outSecond += float64(inTime.Second())
	outSecond += float64(inTime.Nanosecond()) / 1000 / 1000 / 1000

	return outSecond
}

func Time2Duration(inTime time.Time) time.Duration {
	return time.Duration(Time2SecondNumber(inTime) * math.Pow10(9))
}

func Second2Time(sec int64) time.Time {
	return time.Unix(sec, 0)
}

// ReplaceSpecString 替换特殊的字符
func ReplaceSpecString(inString string, rep string) string {
	return regex_things.RegMatchSpString.ReplaceAllString(inString, rep)
}

func Bool2Int(inBool bool) int {
	if inBool == true {
		return 1
	} else {
		return 0
	}
}

// Round 取整
func Round(x float64) int64 {

	if x-float64(int64(x)) > 0 {
		return int64(x) + 1
	} else {
		return int64(x)
	}

	//return int64(math.Floor(x + 0.5))
}

// MakePowerOfTwo 2的整次幂数 buffer length is not a power of two
func MakePowerOfTwo(x int64) int64 {

	power := math.Log2(float64(x))
	tmpRound := Round(power)

	return int64(math.Pow(2, float64(tmpRound)))
}

// MakeCeil10msMultipleFromFloat 将传入的秒，规整到 10ms 的倍数，返回依然是 秒，向上取整
func MakeCeil10msMultipleFromFloat(input float64) float64 {
	const bb = 100
	// 先转到 10 ms 单位，比如传入是 1.912 - > 191.2
	t10ms := input * bb
	// 191.2 - > 192.0
	newT10ms := math.Ceil(t10ms)
	// 转换回来
	return newT10ms / bb
}

// MakeFloor10msMultipleFromFloat 将传入的秒，规整到 10ms 的倍数，返回依然是 秒，向下取整
func MakeFloor10msMultipleFromFloat(input float64) float64 {
	const bb = 100
	// 先转到 10 ms 单位，比如传入是 1.912 - > 191.2
	t10ms := input * bb
	// 191.2 - > 191.0
	newT10ms := math.Floor(t10ms)
	// 转换回来
	return newT10ms / bb
}

// MakeCeil10msMultipleFromTime 向上取整，规整到 10ms 的倍数
func MakeCeil10msMultipleFromTime(input time.Time) time.Time {

	nowTime := MakeCeil10msMultipleFromFloat(Time2SecondNumber(input))
	newTime := time.Time{}.Add(time.Duration(nowTime * math.Pow10(9)))
	return newTime
}

// MakeFloor10msMultipleFromTime 向下取整，规整到 10ms 的倍数
func MakeFloor10msMultipleFromTime(input time.Time) time.Time {

	nowTime := MakeFloor10msMultipleFromFloat(Time2SecondNumber(input))
	newTime := time.Time{}.Add(time.Duration(nowTime * math.Pow10(9)))
	return newTime
}

// Time2SubTimeString 时间转字幕格式的时间字符串
func Time2SubTimeString(inTime time.Time, timeFormat string) string {
	/*
		这里进行时间转字符串的时候有一点比较特殊
		正常来说输出的格式是类似 15:04:05.00
		那么有个问题，字幕的时间格式是 0:00:12.00， 小时，是个数，除非有跨度到 20 小时的视频，不然小时就应该是个数
		这就需要一个额外的函数去处理这些情况
	*/
	outTimeString := inTime.Format(timeFormat)
	if inTime.Hour() > 9 {
		// 小时，两位数
		return outTimeString
	} else {
		// 小时，一位数
		items := strings.SplitN(outTimeString, ":", -1)
		if len(items) == 3 {

			outTimeString = strings.Replace(outTimeString, items[0], fmt.Sprintf("%d", inTime.Hour()), 1)
			return outTimeString
		}

		return outTimeString
	}
}

// IsEqual 比较 float64
func IsEqual(f1, f2 float64) bool {
	const MIN = 0.000001
	if f1 > f2 {
		return math.Dim(f1, f2) < MIN
	} else {
		return math.Dim(f2, f1) < MIN
	}
}

// ParseTime 解析字幕时间字符串，这里可能小数点后面有 2-4 位
func ParseTime(inTime string) (time.Time, error) {

	parseTime, err := time.Parse(common.TimeFormatPoint2, inTime)
	if err != nil {
		parseTime, err = time.Parse(common.TimeFormatPoint3, inTime)
		if err != nil {
			parseTime, err = time.Parse(common.TimeFormatPoint4, inTime)
		}
	}
	return parseTime, err
}

// GetFileSHA1 获取文件的 SHA1 值
func GetFileSHA1(srcFileFPath string) (string, error) {

	infile, err := os.Open(srcFileFPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = infile.Close()
	}()

	h := sha1.New()
	_, err = io.Copy(h, infile)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteFile 写文件
func WriteFile(desFileFPath string, bytes []byte) error {
	var err error
	nowDesPath := desFileFPath
	if filepath.IsAbs(nowDesPath) == false {
		nowDesPath, err = filepath.Abs(nowDesPath)
		if err != nil {
			return err
		}
	}
	// 创建对应的目录
	nowDirPath := filepath.Dir(nowDesPath)
	err = os.MkdirAll(nowDirPath, os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(nowDesPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = file.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

// GetNowTimeString 获取当前的时间，没有秒
func GetNowTimeString() (string, int, int, int) {
	nowTime := time.Now()
	addString := fmt.Sprintf("%d-%d-%d", nowTime.Hour(), nowTime.Minute(), nowTime.Nanosecond())
	return addString, nowTime.Hour(), nowTime.Minute(), nowTime.Nanosecond()
}

// GenerateAccessToken 生成随机的 AccessToken
func GenerateAccessToken() string {
	u4 := uuid.New()
	return u4.String()
}

func Get2UUID() string {
	u4 := uuid.New()
	u5 := uuid.New()
	return u4.String() + u5.String()
}

func UrlJoin(hostUrl, subUrl string) (string, error) {

	u, err := url.Parse(hostUrl)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, subUrl)
	return u.String(), nil
}

// GetFileSHA1String 获取文件的 SHA1 字符串
func GetFileSHA1String(fileFPath string) (string, error) {
	h := sha1.New()

	fp, err := os.Open(fileFPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = fp.Close()
	}()

	partAll, err := io.ReadAll(fp)
	if err != nil {
		return "", err
	}

	h.Write(partAll)
	hashBytes := h.Sum(nil)

	return fmt.Sprintf("%x", md5.Sum(hashBytes)), nil
}

// GetFileSHA256String 获取文件的 SHA256 字符串
func GetFileSHA256String(fileFPath string) (string, error) {

	fp, err := os.Open(fileFPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = fp.Close()
	}()

	partAll, err := io.ReadAll(fp)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(partAll)), nil
}

func GetRestOfDaySec() time.Duration {

	nowTime := time.Now()
	todayLast := nowTime.Format("2006-01-02") + " 23:59:59"
	todayLastTime, _ := time.ParseInLocation("2006-01-02 15:04:05", todayLast, time.Local)
	// 今天剩余的时间（s）
	restOfDaySec := time.Duration(todayLastTime.Unix()-time.Now().Local().Unix()) * time.Second

	return restOfDaySec
}

// IntToBytes 整形转换成字节
func IntToBytes(n int) ([]byte, error) {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	err := binary.Write(bytesBuffer, binary.BigEndian, x)
	if err != nil {
		return nil, err
	}
	return bytesBuffer.Bytes(), nil
}

// BytesToInt 字节转换成整形
func BytesToInt(b []byte) (int, error) {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	err := binary.Read(bytesBuffer, binary.BigEndian, &x)
	if err != nil {
		return 0, err
	}

	return int(x), nil
}

func PrintPanicStack(log *logrus.Logger) {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	log.Errorln(fmt.Sprintf("%s", buf[:n]))
}

func GetMaxSizeFile(path string) string {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return ""
	}
	var maxFile os.FileInfo
	for _, file := range files {
		if maxFile == nil {
			maxFile = file
		} else {
			if file.Size() > maxFile.Size() {
				maxFile = file
			}
		}
	}
	return filepath.Join(path, maxFile.Name())
}

func Sha256File(fileFPath string) (string, int, error) {
	fp, err := os.Open(fileFPath)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		_ = fp.Close()
	}()

	partAll, err := io.ReadAll(fp)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", sha256.Sum256(partAll)), len(partAll), nil
}

var (
	_wantedExtMap = make(map[string]string) // 人工确认的需要监控的视频后缀名
	_defExtMap    = make(map[string]string) // 内置支持的视频后缀名列表
)
