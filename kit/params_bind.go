package kit

import (
	"fmt"
	"net/http"
	"reflect"
)

type errEmpty struct {
	E error
}

// var typError = reflect.TypeFor[error]()
var typError = reflect.TypeOf(errEmpty{}).Field(0).Type

// F is just shortcut for BindFunc.
func F(fn interface{}) http.HandlerFunc {
	return BindFunc(fn)
}

// BindFunc makes any giving function to a http.HandlerFunc.
func BindFunc(fn interface{}) http.HandlerFunc {
	return func(wBase http.ResponseWriter, req *http.Request) {
		w := newMonitoredWriter(wBase)

		typ := reflect.TypeOf(fn)
		if typ.Kind() != reflect.Func {
			simple(w, 400, "invalid binding func.")
			return
		}

		numIn := typ.NumIn()

		args := make([]reflect.Value, numIn)
		if numIn > 0 {
			typHttpReq := reflect.TypeOf(req)
			typCtx := reflect.TypeOf(req.Context())
			typWriter := reflect.TypeOf(w)
			extractor := newValueExtractor(req)
			for i, _ := range args {
				typArg := typ.In(i)
				switch typArg {
				case typHttpReq:
					args[i] = reflect.ValueOf(req)
					continue
				case typWriter:
					args[i] = reflect.ValueOf(w)
					continue
				}
				if typArg.Kind() == reflect.Interface {
					if typWriter.Implements(typArg) {
						args[i] = reflect.ValueOf(w)
						continue
					}
					if typCtx.Implements(typArg) {
						args[i] = reflect.ValueOf(req.Context())
						continue
					}
				}
				isPtr := typArg.Kind() == reflect.Ptr
				if isPtr {
					typArg = typArg.Elem()
				}
				bindable := isPtr && reflect.PointerTo(typArg).Implements(typBindable)
				bindable = bindable || (!isPtr && typArg.Implements(typBindable))
				if bindable {
					arg := reflect.New(typArg)
					var bind reflect.Value
					if isPtr {
						bind = arg.MethodByName("Bind")
					} else {
						bind = arg.Elem().MethodByName("Bind")
					}
					retVals := bind.Call([]reflect.Value{
						reflect.ValueOf(w),
						reflect.ValueOf(req),
					})
					for _, v := range retVals {
						if err, ok := ValueToError(v); ok {
							simple(w, 419, fmt.Sprintf("%v", err))
							return
						}
					}
					if isPtr {
						args[i] = arg
					} else {
						args[i] = arg.Elem()
					}
					continue
				}
				arg, err := extractor.newValueByType(typArg)
				if err != nil {
					simple(w, 419, fmt.Sprintf("%v", err))
					return
				}
				if isPtr {
					args[i] = arg
				} else {
					args[i] = arg.Elem()
				}
			}
		}

		retVals := reflect.ValueOf(fn).Call(args)
		if len(retVals) == 0 {
			simple(w, 200, "")
			return
		}
		retVal := retVals[0]
		if len(retVals) >= 2 {
			// check if the last is an error.
			lastVal := retVals[len(retVals)-1]
			if err, ok := ValueToError(lastVal); ok {
				w.WriteHeader(419)
				msg := fmt.Sprintf("%v", err)
				w.Write([]byte(msg))
				return
			}
			// if lastVal.Type().Kind() == reflect.Interface {
			// 	if !lastVal.IsNil() && lastVal.Type() == typError {
			// 		w.WriteHeader(419)
			// 		msg := fmt.Sprintf("%v", lastVal.Interface())
			// 		w.Write([]byte(msg))
			// 		return
			// 	}
			// }
		}

		w.Header().Set("Cache-Control", "no-store")

		WriteAsResponseAuto(w, retVal)
	}
}
