/* ═══════════════════════════════════════════════════════════════════════════
   SpiderSearch — Frontend JS (SPA Controller)
   Connects to the Python FastAPI backend APIs
   ═══════════════════════════════════════════════════════════════════════════ */

// ── Page Navigation ────────────────────────────────────────────────────────

function showPage(name) {
    document.querySelectorAll('.page').forEach(p => p.style.display = 'none');
    document.querySelectorAll('nav a').forEach(a => a.classList.remove('active'));

    document.getElementById('page-' + name).style.display = 'block';
    document.getElementById('nav-' + name).classList.add('active');

    // Restart animation
    const page = document.getElementById('page-' + name);
    page.style.animation = 'none';
    page.offsetHeight; // trigger reflow
    page.style.animation = null;
}

// ── Toast Notifications ────────────────────────────────────────────────────

function showToast(message, type = 'success') {
    // Remove any existing toasts
    document.querySelectorAll('.toast').forEach(t => t.remove());

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    document.body.appendChild(toast);

    requestAnimationFrame(() => {
        toast.classList.add('visible');
    });

    setTimeout(() => {
        toast.classList.remove('visible');
        setTimeout(() => toast.remove(), 400);
    }, 3500);
}

// ── Status Polling ─────────────────────────────────────────────────────────

async function pollStatus() {
    try {
        const r = await fetch('/status');
        if (!r.ok) return;
        const data = await r.json();

        // Update stat cards
        document.getElementById('stat-indexed').textContent = data.pages_indexed.toLocaleString();
        document.getElementById('stat-active').textContent = data.active_sessions.length;
        document.getElementById('stat-queue').textContent = data.queue_depth.toLocaleString();
        document.getElementById('stat-rate').textContent = data.crawl_rate_per_sec;

        const pressureEl = document.getElementById('stat-pressure');
        if (data.back_pressure_active) {
            pressureEl.innerHTML = '<span class="pressure-badge on">Active</span>';
        } else {
            pressureEl.innerHTML = '<span class="pressure-badge off">Off</span>';
        }

        // Update session list
        const list = document.getElementById('sessionList');

        if (data.active_sessions.length === 0) {
            list.innerHTML = '<div class="empty-state">No crawl sessions detected.</div>';
        } else {
            list.innerHTML = data.active_sessions.map(s => `
                <div class="session-card">
                    <div>
                        <div class="url-text">${escapeHtml(s.origin_url)}</div>
                        <div class="meta-text">ID: ${s.id} • Depth: ${s.max_depth} • Started: ${formatTime(s.started_at)}</div>
                    </div>
                    <div style="text-align: center">
                        <span class="badge badge-${s.status}">${s.status}</span>
                    </div>
                    <div style="text-align: right; min-width: 60px;">
                        <div style="font-size: 11px; color: var(--text-secondary); text-transform: uppercase;">Depth</div>
                        <div style="font-size: 20px; font-weight: 700; color: var(--accent);">${s.max_depth}</div>
                    </div>
                </div>
            `).join('');
        }
    } catch (e) {
        console.error('Status poll failed:', e);
    }
}

// ── Start Crawl ────────────────────────────────────────────────────────────

async function startCrawl() {
    const origin = document.getElementById('origin').value.trim();
    const depth = parseInt(document.getElementById('depth').value, 10);
    const btn = document.getElementById('btn-crawl');

    if (!origin) {
        showToast('Please enter an origin URL.', 'error');
        document.getElementById('origin').focus();
        return;
    }

    // Basic URL validation
    try {
        new URL(origin);
    } catch {
        showToast('Invalid URL. Must be a valid http(s) URL.', 'error');
        return;
    }

    btn.textContent = 'Initializing...';
    btn.disabled = true;

    try {
        const r = await fetch('/index', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ origin, k: depth }),
        });

        if (r.ok) {
            const data = await r.json();
            showToast(`Crawl session #${data.session_id} started!`, 'success');
            document.getElementById('origin').value = '';
            await pollStatus();
        } else {
            const err = await r.json().catch(() => ({ detail: 'Server error' }));
            showToast(err.detail || 'Failed to start crawl.', 'error');
        }
    } catch (e) {
        showToast('Network error — is the server running?', 'error');
    } finally {
        btn.textContent = 'Initialize Neural Crawl Sequence';
        btn.disabled = false;
    }
}

// ── Search ─────────────────────────────────────────────────────────────────

async function doSearch() {
    const q = document.getElementById('search-q').value.trim();
    if (!q) return;

    const container = document.getElementById('results');
    const countLabel = document.getElementById('results-count');

    container.style.display = 'block';
    container.innerHTML = `
        <div style="padding: 80px; text-align: center;">
            <div style="color: var(--accent); margin-bottom: 16px; font-weight: 600;">ACCESSING DISTRIBUTED DATABASE...</div>
            <div class="loading-dots"><span></span><span></span><span></span></div>
            <div style="font-size: 13px; color: var(--text-secondary); margin-top: 12px;">Querying TF-IDF index</div>
        </div>
    `;

    try {
        const params = new URLSearchParams({ q });
        const r = await fetch('/search?' + params.toString());
        const data = await r.json();

        container.innerHTML = '';

        // Handle both array and object responses
        const results = Array.isArray(data) ? data : (data.results || []);
        const total = Array.isArray(data) ? data.length : (data.total || results.length);

        if (!results.length) {
            countLabel.style.display = 'none';
            container.innerHTML = `
                <div style="padding: 80px 40px; text-align: center;">
                    <div style="font-size: 48px; margin-bottom: 24px;">📡</div>
                    <div style="font-size: 22px; font-weight: 600; color: var(--text);">No matching signals found</div>
                    <div style="color: var(--text-secondary); margin-top: 12px; max-width: 400px; margin-left: auto; margin-right: auto;">
                        The search query "${escapeHtml(q)}" did not return any results from the current index. Try crawling more pages first.
                    </div>
                </div>
            `;
            return;
        }

        countLabel.textContent = `${total} indexed page${total !== 1 ? 's' : ''} relevant to your query`;
        countLabel.style.display = 'block';

        results.forEach(res => {
            const div = document.createElement('div');
            div.className = 'search-result';
            div.innerHTML = `
                <a href="${escapeHtml(res.relevant_url || res.url)}" target="_blank" class="search-result-url">
                    ${escapeHtml(res.relevant_url || res.url)}
                </a>
                <div class="search-result-meta">
                    ${res.score != null ? `<div>Score: <b>${res.score.toFixed(4)}</b></div>` : ''}
                    ${res.depth != null ? `<div>Depth: <b>${res.depth}</b></div>` : ''}
                    ${res.origin_url ? `<div style="opacity: 0.6">Origin: <b>${escapeHtml(new URL(res.origin_url).hostname)}</b></div>` : ''}
                </div>
            `;
            container.appendChild(div);
        });
    } catch (e) {
        container.innerHTML = `
            <div style="padding: 40px; text-align: center; color: var(--error)">
                DATABASE CONNECTION LOST: Unable to fetch search results.
            </div>
        `;
    }
}

// ── Helpers ─────────────────────────────────────────────────────────────────

function escapeHtml(text) {
    const el = document.createElement('span');
    el.textContent = text;
    return el.innerHTML;
}

function formatTime(ts) {
    try {
        return new Date(ts).toLocaleString();
    } catch {
        return ts;
    }
}

// ── Auto-Poll Loop ─────────────────────────────────────────────────────────

pollStatus();
setInterval(pollStatus, 3000);
