`go run main.go`

TODO
- mess with tabStates a big...not quite defined right
- investigate going back to webRequestBlocking (need to do manifest v2 if on chrome, which will upset rest of stuff). but is supported on firefox...
- figure out what the deal is with the redirects that are getting made out to be relative paths, when the browser ultimately gets them from CDN. could it have to do with `<script id="webpack-public-path" type="text/uri-list">https://cdn.sstatic.net/</script>`? or is it because of the `../..` here: `background:url('../../Img/filter-sprites.png?v=25267dbcd657')` ([src](https://cdn.sstatic.net/Sites/stackoverflow/primary.css?v=c05ce93d5306))
- make the base64/gzip stuff not gross
x extension for saving .har files (explore side)
- make the extension url-encode full url param so we can parse it properly in request
- improved matching
  x http vs https
- content-length
- POST/etc.
x fonts not working well.
- multiple pages
- combine extensions
  - perhaps this ought to save the har on backend as we click around
- reduce dependence on "rewritten_from" sep string
- test/fix for firefox (longer maintennance of relevant v2 APIs...)
- i think this should be converted away from a popup script, unfortunately
  - since it prevents actions when popup closes. stuff should probably live in background script instead, though then have to sort out the toggling action. maybe it should be a devtools script, because that persists a little better... 
- normalize urls (mostly query param issues) on ingestion
- some things i'm not sure what to do about:
  - sometimes scripts (particularly analytics ones) attach stuff to the URIs they request randomly or using time, or something else computed in js. this means we don't recognize these. maybe the solution here is just to be lax about end of stuff, or use a heuristic or something
  - sometimes a script will do something that depends on it using https (e.g. one of the tracking scripts does some funky stuff with an SSL subdomain [here](https://static.www.calottery.com/-/media/Base-Themes/Main-Theme/scripts/tracking.js?rev=dc6dddae1bca404db5fb59c0fe175fbf)). should we be https when running? what if another script needs us to be http??
  - in both of above cases, i'm not really worried about the behavior too much from what i've seen, since it's pretty much just affecting analytics scripts. but if it were to not...the first case in particular seems basically impossible to solve.
- so much more....