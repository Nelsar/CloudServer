package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

func CreateFolder(path string, l *log.Entry) (string, error) {
	uploadPath := viper.GetString("UploadPath")

	fullpath := filepath.Join(uploadPath, path)

	exist, err := PathExists(fullpath)
	if err != nil {
		l.WithFields(log.Fields{
			"error":    err,
			"fullpath": fullpath,
		}).Error("CreateFolder: Can't check fullpath existing")
		return "", err
	}
	if exist {
		return fullpath, nil
	}

	var perm os.FileMode = 0755

	err = os.MkdirAll(fullpath, perm)
	if err != nil {
		l.WithFields(log.Fields{
			"error":    err,
			"fullpath": fullpath,
		}).Error("CreateFolder: Can't create folder")
		return "", err
	}
	return fullpath, nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return err != nil, err
}
