const isLocal = ["localhost", "127.0.0.1"].includes(window.location.hostname);
const API_BASE = isLocal ? "http://localhost:8080" : "";

const state = {
    user: null,
    token: localStorage.getItem("metrix_token"),
    currentView: "overview",
    workspace: "workspace-solo",
    platform: "all",
    timeframe: "7d",
    charts: {},
    loading: false
};

// DOM Elements
const loginOverlay = document.getElementById("login-overlay");
const appContent = document.getElementById("app-content");
const loadingOverlay = document.getElementById("loading-overlay");
const loginForm = document.getElementById("login-form");
const logoutBtn = document.getElementById("logout-btn");
const healthStatusEl = document.getElementById("health-status");
const navItems = document.querySelectorAll("#nav-list li");
const workspaceSelect = document.getElementById("workspace-select");
const platformSelect = document.getElementById("platform-select");
const timeframeSelect = document.getElementById("timeframe-select");

// Initialization
async function init() {
    setupEventListeners();
    if (state.token) {
        showApp();
        loadDashboardData();
    } else {
        showLogin();
    }
    checkHealth();
}

function setupEventListeners() {
    // Navigation
    navItems.forEach(item => {
        item.addEventListener("click", () => {
            switchView(item.dataset.view);
        });
    });

    // Filters
    workspaceSelect.addEventListener("change", (e) => {
        state.workspace = e.target.value;
        loadDashboardData();
    });
    platformSelect.addEventListener("change", (e) => {
        state.platform = e.target.value;
        loadDashboardData();
    });
    timeframeSelect.addEventListener("change", (e) => {
        state.timeframe = e.target.value;
        loadDashboardData();
    });

    // Auth
    loginForm.addEventListener("submit", handleLogin);
    logoutBtn.addEventListener("click", handleLogout);
}

function showLogin() {
    loginOverlay.classList.remove("hidden");
    appContent.classList.add("hidden");
}

function showApp() {
    loginOverlay.classList.add("hidden");
    appContent.classList.remove("hidden");
}

function setLoading(isLoading) {
    state.loading = isLoading;
    if (isLoading) loadingOverlay.classList.remove("hidden");
    else loadingOverlay.classList.add("hidden");
}

async function handleLogin(e) {
    e.preventDefault();
    setLoading(true);
    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;

    try {
        const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, password })
        });
        const data = await res.json();
        state.token = data.token;
        state.user = data.user;
        localStorage.setItem("metrix_token", data.token);
        showApp();
        loadDashboardData();
    } catch (err) {
        alert("Login failed. Check backend.");
    } finally {
        setLoading(false);
    }
}

function handleLogout() {
    state.token = null;
    localStorage.removeItem("metrix_token");
    showLogin();
}

function switchView(viewId) {
    state.currentView = viewId;
    
    // Update Sidebar
    navItems.forEach(li => {
        li.classList.toggle("active", li.dataset.view === viewId);
    });

    // Update Content
    document.querySelectorAll(".view").forEach(view => {
        view.classList.toggle("hidden", view.id !== `view-${viewId}`);
    });

    // Update Header
    const titleMap = {
        overview: "Dashboard Overview",
        platforms: "Social Platforms",
        content: "Top Content",
        audience: "Audience Insights",
        reports: "Performance Reports",
        settings: "Settings"
    };
    document.getElementById("page-title").textContent = titleMap[viewId];

    // Load data specific to view
    loadDashboardData();
}

// Data Loading
async function loadDashboardData() {
    const queryParams = `?workspace_id=${state.workspace}&platform=${state.platform}&timeframe=${state.timeframe}`;
    
    if (state.currentView === "overview") {
        fetchSummary(queryParams);
        fetchTimeSeries(queryParams);
    } else if (state.currentView === "platforms") {
        fetchPlatforms(queryParams);
    } else if (state.currentView === "content") {
        fetchTopContent(queryParams);
    } else if (state.currentView === "audience") {
        fetchAudience(queryParams);
    }
}

async function fetchSummary(params) {
    try {
        const res = await fetch(`${API_BASE}/api/v1/metrics/summary${params}`);
        const data = await res.json();
        
        updateKPI("reach", data.total_reach, data.reach_delta);
        updateKPI("engagement", data.avg_engagement, data.engage_delta, true);
        updateKPI("growth", data.follower_growth, data.growth_delta);
    } catch (err) { console.error(err); }
}

function updateKPI(id, value, delta, isPercent = false) {
    const valEl = document.getElementById(`kpi-${id}`);
    const deltaEl = document.getElementById(`kpi-${id}-delta`);
    
    valEl.textContent = isPercent ? `${value}%` : value.toLocaleString();
    deltaEl.textContent = `${delta > 0 ? '↑' : '↓'} ${Math.abs(delta)}%`;
    deltaEl.className = `delta ${delta >= 0 ? 'positive' : 'negative'}`;
}

