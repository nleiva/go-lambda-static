package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/apex/gateway"
	_ "github.com/nleiva/go-lambda-static/statik"
	"github.com/rakyll/statik/fs"
)

type data struct {
	IP       string `json:"This is your IP address"`
	Country  string `json:"You are visiting us from"`
	Platf    string `json:"Platform"`
	OS       string `json:"OS"`
	Browser  string `json:"Browser"`
	Bversion string `json:"Browser Version"`
	Host     string `json:"Target host"`
	Mob      bool   `json:"Mobile"`
	Bot      bool   `json:"Bot"`
}

func main() {
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	// E.g.: https://owmcrn7h35.execute-api.us-east-1.amazonaws.com/v1/files/test.txt
	// https://owmcrn7h35.execute-api.us-east-1.amazonaws.com/v1/files/images/GOPHER_MIC_DROP.png
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(statikFS)))
	http.HandleFunc("/", hello)
	log.Fatal(gateway.ListenAndServe("", nil))
}

func hello(w http.ResponseWriter, r *http.Request) {
	// example retrieving values from the api gateway proxy request context.
	requestContext, ok := gateway.RequestContext(r.Context())
	fmt.Printf("Processing request data for request %s, from IP %s.\n",
		requestContext.RequestID,
		requestContext.Identity.SourceIP)

	check := func(err error) {
		if err != nil {
			// Log the detailed error
			fmt.Println(err.Error())
			// Return a generic "Internal Server Error" message
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}
	}

	statikFS, err := fs.New()
	check(err)

	// Return a 404 if the requested template doesn't exist.
	// HARDCODED for now -> 500
	fp, err := statikFS.Open(filepath.Join("/templates", "example.html"))
	check(err)
	defer fp.Close()

	lp, err := statikFS.Open("/templates/layout.html")
	check(err)
	defer lp.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(lp)
	lps := buf.String()

	// https://play.golang.org/p/DUkUAHdIGo3
	t, err := template.New("base").Parse(lps)
	check(err)

	buf = new(bytes.Buffer)
	buf.ReadFrom(fp)
	fps := buf.String()

	tmpl, err := t.New("layout").Parse(fps)
	check(err)

	d := data{
		IP: "1.1.1.1",
	}

	var b strings.Builder
	err = tmpl.ExecuteTemplate(&b, "layout", d)
	check(err)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, err = io.WriteString(w, b.String())
	check(err)

	if !ok || requestContext.Authorizer["sub"] == nil {
		// fmt.Fprint(w, "Hello World from Go")
		return
	}

	userID := requestContext.Authorizer["sub"].(string)
	fmt.Fprintf(w, "Hello %s from Go", userID)
}
