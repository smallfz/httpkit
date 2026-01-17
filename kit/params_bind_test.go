package kit

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestParamsBind1(t *testing.T) {
	textContent := "hello!"

	mux := http.NewServeMux()
	mux.HandleFunc("/", BindFunc(func() string { return textContent }))
	svr := httptest.NewServer(mux)

	req, err := http.NewRequest("GET", svr.URL, nil)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
		return
	}

	cl := svr.Client()

	if resp, err := cl.Do(req); err != nil {
		t.Fatalf("cl.Do: %v", err)
		return
	} else {
		if resp.StatusCode != 200 {
			t.Fatalf("expects 200. got %d", resp.StatusCode)
			return
		}
		dat, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("io.ReadAll: %v", err)
			return
		}
		if string(dat) != "hello!" {
			t.Fatalf("body is not expected.")
			return
		}
	}
}

type testPayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func TestParamsBind2(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/mul", F(func(p *testPayload, req *http.Request) int {
		return p.X * p.Y
	}))
	svr := httptest.NewServer(mux)

	uri, _ := url.Parse(svr.URL)
	uri.Path = "/mul"
	qry := make(url.Values)
	qry.Set("x", fmt.Sprintf("%d", 8))
	qry.Set("y", fmt.Sprintf("%d", 9))
	uri.RawQuery = qry.Encode()
	println(uri.String())

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
		return
	}

	cl := svr.Client()

	if resp, err := cl.Do(req); err != nil {
		t.Fatalf("cl.Do: %v", err)
		return
	} else {
		if resp.StatusCode != 200 {
			t.Fatalf("expects 200. got %d", resp.StatusCode)
			return
		}
		dat, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("io.ReadAll: %v", err)
			return
		}
		if string(dat) != "72" {
			t.Fatalf("body is not expected: %s", string(dat))
			return
		}
	}
}

func TestParamsBind3(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/mul", F(func(w http.ResponseWriter, p *testPayload, req *http.Request) int {
		return p.X * p.Y
	}))
	svr := httptest.NewServer(mux)

	uri, _ := url.Parse(svr.URL)
	uri.Path = "/mul"

	body := bytes.NewBuffer([]byte("{\"x\":8,\"y\":9}"))
	req, err := http.NewRequest("POST", uri.String(), body)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	cl := svr.Client()

	if resp, err := cl.Do(req); err != nil {
		t.Fatalf("cl.Do: %v", err)
		return
	} else {
		if resp.StatusCode != 200 {
			t.Fatalf("expects 200. got %d", resp.StatusCode)
			return
		}
		dat, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("io.ReadAll: %v", err)
			return
		}
		if string(dat) != "72" {
			t.Fatalf("body is not expected: %s", string(dat))
			return
		}
	}
}

type testSession struct {
	Uid string
}

var _ Bindable = (*testSession)(nil)

func (s *testSession) Bind(w http.ResponseWriter, req *http.Request) error {
	for k, _ := range req.Header {
		println(k, req.Header.Get(k))
	}
	s.Uid = req.Header.Get("x-uid")
	return nil
}

func TestParamsBindCustom(t *testing.T) {
	uid := fmt.Sprintf("%d", time.Now().Unix())

	mux := http.NewServeMux()
	mux.HandleFunc("/uid", F(func(s *testSession) string {
		return s.Uid
	}))
	svr := httptest.NewServer(mux)

	uri, _ := url.Parse(svr.URL)
	uri.Path = "/uid"

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
		return
	}
	req.Header.Set("x-uid", uid)

	cl := svr.Client()

	if resp, err := cl.Do(req); err != nil {
		t.Fatalf("cl.Do: %v", err)
		return
	} else {
		if resp.StatusCode != 200 {
			t.Fatalf("expects 200. got %d", resp.StatusCode)
			return
		}
		dat, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("io.ReadAll: %v", err)
			return
		}
		if string(dat) != uid {
			t.Fatalf("body is not expected: %s(%d bytes)", string(dat), len(dat))
			return
		}
	}
}
