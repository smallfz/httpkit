package kit

import (
	"net/http"
	"reflect"
)

// Bindable is the interface that a param should implements if it is needed to be inject to the bind func.
// Example:
//
//	type Session struct{
//	    Uid string
//	}
//
//	func (s *Session) Bind(w http.ResponseWriter, req *http.Request) error {
//	    /* todo: .. initial the session */
//	    s.Uid = lookupUidByToken(req.Header.Get("X-Token"))
//	    return nil
//	}
//
// mux := http.NewServeMux()
//
//	mux.Handle("/restricted-res/", httpkit.F(func(s *Session) string {
//	    return s.Uid
//	}))
type Bindable interface {
	Bind(http.ResponseWriter, *http.Request) error
}

var typBindable = reflect.TypeOf((*Bindable)(nil)).Elem()
