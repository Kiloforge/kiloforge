package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
)

const detailPageCSS = `
:root{--bg:#0f1117;--surface:#1a1d27;--border:#2a2d3a;--text:#e1e4ed;--text-dim:#8b8fa3;--accent:#6c8cff;--green:#3dd68c;--radius:8px}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:var(--bg);color:var(--text);line-height:1.5}
header{display:flex;align-items:center;justify-content:space-between;padding:12px 24px;background:var(--surface);border-bottom:1px solid var(--border)}
.header-left{display:flex;align-items:center;gap:12px}
header h1{font-size:18px;font-weight:600;letter-spacing:-0.5px}
nav a{color:var(--accent);text-decoration:none;font-size:14px}
.badge{font-size:11px;padding:2px 8px;border-radius:10px;background:var(--border);color:var(--text-dim)}
main{padding:24px;max-width:1200px;margin:0 auto}
.panel{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:20px;margin-bottom:16px}
.panel h2{font-size:14px;color:var(--text-dim);text-transform:uppercase;letter-spacing:0.5px;margin-bottom:16px}
.empty-state{color:var(--text-dim);font-size:14px;text-align:center;padding:20px}
.agent-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:12px}
.agent-card{background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);padding:14px;cursor:pointer}
.agent-header{display:flex;justify-content:space-between;align-items:center;margin-bottom:6px}
.agent-meta{font-size:12px;color:var(--text-dim)}
.log-viewer{font-family:"SF Mono","Fira Code",monospace;font-size:12px;line-height:1.6;padding:12px 16px;color:var(--text-dim);white-space:pre-wrap;word-break:break-all}
`

const trackDetailHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Track: {{.TrackID}} — crelay</title>
  <style>` + detailPageCSS + `</style>
</head>
<body>
  <header>
    <div class="header-left">
      <h1><a href="/-/" style="color:inherit;text-decoration:none">crelay</a></h1>
      <span class="badge" id="conn-status">connecting</span>
    </div>
    <nav><a href="/-/">Dashboard</a></nav>
  </header>
  <main>
    <section class="panel">
      <h2>Track: {{.TrackID}}</h2>
      <div style="margin:1rem 0">
        <img src="/-/api/badges/track/{{.TrackID}}" alt="status badge" style="height:20px">
      </div>
      <div id="agent-grid" class="agent-grid">
        <p class="empty-state">Loading agent data...</p>
      </div>
    </section>
    <section class="panel" id="log-section" hidden>
      <h2>Agent Log</h2>
      <pre class="log-viewer" id="log-viewer" style="max-height:400px;overflow:auto"></pre>
    </section>
  </main>
  <script>
    const TRACK_ID = "{{.TrackID}}";
    async function loadAgents() {
      const resp = await fetch('/-/api/agents');
      const agents = await resp.json();
      const matched = agents.filter(a => a.ref === TRACK_ID);
      const grid = document.getElementById('agent-grid');
      if (matched.length === 0) {
        grid.innerHTML = '<p class="empty-state">No agent assigned yet</p>';
        return;
      }
      grid.innerHTML = matched.map(agentCard).join('');
    }
    function agentCard(a) {
      const c = {running:'#4c1',waiting:'#dfb317',completed:'#007ec6',failed:'#e05d44',halted:'#fe7d37'}[a.status]||'#9f9f9f';
      return '<div class="agent-card" onclick="showLog(\''+a.id+'\')">' +
        '<div class="agent-header"><span class="badge" style="background:'+c+'">'+a.status+'</span> ' +
        '<strong>'+a.role+'</strong></div>' +
        '<div class="agent-meta">ID: '+a.id.slice(0,8)+' · PID: '+(a.pid||'—')+'</div></div>';
    }
    async function showLog(id) {
      document.getElementById('log-section').hidden = false;
      const resp = await fetch('/-/api/agents/'+id+'/log?lines=200');
      const data = await resp.json();
      document.getElementById('log-viewer').textContent = (data.lines||[]).join('\n');
    }
    loadAgents();
    const es = new EventSource('/-/events');
    es.onopen = () => document.getElementById('conn-status').textContent = 'live';
    es.addEventListener('agent_update', () => loadAgents());
  </script>
</body>
</html>`

const prDetailHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>PR #{{.PRNumber}} — crelay</title>
  <style>` + detailPageCSS + `</style>
</head>
<body>
  <header>
    <div class="header-left">
      <h1><a href="/-/" style="color:inherit;text-decoration:none">crelay</a></h1>
      <span class="badge" id="conn-status">connecting</span>
    </div>
    <nav><a href="/-/">Dashboard</a></nav>
  </header>
  <main>
    <section class="panel">
      <h2>PR #{{.PRNumber}} — {{.Slug}}</h2>
      <div style="margin:1rem 0">
        <img src="/-/api/badges/pr/{{.Slug}}/{{.PRNumber}}" alt="status badge" style="height:20px">
      </div>
      <div id="agent-grid" class="agent-grid">
        <p class="empty-state">Loading agent data...</p>
      </div>
    </section>
    <section class="panel" id="log-section" hidden>
      <h2>Agent Log</h2>
      <pre class="log-viewer" id="log-viewer" style="max-height:400px;overflow:auto"></pre>
    </section>
  </main>
  <script>
    async function loadAgents() {
      const resp = await fetch('/-/api/agents');
      const agents = await resp.json();
      const matched = agents.filter(a => a.ref && a.ref.includes('#{{.PRNumber}}'));
      const grid = document.getElementById('agent-grid');
      if (matched.length === 0) {
        grid.innerHTML = '<p class="empty-state">No agents found for this PR</p>';
        return;
      }
      grid.innerHTML = matched.map(agentCard).join('');
    }
    function agentCard(a) {
      const c = {running:'#4c1',waiting:'#dfb317',completed:'#007ec6',failed:'#e05d44',halted:'#fe7d37'}[a.status]||'#9f9f9f';
      return '<div class="agent-card" onclick="showLog(\''+a.id+'\')">' +
        '<div class="agent-header"><span class="badge" style="background:'+c+'">'+a.status+'</span> ' +
        '<strong>'+a.role+'</strong></div>' +
        '<div class="agent-meta">ID: '+a.id.slice(0,8)+' · PID: '+(a.pid||'—')+'</div></div>';
    }
    async function showLog(id) {
      document.getElementById('log-section').hidden = false;
      const resp = await fetch('/-/api/agents/'+id+'/log?lines=200');
      const data = await resp.json();
      document.getElementById('log-viewer').textContent = (data.lines||[]).join('\n');
    }
    loadAgents();
    const es = new EventSource('/-/events');
    es.onopen = () => document.getElementById('conn-status').textContent = 'live';
    es.addEventListener('agent_update', () => loadAgents());
  </script>
</body>
</html>`

var (
	trackTmpl = template.Must(template.New("track").Parse(trackDetailHTML))
	prTmpl    = template.Must(template.New("pr").Parse(prDetailHTML))
)

func (s *Server) handleTrackDetail(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	if trackID == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	trackTmpl.Execute(w, map[string]string{"TrackID": trackID})
}

func (s *Server) handlePRDetail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	prNumStr := r.PathValue("prNumber")
	prNum, err := strconv.Atoi(prNumStr)
	if err != nil || slug == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	prTmpl.Execute(w, map[string]any{
		"Slug":     slug,
		"PRNumber": fmt.Sprintf("%d", prNum),
	})
}
