package httpjson

import (
	"reflect"
	"strings"
)

type fieldNames map[string]struct{}

func (n fieldNames) contains(key string) bool {
	_, found := n[key]
	return found
}

func exactFieldNames(target any) (fieldNames, bool) {
	collector := nameCollector{names: fieldNames{}, visited: map[reflect.Type]bool{}}
	exact := collector.collect(reflect.TypeOf(target))
	return collector.names, exact
}

type nameCollector struct {
	names   fieldNames
	visited map[reflect.Type]bool
}

func (c nameCollector) collect(t reflect.Type) bool {
	t = elementType(t)
	if t == nil || t.Kind() == reflect.Map || t.Kind() == reflect.Interface {
		return false
	}
	if t.Kind() != reflect.Struct || c.visited[t] {
		return true
	}
	c.visited[t] = true
	for index := range t.NumField() {
		if !c.collectField(t.Field(index)) {
			return false
		}
	}
	return true
}

func (c nameCollector) collectField(field reflect.StructField) bool {
	name, tagged := jsonName(field)
	if name == "-" {
		return true
	}
	if field.Anonymous && !tagged {
		return c.collect(field.Type)
	}
	if !field.IsExported() {
		return true
	}
	c.names[name] = struct{}{}
	return c.collect(field.Type)
}

func jsonName(field reflect.StructField) (string, bool) {
	tag, tagged := field.Tag.Lookup("json")
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return field.Name, false
	}
	return name, tagged
}

func elementType(t reflect.Type) reflect.Type {
	for t != nil && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array) {
		t = t.Elem()
	}
	return t
}
