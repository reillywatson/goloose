package goloose

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"unicode"
	"unicode/utf8"
)

type typespec struct {
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Value     any        `json:"value"`
	StructVal []typespec `json:"structval"`
}

func FuzzGoloose(f *testing.F) {
	testcases := []string{
		`[[
			{"name": "A", "type": "int", "value": 3},
			{"name": "B", "type": "string", "value": "foo"}
		],
		[
			{"name": "A", "type": "int"},
			{"name": "B", "type": "string"}
		]]`,
		`[[
			{"name":"A", "type": "struct", "structval": [{"name": "B", "type": "int", "value": 3}]}
		],
		[
			{"name": "A", "type": "struct", "structval": [{"name": "B", "type": "float64"}]}
		]]`,
	}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, code string) {
		var typeDesc [][]typespec
		if err := json.Unmarshal([]byte(code), &typeDesc); err != nil {
			return
		}
		if len(typeDesc) != 2 {
			return
		}
		in, err := loadTypespec(typeDesc[0])
		if err != nil {
			return
		}
		out, err := loadTypespec(typeDesc[1])
		if err != nil {
			return
		}
		outSlow, err := loadTypespec(typeDesc[1])
		if err != nil {
			return
		}
		if err := ToStruct(in, out); err != nil {
			t.Error(err)
		}
		if err := toStructSlow(in, outSlow); err != nil {
			return // can't JSON compare!
		}
		if !reflect.DeepEqual(out, outSlow) {
			t.Errorf("Got %+v\nExpected %+v", out, outSlow)
		}
	})
}

func loadTypespec(spec []typespec) (any, error) {
	var values []reflect.Value
	var fields []reflect.StructField
	fieldNames := map[string]bool{}
	for _, f := range spec {
		if !isValidFieldName(f.Name) {
			return nil, fmt.Errorf("invalid field name %q", f.Name)
		}
		if fieldNames[f.Name] {
			return nil, fmt.Errorf("duplicate field name %q", f.Name)
		}
		fieldNames[f.Name] = true
		var value any
		switch f.Type {
		case "int":
			asFloat, _ := f.Value.(float64)
			value = int(asFloat)
		case "float64":
			value, _ = f.Value.(float64)
		case "string":
			value, _ = f.Value.(string)
		case "struct":
			var err error
			value, err = loadTypespec(f.StructVal)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown type %q", f.Type)
		}
		c := f.Name[0]
		var pkgPath string
		if 'a' <= c && c <= 'z' || c == '_' {
			pkgPath = "goloose"
		}
		fields = append(fields, reflect.StructField{
			Name:    f.Name,
			Type:    reflect.TypeOf(value),
			PkgPath: pkgPath,
		})
		values = append(values, reflect.ValueOf(value))
	}
	v := reflect.New(reflect.StructOf(fields)).Elem()
	for i, val := range values {
		if val.IsValid() {
			if v.Field(i).CanSet() {
				v.Field(i).Set(val)
			}
		}
	}
	return v.Addr().Interface(), nil
}

func isValidFieldName(fieldName string) bool {
	for i, c := range fieldName {
		if i == 0 && !isLetter(c) {
			return false
		}

		if !(isLetter(c) || unicode.IsDigit(c)) {
			return false
		}
	}

	return len(fieldName) > 0
}
func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}
