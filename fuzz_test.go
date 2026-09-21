package goloose

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
)

// FuzzGoloose treats its input as a recipe for Go types and values, not Go source.
// Every mutation produces a valid value, including empty and truncated inputs.
func FuzzGoloose(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		c := generateFuzzCase(data)
		encoded, err := json.Marshal(c.in)
		if err != nil {
			t.Fatalf("generator produced a non-JSON input: %v", err)
		}
		// Goloose deliberately accepts some conversions JSON rejects (e.g.
		// strings to bools). Compare the domain where JSON conversion succeeds.
		if err := json.Unmarshal(encoded, c.want.Interface()); err != nil {
			return
		}
		initial := fmt.Sprintf("%#v", c.got.Elem().Interface())
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("ToStruct panicked: %v\ninput (%T): %#v\nJSON: %s\ndestination: %v\ninitial: %s",
					p, c.in, c.in, encoded, c.got.Elem().Type(), initial)
			}
		}()
		if err := ToStruct(c.in, c.got.Interface()); err != nil {
			t.Fatalf("ToStruct failed: %v\ninput (%T): %#v\nJSON: %s\ndestination: %v\ninitial: %s",
				err, c.in, c.in, encoded, c.got.Elem().Type(), initial)
		}
		if !reflect.DeepEqual(c.got.Elem().Interface(), c.want.Elem().Interface()) {
			t.Fatalf("input (%T): %#v\nJSON: %s\ndestination: %v\ninitial: %s\ngot: %#v\nwant: %#v",
				c.in, c.in, encoded, c.got.Elem().Type(), initial, c.got.Elem().Interface(), c.want.Elem().Interface())
		}
	})
}

// The first two bytes select the conversion mode and whether to populate the
// destination. The rest select types, then input and destination values. Keep
// these seeds small so mutations and minimization can change individual choices.
var fuzzSeeds = []string{
	"",                                                     // string
	"\x00\x00\x01\x01",                                     // bool
	"\x00\x00\x02\x00\x01\x00\x2a",                         // int to int8
	"\x00\x00\x03\x00\x03\x01a\x01b",                       // []string
	"\x00\x00\x04\x00\x00\x00\x02\x01k\x01v",               // map[string]string
	"\x00\x00\x05\x00\x01\x00\x03foo",                      // tagged struct
	"\x01\x00\x05\x00\x01\x00\x03foo",                      // struct to any
	"\x02\x00\x05\x00\x01\x00\x03foo",                      // struct to map
	"\x03\x00\x05\x00\x01\x00\x03foo",                      // JSON map to struct
	"\x00\x00\x06\x00\x01\x01x",                            // *string
	"\x00\x00\x07\x01\x01x",                                // interface containing a string
	"\x00\x01\x00\x03new\x03old",                           // populated destination
	"\x00\x00\x05\x00\x01\x03\x02\x00\x0b\x02\x00\x2a",     // struct containing []int to []float64
	"\x03\x00\x05\x00\x01\x06\x00\x01\x03foo",              // JSON map to struct with *string
	"\x03\x00\x05\x00\x01\x05\x00\x01\x00\x03foo",          // JSON map to nested struct
	"\x00\x00\x03\x00\x00",                                 // nil slice
	"\x00\x00\x03\x00\x01",                                 // empty slice
	"\x01\x00\x05\x00\x02\x00",                             // omitempty
	"\x00\x00\x05\x00\x04\x01\x01",                         // bool with ,string
	"\x00\x01\x03\x00\x02\x01x\x02\x01y",                   // populated slice
	"\x00\x01\x04\x00\x00\x00\x02\x01k\x01v\x02\x01m\x01n", // populated map
	"\x00\x00\x02\x04\x04\x01",                             // max int64, without float64 precision loss
}

const (
	fuzzMaxDepth = 3
	fuzzMaxInput = 4096
)

type fuzzGenerator struct {
	data string
	pos  int
}

// Exhaustion supplies zeros instead of rejecting short inputs or reseeding a
// PRNG: nearby byte mutations should make nearby changes to a generated case.
func (g *fuzzGenerator) next() byte {
	if g.pos == len(g.data) {
		return 0
	}
	b := g.data[g.pos]
	g.pos++
	return b
}

func (g *fuzzGenerator) choose(n int) int { return int(g.next()) % n }

var fuzzNumbers = []reflect.Type{
	reflect.TypeOf(int(0)), reflect.TypeOf(int8(0)), reflect.TypeOf(int16(0)),
	reflect.TypeOf(int32(0)), reflect.TypeOf(int64(0)), reflect.TypeOf(uint(0)),
	reflect.TypeOf(uint8(0)), reflect.TypeOf(uint16(0)), reflect.TypeOf(uint32(0)),
	reflect.TypeOf(uint64(0)), reflect.TypeOf(float32(0)), reflect.TypeOf(float64(0)),
}

var fuzzMapKeys = []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(int(0)), reflect.TypeOf(uint(0))}
var fuzzAnyType = reflect.TypeFor[any]()

