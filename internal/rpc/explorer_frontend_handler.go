package rpc

import (
	"fmt"
	"net/http"
)

func ExplorerHomePageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Sila Explorer</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { font-family: Arial, sans-serif; margin: 0; background: #0b1020; color: #e8ecf1; }
    .wrap { max-width: 1100px; margin: 0 auto; padding: 24px; }
    .card { background: #121a2b; border: 1px solid #24304a; border-radius: 16px; padding: 18px; margin-bottom: 18px; }
    h1,h2 { margin-top: 0; }
    input, button { padding: 10px 12px; border-radius: 10px; border: 1px solid #33415f; background: #0f1726; color: #fff; }
    button { cursor: pointer; }
    .row { display: flex; gap: 10px; flex-wrap: wrap; }
    .row input { flex: 1 1 220px; }
    pre { background: #0a0f1a; border: 1px solid #24304a; border-radius: 12px; padding: 14px; overflow: auto; white-space: pre-wrap; }
    a { color: #8ab4ff; text-decoration: none; }
    .muted { color: #a9b4c7; }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>Sila Explorer</h1>
    <p class="muted">Contract pages, VM transaction inspection, and event log search.</p>

    <div class="card">
      <h2>Contract Explorer</h2>
      <div class="row">
        <input id="contractAddress" placeholder="Contract address" />
        <button onclick="loadContract()">Load Contract</button>
      </div>
      <pre id="contractOut">No contract loaded.</pre>
    </div>

    <div class="card">
      <h2>VM Transaction Explorer</h2>
      <div class="row">
        <input id="txHash" placeholder="Transaction hash" />
        <button onclick="loadTxVM()">Load VM Tx</button>
      </div>
      <pre id="txOut">No VM transaction loaded.</pre>
    </div>

    <div class="card">
      <h2>Logs Explorer</h2>
      <div class="row">
        <input id="logsAddress" placeholder="Address" />
        <input id="logsEvent" placeholder="Event" />
        <input id="topic0" placeholder="topic0" />
        <input id="topic1" placeholder="topic1" />
      </div>
      <div class="row" style="margin-top:10px;">
        <input id="topic2" placeholder="topic2" />
        <input id="topic3" placeholder="topic3" />
        <button onclick="loadLogs()">Search Logs</button>
      </div>
      <pre id="logsOut">No logs query yet.</pre>
    </div>
  </div>

<script>
async function fetchJSON(url) {
  const res = await fetch(url);
  const text = await res.text();
  try { return JSON.stringify(JSON.parse(text), null, 2); }
  catch { return text; }
}

async function loadContract() {
  const address = document.getElementById('contractAddress').value.trim();
  const out = document.getElementById('contractOut');
  if (!address) { out.textContent = 'Missing contract address'; return; }
  out.textContent = 'Loading...';
  out.textContent = await fetchJSON('/explorer/contract?address=' + encodeURIComponent(address));
}

async function loadTxVM() {
  const hash = document.getElementById('txHash').value.trim();
  const out = document.getElementById('txOut');
  if (!hash) { out.textContent = 'Missing tx hash'; return; }
  out.textContent = 'Loading...';
  out.textContent = await fetchJSON('/explorer/tx-vm?hash=' + encodeURIComponent(hash));
}

async function loadLogs() {
  const params = new URLSearchParams();
  const fields = ['logsAddress','logsEvent','topic0','topic1','topic2','topic3'];
  const mapping = {
    logsAddress: 'address',
    logsEvent: 'event',
    topic0: 'topic0',
    topic1: 'topic1',
    topic2: 'topic2',
    topic3: 'topic3'
  };
  for (const f of fields) {
    const v = document.getElementById(f).value.trim();
    if (v) params.set(mapping[f], v);
  }
  const out = document.getElementById('logsOut');
  out.textContent = 'Loading...';
  out.textContent = await fetchJSON('/explorer/logs?' + params.toString());
}
</script>
</body>
</html>`

		_, _ = fmt.Fprint(w, html)
	}
}
