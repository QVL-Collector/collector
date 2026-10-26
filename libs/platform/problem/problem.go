package problem

import (
	"encoding/json"
	"net/http"
)

const (
	ContentType     = "application/problem+json"
	blankType       = "about:blank"
	validationType  = "urn:collector:probleme:saisie-invalide"
	validationTitle = "Saisie invalide"
)

type FieldError struct {
	Field string `json:"champ"`
	Code  string `json:"code"`
}

type Problem struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"erreurs,omitempty"`
}

func New(status int, detail string) Problem {
	return Problem{Type: blankType, Title: http.StatusText(status), Status: status, Detail: detail}
}

func Typed(typeURI, title string, status int, detail string) Problem {
	return Problem{Type: typeURI, Title: title, Status: status, Detail: detail}
}

func Validation(fieldErrors []FieldError) Problem {
	invalid := Typed(validationType, validationTitle, http.StatusBadRequest, "Un ou plusieurs champs sont invalides.")
	invalid.Errors = fieldErrors
	return invalid
}

func Unauthorized() Problem {
	return New(http.StatusUnauthorized, "Authentification requise.")
}

func Forbidden() Problem {
	return New(http.StatusForbidden, "Droits insuffisants.")
}

func NotFound() Problem {
	return New(http.StatusNotFound, "Ressource introuvable.")
}

func Internal() Problem {
	return New(http.StatusInternalServerError, "Erreur interne.")
}

func Unavailable() Problem {
	return New(http.StatusServiceUnavailable, "Service indisponible.")
}

func Write(w http.ResponseWriter, p Problem) {
	body, err := json.Marshal(p)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)
	_, _ = w.Write(body)
}