// Generate related types so fields match and most inputs can be converted by
// JSON. Numeric types can differ; containers compose recursively. Depth and
// width bounds limit type sizes. Fixed names/tags keep value mutations from
// creating unnecessary entries in reflection's permanent type cache.
func (g *fuzzGenerator) types(depth int) (reflect.Type, reflect.Type) {
	kinds := 8
	if depth >= fuzzMaxDepth {
		kinds = 3
	}
	switch g.choose(kinds) {
	case 0:
		return reflect.TypeOf(""), reflect.TypeOf("")
	case 1:
		return reflect.TypeOf(false), reflect.TypeOf(false)
	case 2:
		return fuzzNumbers[g.choose(len(fuzzNumbers))], fuzzNumbers[g.choose(len(fuzzNumbers))]
	case 3:
		in, out := g.types(depth + 1)
		return reflect.SliceOf(in), reflect.SliceOf(out)
	case 4:
		inKey, outKey := fuzzMapKeys[g.choose(len(fuzzMapKeys))], fuzzMapKeys[g.choose(len(fuzzMapKeys))]
		in, out := g.types(depth + 1)
		return reflect.MapOf(inKey, in), reflect.MapOf(outKey, out)
	case 5:
		inFields := make([]reflect.StructField, 1+g.choose(3))
		outFields := make([]reflect.StructField, len(inFields))
		for i := range inFields {
			name := fmt.Sprintf("F%d", i)
			jsonName := fmt.Sprintf("field%d", i)
			tags := []string{"", jsonName, jsonName + ",omitempty", "-", jsonName + ",string", "FIELD" + fmt.Sprint(i), "shared"}
			tag := g.choose(len(tags))
			in, out := g.types(depth + 1)
			field := reflect.StructField{Name: name, Type: in}
			if tag != 0 {
				field.Tag = reflect.StructTag(fmt.Sprintf("json:%q", tags[tag]))
			}
			inFields[i] = field
			field.Type = out
			outFields[i] = field
		}
		return reflect.StructOf(inFields), reflect.StructOf(outFields)
	case 6:
		in, out := g.types(depth + 1)
		return reflect.PointerTo(in), reflect.PointerTo(out)
	default:
		return fuzzAnyType, fuzzAnyType
	}
}

func (g *fuzzGenerator) value(typ reflect.Type) reflect.Value {
	v := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.String:
		n := g.choose(33)
		if n > len(g.data)-g.pos {
			n = len(g.data) - g.pos
		}
		v.SetString(g.data[g.pos : g.pos+n])
		g.pos += n
	case reflect.Bool:
		v.SetBool(g.choose(2) != 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var n int64
		switch g.choose(4) {
		case 0:
			n = int64(int8(g.next()))
		case 1:
			n = 1<<(typ.Bits()-1) - 1
		case 2:
			n = -1 << (typ.Bits() - 1)
		case 3:
			n = 0
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var n uint64
		switch g.choose(3) {
		case 0:
			n = uint64(g.next())
		case 1:
			n = 1<<typ.Bits() - 1
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		var n float64
		switch g.choose(5) {
		case 0:
			n = float64(int8(g.next()))
		case 1:
			n = float64(int8(g.next())) / 10
		case 2:
			n = math.Copysign(0, -1)
		case 3:
			n = math.SmallestNonzeroFloat32
		case 4:
			n = math.MaxFloat32
		}
		v.SetFloat(n)
	case reflect.Ptr:
		if g.choose(4) != 0 {
			v.Set(reflect.New(typ.Elem()))
			v.Elem().Set(g.value(typ.Elem()))
		}
	case reflect.Slice, reflect.Map:
		// Distinguish nil, empty, and 1-3 elements.
		n := g.choose(5)
		if n == 0 {
			return v
		}
		n--
		if typ.Kind() == reflect.Slice {
			v.Set(reflect.MakeSlice(typ, n, n))
			for i := 0; i < n; i++ {
				v.Index(i).Set(g.value(typ.Elem()))
			}
		} else {
			v.Set(reflect.MakeMapWithSize(typ, n))
			for i := 0; i < n; i++ {
				key := g.value(typ.Key())
				v.SetMapIndex(key, g.value(typ.Elem()))
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			v.Field(i).Set(g.value(typ.Field(i).Type))
		}
	case reflect.Interface:
		// A finite set of concrete payloads keeps interface recursion bounded.
		types := []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(false), reflect.TypeOf(int(0)),
			reflect.TypeOf(float64(0)), reflect.TypeOf([]string{}), reflect.TypeOf(map[string]int{})}
		if kind := g.choose(len(types) + 1); kind != 0 {
			v.Set(g.value(types[kind-1]))
		}
	default:
		panic(fmt.Sprintf("unexpected generated type %v", typ))
	}
	return v
}

type fuzzCase struct {
	in        any
	got, want reflect.Value // independent pointers to the actual destination type
}

func generateFuzzCase(data string) fuzzCase {
	if len(data) > fuzzMaxInput {
		data = data[:fuzzMaxInput]
	}
	g := fuzzGenerator{data: data}
	mode, populated := g.choose(4), g.choose(2) != 0
	inType, outType := g.types(0)
	in := g.value(inType).Interface()
	switch mode {
	case 1:
		outType = fuzzAnyType
	case 2:
		switch inType.Kind() {
		case reflect.Struct, reflect.Map:
			outType = reflect.TypeOf(map[string]any{})
		case reflect.Slice:
			outType = reflect.TypeOf([]any{})
		default:
			outType = fuzzAnyType
		}
	case 3:
		// Exercise map/slice/interface inputs going into concrete Go types too.
		encoded, err := json.Marshal(in)
		if err != nil {
			panic(err)
		}
		in = nil
		if err := json.Unmarshal(encoded, &in); err != nil {
			panic(err)
		}
	}
	got, want := reflect.New(outType), reflect.New(outType)
	if populated {
		// Replaying value generation allocates separate maps, slices, and
		// pointers. A shallow copy could let one conversion corrupt the oracle.
		initial := g
		got.Elem().Set(g.value(outType))
		want.Elem().Set(initial.value(outType))
	}
	return fuzzCase{in: in, got: got, want: want}
}
