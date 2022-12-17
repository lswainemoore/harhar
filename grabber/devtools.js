// helpful article, including on how to access logs:
// https://www.raymondcamden.com/2012/07/15/How-to-add-a-panel-to-Chrome-Dev-Tools

chrome.devtools.panels.create(
  "Grabber",
  "todo icon",
  "devtools.html",
  function(panel) {
    console.log("panel created");
    panel.onShown.addListener(function (panelWindow) {
      // seems to be important to add listener here:
      // see: https://stackoverflow.com/questions/11624307/how-to-modify-content-under-a-devtools-panel-in-a-chrome-extension
      panelWindow.document.querySelector('#download').addEventListener('click', downloadHar);
    });
  }
)

const downloadHar = () => {
  chrome.devtools.network.getHAR(
    function (harLog) {
      const contentFetchs = harLog.entries.map(request => {
        // boy, what a pain that getContent doesn't return a promise...
        // see: https://stackoverflow.com/a/36072263
        var promiseResolve, promiseReject;
        var promise = new Promise(function(resolve, reject){
          promiseResolve = resolve;
          promiseReject = reject;
        });
        request.getContent(
          function (content, encoding) {
            request.response.content.text = content;
            if (encoding) {
              request.response.content.encoding = encoding;
            }
            promiseResolve();
          }
        );
        return promise;
      });
      // only download after all the content has been fetched
      Promise.all(contentFetchs).then(() => {
        var blob = new Blob(
          [JSON.stringify({"log": harLog})],
          {
            // doing this allows up to specify the file extension ourselves
            // see: https://stackoverflow.com/a/63046720
            type: 'application/octet-stream'
          }
        );
        var url = URL.createObjectURL(blob);
        chrome.downloads.download({
          url: url,
          filename: "har.har"
        });
      });
    }
  );
}
