// Package cobalt represents a small web toolkit to allow the building of web applications.
// It is primarily intended to be used for JSON based API. It has routing, pre-filters and post filters.
//
// Pre-filters are called after the router identifies the proper route and before the user code (handler) is called.
// Pre-filters allow you to write to the response and end the request chain by returning a value of true from the filter handler.
//
// Route-Filters allow you to write a filter for a specific route. Pre-filters and route-filters return a boolean indicating whether to
// continueing processing the request or to exit. So when a filter returns false the request will end. If a filter returns true it will continue
// processing the request.
//
// Post filters allow you to specify a handler that gets called after the user code (handler) is run.
//
// Context contains the http request and response writer. It also allows parameters to be added to the context as well. Context is passed to
// all prefilters, route handler and post filters. Context contains helper methods to extract the route parameters from the request.
package cobalt

import (
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"bitbucket.org/ardanlabs/cobalt/httprouter"
)

const (
	// GetMethod is a http GET
	GetMethod = "GET"

	// PostMethod is a http POST
	PostMethod = "POST"

	// PutMethod is a http PUT
	PutMethod = "PUT"

	// DeleteMethod is a http DELETE
	DeleteMethod = "DELETE"

	// OptionsMethod is a http OPTIONS
	OptionsMethod = "OPTIONS"

	// HeadMethod is a http HEAD
	HeadMethod = "HEAD"
)

type (
	// Coder is the interface used for the encoder in Cobalt. It allows the use
	// of multiple Encoders within cobalt
	Coder interface {
		Encode(w io.Writer, v interface{}) error
		Decode(r io.Reader, v interface{}) error
		ContentType() string
	}

	// Cobalt is the main data structure that holds all the filters, pointer to routes
	Cobalt struct {
		router      *httprouter.Router
		prefilters  []FilterHandler
		postfilters []Handler
		serverError Handler
		coder       Coder
	}

	// Handler represents a request handler that is called by cobalt
	Handler func(c *Context)

	// FilterHandler is the handler that all pre and route filters implement
	FilterHandler func(c *Context) bool
)

// New creates a new instance of cobalt.
func New(coder Coder) *Cobalt {
	return &Cobalt{router: httprouter.New(), coder: coder}
}

// Coder returns the Coder configured in Cobalt
func (c *Cobalt) Coder() Coder {
	return c.coder
}

// AddPrefilter adds a prefilter hanlder to a dispatcher instance.
func (c *Cobalt) AddPrefilter(h FilterHandler) {
	c.prefilters = append(c.prefilters, h)
}

// AddPostfilter adds a post processing handler to a diaptcher instance.
func (c *Cobalt) AddPostfilter(h Handler) {
	c.postfilters = append(c.postfilters, h)
}

// AddServerErrHanlder add handler for server err.
func (c *Cobalt) AddServerErrHanlder(h Handler) {
	c.serverError = h
}

// AddNotFoundHandler adds a not found handler
func (c *Cobalt) AddNotFoundHandler(h Handler) {
	t := func(w http.ResponseWriter, req *http.Request) {
		ctx := NewContext(req, w, nil, c.coder)
		h(ctx)
	}

	c.router.NotFound = http.HandlerFunc(t)
}

// Get adds a route with an associated handler that matches a GET verb in a request.
func (c *Cobalt) Get(route string, h Handler, f ...FilterHandler) {
	c.addroute(GetMethod, route, h, f)
}

// Post adds a route with an associated handler that matches a POST verb in a request.
func (c *Cobalt) Post(route string, h Handler, f ...FilterHandler) {
	c.addroute(PostMethod, route, h, f)
}

// Put adds a route with an associated handler that matches a PUT verb in a request.
func (c *Cobalt) Put(route string, h Handler, f ...FilterHandler) {
	c.addroute(PutMethod, route, h, f)
}

// Delete adds a route with an associated handler that matches a DELETE verb in a request.
func (c *Cobalt) Delete(route string, h Handler, f ...FilterHandler) {
	c.addroute(DeleteMethod, route, h, f)
}

// Options adds a route with an associated handler that matches a OPTIONS verb in a request.
func (c *Cobalt) Options(route string, h Handler, f ...FilterHandler) {
	c.addroute(OptionsMethod, route, h, f)
}

// Head adds a route with an associated handler that matches a HEAD verb in a request.
func (c *Cobalt) Head(route string, h Handler, f ...FilterHandler) {
	c.addroute(HeadMethod, route, h, f)
}

// ServeHTTP implements the HandlerFunc that process the http request.
func (c *Cobalt) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// future use for middleware.
	c.router.ServeHTTP(w, req)
}

// Run runs the dispatcher which starts an http server to listen and serve.
func (c *Cobalt) Run(addr string) {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	log.SetPrefix("[cobalt] ")
	log.Printf("starting, listening on %s", addr)

	//http.Handle("/", c.Router)
	err := http.ListenAndServe(addr, c)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

// addRoute adds a route with an asscoiated method, handler and route filters.. It Builds a function which is then passed to the router.
func (c *Cobalt) addroute(method, route string, h Handler, filters []FilterHandler) {

	f := func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		ctx := NewContext(req, w, p, c.coder)

		// Handle panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("cobalt: Panic Error => %v\n", r)
				log.Printf("cobalt: Panic, Recovering\n")
				buf := make([]byte, 10000)
				runtime.Stack(buf, false)
				log.Printf("%s\n", string(buf))
				if c.serverError != nil {
					log.Printf("cobalt: Panic, Recovering")
					c.serverError(ctx)
					return
				}
			}
			log.Printf("%s => complete: %d %s %s - %s\n", ctx.ID, ctx.status, req.Method, req.RequestURI, req.RemoteAddr)
		}()

		log.Printf("%s =>  %s %s - %s", ctx.ID, req.Method, req.RequestURI, req.RemoteAddr)
		w.Header().Set("X-Request-Id", ctx.ID)
		// global filters.
		for _, pf := range c.prefilters {
			if keepGoing := pf(ctx); !keepGoing {
				return
			}
		}

		// route specific filters.
		for _, f := range filters {
			keepGoing := f(ctx)
			if !keepGoing {
				return
			}
		}

		// call route handler
		h(ctx)

		// handle any post handler filters
		for _, f := range c.postfilters {
			f(ctx)
		}
	}

	c.router.Handle(method, route, f)
}
