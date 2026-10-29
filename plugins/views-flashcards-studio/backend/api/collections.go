package api

import "reflect"

// FillCollections replaces every nil slice and nil map reachable from the pointer with an empty
// one, so that a collection is encoded as `[]` or `{}` and never as `null`.
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
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fillValue(v.Field(i))
			}
		}
	case reflect.Slice:
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
	case reflect.Map:
		if v.CanSet() && v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	default:
	}
}
