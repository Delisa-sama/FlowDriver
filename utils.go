package flowdriver

import (
	"encoding/json"
	"github.com/Delisa-sama/FlowDriver/flowerror"
	"net/http"
)

type EmptyStruct struct{}

func WriteJSONError(w http.ResponseWriter, err flowerror.FlowError, status int) error {
	return WriteJSONResponse(w, err, status)
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
