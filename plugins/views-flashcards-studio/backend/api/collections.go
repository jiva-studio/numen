package api

import "reflect"

// FillCollections replaces every nil slice and nil map reachable from the pointer
// with an empty one: a collection is encoded as `[]` or `{}` and never as `null`.
func FillCollections(target any) {
	fillValue(reflect.ValueOf(target))
}

func fillValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			fillValue(v.Elem())
		}
	case reflect.Struct:
		fillStruct(v)
	case reflect.Slice:
		fillSlice(v)
	case reflect.Map:
		if v.CanSet() && v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.String, reflect.UnsafePointer:
	default:
	}
}

func fillStruct(v reflect.Value) {
	for i := range v.NumField() {
		if v.Type().Field(i).IsExported() {
			fillValue(v.Field(i))
		}
	}
}

func fillSlice(v reflect.Value) {
	if !v.CanSet() {
		return
	}
	if v.IsNil() {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		return
	}
	for i := range v.Len() {
		fillValue(v.Index(i))
	}
}
