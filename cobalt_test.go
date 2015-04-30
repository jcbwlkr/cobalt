package cobalt

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"bitbucket.org/ardanlabs/msgpack"
)

var r = map[int][]string{
	1: []string{"/", "Get"},
	2: []string{"/foo", "Get"},
	3: []string{"/", "Post"},
	4: []string{"/foo", "Post"},
	5: []string{"/", "Put"},
	6: []string{"/foo", "Put"}}

func newRequest(method, path string, body io.Reader) *http.Request {
	r, _ := http.NewRequest(method, path, body)
	u, _ := url.Parse(path)
	r.URL = u
	r.RequestURI = path
	return r
}

// Test_PreFilters tests pre-filters
func Test_PreFilters(t *testing.T) {
	//setup request
	r := newRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	data := "PREFILTER"
	// pre filters
	c := New(&JSONEncoder{})

	c.AddPrefilter(func(ctx *Context) bool {
		ctx.SetData("PRE", data)
		return true
	})

	c.Get("/", func(ctx *Context) {
		v := ctx.GetData("PRE")
		if v != data {
			t.Errorf("expected %s got %s", data, v)
		}
		ctx.Response.Write([]byte(data))
	})

	c.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("expected status code to be 200 instead got %d", w.Code)
	}
	if w.Body.String() != data {
		t.Errorf("expected body to be %s instead got %s", data, w.Body.String())
	}
}

// Test_PreFiltersExit tests pre-filters stopping the request.
func Test_PreFiltersExit(t *testing.T) {
	r := newRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	data := "PREFILTER_EXIT"
	code := http.StatusBadRequest
	c := New(&JSONEncoder{})

	c.AddPrefilter(func(ctx *Context) bool {
		ctx.Response.WriteHeader(code)
		ctx.Response.Write([]byte(data))
		return false
	})

	c.Get("/", func(ctx *Context) {
		v := ctx.GetData("PRE")
		if v != data {
			t.Errorf("expected %s got %s", data, v)
		}
		ctx.Response.Write([]byte(data))
	}, nil)

	c.ServeHTTP(w, r)

	if w.Code != code {
		t.Errorf("expected status code to be 200 instead got %d", w.Code)
	}
	if w.Body.String() != data {
		t.Errorf("expected body to be %s instead got %s", data, w.Body.String())
	}
}

// Test_Routes tests the routing of requests.
func Test_Routes(t *testing.T) {
	c := New(&JSONEncoder{})

	// GET
	c.Get("/", func(ctx *Context) {
		ctx.Response.Write([]byte("Get/"))
	})
	c.Get("/foo", func(ctx *Context) {
		ctx.Response.Write([]byte("Get/foo"))
	})

	// POST
	c.Post("/", func(ctx *Context) {
		ctx.Response.Write([]byte("Post/"))
	})
	c.Post("/foo", func(ctx *Context) {
		ctx.Response.Write([]byte("Post/foo"))
	})

	// PUT
	c.Put("/", func(ctx *Context) {
		ctx.Response.Write([]byte("Put/"))
	})
	c.Put("/foo", func(ctx *Context) {
		ctx.Response.Write([]byte("Put/foo"))
	})

	// Delete
	c.Delete("/", func(ctx *Context) {
		ctx.Response.Write([]byte("Delete/"))
	})
	c.Delete("/foo", func(ctx *Context) {
		ctx.Response.Write([]byte("Delete/foo"))
	})

	// OPTIONS
	c.Options("/", func(ctx *Context) {
		ctx.Response.Write([]byte("Options/"))
	})
	c.Options("/foo", func(ctx *Context) {
		ctx.Response.Write([]byte("Options/foo"))
	})

	// HEAD
	c.Head("/", func(ctx *Context) {
		ctx.Response.Write([]byte("Head/"))
	})
	c.Head("/foo", func(ctx *Context) {
		ctx.Response.Write([]byte("Head/foo"))
	})

	for _, v := range r {
		AssertRoute(v[0], v[1], c, t)
	}
}

