package goloose

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestToStructStringUTF8(t *testing.T) {
	type namedString string
	type message struct {
		Value string `json:"value"`
	}
	strings := []struct {
		name, value string
	}{
		{"empty", ""},
		{"ASCII", "hello\x00\n\"<>&"},
		{"Unicode", "héllo 世界 🌍 �"},
		{"fuzz regression", "0000\x91"},
		{"consecutive invalid bytes", "a\xff\xfe\x80z"},
		{"truncated sequence", "a\xe2\x82"},
		{"overlong encoding", "\xc0\xaf"},
		{"surrogate", "\xed\xa0\x80"},
		{"out of range", "\xf4\x90\x80\x80"},
		{"mixed valid and invalid", "é\xff世\xfe🌍"},
	}
	for _, str := range strings {
		t.Run(str.name, func(t *testing.T) {
			s := str.value
			cases := []struct {
				name    string
				in, out any
			}{
				{"string", s, new(string)},
				{"interface", s, new(any)},
				{"named string", namedString(s), new(namedString)},
				{"string to named string", s, new(namedString)},
				{"named string to string", namedString(s), new(string)},
				{"pointer", &s, new(*string)},
				{"struct", message{s}, new(message)},
				{"struct to map", message{s}, new(map[string]any)},
				{"slice", []string{s}, new([]string)},
				{"slice to interface", []string{s}, new([]any)},
				{"map", map[string]any{"value": s}, new(map[string]string)},
				{"fast map", map[string]string{"value": s}, new(map[string]any)},
				{"fast map to interface", map[string]string{"value": s}, new(any)},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					want := reflect.New(reflect.TypeOf(tc.out).Elem()).Interface()
					encoded, err := json.Marshal(tc.in)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(encoded, want); err != nil {
						t.Fatal(err)
					}
					if err := ToStruct(tc.in, tc.out); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(tc.out, want) {
						t.Fatalf("got %#v, want %#v", reflect.ValueOf(tc.out).Elem().Interface(), reflect.ValueOf(want).Elem().Interface())
					}
				})
			}
		})
	}
}

func TestToStructStringUTF8AfterTransform(t *testing.T) {
	options := Options{Transforms: []TransformFunc{func(v any) any {
		if _, ok := v.(string); ok {
			return "\xff\xfe"
		}
		return v
	}}}
	for _, in := range []any{
		map[string]string{"value": "valid"}, // fast path
		map[string]any{"value": "valid"},
	} {
		var got map[string]any
		if err := ToStruct(in, &got, options); err != nil {
			t.Fatal(err)
		}
		if got["value"] != "��" {
			t.Fatalf("input %T: got %#v, want two replacement characters", in, got)
		}
	}
}
