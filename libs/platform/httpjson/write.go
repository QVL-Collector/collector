package httpjson

import (
	"encoding/json"
	"net/http"

	"collector/libs/platform/problem"
)

func Write(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		problem.Write(w, problem.Internal())
		return
	}
	w.Header().Set("Content-Type", mediaType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
