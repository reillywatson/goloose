package goloose

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func FuzzGoloose(f *testing.F) {
	debugging := false

	testcases := []string{
		`var in = struct{
	A string ` + "`" + `json:"a"` + "`" + `
	B int ` + "`" + `json:"b"` + "`" + `
		}{"foo", 1}
var out = struct{
	A string ` + "`" + `json:"a"` + "`" + `
	B int ` + "`" + `json:"b"` + "`" + `
}{}`,
		`var in = struct{
	A string ` + "`" + `json:"a"` + "`" + `
	B []int ` + "`" + `json:"b"` + "`" + `
		}{"foo", []int{1}}
var out = struct{
	A string ` + "`" + `json:"a"` + "`" + `
	B []int ` + "`" + `json:"b"` + "`" + `
}{}`,
		`var in = map[string]any{"a": "foo", "b": "bar"}
	var out = struct{
	A string ` + "`" + `json:"a"` + "`" + `
}{}`,
		`var in = map[string]any{"a": "foo", "b": map[string]any{"bar":"baz"}}
	var out = struct{
	A string ` + "`" + `json:"a"` + "`" + `
	B struct {
		Bar string ` + "`" + `json:"bar"` + "`" + `
	} ` + "`" + `json:"b"` + "`" + `
}{}`,
		`var in = map[string]any{"a": "foo", "b": map[string]any{"bar":"baz"}}
var out = struct{
A *string ` + "`" + `json:"a"` + "`" + `
B *struct {
	Bar string ` + "`" + `json:"bar"` + "`" + `
} ` + "`" + `json:"b"` + "`" + `
}{}`,
	}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, code string) {
		src := fmt.Sprintf("package main\n\n%s", code)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "", src, parser.DeclarationErrors|parser.AllErrors)
		if err != nil {
			if stringInList(testcases, code) {
				t.Errorf("seed corpus entry failed to parse! Error:\n%v", err)
			}
			return
		}
		if len(file.Decls) != 2 {
			return
		}
		validateFuzz(t, file.Decls[0], file.Decls[1], code, debugging)
		validateFuzz(t, file.Decls[1], file.Decls[0], code, debugging)
	})
}

func validateFuzz(t *testing.T, inDecl, outDecl ast.Decl, code string, debugging bool) {
	parseErr := func(code string, err error) {
		if debugging {
			fmt.Println("Parse error: ", err)
			fmt.Println(code)
		}
	}
	in, err := loadTypespecFromAST(inDecl)
	if err != nil {
		parseErr(code, err)
		return
	}
	out, err := loadTypespecFromAST(outDecl)
	if err != nil {
		parseErr(code, err)
		return
	}
	outSlow, err := loadTypespecFromAST(outDecl)
	if err != nil {
		parseErr(code, err)
		return
	}
	if in == nil || out == nil || outSlow == nil {
		parseErr(code, fmt.Errorf("didn't generate types"))
		return
	}
	if strings.Contains(reflect.TypeOf(in).String(), "reflect.") || strings.Contains(reflect.TypeOf(out).String(), "reflect.") {
		parseErr(code, fmt.Errorf("unexpected results! In: %+#v, Out: %+#v", in, out))
		return
	}
	if err := ToStruct(in, &out); err != nil {
		if err := toStructSlow(in, &outSlow); err == nil {
			t.Errorf("ToStruct failed but toStructSlow succeeded! Error: %v", err)
		}
		return
	}
	if debugging {
		fmt.Println("IN:", toJson(in), "OUT:", toJson(out))
	}
	if err := toStructSlow(in, &outSlow); err != nil {
		return // can't JSON compare!
	}
	if !reflect.DeepEqual(out, outSlow) {
		t.Errorf("Got %+v\nExpected %+v", out, outSlow)
	}
}