async function fetchTimeSeries(params) {
    try {
        const res = await fetch(`${API_BASE}/api/v1/metrics/timeseries${params}`);
        const data = await res.json();
        renderReachChart(data.data);
    } catch (err) { console.error(err); }
    renderPlatformChart();
}

async function fetchPlatforms(params) {
    const list = document.getElementById("platform-list");
    list.innerHTML = `<div class="spinner"></div>`;
    try {
        const res = await fetch(`${API_BASE}/api/v1/platform-accounts${params}`);
        const data = await res.json();
        list.innerHTML = data.map(acc => `
            <div class="platform-card">
                <div class="platform-header">
                    <div class="platform-icon ${acc.platform.substring(0,2)}">${acc.platform[0].toUpperCase()}</div>
                    <span class="status">${acc.status.replace('_', ' ')}</span>
                </div>
                <h4>${acc.displayName}</h4>
                <p>${acc.platform.toUpperCase()}</p>
                <button class="btn-connect">Manage Connection</button>
            </div>
        `).join("");
    } catch (err) { list.innerHTML = "Failed to load platforms."; }
}

async function fetchTopContent(params) {
    const tbody = document.querySelector("#content-table tbody");
    try {
        const res = await fetch(`${API_BASE}/api/v1/metrics/top-content${params}`);
        const data = await res.json();
        tbody.innerHTML = data.map(item => `
            <tr>
                <td style="font-weight: 600;">${item.title}</td>
                <td>${item.platform.toUpperCase()}</td>
                <td style="color: var(--primary); font-weight: 600;">${item.engagement}%</td>
                <td>${item.reach.toLocaleString()}</td>
            </tr>
        `).join("");
    } catch (err) { console.error(err); }
}

async function fetchAudience(params) {
    try {
        const res = await fetch(`${API_BASE}/api/v1/audience/insights${params}`);
        const data = await res.json();
        renderAudienceCharts(data);
    } catch (err) { console.error(err); }
}

// Charts
function renderReachChart(points) {
    const ctx = document.getElementById('reachChart').getContext('2d');
    if (state.charts.reach) state.charts.reach.destroy();
    
    state.charts.reach = new Chart(ctx, {
        type: 'line',
        data: {
            labels: points.map(p => p.date),
            datasets: [{
                label: 'Reach',
                data: points.map(p => p.value),
                borderColor: '#3b82f6',
                borderWidth: 3,
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 4,
                pointBackgroundColor: '#3b82f6'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { grid: { color: '#334155' }, ticks: { color: '#94a3b8' } },
                x: { grid: { display: false }, ticks: { color: '#94a3b8' } }
            }
        }
    });
}

function renderPlatformChart() {
    const ctx = document.getElementById('platformChart').getContext('2d');
    if (state.charts.platform) state.charts.platform.destroy();

    state.charts.platform = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['YouTube', 'Instagram', 'TikTok'],
            datasets: [{
                data: [45, 25, 30],
                backgroundColor: ['#ef4444', '#ec4899', '#000000'],
                borderWidth: 0,
                hoverOffset: 10
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { position: 'bottom', labels: { color: '#94a3b8', padding: 20, usePointStyle: true } } },
            cutout: '70%'
        }
    });
}

function renderAudienceCharts(data) {
    const ageData = data.demographics.filter(d => d.category === 'age');
    const ctx = document.getElementById('audienceChart').getContext('2d');
    if (state.charts.audience) state.charts.audience.destroy();

    state.charts.audience = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: ageData.map(d => d.label),
            datasets: [{
                data: ageData.map(d => d.value),
                backgroundColor: '#3b82f6',
                borderRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { grid: { color: '#334155' }, ticks: { color: '#94a3b8' } },
                x: { grid: { display: false }, ticks: { color: '#94a3b8' } }
            }
        }
    });

    const geoData = data.geography;
    const ctxGeo = document.getElementById('geoChart').getContext('2d');
    if (state.charts.geo) state.charts.geo.destroy();
    state.charts.geo = new Chart(ctxGeo, {
        type: 'polarArea',
        data: {
            labels: geoData.map(d => d.label),
            datasets: [{
                data: geoData.map(d => d.value),
                backgroundColor: ['#3b82f6', '#4ade80', '#f87171', '#fbbf24']
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { position: 'right', labels: { color: '#94a3b8' } } },
            scales: { r: { grid: { color: '#334155' }, ticks: { display: false } } }
        }
    });
}

async function checkHealth() {
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();
        healthStatusEl.textContent = "ONLINE";
        healthStatusEl.style.color = "#4ade80";
    } catch (err) {
        healthStatusEl.textContent = "OFFLINE";
        healthStatusEl.style.color = "#f87171";
    }
}

init();
