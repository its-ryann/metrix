const isLocal = ["localhost", "127.0.0.1"].includes(window.location.hostname);
const API_BASE = isLocal ? "http://localhost:8080" : "";

const state = {
    user: null,
    token: localStorage.getItem("metrix_token"),
    charts: {}
};

// DOM Elements
const loginOverlay = document.getElementById("login-overlay");
const appContent = document.getElementById("app-content");
const loginForm = document.getElementById("login-form");
const logoutBtn = document.getElementById("logout-btn");
const healthStatusEl = document.getElementById("health-status");

// Initialization
async function init() {
    if (state.token) {
        showApp();
        loadDashboardData();
    } else {
        showLogin();
    }
    checkHealth();
}

function showLogin() {
    loginOverlay.classList.remove("hidden");
    appContent.classList.add("hidden");
}

function showApp() {
    loginOverlay.classList.add("hidden");
    appContent.classList.remove("hidden");
}

// Auth Handlers
loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
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
    }
});

logoutBtn.addEventListener("click", () => {
    state.token = null;
    localStorage.removeItem("metrix_token");
    showLogin();
});

// Data Loading
async function loadDashboardData() {
    fetchSummary();
    fetchTimeSeries();
    fetchTopContent();
}

async function fetchSummary() {
    try {
        const res = await fetch(`${API_BASE}/api/v1/metrics/summary`);
        const data = await res.json();
        
        document.getElementById("kpi-reach").textContent = data.total_reach.toLocaleString();
        document.getElementById("kpi-reach-delta").textContent = `${data.reach_delta > 0 ? '+' : ''}${data.reach_delta}%`;
        
        document.getElementById("kpi-engagement").textContent = `${data.avg_engagement}%`;
        document.getElementById("kpi-engagement-delta").textContent = `${data.engage_delta > 0 ? '+' : ''}${data.engage_delta}%`;
        
        document.getElementById("kpi-growth").textContent = data.follower_growth.toLocaleString();
        document.getElementById("kpi-growth-delta").textContent = `${data.growth_delta > 0 ? '+' : ''}${data.growth_delta}%`;
    } catch (err) {
        console.error("Failed to fetch summary:", err);
    }
}

async function fetchTimeSeries() {
    try {
        const res = await fetch(`${API_BASE}/api/v1/metrics/timeseries`);
        const data = await res.json();
        renderReachChart(data.data);
    } catch (err) {
        console.error("Failed to fetch timeseries:", err);
    }
    renderPlatformChart(); // Mocking this one locally
}

async function fetchTopContent() {
    try {
        const res = await fetch(`${API_BASE}/api/v1/metrics/top-content`);
        const data = await res.json();
        const tbody = document.querySelector("#content-table tbody");
        tbody.innerHTML = data.map(item => `
            <tr>
                <td>${item.title}</td>
                <td>${item.platform.toUpperCase()}</td>
                <td>${item.engagement}%</td>
                <td>${item.reach.toLocaleString()}</td>
            </tr>
        `).join("");
    } catch (err) {
        console.error("Failed to fetch top content:", err);
    }
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
                label: 'Daily Reach',
                data: points.map(p => p.value),
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4
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
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { position: 'bottom', labels: { color: '#94a3b8' } } }
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