func loadTypespecFromAST(expr any) (any, error) {
	switch expr := expr.(type) {
	case *ast.ValueSpec:
		if len(expr.Values) != 1 {
			return nil, fmt.Errorf("unexpected values count %d", len(expr.Values))
		}
		return loadTypespecFromAST(expr.Values[0])
	case *ast.CompositeLit:
		typedef, err := loadTypespecFromAST(expr.Type)
		if err != nil {
			return nil, err
		}
		structDef, ok := typedef.(reflect.Type)
		if !ok {
			return nil, fmt.Errorf("expected reflect.Type, got %q", reflect.TypeOf(typedef))
		}
		val := reflect.New(structDef).Elem()
		if len(expr.Elts) == 0 {
			return val.Interface(), err
		}
		if val.Kind() == reflect.Struct && len(expr.Elts) != val.NumField() {
			return nil, fmt.Errorf("invalid field count (expected %d, got %d)", val.NumField(), len(expr.Elts))
		}
		for i, elt := range expr.Elts {
			f, err := loadTypespecFromAST(elt)
			if err != nil {
				return nil, err
			}
			inVal := reflect.ValueOf(f)
			if !inVal.IsValid() {
				return nil, fmt.Errorf("invalid value")
			}
			switch val.Kind() {
			case reflect.Slice:
				if inVal.CanConvert(val.Type()) {
					val = reflect.Append(val, inVal)
				}
			case reflect.Struct:
				if val.Field(i).CanSet() && inVal.CanConvert(val.Field(i).Type()) {
					val.Field(i).Set(inVal.Convert(val.Field(i).Type()))
				}
			case reflect.Map:
				if inVal.Kind() != reflect.Map {
					return nil, fmt.Errorf("invalid conversion: map to %q", inVal.Kind())
				}
				for _, key := range inVal.MapKeys() {
					mapVal := inVal.MapIndex(key)
					if val.IsNil() {
						val = reflect.MakeMap(val.Type())
					}
					if key.CanConvert(val.Type().Key()) && mapVal.CanConvert(val.Type().Elem()) {
						val.SetMapIndex(key.Convert(val.Type().Key()), mapVal.Convert(val.Type().Elem()))
					}
				}
			default:
				return nil, fmt.Errorf("unexpected kind %q", val.Kind())
			}
		}
		if val.CanAddr() {
			return val.Addr().Interface(), nil
		}
		return val.Interface(), nil
	case *ast.BasicLit:
		switch expr.Kind {
		case token.STRING:
			val, err := strconv.Unquote(expr.Value)
			if err != nil {
				return nil, err
			}
			return val, nil
		case token.INT:
			val, err := strconv.Atoi(expr.Value)
			if err != nil {
				return nil, err
			}
			return val, nil
		case token.FLOAT:
			val, err := strconv.ParseFloat(expr.Value, 64)
			if err != nil {
				return nil, err
			}
			return val, nil
		}
	case *ast.GenDecl:
		if len(expr.Specs) != 1 {
			return nil, fmt.Errorf("unexpected specs count %d", len(expr.Specs))
		}
		return loadTypespecFromAST(expr.Specs[0])
	case *ast.StructType:
		fields, err := loadTypespecFromAST(expr.Fields)
		if err != nil {
			return nil, err
		}
		structFields := fields.([]reflect.StructField)
		seen := map[string]bool{}
		for _, f := range structFields {
			if seen[f.Name] {
				return nil, fmt.Errorf("duplicate field name %q", f.Name)
			}
			seen[f.Name] = true
			if !isValidFieldName(f.Name) {
				return nil, fmt.Errorf("invalid field name %q", f.Name)
			}
		}
		return reflect.StructOf(fields.([]reflect.StructField)), nil
	case *ast.ArrayType:
		arrayType, err := loadTypespecFromAST(expr.Elt)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(arrayType.(reflect.Type)), nil
	case *ast.MapType:
		keyType, err := loadTypespecFromAST(expr.Key)
		if err != nil {
			return nil, err
		}
		if !keyType.(reflect.Type).Comparable() {
			return nil, fmt.Errorf("invalid key type %v", keyType)
		}
		valType, err := loadTypespecFromAST(expr.Value)
		if err != nil {
			return nil, err
		}
		mapType := reflect.MapOf(keyType.(reflect.Type), valType.(reflect.Type))
		return mapType, nil
	case *ast.KeyValueExpr:
		keyVal, err := loadTypespecFromAST(expr.Key)
		if err != nil {
			return nil, err
		}
		valVal, err := loadTypespecFromAST(expr.Value)
		if err != nil {
			return nil, err
		}
		key := reflect.ValueOf(keyVal)
		if !key.IsValid() {
			return nil, fmt.Errorf("Invalid key %v", keyVal)
		}
		val := reflect.ValueOf(valVal)
		if !val.IsValid() {
			return nil, fmt.Errorf("invalid val %v", valVal)
		}
		mapVal := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(keyVal), reflect.TypeOf(valVal)))
		mapVal.SetMapIndex(reflect.ValueOf(keyVal), reflect.ValueOf(valVal))
		return mapVal.Interface(), nil
	case *ast.FieldList:
		var fields []reflect.StructField
		for _, field := range expr.List {
			val, err := loadTypespecFromAST(field)
			if err != nil {
				return nil, err
			}
			fields = append(fields, val.(reflect.StructField))
		}
		return fields, nil
	case *ast.StarExpr:
		val, err := loadTypespecFromAST(expr.X)
		if err != nil {
			return nil, err
		}
		if asType, ok := val.(reflect.Type); ok {
			return reflect.PointerTo(asType), nil
		}
		return nil, fmt.Errorf("unexpected return type %v", reflect.TypeOf(val))
	case *ast.Field:
		if len(expr.Names) != 1 {
			return nil, fmt.Errorf("unexpected number of names %d", len(expr.Names))
		}
		if expr.Names[0].Name == "" {
			return nil, fmt.Errorf("no name for struct field")
		}
		var tag reflect.StructTag
		if expr.Tag != nil {
			tagVal, err := loadTypespecFromAST(expr.Tag)
			if err != nil {
				return nil, err
			}
			tag = reflect.StructTag(tagVal.(string))
		}
		fieldType, err := loadTypespecFromAST(expr.Type)
		if err != nil {
			return nil, err
		}
		pkgPath := ""
		c := expr.Names[0].Name[0]
		if 'a' <= c && c <= 'z' || c == '_' {
			pkgPath = "goloose"
		}
		val := reflect.StructField{
			Name:    expr.Names[0].Name,
			Type:    fieldType.(reflect.Type),
			Tag:     tag,
			PkgPath: pkgPath,
		}
		return val, nil
	case *ast.Ident:
		switch expr.Name {
		case "int":
			return reflect.TypeOf(int(0)), nil
		case "string":
			return reflect.TypeOf(""), nil
		case "float64":
			return reflect.TypeOf(float64(0)), nil
		case "any":
			return reflect.TypeOf(new(any)).Elem(), nil
		}
		return nil, fmt.Errorf("unexpected ident name %q", expr.Name)
	default:
		return nil, fmt.Errorf("unhandled type %q", reflect.TypeOf(expr))
	}
	return nil, fmt.Errorf("shouldn't get here")
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

func stringInList(strs []string, s string) bool {
	for _, str := range strs {
		if str == s {
			return true
		}
	}
	return false
}
