package goloose

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	StringToFloat64 bool // controls whether goloose will convert strings to floats if required; note this breaks the json.Unmarshal paradigm

	Transforms []TransformFunc
}
type TransformFunc func(interface{}) interface{}

// ConvertTo tries to convert in into a T, using JSON marshal/unmarshal semantics.
func ConvertTo[T any](in any, options ...Options) (T, error) {
	var res T
	err := ToStruct(in, &res, options...)
	return res, err
}

func ToStruct(in, out interface{}, options ...Options) error {
	inVal := reflect.ValueOf(in)
	if isNil(inVal) {
		return nil
	}
	outVal := reflect.ValueOf(out)
	if outVal.Kind() != reflect.Ptr {
		return fmt.Errorf("out should be a pointer!")
	}

	var opt Options
	if len(options) > 1 {
		return fmt.Errorf("pass at most one Options struct")
	} else if len(options) == 1 {
		opt = options[0]
	}

	err := toStructImpl(inVal, outVal, opt)
	return err
}

func toStructImpl(in, out reflect.Value, options Options) error {
	if !in.IsValid() || !in.CanInterface() {
		return nil
	}
	for _, fn := range options.Transforms {
		in = reflect.ValueOf(fn(in.Interface()))
		if !in.IsValid() {
			return nil
		}
	}
	inType := in.Type()
	outType := out.Type()
	if handled, err := customJson(in, inType, out, outType); handled {
		return err
	}

	if out.Kind() == reflect.Ptr {
		if (in.Kind() == reflect.Ptr || in.Kind() == reflect.Interface) && in.IsNil() && out.CanAddr() {
			out.Set(reflect.Zero(outType))
			return nil
		}
		if out.IsNil() {
			out.Set(reflect.New(outType.Elem()))
		}
		return toStructImpl(in, out.Elem(), options)
	}
	if isNil(in) {
		out.Set(reflect.Zero(outType))
		return nil
	}
	if out.Kind() == reflect.Interface {
		if out.IsNil() {
			var outVal reflect.Value

			for inType.Kind() == reflect.Ptr {
				if isNil(in) {
					return nil
				}
				in = in.Elem()
				inType = in.Type()
			}
			if isNil(in) {
				return nil
			}
			inType = toJsonType(inType)
			switch inType.Kind() {
			case reflect.Struct, reflect.Map:
				outVal = reflect.MakeMap(mapStringInterfaceType)
				err := toStructImpl(in, outVal, options)
				if err != nil {
					return err
				}
				out.Set(outVal)
				return nil
			case reflect.Slice:
				outVal = reflect.New(interfaceSliceType)
			case reflect.Interface:
				return toStructImpl(in.Elem(), out, options)
			default:
				outVal = reflect.New(inType).Elem()
				err := toStructImpl(in, outVal, options)
				if err != nil {
					return err
				}
				out.Set(outVal)
				return nil
			}
			err := toStructImpl(in, outVal, options)
			if err != nil {
				return err
			}
			out.Set(outVal.Elem())
			return nil
		}
		// it would be nice to handle this more performantly, but there are some edge cases that need to be considered more thoroughly!
		return toStructSlow(in.Interface(), out.Addr().Interface())
	}
	var outFields []field

	switch in.Kind() {
	case reflect.Struct:
		if out.Kind() != reflect.Map && out.Kind() != reflect.Struct {
			return nil
		}
		fields := cachedTypeFields(inType)
		for _, field := range fields {
			val := fieldByIndex(in, field.index, false)
			if field.omitEmpty && isEmptyValue(val) {
				continue
			}
			if field.quoted {
				val = dequote(val)
			}
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
			switch out.Kind() {
			case reflect.Map:
				if out.IsNil() {
					outMap := reflect.MakeMap(outType)
					out.Set(outMap)
				}
				outVal := reflect.New(outType.Elem())
				err := toStructImpl(val, outVal, options)
				if err != nil {
					return err
				}
				nameVal := reflect.ValueOf(string(field.name)).Convert(outType.Key())
				out.SetMapIndex(nameVal, outVal.Elem())
			case reflect.Struct:
				if len(outFields) == 0 {
					outFields = cachedTypeFields(outType)
				}
				for _, outfield := range outFields {
					if bytes.Equal(outfield.namelower, field.namelower) {
						if field.quoted {
							val = dequote(val)
						}
						if val.Kind() == reflect.Ptr && val.IsNil() {
							continue
						}
						err := toStructImpl(val, fieldByIndex(out, outfield.index, true), options)
						if err != nil {
							return err
						}
					}
				}
			}
		}

	case reflect.Map:
		if out.Kind() != reflect.Map && out.Kind() != reflect.Struct {
			return nil
		}
		for _, key := range in.MapKeys() {
			if key.Kind() != reflect.String {
				return fmt.Errorf("Only string keys are supported! Kind: %v", key.Kind())
			}
			keyStr := []byte(key.String())
			val := in.MapIndex(key)
			if val.Kind() == reflect.Interface && !val.IsNil() {
				val = val.Elem()
			}
			switch out.Kind() {
			case reflect.Map:
				outVal := reflect.New(toJsonType(outType.Elem()))
				if out.IsNil() {
					outMap := reflect.MakeMap(outType)
					out.Set(outMap)
				}
				err := toStructImpl(val, outVal, options)
				if err != nil {
					return err
				}
				out.SetMapIndex(key.Convert(outType.Key()), outVal.Elem().Convert(outType.Elem()))
			case reflect.Struct:
				// in-place bytes.ToLower
				for i := 0; i < len(keyStr); i++ {
					c := keyStr[i]
					if 'A' <= c && c <= 'Z' {
						keyStr[i] += 'a' - 'A'
					}
				}
				if len(outFields) == 0 {
					outFields = cachedTypeFields(outType)
				}
				for _, field := range outFields {
					if bytes.Equal(field.namelower, keyStr) {
						if field.quoted {
							val = dequote(val)
						}
						if val.Kind() == reflect.Ptr && val.IsNil() {
							continue
						}
						err := toStructImpl(val, fieldByIndex(out, field.index, true), options)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	case reflect.Slice:
		if out.Kind() != reflect.Slice {
			return nil
		}
		if out.IsNil() || out.Len() != in.Len() {
			outSlice := reflect.MakeSlice(outType, in.Len(), in.Cap())
			out.Set(outSlice)
		}
		for i := 0; i < in.Len(); i++ {
			val := in.Index(i)
			err := toStructImpl(val, out.Index(i), options)
			if err != nil {
				return err
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int64, reflect.Uintptr, reflect.Float32,
		reflect.Bool, reflect.String, reflect.Float64, reflect.Complex64, reflect.Complex128:
		tryToConvert(in, inType, out, outType, options)
	case reflect.Array:
		panic("Array not supported yet!")
	case reflect.Chan, reflect.Func:
		// do nothing
	case reflect.Interface:
		return toStructImpl(in.Elem(), out, options)
	case reflect.Ptr:
		return toStructImpl(in.Elem(), out, options)
	case reflect.UnsafePointer:
		panic("UnsafePointer not supported!")
	default:
		panic(fmt.Sprintf("Unknown kind %v", in.Kind()))
	}
	return nil
}

var trueVal = reflect.ValueOf(true)
var falseVal = reflect.ValueOf(false)

func tryToConvert(in reflect.Value, inType reflect.Type, out reflect.Value, outType reflect.Type, options Options) {
	if inType == outType {
		out.Set(in)
		return
	}
	switch in.Kind() {
	case reflect.String:
		if out.Kind() == reflect.Bool {
			switch strings.ToLower(in.String()) {
			case "true":
				out.Set(trueVal)
			case "false":
				out.Set(falseVal)
			}
		}
		if options.StringToFloat64 && out.Kind() == reflect.Float64 {
			if f, err := strconv.ParseFloat(in.String(), 64); err == nil {
				out.Set(reflect.ValueOf(f).Convert(outType))
			}
		}
	}
	if inType.ConvertibleTo(outType) {
		out.Set(in.Convert(outType))
	}
}

// reference version to compare against
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

func toJsonType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int64, reflect.Uintptr, reflect.Float32:
		return float64Type
	case reflect.String:
		return stringType
	case reflect.Map:
		// do something here, we're not converting Messages to map[string]interface{}s
	}
	return t
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func isNil(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		return val.IsNil()
	}
	return false
}

var float64Type = reflect.TypeOf(float64(0))
var stringType = reflect.TypeOf(string(""))
var mapStringInterfaceType = reflect.TypeOf(map[string]interface{}{})
var interfaceSliceType = reflect.TypeOf([]interface{}{})
var timeType = reflect.TypeOf(time.Time{})
var timePtrType = reflect.TypeOf(&time.Time{})
var jsonMarshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()
var jsonUnmarshalerType = reflect.TypeOf(new(json.Unmarshaler)).Elem()
var textMarshalerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
var textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

func customJson(in reflect.Value, inType reflect.Type, out reflect.Value, outType reflect.Type) (bool, error) {
	if !out.CanAddr() {
		return false, nil
	}
	outType = reflect.PointerTo(outType)
	inOk := inType.Implements(jsonMarshalerType) || inType.Implements(textMarshalerType)
	outOk := outType.Implements(jsonUnmarshalerType) || outType.Implements(textUnmarshalerType)
	if inOk || outOk {
		if timeFastPath(in, inType, out, outType) {
			return true, nil
		}

		b, err := json.Marshal(in.Interface())
		if err != nil {
			return true, err
		}
		outInter := out.Addr().Interface()
		err = json.Unmarshal(b, &outInter)
		return true, err
	}
	return false, nil
}

func timeFastPath(in reflect.Value, inType reflect.Type, out reflect.Value, outType reflect.Type) bool {
	switch inType {
	case timeType:
		switch outType {
		case timeType:
			out.Set(in)
			return true
		case stringType:
			t := in.Interface().(time.Time)
			out.Set(reflect.ValueOf(t.Format(time.RFC3339Nano)))
		}
	case timePtrType:
		switch outType {
		case timePtrType:
			if !in.IsNil() {
				outVal := reflect.New(timeType)
				outVal.Elem().Set(in.Elem())
				out.Set(outVal)
			} else {
				out.Set(in)
			}
			return true
		}
	case stringType:
		switch outType {
		case timeType:
			t, err := time.Parse(time.RFC3339Nano, in.String())
			if err == nil {
				out.Set(reflect.ValueOf(t))
				return true
			}
		}
	}
	return false
}

func dequote(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.String {
		return v
	}
	str := v.String()
	if b, err := strconv.ParseBool(str); err == nil {
		return reflect.ValueOf(b)
	}
	if i, err := strconv.ParseInt(str, 10, 64); err == nil {
		return reflect.ValueOf(i)
	}
	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return reflect.ValueOf(f)
	}
	if !strings.HasPrefix(str, `"`) || !strings.HasSuffix(str, `"`) {
		return v
	}
	str = str[1 : len(str)-1]
	return reflect.ValueOf(str)
}
