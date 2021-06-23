package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"gitlab.citicom.kz/CloudServer/server/models"
	"io/ioutil"
	"net/http"
)

func (server *Server) ProcessNewIncomingMessage(
	ctx context.Context,
	m *models.InputMessage,
	oilFieldId int64,
) {
	switch m.Type {
	case models.MessageTypeCloudSyncGzip:
		var input struct {
			FileName string `json:"file_name"`
		}
		if err := json.Unmarshal(m.Body, &input); err != nil {
			return
		}

		oilField, err := server.db.GetOilField(ctx, oilFieldId)
		if err != nil {
			return
		}

		dataForSync, err := syncRequest(oilField.HttpAddress, input.FileName)
		if err != nil {
			outputJson := struct {
				FileName string `json:"file_name"`
			}{
				input.FileName,
			}
			server.SendMessageOilField(models.MessageTypeCloudSyncGzipAck, outputJson, oilFieldId)
			return
		}

		fmt.Println("DATA SIZE: ", len(dataForSync.Data))
		server.db.SynchronizeData(ctx, server.influxDB, dataForSync.Controllers, dataForSync.Data, oilFieldId)

		outputJson := struct {
			FileName string `json:"file_name"`
		}{
			input.FileName,
		}
		server.SendMessageOilField(models.MessageTypeCloudSyncGzipAck, outputJson, oilFieldId)
	default:
		fmt.Printf("Incorrect type: %s", m.Type)
	}

}

func syncRequest(address string,fileName string) (*models.CloudGzipData, error) {
	outputJson := struct {
		FileName string `json:"file_name"`
	}{
		fileName,
	}
	jsonBytes, _ := json.Marshal(outputJson)

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("http://%s/gzipfile", address),
		bytes.NewBuffer(jsonBytes),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return nil, err
	}

	if resp.StatusCode == 500 {
		fmt.Println("ERROR FILE NOT FOUND")
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return nil, err
	}

	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("Empty bodyBytes")
	}

	b := bytes.NewBuffer(bodyBytes)
	r, err := gzip.NewReader(b)
	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	resData := resB.Bytes()

	var outputGzip *models.CloudGzipData
	err = json.Unmarshal(resData, &outputGzip)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return nil, err
	}

	return outputGzip, nil
}
