package upload

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func RemoveFile(filename string, l *log.Entry) error {
	l.Info("RemoveFile: try to remove file with filename '" +
		filename + "'")

	if err := os.Remove(filename); err != nil {
		l.WithFields(log.Fields{
			"error":    err,
			"filename": filename,
		}).Error("RemoveFile: Can't remove file")
		return err
	}

	return nil
}
