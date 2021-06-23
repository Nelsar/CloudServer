package utils

import (
	"encoding/json"
	"net/http"
)

func ParseJsonRaw(m json.RawMessage, dst interface{}) error {
	err := json.Unmarshal(m, &dst)

	return err
}

func ParseJson(r *http.Request, dst interface{}) error {
	// Read the Body content
	//var bodyBytes []byte
	//if r.Body != nil {
	//	bodyBytes, _ = ioutil.ReadAll(r.Body)
	//}
	//
	//// // Restore the io.ReadCloser to its original state
	//r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	//str := string(bodyBytes)

	decoder := json.NewDecoder(r.Body)

	return decoder.Decode(&dst)

}