// Test_RouteFiltersSettingData tests route filters setting data and passing it to handlers.
func Test_RouteFiltersSettingData(t *testing.T) {

	//setup request
	r := newRequest("GET", "/RouteFilter", nil)
	w := httptest.NewRecorder()

	// test route filter setting
	data := "ROUTEFILTER"

	c := New(&JSONEncoder{})

	c.Get("/RouteFilter",

		func(ctx *Context) {
			v := ctx.GetData("PRE")
			if v != data {
				t.Errorf("expected %s got %s", data, v)
			}
			ctx.Response.Write([]byte(data))
		},

		func(c *Context) bool {
			c.SetData("PRE", data)
			return true
		})

	c.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("expected status code to be 200 instead got %d", w.Code)
	}
	if w.Body.String() != data {
		t.Errorf("expected body to be %s instead got %s", data, w.Body.String())
	}
}

// Test_RouteFilterExit tests route filters stopping the request.
func Test_RouteFilterExit(t *testing.T) {
	data := "ROUTEFILTEREXIT"
	//setup request
	r := newRequest("GET", "/RouteFilter", nil)
	w := httptest.NewRecorder()

	c := New(&JSONEncoder{})

	c.Get("/RouteFilter",

		func(ctx *Context) {
			v := ctx.GetData("PRE")
			if v != data {
				t.Errorf("expected %s got %s", data, v)
			}
			ctx.Response.Write([]byte("FOO"))
		},

		func(ctx *Context) bool {
			ctx.Response.WriteHeader(http.StatusUnauthorized)
			ctx.Response.Write([]byte(data))
			return false
		})

	c.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status code to be %d instead got %d", http.StatusUnauthorized, w.Code)
	}
	if w.Body.String() != data {
		t.Errorf("expected body to be %s instead got %s", data, w.Body.String())
	}
}

// AsserRoute is a helper method to tests routes
func AssertRoute(path, verb string, c *Cobalt, t *testing.T) {
	r := newRequest(strings.ToUpper(verb), path, nil)
	w := httptest.NewRecorder()

	c.ServeHTTP(w, r)
	if w.Body.String() != verb+path {
		t.Errorf("expected body to be %s instead got %s", verb+path, w.Body.String())
	}
}

func Test_GroupRoutes(t *testing.T) {

}

func Test_PostFilters(t *testing.T) {
}

func Test_NotFoundHandler(t *testing.T) {
	//setup request
	r := newRequest("GET", "/FOO", nil)
	w := httptest.NewRecorder()

	m := struct{ Message string }{"Not Found"}

	nf := func(c *Context) {
		c.ServeWithStatus(m, http.StatusNotFound)
	}

	c := New(&JSONEncoder{})
	c.AddNotFoundHandler(nf)

	c.Get("/",
		func(ctx *Context) {
			panic("Panic Test")
		},
		nil)

	c.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status code to be 404 instead got %d", w.Code)
	}

	var msg struct{ Message string }
	json.Unmarshal([]byte(w.Body.String()), &msg)

	if msg.Message != m.Message {
		t.Errorf("expected body to be %s instead got %s", msg.Message, m.Message)
	}
}

func Test_ServerErrorHandler(t *testing.T) {
	//setup request
	r := newRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	m := struct{ Message string }{"Internal Error"}

	se := func(c *Context) {
		c.ServeWithStatus(m, http.StatusInternalServerError)
	}

	c := New(&JSONEncoder{})
	c.AddServerErrHanlder(se)

	c.Get("/",
		func(ctx *Context) {
			panic("Panic Test")
		},
		nil)

	c.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status code to be 500 instead got %d", w.Code)
	}

	var msg struct{ Message string }
	json.Unmarshal([]byte(w.Body.String()), &msg)

	if msg.Message != m.Message {
		t.Errorf("expected body to be %s instead got %s", msg.Message, m.Message)
	}
}

type JSONEncoder struct{}

func (enc JSONEncoder) Encode(w io.Writer, val interface{}) error {
	return json.NewEncoder(w).Encode(val)
}

func (enc JSONEncoder) Decode(r io.Reader, val interface{}) error {
	return json.NewDecoder(r).Decode(val)
}

func (enc JSONEncoder) ContentType() string {
	return "application/json;charset=UTF-8"
}

type MPackEncoder struct{}

func (enc MPackEncoder) Encode(w io.Writer, val interface{}) error {
	return msgpack.NewEncoder(w).Encode(val)
}

func (enc MPackEncoder) Decode(r io.Reader, val interface{}) error {
	return msgpack.NewDecoder(r).Decode(val)
}

func (enc MPackEncoder) ContentType() string {
	return "application/x-msgpack"
}
