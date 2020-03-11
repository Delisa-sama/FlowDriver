package flowdriver

import (
	"encoding/json"
	"net/http"
)

type EmptyStruct struct{}

// если ответ предполагает пустой json
var EMPTY = EmptyStruct{}

func WriteJSONError(w http.ResponseWriter, message string, status int) error {
	return WriteJSONResponse(w, map[string]string{"error": message}, status)
}

func WriteJSONResponse(w http.ResponseWriter, v interface{}, status int) error {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		return err
	}
	return nil
}
