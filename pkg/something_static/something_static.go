package something_static

import (
	b64 "encoding/base64"
	"errors"
	"os"
	"path/filepath"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/sirupsen/logrus"
)

func WriteFile(CloneProjectDesSaveDir, enString, nowTime, nowTimeFileNamePrix string) (bool, error) {
	saveFileFPath := filepath.Join(CloneProjectDesSaveDir, nowTimeFileNamePrix+common.StaticFileName00)
	saveFileFPathWait := filepath.Join(CloneProjectDesSaveDir, nowTimeFileNamePrix+common.StaticFileName00+waitExt)

	if pkg.IsFile(saveFileFPath) == true {
		err := writeFile(saveFileFPathWait, enString, nowTime)
		if err != nil {
			return false, err
		}
		orgFileSHA1, err := pkg.GetFileSHA1(saveFileFPath)
		if err != nil {
			return false, err
		}
		waitFileSHA1, err := pkg.GetFileSHA1(saveFileFPathWait)
		if err != nil {
			return false, err
		}
		if orgFileSHA1 == waitFileSHA1 {
			err = os.Remove(saveFileFPathWait)
			if err != nil {
				return false, err
			}
			return false, nil
		}
		err = os.Remove(saveFileFPath)
		if err != nil {
			return false, err
		}
		err = os.Rename(saveFileFPathWait, saveFileFPath)
		if err != nil {
			return false, err
		}
	} else {
		err := writeFile(saveFileFPath, enString, nowTime)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func writeFile(saveFileFPath, enString, nowTime string) error {
	file, err := os.Create(saveFileFPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	_, err = file.WriteString(enString + b64.StdEncoding.EncodeToString([]byte(nowTime)))
	if err != nil {
		return err
	}

	return nil
}

func getCodeFromWeb(l *logrus.Logger, desUrl string) (string, string, error) {
	fileBytes, _, err := pkg.DownFile(l, desUrl)
	if err != nil {
		return "", "", err
	}

	if len(fileBytes) < 24 {
		return "", "", errors.New("fileBytes len < 24")
	}

	timeB64String := fileBytes[len(fileBytes)-16:]
	decodedTime, err := b64.StdEncoding.DecodeString(string(timeB64String))
	if err != nil {
		return "", "", err
	}
	decodeTimeStr := string(decodedTime)

	codeB64String := fileBytes[:len(fileBytes)-16]
	decodedCode, err := b64.StdEncoding.DecodeString(string(codeB64String))
	if err != nil {
		return "", "", err
	}
	decodeCodeStr := string(decodedCode)

	return decodeTimeStr, decodeCodeStr, nil
}

const waitExt = ".wait"
