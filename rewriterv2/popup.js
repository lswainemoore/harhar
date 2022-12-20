const LOCALHOST = 'localhost:8000';

const rewriterRequest = (details) => {
  console.log('request time details: ', details);
  const url = new URL(details.url);
  const { origin, host, hostname, pathname, search, searchParams } = url;
  console.log({ origin, host, hostname, pathname, search, searchParams });

  var rewrittenFrom;

  if (host == LOCALHOST) {
    // two possibilities:
    // 1. the url is already rewritten, 
    //    in which case we don't want to touch it
    if (searchParams.has('rewritten_from')) {
      return
    }

    // 2. the url was provided as a relative path, and so this is assuming localhost as domain,
    //    in which we want to tell the backend to "figure it out" wrt host. 
    //    so we'll drop a dummy host here.
    rewrittenFrom = 'WEDUNNO';
  }
  else {
    rewrittenFrom = origin;
  }

  // const newUrl = new URL(`http://${LOCALHOST}/`);
  // newUrl.pathname = pathname;
  // newUrl.search = search;
  // url.searchParams.set('rewritten_from', rewrittenFrom);

  // sucks that we need to do this here (it'd be nicer to do above)
  // but we have the same problem we have on backend around this re-encoding 
  // stuff that wasn't url-encoded in the first place (happens when you mess w search params)
  // interestingly, this issue we see on backend around changing of param order doesn't
  // seem to happen here (implies there's some sorting to the searchParams)
  // TODO same solution as backend: normalize URLs
  url.host = LOCALHOST;
  url.protocol = 'http';
  // what we want here depends on whether the url already has a query string
  var sep = '?';
  if (url.search) {
    sep = '&';
  }
  // careful! we do want to url-encode our own parameter, just in case.
  url.search = url.search + sep + 'rewritten_from=' + encodeURIComponent(rewrittenFrom);
  console.log('redirecting to: ', url.toString());
  
  return {
    redirectUrl: url.toString(),
  }
}

const startRewriting = () => {
  console.log('starting rewriting...');
  chrome.tabs.query(
    { currentWindow: true, active: true },
    (tabs) => {
      console.log(tabs);
      chrome.webRequest.onBeforeRequest.addListener(
        rewriterRequest,
        {
          urls: ["<all_urls>"],
          tabId: tabs[0].id
        },
        ["blocking", "requestBody", "extraHeaders"]
      );
    }
  );
};

const stopRewriting = () => {
  console.log('stopping rewriting...');
  chrome.webRequest.onBeforeRequest.removeListener(rewriterRequest);
};

window.onload = function() {
  window.document.querySelector('#start-rewriting').addEventListener('click', startRewriting);
  window.document.querySelector('#stop-rewriting').addEventListener('click', stopRewriting);
};