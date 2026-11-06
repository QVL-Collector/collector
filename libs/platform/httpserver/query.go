package httpserver

import (
	"net/url"
	"strconv"

	"collector/libs/platform/problem"
)

const (
	pageParameter     = "page"
	pageSizeParameter = "taille"
	firstPage         = 1
)

type PageLimits struct {
	DefaultSize int
	MaxSize     int
	MaxNumber   int
}

type Pagination struct {
	Number int
	Size   int
}

func PaginationOf(query url.Values, limits PageLimits) (Pagination, []problem.FieldError) {
	number, numberErrors := BoundedQueryInt(query, pageParameter, firstPage, limits.MaxNumber)
	size, sizeErrors := BoundedQueryInt(query, pageSizeParameter, limits.DefaultSize, limits.MaxSize)
	return Pagination{Number: number, Size: size}, append(numberErrors, sizeErrors...)
}

func BoundedQueryInt(query url.Values, name string, fallback, maximum int) (int, []problem.FieldError) {
	raw, present, fieldErrors := SingleQueryParameter(query, name)
	if !present || len(fieldErrors) > 0 {
		return fallback, fieldErrors
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, []problem.FieldError{{Field: name, Code: problem.CodeInvalid}}
	}
	if value < 1 || value > maximum {
		return 0, []problem.FieldError{{Field: name, Code: problem.CodeOutOfBounds}}
	}
	return value, nil
}

func SingleQueryParameter(query url.Values, name string) (string, bool, []problem.FieldError) {
	values, present := query[name]
	if !present {
		return "", false, nil
	}
	if len(values) != 1 || values[0] == "" {
		return "", true, []problem.FieldError{{Field: name, Code: problem.CodeInvalid}}
	}
	return values[0], true, nil
}
