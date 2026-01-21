package kit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func simple(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func JSON(w http.ResponseWriter, code int, v interface{}) {
	if dat, err := json.Marshal(v); err != nil {
		http.Error(w, "json marshaling error.", 419)
		slog.Warn("json.Marshal:", "err", err)
	} else {
		w.Write(dat)
	}
}

func Event(w http.ResponseWriter, event, data string) {
	if len(event) > 0 {
		fmt.Fprintf(w, "event: %s\r\n", event)
	}
	if strings.Index(data, "\n") >= 0 {
		lines := strings.Split(data, "\n")
		for i, line := range lines {
			lines[i] = strings.Trim(line, "\r\n")
			fmt.Fprintf(w, "data: %s\r\n", data)
		}
	} else {
		fmt.Fprintf(w, "data: %s\r\n", data)
	}
	fmt.Fprintf(w, "\r\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func WriteAsResponseAuto(w http.ResponseWriter, val reflect.Value) {
	switch val.Type().Kind() {
	case reflect.String:
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Write([]byte(val.String()))
	default:
		if val.CanInterface() {
			w.Header().Set("Content-Type", "application/json")
			vi := val.Interface()
			if dat, err := json.Marshal(vi); err != nil {
				slog.Warn("json.Marshal:", "value", vi, "err", err)
			} else {
				w.Write(dat)
			}
		}
	}
}

func ValueToError(v reflect.Value) (error, bool) {
	typ := v.Type()
	if typ.Kind() == reflect.Interface {
		if !v.IsNil() {
			if typ == typError || typ.Implements(typError) {
				ev := v.Interface()
				if err, ok := ev.(error); ok {
					return err, true
				}
			}
		}
	}
	return nil, false
}

type ParamFinder func(string) interface{}

func UnmarshalParams(target interface{}, finder ParamFinder) error {
	if target == nil {
		return fmt.Errorf("nil input.")
	}
	tv := reflect.ValueOf(target)
	return unmarshalParams(tv, finder)
}

func unmarshalParams(tv reflect.Value, finder ParamFinder) error {
	if finder == nil {
		return fmt.Errorf("a finder func required.")
	}
	tvType := tv.Type()
	if tvType.Kind() == reflect.Ptr {
		if tvType.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("target must be a struct or *struct.")
		}
		return unmarshalParams(tv.Elem(), finder)
	}
	if tvType.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct or *struct.")
	}
	for i := 0; i < tvType.NumField(); i++ {
		f := tvType.Field(i)
		varName := f.Name
		tagReq := f.Tag.Get("req")
		tagJson := f.Tag.Get("json")
		if len(tagReq) > 0 {
			varName = tagReq
		} else if len(tagJson) > 0 {
			varName = tagJson
		}

		sv := finder(varName)
		if sv == nil {
			continue
		}
		sVal := reflect.ValueOf(sv)
		sValKind := sVal.Type().Kind()
		val := ""
		if sValKind == reflect.String {
			val = sv.(string)
		}

		fv := tv.FieldByName(f.Name)
		if !fv.CanSet() {
			continue
		}
		switch fv.Type().Kind() {
		case reflect.String:
			if sValKind == reflect.String {
				fv.Set(sVal)
			}
			break
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			if sVal.Type().AssignableTo(fv.Type()) {
				fv.Set(sVal)
				break
			} else if sVal.Type().ConvertibleTo(fv.Type()) {
				vtmp := sVal.Convert(fv.Type())
				fv.Set(vtmp)
				break
			}
			if flt, err := strconv.ParseFloat(val, 64); err == nil {
				iv := reflect.ValueOf(flt)
				if iv.Type().AssignableTo(fv.Type()) {
					fv.Set(iv)
				} else if iv.Type().ConvertibleTo(fv.Type()) {
					vtmp := iv.Convert(fv.Type())
					fv.Set(vtmp)
				}
			}
			break
		default:
			if sVal.Type().AssignableTo(fv.Type()) {
				fv.Set(sVal)
				break
			} else if sVal.Type().ConvertibleTo(fv.Type()) {
				vtmp := sVal.Convert(fv.Type())
				fv.Set(vtmp)
				break
			}
			if intVal, err := strconv.Atoi(val); err == nil {
				iv := reflect.ValueOf(intVal)
				if iv.Type().AssignableTo(fv.Type()) {
					fv.Set(iv)
				} else if iv.Type().ConvertibleTo(fv.Type()) {
					vtmp := iv.Convert(fv.Type())
					fv.Set(vtmp)
				}
			}
			break
		}
	}
	return nil
}
