package upload

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/models"
	"gitlab.citicom.kz/CloudServer/server/utils"
)

type UploadError struct {
	ErrorString string
}

func (err UploadError) Error() string {
	return err.ErrorString
}

func FileExists(
	r *http.Request,
	fileKey string,
) bool {
	infile, _, err := r.FormFile(fileKey)
	if err != nil {
		return false
	}
	defer infile.Close()

	return true
}

func UploadFile(
	r *http.Request,
	fileKey string,
	path string,
	l *log.Entry,
) (*models.FileData, error) {
	l.Info("UploadFile: try to upload file with key '" + fileKey + "' to folder: " + path)

	infile, fileHandler, err := r.FormFile(fileKey)
	if err != nil {
		l.Errorf("UploadFile(48): %v")
		return nil, err
	}
	defer infile.Close()

	folderPath, err := utils.CreateFolder(path, l)
	if err != nil {
		l.Errorf("Create Folder(load:56): %v", err)
		return nil, err
	}

	ext := filepath.Ext(fileHandler.Filename)
	filename := xid.New().String() + ext
	filepath := filepath.Join(folderPath, filename)

	fmt.Println("file name: ", filename)

	outfile, err := os.Create(filepath)
	if err != nil {
		l.Errorf("File Create(load:68): %v", err)
		return nil, err
	}
	defer outfile.Close()

	_, err = io.Copy(outfile, infile)
	if err != nil {
		l.Errorf("File Copy(load:75): %v", err)
		return nil, err
	}

	fileURL := fmt.Sprintf("/%s/%s", path, filename)

	return &models.FileData{
		FilePath: filepath,
		FileName: filename,
		Url:      fileURL,
	}, nil
}

func UploadImage(
	r *http.Request,
	fileKey string,
	path string,
	l *log.Entry,
) (*models.ImageFileData, error) {
	l.Info("UploadImage: try to upload image with key '" + fileKey + "' to folder: " + path)

	infile, _, err := r.FormFile(fileKey)
	if err != nil {
		return nil, err
	}
	defer infile.Close()

	buff := make([]byte, 512) // docs tell that it take only first 512 bytes into consideration
	if _, err = infile.Read(buff); err != nil {
		return nil, err
	}
	infile.Seek(0, 0)

	mimeType := http.DetectContentType(buff)
	var ext string

	if mimeType == "image/png" {
		ext = "png"
	} else if mimeType == "image/jpeg" {
		ext = "jpg"
	} else {
		return nil, UploadError{ErrorString: "Unsupported MIME type: '" + mimeType + "'"}
	}

	filename := xid.New().String()

	folderPath, err := utils.CreateFolder(path, l)
	if err != nil {
		return nil, err
	}

	filepath := filepath.Join(folderPath, filename+"."+ext)

	outfile, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	defer outfile.Close()

	_, err = io.Copy(outfile, infile)
	if err != nil {
		return nil, err
	}

	fileUrl := fmt.Sprintf("/%s/%s.%s", path, filename, ext)

	return &models.ImageFileData{
		FilePath:  filepath,
		FileName:  filename,
		Extension: ext,
		Url:       fileUrl,
	}, nil

}

func RemoveUploadedFile(filePath string) {
	os.Remove("." + filePath)
}

func SaveImage(
	base64String string,
	fileName string,
	path string,
	l *log.Entry,
) (*models.ImageFileData, error) {
	unbased, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(unbased)

	ext := "jpg"
	folderPath, err := utils.CreateFolder(path, l)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(folderPath, fileName+"."+ext)
	outfile, err := os.Create(filePath)

	if err != nil {
		return nil, err
	}
	defer outfile.Close()

	_, err = io.Copy(outfile, r)
	if err != nil {
		return nil, err
	}

	fileUrl := fmt.Sprintf("/%s/%s.%s", path, fileName, ext)

	return &models.ImageFileData{
		FilePath:  filePath,
		FileName:  fileName,
		Extension: ext,
		Url:       fileUrl,
	}, nil
}
