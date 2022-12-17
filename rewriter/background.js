tabStates = {};
chrome.runtime.onInstalled.addListener(() => {
  tabStates = {};
  chrome.declarativeNetRequest.updateSessionRules({
    removeRuleIds: [1],
  });
});

const updateBadge = async (tab) => {
  await chrome.action.setBadgeText({
    tabId: tab.id,
    text: tabStates[tab.id] || 'OFF',
  });
}

chrome.tabs.onUpdated.addListener(async function (tabId, changeInfo, tab) {
  if (changeInfo.status == 'complete') {
    console.log(tabStates)
    await updateBadge(tab);
  }
})

chrome.action.onClicked.addListener(async (tab) => {
  const prevState = tabStates[tab.id] || 'OFF';
  const nextState = prevState === 'ON' ? 'OFF' : 'ON'

  tabStates[tab.id] = nextState;
  updateBadge(tab); 

  const tabIds = Object.entries(tabStates).filter(([tabId, state]) => state === 'ON').map(([tabId, state]) => parseInt(tabId))

  chrome.declarativeNetRequest.updateSessionRules({
    removeRuleIds: [1],
  })

  if (tabIds.length > 0) {
    chrome.declarativeNetRequest.updateSessionRules({
      addRules: [{
        "id": 1,
        "priority": 1,
        "condition": {
          "regexFilter": "^(.*)$",
          "excludedRequestDomains": [
            "localhost"
          ],
          "tabIds": tabIds,
        },
        "action": {
          "type": "redirect",
          "redirect": {
            "regexSubstitution": "http://localhost:8000/?url=\\1"
          }
        }
      }],
    })
    return
  }


  // old: manifest v2
  // // If the extension is 'ON' then add the listener,
  // // otherwise remove it
  // if (nextState === 'ON') {
  //   chrome.webRequest.onBeforeRequest.addListener(
  //     rewriter,
  //     { urls: ["<all_urls>"] },
  //     // ['blocking']
  //   );
  // } else {
  //   chrome.webRequest.onBeforeRequest.removeListener(rewriter);
  // }

  // create a handler for chrome.webRequest.onBeforeRequest event
// const rewriter = (details) => {
//   const { tabId, url } = details;
//   console.log('url: ' + url);
//   // const { origin, pathname } = new URL(url);
//   const newUrl = `http://localhost:8000?url=${url}`;
//   console.log('Rewriting', url, 'to', newUrl);
//   return { redirectUrl: newUrl };
// }

// chrome.webRequest.onBeforeRequest.addListener(
//   rewriter,
//   { urls: ["<all_urls>"] },
//   ['blocking']
// );
});
