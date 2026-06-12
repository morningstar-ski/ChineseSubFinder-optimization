package filter

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func SkipFileInfo(l *logrus.Logger, curFile os.DirEntry, fileFullPath string) bool {
	return skipFileInfo(l, curFile, fileFullPath, false)
}

func SkipFileInfo4Sub(l *logrus.Logger, curFile os.DirEntry, fileFullPath string) bool {
	return skipFileInfo(l, curFile, fileFullPath, true)
}

func skipFileInfo(l *logrus.Logger, curFile os.DirEntry, fileFullPath string, allowSmallSub bool) bool {
	if curFile.IsDir() == true {
		// 鎺掗櫎缂撳瓨鏂囦欢澶癸紝瑙?#532
		if strings.HasPrefix(curFile.Name(), ".@__thumb") == true {
			l.Debugln("curFile is dir and match `.@__thumb`, skip")
			return true
		}
	}

	// 璺宠繃涓嶇鍚堢殑鏂囦欢锛屾瘮濡?MAC OS 涓嬪彲鑳芥湁缂撳瓨鏂囦欢锛岃 #138
	fi, err := curFile.Info()
	if err != nil {
		l.Errorln("curFile.Info:", curFile.Name(), err)
		return true
	}

	// 灏侀潰缂撳瓨鏂囦欢澶逛腑鐨勬枃浠堕兘瑕佽烦杩?.@__thumb  #581
	parentFolderName := filepath.Base(filepath.Dir(fileFullPath))
	if strings.HasPrefix(parentFolderName, ".@__thumb") == true {
		l.Debugln("curFile is in .@__thumb folder, skip")
		return true
	}

	// 杞摼鎺ラ棶棰?#558
	if fi.Size() < 1000 {
		fileInfo, err := os.Lstat(fileFullPath)
		if err != nil {
			l.Errorln("os.Lstat:", fileFullPath, err)
			return true
		}
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			// 纭鏄蒋杩炴帴
			l.Debugln("curFile is symlink,", fileFullPath)
		} else if allowSmallSub == true {
			l.Debugln("curFile.Size() < 1000 but allowed for subtitle:", curFile.Name())
		} else {
			l.Debugln("curFile.Size() < 1000:", curFile.Name())
			return true
		}
	}

	if fi.Size() == 4096 && strings.HasPrefix(curFile.Name(), "._") == true {
		l.Debugln("curFile.Size() == 4096 && Prefix Name == ._*", curFile.Name())
		return true
	}
	// 璺宠繃棰勫憡鐗囷紝瑙?#315
	if strings.HasSuffix(strings.ReplaceAll(curFile.Name(), filepath.Ext(curFile.Name()), ""), "-trailer") == true {
		l.Debugln("curFile Name has -trailer:", curFile.Name())
		return true
	}

	return false
}
