package goloose

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
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

func TestConvertTo(t *testing.T) {
	a := []int{1, 2, 3}
	got, err := ConvertTo[[]float64](a)
	if err != nil {
		t.Fatal(err)
	}
	exp := []float64{1, 2, 3}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Got %v\nExpected %v", got, exp)
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

func TestEmbeddedNilPointer(t *testing.T) {
	type Bar struct {
		Baz string
	}
	type Foo struct {
		*Bar
	}
	var m Foo
	var a, b Foo
	errA := ToStruct(m, &a)
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
	ToStruct(m, &a)
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
	ToStruct(m, &a)
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
	ToStruct(a, &m1)
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
	ToStruct(map[string]interface{}{"bar": "true", "baz": "true", "qux": "true"}, &a)
	if !reflect.DeepEqual(a, expected) {
		t.Errorf("Got %+v\nExpected: %+v", a, expected)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestBoolCapitalization(t *testing.T) {
	type foo struct {
		Bar bool `json:"bar"`
		Baz bool `json:"baz"`
		Qux bool `json:"qux"`
	}
	var a foo
	expected := foo{Bar: true, Baz: true, Qux: true}
	ToStruct(map[string]interface{}{"bar": "true", "baz": "TRUE", "qux": "tRuE"}, &a)
	if !reflect.DeepEqual(a, expected) {
		t.Errorf("Got %+v\nExpected: %+v", a, expected)
	}
}

func TestErrorInMap(t *testing.T) {
	a := map[string]interface{}{"foo": fmt.Errorf("bar")}
	expected := map[string]interface{}{"foo": "bar"}
	var b map[string]interface{}
	ToStruct(a, &b, Options{Transforms: []TransformFunc{convertErrors}})
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

func BenchmarkToStructSlow(b *testing.B) {
	type Foo struct {
		A string `json:"a"`
		B string `json:"b"`
		C string `json:"c"`
	}
	type Foo2 struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	f := Foo{A: "some a", B: "some b", C: "some C"}
	for i := 0; i < b.N; i++ {
		var foo2 Foo2
		_ = toStructSlow(f, &foo2)
		_ = foo2
	}
}

func BenchmarkToStruct(b *testing.B) {
	type Foo struct {
		A string `json:"a"`
		B string `json:"b"`
		C string `json:"c"`
	}
	type Foo2 struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	f := Foo{A: "some a", B: "some b", C: "some C"}
	for i := 0; i < b.N; i++ {
		var foo2 Foo2
		_ = ToStruct(f, &foo2)
		_ = foo2
	}
}

func BenchmarkStringMapToMapAny(b *testing.B) {
	in := map[string]string{}
	for x := 0; x < 1000; x++ {
		s := strconv.Itoa(x)
		in[s] = s
	}
	for i := 0; i < b.N; i++ {
		var out map[string]any
		if err := ToStruct(in, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func TestStringMapToMapAny(t *testing.T) {
	in := map[string]string{}
	for x := 0; x < 1000; x++ {
		s := strconv.Itoa(x)
		in[s] = s
	}
	var out, outSlow map[string]any
	if err := ToStruct(in, &out); err != nil {
		t.Fatal(err)
	}
	toStructSlow(in, &outSlow)
	if !reflect.DeepEqual(out, outSlow) {
		t.Errorf("Got %+v\nExpected: %+v", out, outSlow)
	}
}

func TestNestedMap(t *testing.T) {
	in := map[string]any{
		"foo": map[string]int{"a": 1},
		"bar": map[string]string{"b": "c"},
	}
	var out, outSlow map[string]any
	if err := ToStruct(in, &out); err != nil {
		t.Fatal(err)
	}
	toStructSlow(in, &outSlow)
	if !reflect.DeepEqual(out, outSlow) {
		t.Errorf("Got %+v\nExpected: %+v", out, outSlow)
	}
}

func BenchmarkNestedMap(b *testing.B) {
	bigMap := map[string]string{}
	for x := 0; x < 1000; x++ {
		s := strconv.Itoa(x)
		bigMap[s] = s
	}
	in := map[string]any{
		"big": bigMap,
		"foo": map[string]int{"a": 1},
		"bar": map[string]string{"b": "c"},
	}
	for i := 0; i < b.N; i++ {
		var out map[string]any
		if err := ToStruct(in, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMapToStructLarge(b *testing.B) {
	type Big struct {
		F0  int `json:"f0"`
		F1  int `json:"f1"`
		F2  int `json:"f2"`
		F3  int `json:"f3"`
		F4  int `json:"f4"`
		F5  int `json:"f5"`
		F6  int `json:"f6"`
		F7  int `json:"f7"`
		F8  int `json:"f8"`
		F9  int `json:"f9"`
		F10 int `json:"f10"`
		F11 int `json:"f11"`
		F12 int `json:"f12"`
		F13 int `json:"f13"`
		F14 int `json:"f14"`
		F15 int `json:"f15"`
		F16 int `json:"f16"`
		F17 int `json:"f17"`
		F18 int `json:"f18"`
		F19 int `json:"f19"`
		F20 int `json:"f20"`
		F21 int `json:"f21"`
		F22 int `json:"f22"`
		F23 int `json:"f23"`
		F24 int `json:"f24"`
		F25 int `json:"f25"`
		F26 int `json:"f26"`
		F27 int `json:"f27"`
		F28 int `json:"f28"`
		F29 int `json:"f29"`
		F30 int `json:"f30"`
		F31 int `json:"f31"`
	}
	in := make(map[string]any, 32)
	for i := 0; i < 32; i++ {
		in["f"+strconv.Itoa(i)] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out Big
		if err := ToStruct(in, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func TestStringMapToMapAnyWithTransforms(t *testing.T) {
	in := map[string]string{"a": "foo", "b": "bar"}
	var out map[string]any
	if err := ToStruct(in, &out, Options{Transforms: []TransformFunc{func(i any) any {
		if str, ok := i.(string); ok {
			return str + "baz"
		}
		return i
	}}}); err != nil {
		t.Fatal(err)
	}
	exp := map[string]any{"a": "foobaz", "b": "barbaz"}

	if !reflect.DeepEqual(out, exp) {
		t.Errorf("Got %+v\nExpected: %+v", out, exp)
	}

	// validate we can return a whole different type in our TransformFunc
	out = nil
	if err := ToStruct(in, &out, Options{Transforms: []TransformFunc{func(i any) any {
		if _, ok := i.(string); ok {
			return 3
		}
		return i
	}}}); err != nil {
		t.Fatal(err)
	}
	exp = map[string]any{"a": 3, "b": 3}

	if !reflect.DeepEqual(out, exp) {
		t.Errorf("Got %+v\nExpected: %+v", out, exp)
	}
}

func TestStringIntMapToMapAny(t *testing.T) {
	in := map[string]int{}
	for x := 0; x < 5; x++ {
		s := strconv.Itoa(x)
		in[s] = x
	}
	var out, outSlow map[string]any
	if err := ToStruct(in, &out); err != nil {
		t.Fatal(err)
	}
	toStructSlow(in, &outSlow)
	if !reflect.DeepEqual(out, outSlow) {
		t.Errorf("Got %+v\nExpected: %+v", out, outSlow)
	}
}

func TestStringIntMapToPopulatedMap(t *testing.T) {
	in := map[string]int{"bar": 1, "baz": 2}
	out := map[string]any{"existing": "foo", "baz": "3"}
	outSlow := map[string]any{"existing": "foo", "baz": "3"}
	if err := ToStruct(in, &out); err != nil {
		t.Fatal(err)
	}
	toStructSlow(in, &outSlow)
	if !reflect.DeepEqual(out, outSlow) {
		t.Errorf("Got %+v\nExpected: %+v", out, outSlow)
	}
}

func TestNilInputFastPath(t *testing.T) {
	var in map[string]int
	var out, outSlow map[string]any
	if err := ToStruct(in, &out); err != nil {
		t.Fatal(err)
	}
	toStructSlow(in, &outSlow)
	if !reflect.DeepEqual(out, outSlow) {
		t.Errorf("Got %+v\nExpected: %+v", out, outSlow)
	}
}

func TestOverwriteBoolPtrWithNil(t *testing.T) {
	type Foo struct {
		B *bool `json:"b"`
	}
	x := Foo{boolPtr(false)}
	y := Foo{boolPtr(false)}
	msg := map[string]interface{}{"b": nil}
	ToStruct(msg, &x)
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
	ToStruct(msg, &x, Options{Transforms: []TransformFunc{nilTransform}})
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
	ToStruct(msg, &x)
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

	err := ToStruct(msg, &x)
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

func TestStringToFloat64ConvertsAliases(t *testing.T) {
	type MyFloat float64
	type Foo struct {
		A MyFloat `json:"a"`
	}
	expected := Foo{A: 3.14159}
	foo := Foo{}
	msg := map[string]interface{}{"a": "3.14159"}
	err := ToStruct(msg, &foo, Options{StringToFloat64: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(foo, expected) {
		t.Errorf("\nstringToFloat64 true should convert strings to floats\nexpected: %v\nreceived: %v\n", expected, foo)
	}
}

func TestConvertPtrToNilStruct(t *testing.T) {
	type MyStruct struct {
		A int `json:"a"`
	}
	var nilStruct *MyStruct
	a := struct {
		Struct **MyStruct `json:"struct"`
	}{Struct: &nilStruct}
	var b, c map[string]interface{}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestConvertPtrToNilPtr(t *testing.T) {
	var nilInt *int
	a := struct {
		MyInt **int `json:"my_int"`
	}{MyInt: &nilInt}
	var b, c map[string]interface{}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestConvertPtrToNilSlice(t *testing.T) {
	var nilInt []int
	a := struct {
		MyInt *[]int `json:"my_int"`
	}{MyInt: &nilInt}
	var b, c map[string]interface{}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

func TestConvertPtrToNilMap(t *testing.T) {
	var nilMap map[string]interface{}
	a := struct {
		MyMap *map[string]interface{} `json:"my_map"`
	}{MyMap: &nilMap}
	var b, c map[string]interface{}
	ToStruct(a, &b)
	toStructSlow(a, &c)
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %v\nExpected %v", b, c)
	}
}

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func (d Date) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)), nil
}
func (d *Date) UnmarshalText(data []byte) error {
	t, err := time.Parse("2006-01-02", string(data))
	if err != nil {
		return err
	}
	d.Year, d.Month, d.Day = t.Date()
	return nil
}

func TestCustomTextMarshaler(t *testing.T) {
	type Foo struct {
		Date Date `json:"date"`
	}
	var a, b Foo
	m := map[string]interface{}{
		"date": "2001-01-01",
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

func TestRecursiveDataStructureDoesntPanic(t *testing.T) {
	type RecursiveStructure struct {
		Recurse *RecursiveStructure
	}
	r := &RecursiveStructure{}
	r.Recurse = r
	var m map[string]any
	if err := ToStruct(r, &m); err == nil {
		t.Error("Expected an error when converting a recursive structure, got none")
	}
}

func TestHugeNumber(t *testing.T) {
	in := struct{ HugeNumber *big.Int }{}
	in.HugeNumber = big.NewInt(1)
	in.HugeNumber, _ = in.HugeNumber.SetString("29157745120982650805953014097632160709947782099779208142451183556177980040665409506504745020886944175204497974977637090690201434791993575900128193797677872701163698274430343501746642052167702180396051675274748307985868208114311186980979298113930938344249665181686754463642448656385800073060075432884342393421100728585824378649245432727133022814294795072203332739083239001036325769762967358033340745798274757230450279572826675736895665366767754201612686011259638464597131009700878517633874689926429545413261363161426209942598516697451476197397103444680968659364909854234919768224131961929083663418258486694435258573213", 10)
	var a, b map[string]any
	err := ToStruct(in, &a)
	err2 := toStructSlow(in, &b)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %+v\nExpected %+v", a, b)
	}
}

func TestHugeNumberInStruct(t *testing.T) {
	in := struct{ HugeNumber *big.Int }{}
	type outType struct{ HugeNumber *big.Int }

	in.HugeNumber = big.NewInt(1)
	in.HugeNumber, _ = in.HugeNumber.SetString("29157745120982650805953014097632160709947782099779208142451183556177980040665409506504745020886944175204497974977637090690201434791993575900128193797677872701163698274430343501746642052167702180396051675274748307985868208114311186980979298113930938344249665181686754463642448656385800073060075432884342393421100728585824378649245432727133022814294795072203332739083239001036325769762967358033340745798274757230450279572826675736895665366767754201612686011259638464597131009700878517633874689926429545413261363161426209942598516697451476197397103444680968659364909854234919768224131961929083663418258486694435258573213", 10)
	var a, b outType
	err := ToStruct(in, &a)
	err2 := toStructSlow(in, &b)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %+v\nExpected %+v", a, b)
	}
}

func TestNonStringKeysInMaps(t *testing.T) {
	type Foo struct {
		A map[any]any `json:"a"`
	}
	a := Foo{A: map[any]any{"foo": "bar", 1: "baz"}}
	var b, c map[string]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

type cantMarshal struct{}

func (cantMarshal) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("custom error")
}

type cantUnmarshal struct{}

func (cantUnmarshal) UnmarshalJSON(data []byte) error {
	return fmt.Errorf("custom error")
}

func TestCustomJsonMarshalerReturningError(t *testing.T) {
	type CustomMarshaler struct {
		Foo cantMarshal `json:"foo"`
	}
	var a CustomMarshaler
	var b, c map[string]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestCustomJsonMarshalerReturningErrorInMap(t *testing.T) {
	a := map[string]any{"foo": cantMarshal{}}
	var b, c map[string]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestCustomJsonUnmarshalerReturningError(t *testing.T) {
	type CustomMarshaler struct {
		Foo cantUnmarshal `json:"foo"`
	}
	var a CustomMarshaler
	var b, c map[string]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestCustomJsonUnmarshalerReturningErrorInMap(t *testing.T) {
	a := map[string]any{"foo": cantUnmarshal{}}
	var b, c map[string]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestIntMapToMapIntAny(t *testing.T) {
	a := map[int]string{1: "foo"}
	var b, c map[int]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestIntMapToMapStringAny(t *testing.T) {
	a := map[int]string{1: "foo"}
	var b, c map[string]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestStringMapToMapIntAnyWithNumericKeys(t *testing.T) {
	a := map[string]string{"1": "foo", "2": "bar"}
	var b, c map[int]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestStringMapToMapIntAnyWithMixedNumericAndNonNumericKeys(t *testing.T) {
	a := map[string]string{"1": "foo", "bar": "baz"}
	var b, c map[int]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestMapAnyAnyToMapIntAny(t *testing.T) {
	a := map[any]any{"1": "foo", 2: "baz"}
	var b, c map[int]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestMapAnyAnyToMapAnyAny(t *testing.T) {
	a := map[any]any{"1": "foo", 2: "baz"}
	var b, c map[any]any
	err := ToStruct(a, &b)
	err2 := toStructSlow(a, &c)
	if (err != nil) != (err2 != nil) {
		t.Errorf("Got %+v\nExpected: %+v", err, err2)
	}
	if !reflect.DeepEqual(b, c) {
		t.Errorf("Got %+v\nExpected %+v", b, c)
	}
}

func TestToStructRejectsInvalidArguments(t *testing.T) {
	for _, out := range []any{nil, 0, (*int)(nil)} {
		if err := ToStruct(1, out); err == nil || !strings.Contains(err.Error(), "non-pointer") {
			t.Fatalf("out %T: %v", out, err)
		}
	}
	var out int
	if err := ToStruct(1, &out, Options{}, Options{}); err == nil || !strings.Contains(err.Error(), "at most one") {
		t.Fatalf("got %v", err)
	}
}

func TestFloatMapFastPaths(t *testing.T) {
	in := map[string]float64{"a": 1.25, "b": -2}
	var got, want any
	checkJSONConversion(t, in, &got, &want)
	var gotMap, wantMap map[string]any
	checkJSONConversion(t, in, &gotMap, &wantMap)
	gotMap, wantMap = map[string]any{"keep": true}, map[string]any{"keep": true}
	checkJSONConversion(t, in, &gotMap, &wantMap)
	for _, out := range []any{(*any)(nil), (*map[string]any)(nil)} {
		if fastPathMapStringAny(in, out, Options{}) {
			t.Fatalf("nil destination %T was handled", out)
		}
	}
}

func TestUnsupportedMapKeys(t *testing.T) {
	for _, in := range []any{map[bool]int{true: 1}, map[any]int{nil: 1}, map[any]int{true: 1}} {
		var out map[string]int
		var unsupported *json.UnsupportedTypeError
		if err := ToStruct(in, &out); !errors.As(err, &unsupported) {
			t.Fatalf("input %T: got %v", in, err)
		}
		if out != nil {
			t.Fatalf("unsupported key inserted: %v", out)
		}
	}
}

func TestNumericMapKeys(t *testing.T) {
	for _, in := range []any{map[uint]int{12: 3}, map[any]int{uint(12): 3}} {
		var got map[string]int
		if err := ToStruct(in, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, map[string]int{"12": 3}) {
			t.Fatalf("got %v", got)
		}
	}
	var got, want map[uint8]int
	checkJSONConversion(t, map[string]int{"0": 1, "12": 2, "255": 3}, &got, &want)
	for _, key := range []string{"256", "-1", "invalid"} {
		var out map[uint8]int
		var typeErr *json.UnmarshalTypeError
		if err := ToStruct(map[string]int{key: 1}, &out); !errors.As(err, &typeErr) {
			t.Fatalf("key %q: %v", key, err)
		}
		if len(out) != 0 {
			t.Fatalf("invalid key inserted: %v", out)
		}
	}
	for _, out := range []any{new(map[fmt.Stringer]int), new(map[bool]int)} {
		var typeErr *json.UnmarshalTypeError
		if err := ToStruct(map[string]int{"1": 2}, out); !errors.As(err, &typeErr) {
			t.Fatalf("out %T: %v", out, err)
		}
	}
}

func TestConversionErrorsPropagate(t *testing.T) {
	cases := []struct {
		name    string
		in, out any
	}{
		{"base64", "%%%", new([]byte)},
		{"quoted input", struct {
			Value float64 `json:",string"`
		}{math.Inf(1)}, new(map[string]any)},
		{"struct field", struct{ Value cantMarshal }{}, new(struct{ Value int })},
		{"slice element", []cantMarshal{{}}, new([]int)},
		{"quoted marshaler", map[string]any{"Value": cantMarshal{}}, new(struct {
			Value int `json:",string"`
		})},
		{"quoted nonstring", map[string]any{"Value": 123}, new(struct {
			Value int `json:",string"`
		})},
		{"quoted invalid literal", map[string]any{"Value": "invalid"}, new(struct {
			Value int `json:",string"`
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ToStruct(tc.in, tc.out); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

type nullJSONValue int

func (nullJSONValue) MarshalJSON() ([]byte, error) { return []byte("null"), nil }

func TestQuotedNull(t *testing.T) {
	for _, value := range []any{"null", nullJSONValue(1), nil} {
		in := map[string]any{"Value": value}
		got, want := struct {
			Value int `json:",string"`
		}{42}, struct {
			Value int `json:",string"`
		}{42}
		checkJSONConversion(t, in, &got, &want)
	}
}

func TestStructInterfaceFields(t *testing.T) {
	in := struct {
		Value any
		Null  any
	}{Value: "hello"}
	var got, want map[string]any
	checkJSONConversion(t, in, &got, &want)
}

func TestMismatchedContainersLeaveScalarUnchanged(t *testing.T) {
	for _, in := range []any{struct{ Value int }{1}, map[string]int{"value": 1}, []int{1}} {
		out := 42
		if err := ToStruct(in, &out); err != nil {
			t.Fatal(err)
		}
		if out != 42 {
			t.Fatalf("%T changed destination: %v", in, out)
		}
	}
}

func TestUnsupportedKinds(t *testing.T) {
	for _, tc := range []struct {
		in      any
		message string
	}{
		{[1]int{1}, "Array not supported yet!"},
		{unsafe.Pointer(new(int)), "UnsafePointer not supported!"},
	} {
		t.Run(tc.message, func(t *testing.T) {
			defer func() {
				if got := recover(); got != tc.message {
					t.Fatalf("panic = %v, want %q", got, tc.message)
				}
			}()
			var out int
			_ = ToStruct(tc.in, &out)
		})
	}
	for _, in := range []any{make(chan int), func() {}} {
		out := 42
		if err := ToStruct(in, &out); err != nil || out != 42 {
			t.Fatalf("%T: out %v, err %v", in, out, err)
		}
	}
}

func TestNamedStringMapNormalization(t *testing.T) {
	type key string
	type namedMap map[key]int
	for _, in := range []any{namedMap{"\xff": 2, "\xfe": 1}, map[string]float64{"\xff": 2}} {
		var got, want map[string]any
		checkJSONConversion(t, in, &got, &want)
	}
}

func TestFalseStringToBool(t *testing.T) {
	out := true
	if err := ToStruct("FaLsE", &out); err != nil || out {
		t.Fatalf("got %v, err %v", out, err)
	}
}

func TestSkipValueError(t *testing.T) {
	cause := errors.New("failed to encode value")
	err := &skipValError{err: cause}
	if err.Error() != cause.Error() || !errors.Is(err, cause) {
		t.Fatalf("wrapper lost cause: %v", err)
	}
}

func TestUnexportedInputValueIsIgnored(t *testing.T) {
	in := reflect.ValueOf(struct{ hidden int }{42}).Field(0)
	out := 3
	if err := toStructImpl(in, reflect.ValueOf(&out).Elem(), Options{}, 0); err != nil || out != 3 {
		t.Fatalf("got %v, err %v", out, err)
	}
}

func TestNullAndEmptyHelpers(t *testing.T) {
	if !isJSONNull(reflect.Value{}) {
		t.Fatal("invalid value should represent null")
	}
	out := 42
	if err := toStructSlow(nil, &out); err != nil || out != 42 {
		t.Fatalf("got %v, err %v", out, err)
	}
	for _, tc := range []struct {
		v     any
		empty bool
	}{
		{false, true}, {true, false}, {int64(0), true}, {int64(-1), false},
		{uint64(0), true}, {uint64(1), false}, {float64(0), true}, {float64(1), false},
		{(*int)(nil), true}, {new(int), false}, {struct{}{}, false},
	} {
		if got := isEmptyValue(reflect.ValueOf(tc.v)); got != tc.empty {
			t.Fatalf("%#v: empty %v, want %v", tc.v, got, tc.empty)
		}
	}
}

func TestInterfaceRecursionLimitPreservesDestination(t *testing.T) {
	var out any
	err := toStructImpl(reflect.ValueOf(42), reflect.ValueOf(&out).Elem(), Options{}, maxRecursionLevel)
	if err == nil || !strings.Contains(err.Error(), "maximum recursion level") {
		t.Fatalf("expected recursion limit error, got %v", err)
	}
	if out != nil {
		t.Fatalf("failed conversion changed destination: %v", out)
	}
}

type nilMapWithJSONMethod map[string]int

func (nilMapWithJSONMethod) MarshalJSON() ([]byte, error) { return []byte(`{"custom":42}`), nil }

func TestNilMapMarshalerTakesPrecedence(t *testing.T) {
	var in nilMapWithJSONMethod
	var got, want any
	checkJSONConversion(t, in, &got, &want)
	var gotMap, wantMap map[string]int
	checkJSONConversion(t, in, &gotMap, &wantMap)
}

func TestFieldIndexOrdering(t *testing.T) {
	prefix := byIndex{{index: []int{1}}, {index: []int{1, 2}}}
	if !prefix.Less(0, 1) || prefix.Less(1, 0) || prefix.Less(0, 0) {
		t.Fatal("a prefix must sort before its extensions, but not before itself")
	}
	fields := byIndex{{index: []int{1, 2}}, {index: []int{0, 1}}, {index: []int{1}}, {index: []int{0}}, {index: []int{1}}}
	sort.Sort(fields)
	want := [][]int{{0}, {0, 1}, {1}, {1}, {1, 2}}
	for i, f := range fields {
		if !reflect.DeepEqual(f.index, want[i]) {
			t.Fatalf("field %d: %v, want %v", i, f.index, want[i])
		}
	}
}

func TestFieldTagParsing(t *testing.T) {
	for _, tc := range []struct {
		tag   string
		valid bool
	}{
		{"", false}, {"field", true}, {"日本語", true}, {"a b!", true}, {"bad\\name", false}, {"bad\"name", false}, {"bad\nname", false},
	} {
		if got := isValidTag(tc.tag); got != tc.valid {
			t.Fatalf("%q: valid %v", tc.tag, got)
		}
	}
	name, opts := parseTag("value,omitempty,string")
	if name != "value" || !opts.Contains("omitempty") || !opts.Contains("string") || opts.Contains("omit") {
		t.Fatalf("got name %q, options %q", name, opts)
	}
	if fieldByIndex(reflect.Value{}, []int{0}, false).IsValid() {
		t.Fatal("invalid root should yield an invalid field")
	}
}

func TestEmbeddedFieldDominance(t *testing.T) {
	type Plain struct{ Value int }
	type Tagged struct {
		Other int `json:"Value"`
	}
	type Left struct{ Plain }
	type Right struct{ Plain }
	type Recursive struct {
		*Recursive
		Value int
	}
	for _, in := range []any{
		struct {
			Plain
			Value int
		}{Plain{1}, 2},
		struct {
			Plain
			Tagged
		}{Plain{1}, Tagged{2}},
		struct {
			Left
			Right
		}{Left{Plain{1}}, Right{Plain{2}}},
		struct {
			Plain
			Left
		}{Plain{1}, Left{Plain{2}}},
		Recursive{Value: 3},
	} {
		t.Run(reflect.TypeOf(in).String(), func(t *testing.T) {
			var got, want map[string]any
			checkJSONConversion(t, in, &got, &want)
		})
	}
}

func TestDominantFieldSelection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fields []field
		want   string
		ok     bool
	}{
		{"single", []field{{name: "first", index: []int{0}}}, "first", true},
		{"shallower", []field{{name: "first", index: []int{0}}, {name: "deep", index: []int{1, 0}}}, "first", true},
		{"tag wins", []field{{name: "plain", index: []int{0}}, {name: "tagged", index: []int{1}, tag: true}}, "tagged", true},
		{"plain conflict", []field{{index: []int{0}}, {index: []int{1}}}, "", false},
		{"tag conflict", []field{{index: []int{0}, tag: true}, {index: []int{1}, tag: true}}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := dominantField(tc.fields)
			if ok != tc.ok || got.name != tc.want {
				t.Fatalf("got %q, %v; want %q, %v", got.name, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestConcurrentFieldCache(t *testing.T) {
	// Large fresh types make concurrent callers compute and publish the same
	// metadata. All callers must observe identical complete entries.
	fields := make([]reflect.StructField, 256)
	for i := range fields {
		fields[i] = reflect.StructField{Name: "Field" + strconv.Itoa(i), Type: reflect.TypeFor[int]()}
	}
	typ := reflect.StructOf(fields)
	const workers = 32
	start := make(chan struct{})
	results := make(chan fieldCacheEntry, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- cachedTypeFieldsEntry(typ) }()
	}
	close(start)
	wg.Wait()
	close(results)
	for entry := range results {
		if len(entry.fields) != len(fields) || len(entry.byLower) != len(fields) {
			t.Fatalf("incomplete entry: %d fields, %d names", len(entry.fields), len(entry.byLower))
		}
	}
	if got := cachedTypeFieldsByLower(typ)["field0"]; !reflect.DeepEqual(got, []int{0}) {
		t.Fatalf("wrong cached field index: %v", got)
	}
}

func TestFieldCacheKeepsFirstEntry(t *testing.T) {
	type record struct {
		Value int `json:"stored"`
	}
	typ := reflect.TypeFor[record]()
	first := cachedTypeFieldsEntry(typ)
	// Simulate a second writer reaching publication after the first one. This
	// checks the lost-race path deterministically, even with GOMAXPROCS=1.
	got := storeTypeFieldsEntry(typ, fieldCacheEntry{})
	if !reflect.DeepEqual(got, first) || len(got.fields) != 1 {
		t.Fatalf("second writer replaced the published entry: %#v", got)
	}
	if !reflect.DeepEqual(cachedTypeFieldsEntry(typ), first) {
		t.Fatal("second writer changed the cached entry")
	}
}
