package goloose

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

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
	ToStruct(a, &b, DefaultOptions())
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestToStructConvertsTypes(t *testing.T) {
	a := []int{1, 2, 3}
	var b, c []float64
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
	var d, e map[string]interface{}
	ToStruct(a, &d, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
	var d, e map[string]interface{}
	ToStruct(a, &d, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
	var d, e map[string]interface{}
	ToStruct(a, &d, DefaultOptions())
	toStructSlow(a, &e)
	if !reflect.DeepEqual(d, e) {
		t.Errorf("Got %v\nExpected %v", d, e)
	}
}

func TestToStructEmptyMap(t *testing.T) {
	var emptyMap map[string]interface{}
	a := map[string]interface{}{"a": map[string]interface{}{}, "b": emptyMap}
	var b, c map[string]interface{}
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(a, &b, DefaultOptions())
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
	ToStruct(m, &a, DefaultOptions())
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
	ToStruct(m, &a, DefaultOptions())
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}

func TestEmbeddedNilPointer(t *testing.T) {
	type Bar struct {
		Baz string
	}
	type Foo struct {
		*Bar
	}
	var m Foo
	var a, b Foo
	errA := ToStruct(m, &a, DefaultOptions())
	errB := toStructSlow(m, &b)
	if errA != errB {
		t.Errorf("Got err %v, expected %v", errA, errB)
	}
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
	aErr := ToStruct(m, &a, DefaultOptions())
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
	ToStruct(m, &a, DefaultOptions())
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
	ToStruct(m, &a, DefaultOptions())
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}

func TestEmbeddedFunc(t *testing.T) {
	type Foo struct {
		Bar string     `json:"bar"`
		Fn  func() int `json:"-"`
	}
	fn := func() int { return 1 }
	var a, b Foo
	a.Fn = fn
	b.Fn = fn
	m := map[string]interface{}{"bar": "a", "Fn": "2"}
	ToStruct(m, &a, DefaultOptions())
	toStructSlow(m, &b)
	// verify neither func is nil
	if a.Fn() != 1 {
		t.Errorf("Expected 1, got %d", a.Fn())
	}
	if b.Fn() != 1 {
		t.Errorf("Expected 1, got %d", b.Fn())
	}
	if a.Bar != b.Bar {
		t.Errorf("Expected %s, got %s", b.Bar, a.Bar)
	}
}

func TestNilNondestructive(t *testing.T) {
	type Foo struct {
		Bar string `json:"bar"`
	}
	var a, b Foo
	a.Bar = "test"
	b.Bar = "test"

	var m map[string]interface{}
	ToStruct(m, &a, DefaultOptions())
	toStructSlow(m, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %v\nExpected: %v", a, b)
	}
}

func TestInterfaceArray(t *testing.T) {
	type Foo struct {
		Bar []interface{} `json:"bar"`
	}
	type X struct {
		X int `json:"x"`
	}
	a := Foo{Bar: []interface{}{X{X: 1}, 5}}
	m1 := map[string]interface{}{}
	m2 := map[string]interface{}{}
	ToStruct(a, &m1, DefaultOptions())
	toStructSlow(a, &m2)
	if !reflect.DeepEqual(m1, m2) {
		t.Errorf("Got %v\nExpected: %v", m1, m2)
	}
	_ = fmt.Println
}

func TestConvertTrueAndFalseStringsToBool(t *testing.T) {
	type foo struct {
		Bar string `json:"bar"`
		Baz bool   `json:"baz"`
		Qux *bool  `json:"qux"`
	}
	var a foo
	expected := foo{Bar: "true", Baz: true, Qux: boolPtr(true)}
	ToStruct(map[string]interface{}{"bar": "true", "baz": "true", "qux": "true"}, &a, DefaultOptions())
	if !reflect.DeepEqual(a, expected) {
		t.Errorf("Got %+v\nExpected: %+v", a, expected)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestErrorInMap(t *testing.T) {
	a := map[string]interface{}{"foo": fmt.Errorf("bar")}
	expected := map[string]interface{}{"foo": "bar"}
	var b map[string]interface{}
	ToStructWithTransforms(a, &b, []TransformFunc{convertErrors}, DefaultOptions())
	if !reflect.DeepEqual(expected, b) {
		t.Errorf("Got %+v\nExpected: %+v", b, expected)
	}
}

func convertErrors(in interface{}) interface{} {
	err, ok := in.(error)
	if ok {
		return err.Error()
	}
	return in
}

func TestOverwriteBoolPtrWithNil(t *testing.T) {
	type Foo struct {
		B *bool `json:"b"`
	}
	x := Foo{boolPtr(false)}
	y := Foo{boolPtr(false)}
	msg := map[string]interface{}{"b": nil}
	ToStruct(msg, &x, DefaultOptions())
	toStructSlow(msg, &y)
	if !reflect.DeepEqual(x, y) {
		t.Errorf("Got %+v\nExpected: %+v", x, y)
	}
}

func TestWriteNilWithTransforms(t *testing.T) {
	type Foo struct {
		B *bool `json:"b"`
	}
	msg := map[string]interface{}{"b": nil}
	var x, y Foo
	ToStructWithTransforms(msg, &x, []TransformFunc{nilTransform}, DefaultOptions())
	toStructSlow(msg, &y)
	if !reflect.DeepEqual(x, y) {
		t.Errorf("Got %+v\nExpected: %+v", x, y)
	}
}

func nilTransform(in interface{}) interface{} {
	return in
}

func TestSetExistingInterfaceInSlice(t *testing.T) {
	type Foo struct {
		Val interface{} `json:"val"`
	}
	type Bar struct {
		Foo []Foo `json:"foo"`
	}
	msg := map[string]interface{}{"foo": []map[string]interface{}{{"val": "floob"}}}
	x := Bar{Foo: []Foo{{Val: "qux"}}}
	y := Bar{Foo: []Foo{{Val: "qux"}}}
	ToStruct(msg, &x, DefaultOptions())
	toStructSlow(msg, &y)
	if !reflect.DeepEqual(x, y) {
		t.Errorf("Got %+v\nExpected: %+v", x, y)
	}
}

func TestStringToFloat64(t *testing.T) {
	// note: all tests also confirm that we don't effect strings containing only numbers

	type Foo struct {
		FloatVal  float64 `json:"float_val"`
		StringVal string  `json:"string_val"`
	}
	var x, y, z Foo

	// test 1: default options don't convert string
	msg := map[string]interface{}{
		"float_val":  "3.14159",
		"string_val": "42",
	}
	expected := Foo{
		FloatVal:  0,
		StringVal: "42",
	}

	err := ToStruct(msg, &x, DefaultOptions())
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(x, expected) {
		t.Errorf("\ndefault options shouldn't convert strings to floats\nexpected: %v\nreceived: %v\n", expected, x)
	}

	// test 2: false doesn't convert string
	msg = map[string]interface{}{
		"float_val":  "3.14159",
		"string_val": "42",
	}
	expected = Foo{
		FloatVal:  0,
		StringVal: "42",
	}

	err = ToStruct(msg, &y, Options{StringToFloat64: false})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(y, expected) {
		t.Errorf("\nstringToFloat64 false shouldn't convert strings to floats\nexpected: %v\nreceived: %v\n", expected, x)
	}

	// test 3: true converts string
	msg = map[string]interface{}{
		"float_val":  "3.14159",
		"string_val": "42",
	}
	expected = Foo{
		FloatVal:  3.14159,
		StringVal: "42",
	}

	err = ToStruct(msg, &z, Options{StringToFloat64: true})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(z, expected) {
		t.Errorf("\nstringToFloat64 true should convert strings to floats\nexpected: %v\nreceived: %v\n", expected, x)
	}
}
