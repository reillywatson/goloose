package goloose

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func ToStruct(in, out interface{}) error {
	return ToStructWithTransforms(in, out, nil)
}

type TransformFunc func(interface{}) interface{}

func ToStructWithTransforms(in, out interface{}, transforms []TransformFunc) error {
	if isNil(reflect.ValueOf(in)) {
		return nil
	}

	outVal := reflect.ValueOf(out)
	if outVal.Kind() != reflect.Ptr {
		return fmt.Errorf("out should be a pointer!")
	}

	err := toStructImpl(reflect.ValueOf(in), outVal, transforms)
	return err
}

func toStructImpl(in, out reflect.Value, transforms []TransformFunc) error {
	if !in.IsValid() || !in.CanInterface() {
		return nil
	}
	for _, fn := range transforms {
		in = reflect.ValueOf(fn(in.Interface()))
		if !in.IsValid() {
			return nil
		}
	}
	if handled, err := customJson(in, out); handled {
		return err
	}

	if out.Kind() == reflect.Ptr {
		if (in.Kind() == reflect.Ptr || in.Kind() == reflect.Interface) && in.IsNil() && out.CanAddr() {
			out.Set(reflect.Zero(out.Type()))
			return nil
		}
		if out.IsNil() {
			out.Set(reflect.New(out.Type().Elem()))
		}
		return toStructImpl(in, out.Elem(), transforms)
	}
	if isNil(in) {
		out.Set(reflect.Zero(out.Type()))
		return nil
	}
	if out.Kind() == reflect.Interface {
		inType := in.Type()
		if out.IsNil() {
			var outVal reflect.Value
			for inType.Kind() == reflect.Ptr {
				inType = inType.Elem()
			}
			inType = toJsonType(inType)
			switch inType.Kind() {
			case reflect.Struct, reflect.Map:
				outVal = reflect.MakeMap(mapStringInterfaceType)
				err := toStructImpl(in, outVal, transforms)
				if err != nil {
					return err
				}
				out.Set(outVal)
				return nil
			case reflect.Slice:
				outVal = reflect.New(interfaceSliceType)
			case reflect.Interface:
				return toStructImpl(in.Elem(), out, transforms)
			default:
				outVal = reflect.New(inType).Elem()
				err := toStructImpl(in, outVal, transforms)
				if err != nil {
					return err
				}
				out.Set(outVal)
				return nil
			}
			err := toStructImpl(in, outVal, transforms)
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
		fields := cachedTypeFields(in.Type())
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
					outMap := reflect.MakeMap(out.Type())
					out.Set(outMap)
				}
				outVal := reflect.New(out.Type().Elem())
				err := toStructImpl(val, outVal, transforms)
				if err != nil {
					return err
				}
				nameVal := reflect.ValueOf(field.name).Convert(out.Type().Key())
				out.SetMapIndex(nameVal, outVal.Elem())
			case reflect.Struct:
				if len(outFields) == 0 {
					outFields = cachedTypeFields(out.Type())
				}
				for _, outfield := range outFields {
					if outfield.namelower == field.namelower {
						if field.quoted {
							val = dequote(val)
						}
						if val.Kind() == reflect.Ptr && val.IsNil() {
							continue
						}
						err := toStructImpl(val, fieldByIndex(out, outfield.index, true), transforms)
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
			keyStr := key.String()
			val := in.MapIndex(key)
			if val.Kind() == reflect.Interface && !val.IsNil() {
				val = val.Elem()
			}
			switch out.Kind() {
			case reflect.Map:
				outVal := reflect.New(toJsonType(out.Type().Elem()))
				if out.IsNil() {
					outMap := reflect.MakeMap(out.Type())
					out.Set(outMap)
				}
				err := toStructImpl(val, outVal, transforms)
				if err != nil {
					return err
				}
				out.SetMapIndex(key.Convert(out.Type().Key()), outVal.Elem().Convert(out.Type().Elem()))
			case reflect.Struct:
				keyStr = strings.ToLower(keyStr)
				if len(outFields) == 0 {
					outFields = cachedTypeFields(out.Type())
				}
				for _, field := range outFields {
					if field.namelower == keyStr {
						if field.quoted {
							val = dequote(val)
						}
						if val.Kind() == reflect.Ptr && val.IsNil() {
							continue
						}
						err := toStructImpl(val, fieldByIndex(out, field.index, true), transforms)
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
			outSlice := reflect.MakeSlice(out.Type(), in.Len(), in.Cap())
			out.Set(outSlice)
		}
		for i := 0; i < in.Len(); i++ {
			val := in.Index(i)
			err := toStructImpl(val, out.Index(i), transforms)
			if err != nil {
				return err
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int64, reflect.Uintptr, reflect.Float32,
		reflect.Bool, reflect.String, reflect.Float64, reflect.Complex64, reflect.Complex128:
		tryToConvert(in, out)
	case reflect.Array:
		panic("Array not supported yet!")
	case reflect.Chan, reflect.Func:
		// do nothing
	case reflect.Interface:
		return toStructImpl(in.Elem(), out, transforms)
	case reflect.Ptr:
		return toStructImpl(in.Elem(), out, transforms)
	case reflect.UnsafePointer:
		panic("UnsafePointer not supported!")
	default:
		panic(fmt.Sprintf("Unknown kind %v", in.Kind()))
	}
	return nil
}

func tryToConvert(in, out reflect.Value) {
	switch in.Kind() {
	case reflect.String:
		if out.Kind() == reflect.Bool {
			switch in.String() {
			case "true":
				out.Set(reflect.ValueOf(true))
			case "false":
				out.Set(reflect.ValueOf(false))
			}
		}
	}
	if in.Type().ConvertibleTo(out.Type()) {
		out.Set(in.Convert(out.Type()))
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

func customJson(in, out reflect.Value) (bool, error) {
	if !out.CanAddr() {
		return false, nil
	}
	inOk := in.Type().Implements(jsonMarshalerType)
	outOk := out.Addr().Type().Implements(jsonUnmarshalerType)
	if inOk || outOk {
		if timeFastPath(in, out) {
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

func timeFastPath(in, out reflect.Value) bool {
	switch in.Type() {
	case timeType:
		switch out.Type() {
		case timeType:
			out.Set(in)
			return true
		case stringType:
			t := in.Interface().(time.Time)
			out.Set(reflect.ValueOf(t.Format(time.RFC3339Nano)))
		}
	case timePtrType:
		switch out.Type() {
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
		switch out.Type() {
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
