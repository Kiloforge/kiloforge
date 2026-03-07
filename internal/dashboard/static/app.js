(function() {
  "use strict";

  var state = { agents: [], quota: null, tracks: [] };

  // DOM refs
  var agentGrid = document.getElementById("agent-grid");
  var trackList = document.getElementById("track-list");
  var totalAgents = document.getElementById("total-agents");
  var totalCost = document.getElementById("total-cost");
  var rateStatus = document.getElementById("rate-status");
  var tokenCounts = document.getElementById("token-counts");
  var connStatus = document.getElementById("conn-status");
  var logModal = document.getElementById("log-modal");
  var logViewer = document.getElementById("log-viewer");
  var logAgentId = document.getElementById("log-agent-id");
  var logClose = document.getElementById("log-close");

  function formatCost(usd) {
    return "$" + (usd || 0).toFixed(2);
  }

  function formatTokens(n) {
    if (!n) return "0";
    if (n >= 1000000) return (n / 1000000).toFixed(1) + "M";
    if (n >= 1000) return (n / 1000).toFixed(1) + "k";
    return String(n);
  }

  function formatUptime(seconds) {
    if (!seconds || seconds < 0) return "-";
    var h = Math.floor(seconds / 3600);
    var m = Math.floor((seconds % 3600) / 60);
    var s = seconds % 60;
    if (h > 0) return h + "h " + m + "m";
    if (m > 0) return m + "m " + s + "s";
    return s + "s";
  }

  // Render agents
  function renderAgents() {
    if (!state.agents.length) {
      agentGrid.innerHTML = '<p class="empty-state">No agents running</p>';
      totalAgents.textContent = "0";
      return;
    }
    totalAgents.textContent = String(state.agents.length);
    agentGrid.innerHTML = state.agents.map(function(a) {
      return '<div class="agent-card">' +
        '<div class="agent-header">' +
          '<span class="agent-id">' + esc(a.id) + '</span>' +
          '<span class="agent-role ' + esc(a.role) + '">' + esc(a.role) + '</span>' +
        '</div>' +
        '<div class="agent-header">' +
          '<span class="status-badge ' + esc(a.status) + '">' + esc(a.status) + '</span>' +
          (a.cost_usd ? '<span style="font-size:12px;color:var(--text-dim)">' + formatCost(a.cost_usd) + '</span>' : '') +
        '</div>' +
        '<div class="agent-meta">' +
          (a.ref ? '<span>ref: ' + esc(a.ref) + '</span>' : '') +
          (a.uptime_seconds ? '<span>uptime: ' + formatUptime(a.uptime_seconds) + '</span>' : '') +
          (a.pid ? '<span>PID: ' + a.pid + '</span>' : '') +
        '</div>' +
        '<div class="agent-actions">' +
          (a.log_file ? '<button class="btn" onclick="window.__viewLog(\'' + esc(a.id) + '\')">View Log</button>' : '') +
        '</div>' +
      '</div>';
    }).join("");
  }

  // Render tracks
  function renderTracks() {
    if (!state.tracks.length) {
      trackList.innerHTML = '<p class="empty-state">No tracks found</p>';
      return;
    }
    trackList.innerHTML = state.tracks.map(function(t) {
      return '<div class="track-row">' +
        '<span class="track-status ' + esc(t.status) + '"></span>' +
        '<span class="track-id">' + esc(t.id.substring(0, 20)) + '</span>' +
        '<span class="track-title">' + esc(t.title) + '</span>' +
      '</div>';
    }).join("");
  }

  // Render quota
  function renderQuota() {
    if (!state.quota) return;
    totalCost.textContent = formatCost(state.quota.total_cost_usd);
    tokenCounts.textContent = formatTokens(state.quota.input_tokens) + " / " + formatTokens(state.quota.output_tokens);
    if (state.quota.rate_limited) {
      rateStatus.textContent = "LIMITED";
      rateStatus.className = "stat-value danger";
    } else {
      rateStatus.textContent = "OK";
      rateStatus.className = "stat-value";
    }
  }

  // Escape HTML
  function esc(s) {
    if (!s) return "";
    var d = document.createElement("div");
    d.appendChild(document.createTextNode(String(s)));
    return d.innerHTML;
  }

  // Fetch helpers
  function fetchJSON(url, cb) {
    fetch(url).then(function(r) { return r.json(); }).then(cb).catch(function() {});
  }

  // Initial data load
  function loadAll() {
    fetchJSON("/api/agents", function(data) {
      state.agents = data || [];
      renderAgents();
    });
    fetchJSON("/api/quota", function(data) {
      state.quota = data;
      renderQuota();
    });
    fetchJSON("/api/tracks", function(data) {
      state.tracks = data || [];
      renderTracks();
    });
    fetchJSON("/api/status", function(data) {
      if (data && data.gitea_url) {
        var link = document.getElementById("gitea-link");
        link.href = data.gitea_url;
      }
    });
  }

  // SSE connection
  function connectSSE() {
    var es = new EventSource("/events");

    es.onopen = function() {
      connStatus.textContent = "connected";
      connStatus.className = "badge connected";
    };

    es.onerror = function() {
      connStatus.textContent = "disconnected";
      connStatus.className = "badge disconnected";
    };

    es.addEventListener("agent_update", function(e) {
      var data = JSON.parse(e.data);
      var agent = data.data;
      var idx = -1;
      for (var i = 0; i < state.agents.length; i++) {
        if (state.agents[i].id === agent.id) { idx = i; break; }
      }
      if (idx >= 0) {
        state.agents[idx] = Object.assign(state.agents[idx], agent);
      } else {
        state.agents.push(agent);
      }
      renderAgents();
    });

    es.addEventListener("agent_removed", function(e) {
      var data = JSON.parse(e.data);
      var id = data.data.id;
      state.agents = state.agents.filter(function(a) { return a.id !== id; });
      renderAgents();
    });

    es.addEventListener("quota_update", function(e) {
      var data = JSON.parse(e.data);
      state.quota = data.data;
      renderQuota();
    });
  }

  // Log viewer
  window.__viewLog = function(agentId) {
    logAgentId.textContent = agentId;
    logViewer.textContent = "Loading...";
    logModal.hidden = false;

    fetchJSON("/api/agents/" + encodeURIComponent(agentId) + "/log?lines=200", function(data) {
      if (data && data.lines) {
        logViewer.textContent = data.lines.join("\n");
        logViewer.scrollTop = logViewer.scrollHeight;
      } else {
        logViewer.textContent = "No log data available.";
      }
    });
  };

  logClose.addEventListener("click", function() {
    logModal.hidden = true;
    logViewer.textContent = "";
  });

  // Boot
  loadAll();
  connectSSE();
})();
