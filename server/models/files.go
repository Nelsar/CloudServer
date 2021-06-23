package models

type ImageFileData struct {
	FilePath  string
	FileName  string
	Extension string
	Url       string
}

func (fileData ImageFileData) GetFullFilename() string {
	return fileData.FileName + "." + fileData.Extension
}

type FileData struct {
	FilePath string
	FileName string
	Url      string
}
