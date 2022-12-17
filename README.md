`go run main.go`

TODO
- mess with tabStates a big...not quite defined right
- investigate going back to webRequestBlocking (need to do manifest v2 if on chrome, which will upset rest of stuff). but is supported on firefox...
- figure out what the deal is with the redirects that are getting made out to be relative paths, when the browser ultimately gets them from CDN. could it have to do with `<script id="webpack-public-path" type="text/uri-list">https://cdn.sstatic.net/</script>`? or is it because of the `../..` here: `background:url('../../Img/filter-sprites.png?v=25267dbcd657')` ([src](https://cdn.sstatic.net/Sites/stackoverflow/primary.css?v=c05ce93d5306))
- make the base64/gzip stuff not gross
- extension for saving .har files (explore side)
- make the extension url-encode full url param so we can parse it properly in request
- improved matching
- content-length
- POST/etc.
- fonts not working well.
- so much more....