package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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

type LoadHARRequest struct {
	Filename string `json:"filename"`
}


func main() {
	matchRequest := func(harMap map[string]Entry, url string) (Entry, bool) {
		// basic method: match only on full strings
		val, found := harMap[url]

		if !found {
			// try upgrading to https
			if strings.HasPrefix(url, "http://") {
				url = strings.Replace(url, "http://", "https://", 1)
				val, found = harMap[url]
			}
		}
		return val, found
	}

	var harMap map[string]Entry

	loadHar := func(filename string) map[string]Entry {
		var harMap = make(map[string]Entry)

		// see: https://tutorialedge.net/golang/parsing-json-with-golang/
		jsonFile, _ := os.Open("hars/" + filename)
		byteValue, _ := ioutil.ReadAll(jsonFile)

		var metaLog MetaLog
		json.Unmarshal(byteValue, &metaLog)
		var harLog = metaLog.HARLog

		parsedBase, _ := url.Parse(harLog.Pages[0].Title)
		baseUrl := fmt.Sprintf("%s://%s", parsedBase.Scheme, parsedBase.Hostname())
		log.Println("Base URL: " + baseUrl)

		for i := 0; i < len(harLog.Entries); i++ {
			log.Println("Entry Request URL: " + harLog.Entries[i].Request.URL)
			harMap[harLog.Entries[i].Request.URL] = harLog.Entries[i]
		}
		return harMap
	}

	// harMap = loadHar("archive.har")
	
	loadHARHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
	
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
	
		var req LoadHARRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}
	
		fmt.Println("loading: " + req.Filename)
		harMap = loadHar(req.Filename)
	
		w.Write([]byte("OK"))
	}

	// see: https://www.wolfe.id.au/2020/03/10/starting-a-go-project/
	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		log.Println("received URL: " + req.URL.RequestURI())

		origin := req.URL.Query().Get("rewritten_from")

		// // little unpleasant in go to remove a query param
		// // see: https://johnweldon.com/blog/quick-tip-remove-query-param-from-url-in-go/
		// params := req.URL.Query()
		// params.Del("rewritten_from")
		// req.URL.Query().Del("rewritten_from")
		// req.URL.RawQuery = params.Encode()
		// uri := req.URL.RequestURI()

		// this is pretty gnarly. it would be much better to do it like above,
		// but unfortunately our HAR archives are not necessarily going to have
		// parameters url-encoded, and since the above re-writes and thus encodes them,
		// we'll have some misses.
		// actually, along the same lines, there's a different problem, which is the
		// ordering of params may not be the same when we re-encoded.
		// TODO solution: normalize URLs when reading archive, and here.
		splittened := strings.Split(req.URL.RequestURI(), "rewritten_from=")
		// remove the last character, which is either a `&` or a `?`
		// (depends whether there were other params)
		uri := splittened[0][:len(splittened[0])-1]

		if origin == "WEDUNNO" {
			// generally, this will have a referer header, which has a origin attached to it
			referer := req.Header.Get("Referer")
			if referer != "" {
				refererUrl, _ := url.Parse(referer)
				origin = refererUrl.Query().Get("rewritten_from")

				// redirect with a new rewritten_from with our determined origin
				// (this is important so that future requests can figure out replacement
				// for WEDUNNO in the same way we did here)
				http.Redirect(w, req, uri + string(splittened[0][len(splittened[0])-1]) + "rewritten_from=" + origin, http.StatusMovedPermanently)
				return
			} else {
				// when it doesn't, we'll use our main page's one from HAR
				// (TODO this will change if we support multiple pages)
				// origin = baseUrl
			}
		}

		fullUrl := origin + uri

		log.Println("seeking url: " + fullUrl)

		match, found := matchRequest(harMap, fullUrl)
		if !found {
			log.Println("No match found for: " + fullUrl)
			fmt.Printf("%+v\n", req)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println("Matched: " + match.Request.URL)

		for i := 0; i < len(match.Response.Headers); i++ {
			if contains([]string{"accept-ranges", "content-type", "vary"}, strings.ToLower(match.Response.Headers[i].Name)) {
				w.Header().Set(match.Response.Headers[i].Name, match.Response.Headers[i].Value)
			}
		}
		
		// fmt.Printf("%+v\n", w)

		content := match.Response.Content.Text
		if match.Response.Content.Encoding == "base64" {
			decoded, _ := base64.StdEncoding.DecodeString(content)
			content = string(decoded)
		}

		// TODO we'd like to be able to set this, but it's proving difficult for images
		// w.Header().Set("Content-Length", strconv.Itoa(len(match.Response.Content.Text)))

		io.WriteString(w, content)
	}

	http.HandleFunc("/loadHAR", loadHARHandler)
	http.HandleFunc("/", helloHandler)
	log.Println("Listing for requests at http://localhost:8000/")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
