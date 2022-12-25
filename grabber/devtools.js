// helpful article, including on how to access logs:
// https://www.raymondcamden.com/2012/07/15/How-to-add-a-panel-to-Chrome-Dev-Tools


// our approach to capturing this stuff is a little funny.
// unfortunately, it's not as simple as just saving the result of getHar
// when we click download, because 
// 1) we rely on the "Preserve Log" box being checked
// 2) even when it is, calls to getResponse for older page's requests seem to fail
// so, instead we save every request, and get its content when as that happens
// and save the "pages" components on every request. then we combine these to make a single HAR.
// it would be better to only save the HAR log when we change pages (aka when the network
// tab gets wiped), but we'll maybe come back to that later.
const pages = {};
const harHeaders = {};
const entries = [];

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
      panelWindow.document.querySelector('#start').addEventListener('click', (event) => {
        chrome.devtools.network.onRequestFinished.addListener( saveRequest );
        event.target.disabled = true;
        panelWindow.document.querySelector('#stop').disabled = false;
      });
      panelWindow.document.querySelector('#stop').addEventListener('click', (event) => {
        chrome.devtools.network.onRequestFinished.removeListener( saveRequest );
        event.target.disabled = true;
        panelWindow.document.querySelector('#start').disabled = false;
      });
    });
  }
);

const saveRequest = (request) => {
  chrome.devtools.network.getHAR(
    function(har) {
      har.pages.forEach(page => {
        pages[page.id] = page;
      });
      harHeaders['version'] = har.version;
      harHeaders['browser'] = har.creator;
    }
  )

  // console.log('saving request: ', request)
  entries.push(request);

  // boy, what a pain that getContent doesn't return a promise...
  // see: https://stackoverflow.com/a/36072263
  var promiseResolve, promiseReject;
  var promise = new Promise(function(resolve, reject){
    promiseResolve = resolve;
    promiseReject = reject;
  });

  request.getContent(
    function (content, encoding) {
      try {
        request.response.content.text = content;
        if (encoding) {
          request.response.content.encoding = encoding;
        }
        promiseResolve();
      } catch (e) {
        promiseReject(e);
      }
    }
  )
  request.promise = promise;
};

const downloadHar = () => {
  console.log('attempting downloadHar: ', entries);
  promises = entries.map(request => request.promise);
  Promise.all(promises).then(() => {
    console.log('received all content: ', entries);

    const fullHar = {
      log: {
        version: "1.2",
        creator: {
          name: "Grabber",
          version: "1.0",
        },
        browser: harHeaders.browser,
        pages: Object.values(pages),
        entries: entries,
      }
    }

    var blob = new Blob(
      [JSON.stringify(fullHar)],
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
  // chrome.devtools.network.getHAR(
  //   function (harLog) {
  //     console.log('received harLog: ', harLog)
  //     const contentFetchs = harLog.entries.map(request => {
  //       // boy, what a pain that getContent doesn't return a promise...
  //       // see: https://stackoverflow.com/a/36072263
  //       var promiseResolve, promiseReject;
  //       var promise = new Promise(function(resolve, reject){
  //         promiseResolve = resolve;
  //         promiseReject = reject;
  //       });
  //       if (request._resourceType !== "ping") {
  //         request.getContent(
  //           function (content, encoding) {
  //             try {
  //               request.response.content.text = content;
  //               if (encoding) {
  //                 request.response.content.encoding = encoding;
  //               }
  //               promiseResolve();
  //             } catch (e) {
  //               promiseReject(e);
  //             }
  //           }
  //         )
  //       } else {
  //         promiseResolve();
  //       }
  //       request.promise = promise;
  //       return promise;
  //     });
  //     console.log('promises: ', contentFetchs)
  //     // only download after all the content has been fetched
  //     Promise.all(contentFetchs).then(() => {
  //       console.log('received all content: ', harLog)
  //       var blob = new Blob(
  //         [JSON.stringify({"log": harLog})],
  //         {
  //           // doing this allows up to specify the file extension ourselves
  //           // see: https://stackoverflow.com/a/63046720
  //           type: 'application/octet-stream'
  //         }
  //       );
  //       var url = URL.createObjectURL(blob);
  //       chrome.downloads.download({
  //         url: url,
  //         filename: "har.har"
  //       });
  //     });
  //   }
  // );
}
