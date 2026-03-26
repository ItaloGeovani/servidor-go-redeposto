package utils

import (
	"encoding/json"
	"net/http"
)

func DecodificarJSON(r *http.Request, destino any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(destino)
}
