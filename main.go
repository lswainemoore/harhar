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
	"reflect"
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

type HARMapKey struct {
	URL string
	Method string
}


func filter(entries []Entry, fn func(Entry) bool) []Entry {
	vsf := make([]Entry, 0)
	for _, v := range entries {
		if fn(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}


func main() {
	matchRequest := func(harMap map[HARMapKey][]Entry, url string, r *http.Request) (Entry, bool) {
		// basic method: match only on full strings
		entries, found := harMap[HARMapKey{url, r.Method}]

		if !found {
			// try upgrading to https
			if strings.HasPrefix(url, "http://") {
				url = strings.Replace(url, "http://", "https://", 1)
				entries, found = harMap[HARMapKey{url, r.Method}]
			}
		}
		if !found {
			return Entry{}, false
		}

		// try to filter to matching cookies, but for now let's not be strict about it
		// (both in terms of counting what counts a matching set of cookies, 
		// and how we behave when there's no match)
		cookiesMatch := func(entry Entry) bool {
			// this is ignoring domains and paths and such,
			// so it's a little loose-y goose-y
			entryCookies := make(map[string]string)
			reqCookies := make(map[string]string) 
			for _, cookie := range entry.Request.Cookies {
				entryCookies[cookie.Name] = cookie.Value
			}
			for _, cookie := range r.Cookies() {
				reqCookies[cookie.Name] = cookie.Value
			}
			// fmt.Printf("entry cookies: %v, req cookies: %v", entryCookies, reqCookies)
			return reflect.DeepEqual(entryCookies, reqCookies)
		}
		withMatchingCookies := filter(entries, cookiesMatch)
		if len(withMatchingCookies) > 0 {
			log.Printf(
				"Found %d entries with matching cookies for %s: %v, %v", 
				len(withMatchingCookies),
				url,
				r.Cookies(),
				withMatchingCookies[0].Request.Cookies,
			)
			entries = withMatchingCookies
		} else {
			log.Printf("No entries with matching cookies for %s: %v (but proceeding anyway)", url, r.Cookies())
		}

		// TODO probably don't want to just use first with content
		// maybe something having to do with page?
		// it's pretty hard to manage though. imagine a login/logout sequence:
		// 1) user hits / with no cookie, get cookie-less view
		// 2) user POSTs to /login with data, gets 302 to / with user=blah (plus session) cookie
		// 3) user hits / with user=blah cookie, get user-specific view
		// 4) user hits /logout, gets 302 to / (and server registers the logout for session)
		// 5) user hits / still with cookie, get cookie-less view, and user= cookie back
		// how do you distinguish between 3 and 5? both involve a GET to / with a cookie.
		// presumably you try to use the order of requests, but that's pretty tricky with an async server,
		// and with all the other requests flying around.
		// for an example of this, see the login/logout behavior of Hacker News
		if r.Method == "GET" {
			for i := 0; i < len(entries); i++ {
				if len(entries[i].Response.Content.Text) > 0 {
					return entries[i], true
				}
			}
			return entries[0], true
		} else if r.Method == "POST" {
			fmt.Printf("POST request matching...: %+v\n %+v\n", r, entries)
			return entries[0], found
			// TODO match body data
		} else {
			return entries[0], false
		}
	}

	var harMap map[HARMapKey][]Entry
	var harLog HARLog

	loadHar := func(filename string) map[HARMapKey][]Entry {
		var harMap = make(map[HARMapKey][]Entry)

		// see: https://tutorialedge.net/golang/parsing-json-with-golang/
		jsonFile, _ := os.Open("hars/" + filename)
		byteValue, _ := ioutil.ReadAll(jsonFile)

		var metaLog MetaLog
		json.Unmarshal(byteValue, &metaLog)
		harLog = metaLog.HARLog

		// fill out the harMap, creating and/or appending current entry to it's key's slice
		for i := 0; i < len(harLog.Entries); i++ {
			log.Println("Entry Request URL: " + harLog.Entries[i].Request.URL)
			key := HARMapKey{harLog.Entries[i].Request.URL, harLog.Entries[i].Request.Method}
			harMap[key] = append(harMap[key], harLog.Entries[i])
		}
		
		// for i := 0; i < len(harLog.Entries); i++ {
		// 	log.Println("Entry Request URL: " + harLog.Entries[i].Request.URL)
		// 	harMap[HARMapKey{harLog.Entries[i].Request.URL, harLog.Entries[i].Request.method}] = harLog.Entries[i]
		// }
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

		if len(splittened) == 1 {
			log.Println("No rewritten_from found in uri: " + req.URL.RequestURI())
			w.WriteHeader(http.StatusNotFound)
			return
		}

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
				if origin != "" && origin != "WEDUNNO" {
					http.Redirect(w, req, uri + string(splittened[0][len(splittened[0])-1]) + "rewritten_from=" + origin, http.StatusMovedPermanently)
					return
				} else {
					log.Println("No origin found in referer: " + referer)
				}
			} 
			
			// if we still don't have an origin, we'll use the base URL from the HAR
			if origin == "WEDUNNO" || origin == "" {
				// when it doesn't, we'll use our main page's one from HAR
				// (TODO this may be iffy if we support multiple pages)
				parsedBase, _ := url.Parse(harLog.Pages[0].Title)
				origin = fmt.Sprintf("%s://%s", parsedBase.Scheme, parsedBase.Hostname())
			}
		}

		fullUrl := origin + uri

		log.Println("seeking url: " + fullUrl)

		match, found := matchRequest(harMap, fullUrl, req)
		if !found {
			log.Println("No match found for: " + fullUrl)
			fmt.Printf("%+v\n", req)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println("Matched: " + match.Request.URL)

		for i := 0; i < len(match.Response.Headers); i++ {
			if contains([]string{"accept-ranges", "content-type", "vary", "location", "set-cookie"}, strings.ToLower(match.Response.Headers[i].Name)) {
				w.Header().Set(match.Response.Headers[i].Name, match.Response.Headers[i].Value)
			}
		}

		// set the status code of response to be that of match
		w.WriteHeader(match.Response.Status)
		
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
