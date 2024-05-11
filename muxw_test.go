package muxr_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/btwiuse/muxr"
)

var methods = [...]string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

func TestRouterMethods(t *testing.T) {
	r := muxr.NewRouter()
	testCases := [...]*struct {
		method   string
		register func(string, http.HandlerFunc)
		route    string
	}{
		{http.MethodGet, r.Get, "/hello_get"},
		{http.MethodHead, r.Head, "/hello_head"},
		{http.MethodPost, r.Post, "/hello_post"},
		{http.MethodPut, r.Put, "/hello_put"},
		{http.MethodPatch, r.Patch, "/hello_patch"},
		{http.MethodDelete, r.Delete, "/hello_delete"},
		{http.MethodConnect, r.Connect, "/hello_connect"},
		{http.MethodOptions, r.Options, "/hello_options"},
		{http.MethodTrace, r.Trace, "/hello_trace"},
	}
	var count int
	for _, tc := range testCases {
		tc.register(tc.route, func(w http.ResponseWriter, r *http.Request) {
			count++
		})
	}
	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.route, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if count != 1 {
			t.Fatalf("expected handler %s for method %s to be called exactly once but got %d", tc.route, tc.method, count)
		}
		count = 0

		for _, wrongMethod := range methods {
			if tc.method == wrongMethod {
				continue
			}
			// net/http matches GET /hello for both GET and HEAD requests.
			if wrongMethod == http.MethodHead && tc.method == http.MethodGet {
				continue
			}

			req = httptest.NewRequest(wrongMethod, tc.route, nil)
			w = httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if count != 0 {
				t.Fatalf("expected handler %s for method %s to not be called on wrong method: %s", tc.route, tc.method, wrongMethod)
			}
		}
	}
}

func TestMiddleware(t *testing.T) {
	r := muxr.NewRouter()

	got := []string{}
	want := []string{"one", "two", "three", "four"}
	r.Use(
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = append(got, "one")
				h.ServeHTTP(w, r)
			})
		},
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = append(got, "two")
				h.ServeHTTP(w, r)
			})
		},
	)
	r.Use(
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = append(got, "three")
				h.ServeHTTP(w, r)
			})
		},
	)
	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		got = append(got, "four")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected middleware call order to be %+v but got %+v", want, got)
	}
}

func TestMount(t *testing.T) {
	r := muxr.NewRouter()

	var helloCalled, prefixCalled bool
	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		helloCalled = true
	})

	subRouter := muxr.NewRouter()
	subRouter.Get("/prefix/hello", func(w http.ResponseWriter, r *http.Request) {
		prefixCalled = true
	})
	r.Mount("/prefix", subRouter)

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !helloCalled {
		t.Fatal("expected hello handler to be called")
	}
	if prefixCalled {
		t.Fatal("expected prefix to not be called")
	}

	helloCalled = false
	req = httptest.NewRequest(http.MethodGet, "/prefix/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if helloCalled {
		t.Fatal("expected hello handler to not be called")
	}
	if prefixCalled {
		t.Fatal("expected prefix to not be called")
	}

	req = httptest.NewRequest(http.MethodGet, "/prefix/hello", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if helloCalled {
		t.Fatal("expected hello handler to be called")
	}
	if !prefixCalled {
		t.Fatal("expected prefix to be called")
	}
}

func TestTrailingSlash(t *testing.T) {
	const wantCode = http.StatusCreated

	for _, tc := range [...]struct {
		name     string
		path     string
		requests []string
	}{
		{
			name:     "no_trailing_slash",
			path:     "/hello",
			requests: []string{"/hello", "/hello/"},
		},
		{
			name:     "trailing_slash",
			path:     "/hello/",
			requests: []string{"/hello", "/hello/"},
		},
		{
			name:     "strict_ending",
			path:     "/hello/{$}",
			requests: []string{"/hello", "/hello/"},
		},
		{
			name:     "wildcard",
			path:     "/{hello}",
			requests: []string{"/hello", "/hello/"},
		},
		{
			name:     "wildcard_trailing_slash",
			path:     "/{hello}/",
			requests: []string{"/hello", "/hello/"},
		},
		{
			name:     "wildcard_strict_ending",
			path:     "/{hello}/{$}",
			requests: []string{"/hello", "/hello/"},
		},
		{
			name:     "remainder",
			path:     "/{hello...}",
			requests: []string{"/hello", "/hello/"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := muxr.NewRouter()
			var called int
			r.Get(tc.path, func(w http.ResponseWriter, r *http.Request) {
				called++
				w.WriteHeader(wantCode)
			})

			for _, path := range tc.requests {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != wantCode {
					t.Fatalf("expected status code %d but got %d", wantCode, w.Code)
				}
			}
			if called != 2 {
				t.Fatalf("expected the handler to be called %d times but got %d", len(tc.requests), called)
			}
		})
	}
}

func TestHandleAll(t *testing.T) {
	const wantCode = http.StatusCreated

	for _, tc := range [...]struct {
		name     string
		path     string
		requests []string
	}{
		{
			name:     "catch_all",
			path:     "/",
			requests: []string{"/hello", "/hello/", "/a/b/c", "/"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := muxr.NewRouter()
			var called int
			r.HandleFunc(tc.path, func(w http.ResponseWriter, r *http.Request) {
				called++
				w.WriteHeader(wantCode)
			})

			for _, path := range tc.requests {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != wantCode {
					t.Fatalf("%s expected status code %d but got %d", path, wantCode, w.Code)
				}
			}
			if called != len(tc.requests) {
				t.Fatalf("expected the handler to be called %d times but got %d", len(tc.requests), called)
			}
		})
	}
}
