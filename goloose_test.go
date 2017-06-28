package goloose

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// slow version to compare against
func toStructSlow(in interface{}, out interface{}) error {
	if in == nil {
		return nil
	}
	tmp, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(tmp, &out)
}

func toJson(in interface{}) string {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestToStruct(t *testing.T) {
	a := map[string]interface{}{
		"a": map[string]interface{}{"b": 1},
		"b": struct {
			A string `json:"a"`
			B string `json:"b"`
		}{"1", "2"},
	}
	type bar struct {
		B int `json:"b"`
	}
	type foo struct {
		A bar `json:"a"`
	}
	var b foo
	var c foo
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructConvertsTypes(t *testing.T) {
	a := []int{1, 2, 3}
	var b, c []float64
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructIntToInterface(t *testing.T) {
	type foo struct {
		Dur int `json:"dur"`
	}
	type bar struct {
		Dur interface{} `json:"dur"`
	}
	a := foo{6}
	var b, c bar
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestToStructConvertsTimes(t *testing.T) {
	type foo struct {
		T  time.Time  `json:"t"`
		T2 *time.Time `json:"t2"`
	}
	a := foo{time.Now().UTC(), timePtr(time.Now().UTC())}
	var b, c foo
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructTimeToString(t *testing.T) {
	type foo struct {
		A time.Time  `json:"a"`
		B *time.Time `json:"b"`
		C string     `json:"c"`
		D string     `json:"d"`
	}
	type bar struct {
		A string     `json:"a"`
		B string     `json:"b"`
		C time.Time  `json:"c"`
		D *time.Time `json:"d"`
	}
	a := foo{time.Now(), timePtr(time.Now()), time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339)}
	var b, c bar
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructZerosThingsOut(t *testing.T) {
	type foo struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	a := map[string]interface{}{"a": 1}
	b := foo{1, 1}
	c := foo{1, 1}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructIgnoresCase(t *testing.T) {
	type foo struct {
		A int `json:"ABC_DEF"`
	}
	a := map[string]interface{}{"aBc_DEf": 2}
	b := foo{1}
	c := foo{1}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructDuration(t *testing.T) {
	type foo struct {
		A time.Duration `json:"a"`
	}
	a := foo{1373663273332128183} // this is large enough that a float64 will lose some precision
	var b, c foo
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
	var d, e map[string]interface{}
	ToStruct(a, &d)
	toStructSlow(a, &e)
	if !reflect.DeepEqual(d, e) {
		t.Errorf("Got %v\nExpected %v", d, e)
	}
}

func TestToStructEmbeddedStruct(t *testing.T) {
	type bar struct {
		Baz int `json:"baz"`
	}
	type foo struct {
		Bar bar `json:"bar"`
	}
	a := foo{Bar: bar{Baz: 1}}
	var b map[string]interface{}
	ToStruct(a, &b)
	val := reflect.ValueOf(b["bar"])
	if val.Kind() != reflect.Map {
		t.Errorf("Expected map, got %v", val.Type())
	}
}

func TestToStructInterfaceSlice(t *testing.T) {
	a := map[string]interface{}{
		"a": []interface{}{
			map[string]interface{}{"b": []string{"1"}},
		},
	}
	type foo struct {
		A []struct {
			B []string `json:"b"`
		} `json:"a"`
	}
	var b, c foo
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
	var d, e map[string]interface{}
	ToStruct(a, &d)
	toStructSlow(a, &e)
	if !reflect.DeepEqual(d, e) {
		t.Errorf("Got %v\nExpected %v", d, e)
	}
}

func TestToStructUnexportedFields(t *testing.T) {
	type foo struct {
		A int `json:"a"`
		b int `json:"b"`
	}
	a := foo{A: 1, b: 2}
	var b, c foo
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructPointer(t *testing.T) {
	type bar struct {
		B string `json:"b"`
	}
	type foo struct {
		A *bar `json:"a"`
	}
	a := map[string]interface{}{"a": map[string]interface{}{"b": "b"}}
	var b, c foo
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
	var d, e map[string]interface{}
	ToStruct(a, &d)
	toStructSlow(a, &e)
	if !reflect.DeepEqual(d, e) {
		t.Errorf("Got %v\nExpected %v", d, e)
	}
}

func TestToStructEmptyMap(t *testing.T) {
	var emptyMap map[string]interface{}
	a := map[string]interface{}{"a": map[string]interface{}{}, "b": emptyMap}
	var b, c map[string]interface{}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestPointerToInt(t *testing.T) {
	type foo struct {
		A *int `json:"a"`
	}
	one := 1
	a := foo{A: &one}
	var b, c map[string]interface{}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestEmbeddedStructPtr(t *testing.T) {
	type Bar struct {
		Baz string
	}
	type Foo struct {
		*Bar
	}
	var a, b Foo
	m := map[string]interface{}{
		"Baz": "cancel",
	}
	ToStruct(m, &a)
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}

func TestEmbeddedStructPtrDoesntAllocAbsentFields(t *testing.T) {
	type Bar struct {
		Baz string
	}
	type Quux struct {
		A string
	}
	type Foo struct {
		*Quux
		*Bar
	}
	var a, b Foo
	m := map[string]interface{}{
		"Baz": "cancel",
	}
	ToStruct(m, &a)
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}

func TestInvalidTime(t *testing.T) {
	type Foo struct {
		Time time.Time `json:"time"`
	}
	var a, b Foo
	m := map[string]interface{}{
		"time": "badtime",
	}
	aErr := ToStruct(m, &a)
	bErr := toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
	if !reflect.DeepEqual(aErr, bErr) {
		t.Errorf("Got %v\nExpected: %v", aErr, bErr)
	}
}

func TestStringJSONString(t *testing.T) {
	type Foo struct {
		Bar string `json:"bar,string"`
	}
	var a, b Foo
	m := map[string]interface{}{
		"bar": "\"a\"",
	}
	ToStruct(m, &a)
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}

func TestStringJSONInt64(t *testing.T) {
	type Foo struct {
		Id int64 `json:"id,string"`
	}
	var a, b Foo
	m := map[string]interface{}{
		"id": "131412412412412412",
	}
	ToStruct(m, &a)
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}
