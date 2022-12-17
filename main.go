package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	// "strconv"
	"strings"
)

type MetaLog struct {
	HARLog HARLog `json:"log"`
}

type HARLog struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Pages   []Page  `json:"pages"`
	Entries []Entry `json:"entries"`
}

type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Page struct {
	StartedDateTime string      `json:"startedDateTime"`
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	PageTimings     PageTimings `json:"pageTimings"`
}

type PageTimings struct {
	OnContentLoad float64 `json:"onContentLoad"`
	OnLoad        float64 `json:"onLoad"`
}

type Entry struct {
	Cache           Cache    `json:"cache"`
	Connection      string   `json:"connection"`
	Pageref         string   `json:"pageref"`
	Request         Request  `json:"request"`
	Response        Response `json:"response"`
	ServerIPAddress string   `json:"serverIPAddress"`
	StartedDateTime string   `json:"startedDateTime"`
	Time            float64  `json:"time"`
	Timings         Timings  `json:"timings"`
}

type Cache struct {
}

type Timings struct {
	Blocked float64 `json:"blocked"`
	DNS     int     `json:"dns"`
	SSL     int     `json:"ssl"`
	Connect int     `json:"connect"`
	Send    int     `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

type Request struct {
	Method      string        `json:"method"`
	URL         string        `json:"url"`
	HTTPVersion string        `json:"httpVersion"`
	Headers     []Header      `json:"headers"`
	QueryString []QueryString `json:"queryString"`
	Cookies     []Cookie      `json:"cookies"`
	HeaderSize  int           `json:"headerSize"`
	BodySize    int           `json:"bodySize"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type QueryString struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	Domain   string `json:"domain"`
	Expires  string `json:"expires"`
	HTTPOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
}

type Response struct {
	Status      int      `json:"status"`
	StatusText  string   `json:"statusText"`
	HTTPVersion string   `json:"httpVersion"`
	Headers     []Header `json:"headers"`
	Cookies     []Cookie `json:"cookies"`
	Content     Content  `json:"content"`
	RedirectURL string   `json:"redirectURL"`
	HeadersSize int      `json:"headersSize"`
	BodySize    int      `json:"bodySize"`
}

type Content struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
	Encoding string `json:"encoding"`
}

// see: https://stackoverflow.com/a/10485970
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func main() {
	// see: https://tutorialedge.net/golang/parsing-json-with-golang/
	jsonFile, _ := os.Open("mything.har")
	byteValue, _ := ioutil.ReadAll(jsonFile)

	var metaLog MetaLog
	json.Unmarshal(byteValue, &metaLog)
	var harLog = metaLog.HARLog

	var harMap = make(map[string]Entry)

	for i := 0; i < len(harLog.Entries); i++ {
		log.Println("Entry Request URL: " + harLog.Entries[i].Request.URL)
		harMap[harLog.Entries[i].Request.URL] = harLog.Entries[i]
	}

	matchRequest := func(harMap map[string]Entry, url string) (Entry, bool) {
		// basic method: match only on full strings
		val, found := harMap[url]
		return val, found
	}

	// see: https://www.wolfe.id.au/2020/03/10/starting-a-go-project/

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		log.Println("URL: " + req.URL.RequestURI())

		// TODO this is quite ugly. we'd really rather use
		// `url := req.URL.Query().Get("url")` but my extension doesn't
		// properly URL encode the query param, so subsequent params get chopped
		// when we access it like that.
		splittened := strings.Split(req.URL.RequestURI(), "/?url=")
		var url string
		if len(splittened) >= 2 {
			url = strings.Split(req.URL.RequestURI(), "/?url=")[1]
		} else {
			log.Println("ill-formatted url: " + req.URL.RequestURI())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		match, found := matchRequest(harMap, url)
		if !found {
			log.Println("No match found for: " + url)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println("Matched: " + match.Request.URL)

		for i := 0; i < len(match.Response.Headers); i++ {
			if contains([]string{"accept-ranges", "content-type", "vary"}, match.Response.Headers[i].Name) {
				w.Header().Set(match.Response.Headers[i].Name, match.Response.Headers[i].Value)
			}
		}

		content := match.Response.Content.Text
		if match.Response.Content.Encoding == "base64" {
			decoded, _ := base64.StdEncoding.DecodeString(content)
			content = string(decoded)
		}

		// TODO we'd like to be able to set this, but it's proving difficult for images
		// w.Header().Set("Content-Length", strconv.Itoa(len(match.Response.Content.Text)))

		io.WriteString(w, content)
	}

	http.HandleFunc("/", helloHandler)
	log.Println("Listing for requests at http://localhost:8000/")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
