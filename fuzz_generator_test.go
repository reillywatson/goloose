package goloose

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func TestFuzzGenerator(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	inputs := append([]string(nil), fuzzSeeds...)
	for _, b := range []byte{0, 3, 4, 5, 6, 7, 255} {
		inputs = append(inputs, strings.Repeat(string([]byte{b}), fuzzMaxInput*2))
	}
	for i := 0; i < 1000; i++ {
		data := make([]byte, rng.Intn(512))
		rng.Read(data)
		inputs = append(inputs, string(data))
	}
	accepted := 0
	for i, data := range inputs {
		c, replay := generateFuzzCase(data), generateFuzzCase(data)
		if !reflect.DeepEqual(c.in, replay.in) || c.got.Type() != replay.got.Type() ||
			!reflect.DeepEqual(c.got.Interface(), replay.got.Interface()) ||
			!reflect.DeepEqual(c.got.Interface(), c.want.Interface()) {
			t.Fatalf("generation is not reproducible for %q", data)
		}
		encoded, err := json.Marshal(c.in)
		if err != nil {
			t.Fatalf("input %q could not be marshaled: %v", data, err)
		}
		if err := json.Unmarshal(encoded, c.want.Interface()); err == nil {
			accepted++
		} else if i < len(fuzzSeeds) {
			t.Fatalf("seed %d would skip the comparison: %v", i, err)
		}
		// Running the reference must not mutate the other destination or input.
		if !reflect.DeepEqual(c.got.Interface(), replay.got.Interface()) || !reflect.DeepEqual(c.in, replay.in) {
			t.Fatalf("reference conversion shares storage with input/destination for %q", data)
		}
	}
	if accepted < len(inputs)*3/4 {
		t.Fatalf("only %d/%d generated cases reached the comparison", accepted, len(inputs))
	}
	t.Logf("%d/%d generated cases accepted by encoding/json", accepted, len(inputs))
}

func TestFuzzGeneratorValues(t *testing.T) {
	tests := []struct {
		name string
		data string
		want any
	}{
		{"slice", "\x00\x00\x03\x00\x03\x01a\x01b", []string{"a", "b"}},
		{"nil slice", "\x00\x00\x03\x00\x00", []string(nil)},
		{"empty slice", "\x00\x00\x03\x00\x01", []string{}},
		{"map", "\x00\x00\x04\x00\x00\x00\x02\x01k\x01v", map[string]string{"k": "v"}},
		{"nil map", "\x00\x00\x04\x00\x00\x00\x00", map[string]string(nil)},
		{"empty map", "\x00\x00\x04\x00\x00\x00\x01", map[string]string{}},
		{"negative integer", "\x00\x00\x02\x00\x00\x00\xff", -1},
		{"bool", "\x00\x00\x01\x01", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := generateFuzzCase(tt.data)
			if !reflect.DeepEqual(c.in, tt.want) {
				t.Fatalf("got %#v, want %#v", c.in, tt.want)
			}
		})
	}
}

func TestFuzzGeneratorConcreteDestination(t *testing.T) {
	c := generateFuzzCase("\x03\x00\x05\x00\x01\x00\x03foo")
	if c.got.Elem().Kind() != reflect.Struct {
		t.Fatalf("expected concrete struct destination, got %v", c.got.Type())
	}
	encoded, err := json.Marshal(c.in)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, c.want.Interface()); err != nil {
		t.Fatal(err)
	}
	if got := c.want.Elem().Field(0).String(); got != "foo" {
		t.Fatalf("JSON did not populate the concrete destination: %q", got)
	}
	if got := c.got.Elem().Field(0).String(); got != "" {
		t.Fatalf("destinations share storage: %q", got)
	}
}

func TestFuzzGeneratorIndependentDestinations(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf([]string{}),
		reflect.TypeOf(map[string]string{}),
		reflect.TypeOf(new(string)),
		reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.TypeOf([]string{})}}),
	} {
		t.Run(typ.String(), func(t *testing.T) {
			g := fuzzGenerator{data: strings.Repeat("\x02", 128)}
			replay := g
			got, want := g.value(typ), replay.value(typ)
			initial, err := json.Marshal(got.Interface())
			if err != nil {
				t.Fatal(err)
			}
			switch want.Kind() {
			case reflect.Slice:
				want.Index(0).SetString("changed")
			case reflect.Map:
				want.SetMapIndex(reflect.ValueOf("changed"), reflect.ValueOf("changed"))
			case reflect.Ptr:
				want.Elem().SetString("changed")
			case reflect.Struct:
				want.Field(0).Index(0).SetString("changed")
			}
			after, err := json.Marshal(got.Interface())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(initial, after) {
				t.Fatalf("destinations share mutable storage: %s -> %s", initial, after)
			}
		})
	}
}
