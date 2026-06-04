package pkg

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func ReadCustomPortFile(log *logrus.Logger) int {
	customPortFilePath := resolveCustomFilePath(customPort)
	if IsFile(customPortFilePath) == false {
		return defPort
	} else {
		bytes, err := os.ReadFile(customPortFilePath)
		if err != nil {
			log.Errorln("ReadFile CustomPort Error", err)
			log.Infoln("Use DefPort", defPort)
			return defPort
		}

		atoi, err := strconv.Atoi(string(bytes))
		if err != nil {
			log.Errorln("Atoi CustomPort Error", err)
			log.Infoln("Use DefPort", defPort)
			return defPort
		}

		log.Infoln("Use CustomPort", atoi)
		return atoi
	}
}

func ReadCustomAuthFile(log *logrus.Logger) bool {
	customAuthFilePath := resolveCustomFilePath(customAuth)
	if IsFile(customAuthFilePath) == false {
		return false
	} else {
		bytes, err := os.ReadFile(customAuthFilePath)
		if err != nil {
			log.Errorln("ReadFile CustomAuth Error", err)
			return false
		}

		nowContent := string(bytes)
		authStings := strings.Split(nowContent, "@@@@")
		if len(authStings) != 3 {
			log.Errorln("ReadFile CustomAuth Error", err)
			return false
		}

		SetBaseKey(authStings[0])
		SetAESKey16(authStings[1])
		SetAESIv16(authStings[2])

		log.Infoln("Use CustomAuth")
		return true
	}
}

func resolveCustomFilePath(fileName string) string {
	if IsFile(fileName) {
		return fileName
	}

	configRootDir := ConfigRootDirFPath()
	if configRootDir == "" {
		return fileName
	}

	configRootFilePath := filepath.Join(configRootDir, fileName)
	if IsFile(configRootFilePath) {
		return configRootFilePath
	}

	return fileName
}

const (
	defPort    = 19035
	customPort = "CustomPort"
	customAuth = "CustomAuth"
)
