package response

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type errorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func setupResponse(w *http.ResponseWriter, code int) {
	(*w).Header().Set("Content-Type", "application/json; charset=UTF-8")
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, AuthToken")
	(*w).WriteHeader(code)
}

// ErrorResponse generate a error response
func ErrorResponse(l *log.Entry, w http.ResponseWriter, code int, message string, data interface{}) {
	setupResponse(&w, code)

	response := &errorResponse{
		Code: code,
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		l.WithFields(log.Fields{
			"code":    code,
			"message": message,
			"data":    data,
		}).Error("Error json encode in ErrorResp")
	}
}

func Response(l *log.Entry, w http.ResponseWriter, data interface{}) {
	setupResponse(&w, http.StatusOK)

	response := &errorResponse{
		Code: 0,
		Message: "",
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		l.WithFields(log.Fields{
			"code":    0,
			"message": "",
			"data":    data,
		}).Error("Error json encode in Response")
	}
}