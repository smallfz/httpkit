package kit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

type valueExtractor struct {
	req   *http.Request
	bodyb []byte
}

func newValueExtractor(req *http.Request) *valueExtractor {
	return &valueExtractor{req: req}
}

func (x *valueExtractor) unmarshalPathAndForm(target interface{}) error {
	err := UnmarshalParams(target, func(tag string) interface{} {
		name := tag
		parts := strings.Split(name, ",")
		if len(parts) > 1 {
			name = parts[0]
		}
		if v := x.req.FormValue(name); len(v) > 0 {
			return v
		}
		return x.req.PathValue(name)
	})
	if err != nil {
		return fmt.Errorf("unmarshalRequest: %v", err)
	}
	return nil
}

func (x *valueExtractor) unmarshalJSON(target interface{}) (bool, error) {
	ct := strings.ToLower(x.req.Header.Get("Content-Type"))
	if strings.Index(ct, "application/json") >= 0 {
		if x.bodyb == nil {
			dat, err := io.ReadAll(x.req.Body)
			if err != nil {
				return true, err
			}
			x.bodyb = dat
		}
		if len(x.bodyb) == 0 {
			return true, nil
		}
		return true, json.Unmarshal(x.bodyb, target)
	}
	return false, nil
}

func (x *valueExtractor) newValueByType(typ reflect.Type) (reflect.Value, error) {
	arg := reflect.New(typ)
	switch typ.Kind() {
	case reflect.Struct:
		argVal := arg.Interface()
		if err := x.unmarshalPathAndForm(argVal); err != nil {
			return arg, fmt.Errorf("%v", err)
		}
		if isJson, err := x.unmarshalJSON(argVal); isJson && err != nil {
			return arg, fmt.Errorf("%v", err)
		}
	case reflect.String:
		argVal := ""
		if _, err := x.unmarshalJSON(&argVal); err != nil {
			return arg, fmt.Errorf("%v", err)
		}
		arg.Elem().SetString(argVal)
	}
	return arg, nil
}
