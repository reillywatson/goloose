package goloose

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
)

// Callers supply independent destinations, including any initial contents.
func checkJSONConversion(t *testing.T, in, got, want any) {
	t.Helper()
	encoded, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, want); err != nil {
		t.Fatal(err)
	}
	if err := ToStruct(in, got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("input: %#v\nJSON: %s\ngot: %#v\nwant: %#v", in, encoded,
			reflect.ValueOf(got).Elem().Interface(), reflect.ValueOf(want).Elem().Interface())
	}
}

func TestEmptyObjectToMap(t *testing.T) {
	for _, in := range []any{
		map[string]string{},
		map[int]string{},
		struct{}{},
		struct {
			Value string `json:",omitempty"`
		}{},
	} {
		t.Run(reflect.TypeOf(in).String(), func(t *testing.T) {
			var got, want map[string]string
			checkJSONConversion(t, in, &got, &want)
			got, want = map[string]string{"old": "value"}, map[string]string{"old": "value"}
			checkJSONConversion(t, in, &got, &want)
		})
	}
}

func TestNullConversion(t *testing.T) {
	var nilInterface any
	var nilPointer *string
	for _, in := range []any{nil, nilPointer, &nilPointer, &nilInterface, map[string]string(nil), []string(nil)} {
		t.Run(fmt.Sprintf("%T", in), func(t *testing.T) {
			gotString, wantString := "keep", "keep"
			checkJSONConversion(t, in, &gotString, &wantString)
			gotMap, wantMap := map[string]string{"old": "value"}, map[string]string{"old": "value"}
			checkJSONConversion(t, in, &gotMap, &wantMap)
			var got, want *any
			checkJSONConversion(t, in, &got, &want)
			got, want = new(any), new(any)
			checkJSONConversion(t, in, &got, &want)
		})
	}
}

func TestByteSliceJSONConversion(t *testing.T) {
	for _, in := range [][]byte{nil, {}, {0, 128, 255}, []byte("hello")} {
		var gotString, wantString string
		checkJSONConversion(t, in, &gotString, &wantString)
		var gotAny, wantAny any
		checkJSONConversion(t, in, &gotAny, &wantAny)
		gotAny, wantAny = nil, nil
		checkJSONConversion(t, &in, &gotAny, &wantAny)
		var gotBytes, wantBytes []byte
		checkJSONConversion(t, in, &gotBytes, &wantBytes)
		checkJSONConversion(t, wantString, &gotBytes, &wantBytes)
	}
}

func TestQuotedFieldJSONConversion(t *testing.T) {
	type quoted struct {
		Text   string `json:"text,string"`
		Number int64  `json:"number,string"`
		Bool   bool   `json:"bool,string"`
	}
	in := quoted{Text: "a\n\"b\\c", Number: 9223372036854775807, Bool: true}
	var gotMap, wantMap map[string]any
	checkJSONConversion(t, in, &gotMap, &wantMap)
	var got, want quoted
	checkJSONConversion(t, in, &got, &want)
	checkJSONConversion(t, wantMap, &got, &want)
}

func TestJSONNumericPrecision(t *testing.T) {
	for _, in := range []float32{0.1, math.SmallestNonzeroFloat32, math.MaxFloat32} {
		var got, want float64
		checkJSONConversion(t, in, &got, &want)
	}
	var got, want map[string]int64
	checkJSONConversion(t, map[string]int64{"max": math.MaxInt64, "min": math.MinInt64}, &got, &want)
	var gotUint, wantUint map[string]uint64
	checkJSONConversion(t, map[string]uint64{"max": math.MaxUint64}, &gotUint, &wantUint)
	var gotLarge, wantLarge uint64
	checkJSONConversion(t, float64(1<<63), &gotLarge, &wantLarge)
	var gotInt, wantInt int64
	checkJSONConversion(t, float32(1e18), &gotInt, &wantInt)
}

func TestJSONSliceReusesElements(t *testing.T) {
	for _, length := range []int{1, 3} {
		got, want := make([]map[string]int, length), make([]map[string]int, length)
		got[0], want[0] = map[string]int{"old": 1}, map[string]int{"old": 1}
		checkJSONConversion(t, []map[string]int{{"new": 2}, {"new": 3}}, &got, &want)
	}
}

func TestJSONMapKeyUTF8(t *testing.T) {
	in := map[string]string{"\xff": "last", "\xfe": "middle", "�": "first", "\xff\xff": "two"}
	var got, want map[string]string
	checkJSONConversion(t, in, &got, &want)
	var gotAny, wantAny map[string]any
	checkJSONConversion(t, in, &gotAny, &wantAny)
}

func TestJSONMapKeyCollision(t *testing.T) {
	in := map[string]int{"0": 1, "00": 2, "000": 3}
	for i := 0; i < 100; i++ {
		var got, want map[int]int
		checkJSONConversion(t, in, &got, &want)
	}
}

func TestPointerInterfaceCycle(t *testing.T) {
	var in any
	in = &in
	var out any
	if err := ToStruct(in, &out); err == nil {
		t.Fatal("expected a recursion error for a pointer/interface cycle")
	}
}

type conformanceString string

func (conformanceString) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type conformanceByte byte

func (conformanceByte) MarshalJSON() ([]byte, error) { return []byte(`"byte"`), nil }

type conformanceQuotedString string

func (conformanceQuotedString) MarshalJSON() ([]byte, error) {
	return json.Marshal(`"decoded"`)
}

func TestJSONMarshalerTakesPrecedence(t *testing.T) {
	in := struct {
		Value conformanceString `json:",string"`
	}{"input"}
	var got, want any
	checkJSONConversion(t, in, &got, &want)
	got, want = nil, nil
	checkJSONConversion(t, []conformanceByte{1, 2}, &got, &want)
	type quoted struct {
		Value string `json:",string"`
	}
	var gotQuoted, wantQuoted quoted
	checkJSONConversion(t, map[string]any{"Value": conformanceQuotedString("input")}, &gotQuoted, &wantQuoted)
}
