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
	"github.com/mssola/user_agent"
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
	// Generate files store with "statik -src=./files"
	// statikFS, err := fs.New()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// E.g.: https://owmcrn7h35.execute-api.us-east-1.amazonaws.com/v1/files/test.txt
	// https://owmcrn7h35.execute-api.us-east-1.amazonaws.com/v1/files/images/GOPHER_MIC_DROP.png
	// I no longer need this, sourcing filees from S3.
	// http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(statikFS)))
	http.HandleFunc("/", processInfo)
	log.Fatal(gateway.ListenAndServe("", nil))
}

func processInfo(w http.ResponseWriter, r *http.Request) {
	// example retrieving values from the api gateway proxy request context.
	requestContext, ok := gateway.RequestContext(r.Context())
	if !ok {
		fmt.Println("Could not process request")
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}
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

	ip := r.Header.Get("X-Forwarded-For")

	sys := r.Header.Get("User-Agent")
	ua := user_agent.New(sys)
	br, ver := ua.Browser()

	// Can't forward Host header from CloudFront from API Gateway.
	// It results in {"message":"Forbidden"}.
	// We need something like 'X-Forwarded-Host' instead.
	h := r.Header.Get("Host")

	c := r.Header.Get("CloudFront-Viewer-Country")

	d := data{
		IP:       ip,
		Country:  c,
		Platf:    ua.Platform(),
		OS:       ua.OS(),
		Browser:  br,
		Bversion: ver,
		Mob:      ua.Mobile(),
		Bot:      ua.Bot(),
		Host:     h,
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
