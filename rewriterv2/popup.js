const LOCALHOST = 'localhost:8000';

// annoyingly, need to maintain some state between the two callbacks.
// this is because the header-adjusting stage doesn't have access to what
// was done in the redirect stage.
// we'll solve this by just computing the correct headers at the first stage
// and saving them here. then we can just pull from that in the second stage.
// (note: this wouldn't make sense if we were trying to use the existing headers
// when computing our new ones.)
const newHeadersByRequest = {}

const rewriterRequest = (details) => {
  console.log('request time details: ', details);
  const url = new URL(details.url);
  const { origin, host, hostname, pathname, search, searchParams } = url;
  console.log({ origin, host, hostname, pathname, search, searchParams });

  // TODO this may also need to handle "no host" case
  if (host == LOCALHOST) {
    // two possibilities:
    // 1. the url is already rewritten, 
    //    in which case we don't want to touch it
    if (details.requestId in newHeadersByRequest) {
      return
    }

    // 2. the url was provided as a relative path, and so this is assuming localhost as domain,
    //    in which we want to tell the backend to "figure it out" wrt host. 
    //    so we'll drop a dummy host here.
    newHeadersByRequest[details.requestId] = [{name: 'old-origin', value: 'WEDUNNO'}]
    
    // and then we don't actually need to redirect anywhere
    return
  }

  // if we're here, there's a non-LOCALHOST domain
  newHeadersByRequest[details.requestId] = [{name: 'old-origin', value: origin}]
  const newUrl = new URL(`http://${LOCALHOST}/`);
  newUrl.pathname = pathname;
  newUrl.search = search;

  console.log('redirecting to: ', newUrl.toString());
  
  return {
    redirectUrl: newUrl.toString(),
  }

  // // this is the backwards compatible-ish version
  // // encode a url with base "http://localhost:8000/" and parameters {rewritten: true, url:[url from above]}
  // const newUrl = new URL(`http://${LOCALHOST}/`);
  // newUrl.searchParams.set('rewritten', true);
  // newUrl.searchParams.set('url2', url);

  // // condition 1: if the url is already rewritten, do nothing
  // if (searchParams.get('rewritten') === 'true') {
  //   return;
  // }

  // // condition 2: if the host is localhost, make sure to note that we need
  // // to make it absolute for something else.
  // else if (host == LOCALHOST) {
  //   newUrl.searchParams.set('must_correct_domain', true);
  // }

  // console.log(newUrl);
  
  // return {
  //   redirectUrl: newUrl.toString(),
  //   requestHeaders: [{name: 'X-blah', value: 'test'}]
  // }


  // now for a version that isn't backward compatible and attempts to solve
  // for the relative path issue

}

const rewriterHeaders = (details) => {
  console.log('header time details: ', details);
  for (header of newHeadersByRequest[details.requestId]) {
    details.requestHeaders.push(header);
  }
  return {
    requestHeaders: details.requestHeaders
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
      chrome.webRequest.onBeforeSendHeaders.addListener(
        rewriterHeaders,
        {
          urls: ["<all_urls>"],
          tabId: tabs[0].id
        },
        ["blocking", "requestHeaders"]
      );
    }
  );
};

const stopRewriting = () => {
  console.log('stopping rewriting...');
  chrome.webRequest.onBeforeRequest.removeListener(rewriterRequest);
  chrome.webRequest.onBeforeSendHeaders.removeListener(rewriterHeaders);
};

window.onload = function() {
  window.document.querySelector('#start-rewriting').addEventListener('click', startRewriting);
  window.document.querySelector('#stop-rewriting').addEventListener('click', stopRewriting);
};