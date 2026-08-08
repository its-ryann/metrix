const isLocal = ["localhost", "127.0.0.1"].includes(window.location.hostname);
const API_BASE = isLocal ? "http://localhost:8080" : "";

const state = {
    user: null,
    token: localStorage.getItem("metrix_token"),
    currentView: "overview",
    platform: "all",
    timeframe: "28d",
    charts: {},
    loading: false,
    isRegistering: false
};

const PLATFORM_META = {
    youtube: { label: "YouTube", icon: "play_circle", className: "platform-youtube" },
    instagram: { label: "Instagram", icon: "photo_camera", className: "platform-instagram" },
    tiktok: { label: "TikTok", icon: "music_note", className: "platform-tiktok" }
};

const DISTRIBUTION_COLORS = { youtube: "#ffb4ab", instagram: "#d0bcff", tiktok: "#4edea3" };

const VIEW_META = {
    overview: { title: "Dashboard Overview", subtitle: "Your cross-platform performance at a glance." },
    platforms: { title: "Connected Platforms", subtitle: "Manage your data sources and authentication status across all networks." },
    content: { title: "Top Content", subtitle: "Analyze your best performing posts across all platforms." },
    audience: { title: "Audience Insights", subtitle: "Deep dive into your demographic breakdown and geographic reach." },
    settings: { title: "Settings", subtitle: "Manage your account and preferences." }
};

const loginOverlay = document.getElementById("login-overlay");
const appContent = document.getElementById("app-content");
const loadingOverlay = document.getElementById("loading-overlay");
const loginForm = document.getElementById("login-form");
const logoutBtn = document.getElementById("logout-btn");
const healthStatusEl = document.getElementById("health-status");
const navItems = document.querySelectorAll("#nav-list .nav-item");
const mobNavItems = document.querySelectorAll(".mob-nav");
const platformSelect = document.getElementById("platform-select");
const authTitle = document.getElementById("auth-title");
const authSubmitBtn = document.getElementById("auth-submit-btn");
const authBtnLabel = document.getElementById("auth-btn-label");
const authSpinner = document.getElementById("auth-spinner");
const authToggleBtn = document.getElementById("auth-toggle-btn");
const authNotice = document.getElementById("auth-notice");
const forgotLink = document.getElementById("forgot-link");
const nameGroup = document.getElementById("name-group");
const userGreeting = document.getElementById("user-greeting");
const userAvatar = document.getElementById("user-avatar");
const pageTitle = document.getElementById("page-title");
const pageSubtitle = document.getElementById("page-subtitle");
const filterBar = document.getElementById("filter-bar");

function init() {
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
    navItems.forEach(item => {
        item.addEventListener("click", () => switchView(item.dataset.view));
    });
    mobNavItems.forEach(item => {
        item.addEventListener("click", (e) => {
            e.preventDefault();
            switchView(item.dataset.view);
        });
    });

    platformSelect.addEventListener("change", (e) => {
        state.platform = e.target.value;
        loadDashboardData();
    });

    document.querySelectorAll(".tf-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            document.querySelectorAll(".tf-btn").forEach(b => b.classList.remove("active"));
            btn.classList.add("active");
            state.timeframe = btn.dataset.tf;
            loadDashboardData();
        });
    });

    loginForm.addEventListener("submit", handleAuthSubmit);
    logoutBtn.addEventListener("click", handleLogout);
    authToggleBtn.addEventListener("click", toggleAuthMode);
    if (forgotLink) {
        forgotLink.addEventListener("click", () => {
            showInlineNotice("Password reset isn't available yet. Contact support@metrix.com.");
        });
    }
}

function showInlineNotice(message) {
    if (!authNotice) return;
    authNotice.textContent = message;
    authNotice.classList.remove("hidden");
    clearTimeout(showInlineNotice._timer);
    showInlineNotice._timer = setTimeout(() => authNotice.classList.add("hidden"), 6000);
}

function toggleAuthMode() {
    state.isRegistering = !state.isRegistering;
    if (state.isRegistering) {
        if (authTitle) authTitle.textContent = "Create Account";
        if (authBtnLabel) authBtnLabel.textContent = "CREATE ACCOUNT";
        authToggleBtn.innerHTML = `Already have an account? <span class="text-primary font-semibold">Sign In</span>`;
        nameGroup.classList.remove("hidden");
    } else {
        if (authTitle) authTitle.textContent = "Sign In";
        if (authBtnLabel) authBtnLabel.textContent = "SIGN IN";
        authToggleBtn.innerHTML = `Don't have an account? <span class="text-primary font-semibold">Create Account</span>`;
        nameGroup.classList.add("hidden");
    }
}

function setAuthLoading(isLoading) {
    if (!authSubmitBtn || !authSpinner || !authBtnLabel) return;
    authSubmitBtn.disabled = isLoading;
    authSpinner.classList.toggle("hidden", !isLoading);
    authBtnLabel.textContent = isLoading
        ? (state.isRegistering ? "CREATING ACCOUNT..." : "SIGNING IN...")
        : (state.isRegistering ? "CREATE ACCOUNT" : "SIGN IN");
}

function showLogin() {
    loginOverlay.classList.remove("hidden");
    appContent.classList.add("hidden");
}

function showApp() {
    loginOverlay.classList.add("hidden");
    appContent.classList.remove("hidden");
    if (state.user) {
        userGreeting.textContent = `Welcome back, ${state.user.name}`;
        const initials = state.user.name.split(" ").map(w => w[0]).filter(Boolean).slice(0, 2).join("").toUpperCase();
        userAvatar.textContent = initials || "M";
    }
}

function setLoading(isLoading) {
    state.loading = isLoading;
    if (isLoading) loadingOverlay.classList.remove("hidden");
    else loadingOverlay.classList.add("hidden");
}

async function fetchWithAuth(url, options = {}) {
    if (!options.headers) options.headers = {};
    if (state.token) options.headers["Authorization"] = `Bearer ${state.token}`;
    const res = await fetch(url, options);
    if (res.status === 401) {
        handleLogout();
        throw new Error("Unauthorized access - logging out.");
    }
    return res;
}

async function handleAuthSubmit(e) {
    e.preventDefault();
    setAuthLoading(true);
    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;
    const name = document.getElementById("name").value;

    const endpoint = state.isRegistering ? `${API_BASE}/api/v1/auth/register` : `${API_BASE}/api/v1/auth/login`;
    const payload = state.isRegistering ? { email, password, name } : { email, password };

    try {
        const res = await fetch(endpoint, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(payload)
        });
        const data = await res.json();
        if (!res.ok) {
            alert(data.error || "Authentication failed.");
            return;
        }
        state.token = data.token;
        state.user = data.user;
        localStorage.setItem("metrix_token", data.token);
        showApp();
        loadDashboardData();
    } catch (err) {
        showInlineNotice("Can't reach the backend. Check that the API is running.");
    } finally {
        setAuthLoading(false);
    }
}

function handleLogout() {
    state.token = null;
    state.user = null;
    localStorage.removeItem("metrix_token");
    showLogin();
}

function switchView(viewId) {
    state.currentView = viewId;

    navItems.forEach(li => {
        li.classList.toggle("active", li.dataset.view === viewId);
    });
    mobNavItems.forEach(a => {
        a.classList.toggle("active", a.dataset.view === viewId);
    });

    document.querySelectorAll(".view").forEach(view => {
        view.classList.toggle("hidden", view.id !== `view-${viewId}`);
    });

    const meta = VIEW_META[viewId] || VIEW_META.overview;
    pageTitle.textContent = meta.title;
    pageSubtitle.textContent = meta.subtitle;
    filterBar.classList.toggle("hidden", viewId !== "overview");

    loadDashboardData();
}

// Data Loading
async function loadDashboardData() {
    const queryParams = `?platform=${state.platform}&timeframe=${state.timeframe}`;

    if (state.currentView === "overview") {
        fetchSummary(queryParams);
        fetchTimeSeries(queryParams);
        fetchPlatformDistribution();
    } else if (state.currentView === "platforms") {
        fetchPlatforms(queryParams);
    } else if (state.currentView === "content") {
        fetchTopContent(queryParams);
    } else if (state.currentView === "audience") {
        fetchAudience(queryParams);
        fetchSummary(queryParams);
    }
}

async function fetchSummary(params) {
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/metrics/summary${params}`);
        const data = await res.json();

        updateKPI("reach", data.total_reach, data.reach_delta);
        updateKPI("engagement", data.avg_engagement, data.engage_delta, true);
        updateKPI("growth", data.follower_growth, data.growth_delta);

        const audienceTotal = document.getElementById("kpi-audience-total");
        const audienceEngagement = document.getElementById("kpi-audience-engagement");
        if (audienceTotal) audienceTotal.textContent = Number(data.total_reach).toLocaleString();
        if (audienceEngagement) audienceEngagement.textContent = `${data.avg_engagement}%`;
    } catch (err) { console.error(err); }
}

function updateKPI(id, value, delta, isPercent = false) {
    const valEl = document.getElementById(`kpi-${id}`);
    const deltaEl = document.getElementById(`kpi-${id}-delta`);
    if (!valEl || !deltaEl) return;
    valEl.textContent = isPercent ? `${value}%` : Number(value).toLocaleString();
    const arrow = delta >= 0 ? "↑" : "↓";
    deltaEl.textContent = `${arrow} ${Math.abs(delta)}%`;
    deltaEl.className = `delta-badge ${delta >= 0 ? "positive" : "negative"}`;
}

async function fetchTimeSeries(params) {
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/metrics/timeseries${params}`);
        const data = await res.json();
        renderReachChart(data.data);
    } catch (err) { console.error(err); }
}

async function fetchPlatformDistribution() {
    const platforms = ["youtube", "instagram", "tiktok"];
    try {
        const sums = {};
        for (const p of platforms) {
            const res = await fetchWithAuth(`${API_BASE}/api/v1/metrics/timeseries?platform=${p}&timeframe=${state.timeframe}`);
            const data = await res.json();
            sums[p] = data.data.reduce((a, b) => a + b.value, 0);
        }
        const total = platforms.reduce((a, p) => a + sums[p], 0) || 1;
        const values = platforms.map(p => sums[p]);

        renderPlatformChart(platforms, values, platforms.map(p => PLATFORM_META[p].label));

        const totalEl = document.getElementById("platform-distribution-total");
        if (totalEl) totalEl.textContent = total.toLocaleString();

        document.querySelectorAll("#platform-legend .legend-row").forEach((row, i) => {
            const pct = Math.round((values[i] / total) * 100);
            row.querySelector(".legend-pct").textContent = `${pct}%`;
        });
    } catch (err) { console.error(err); }
}

async function fetchPlatforms(params) {
    const list = document.getElementById("platform-list");
    list.innerHTML = `<div class="col-span-full flex justify-center py-12"><div class="spinner"></div></div>`;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/platform-accounts${params}`);
        const data = await res.json();
        if (!data.length) {
            list.innerHTML = `<div class="col-span-full text-center text-sm text-on-surface-variant py-12">No connected platforms yet.</div>`;
            return;
        }
        list.innerHTML = data.map(acc => {
            const meta = PLATFORM_META[acc.platform] || { label: acc.platform, icon: "hub", className: "" };
            const statusClass = acc.status === "connected" ? "connected" : "warn";
            const statusLabel = acc.status.replace("_", " ");
            return `
                <div class="glass-card rounded-xl p-5 flex flex-col gap-4 hover:bg-white/5 transition-colors ${meta.className}">
                    <div class="flex justify-between items-start">
                        <div class="flex items-center gap-3">
                            <div class="platform-icon"><span class="material-symbols-outlined">${meta.icon}</span></div>
                            <div>
                                <h3 class="font-semibold">${meta.label}</h3>
                                <p class="text-sm text-on-surface-variant mt-0.5">${acc.display_name}</p>
                            </div>
                        </div>
                        <span class="status-badge ${statusClass}">${statusLabel}</span>
                    </div>
                    <div class="mt-auto">
                        <button class="w-full px-4 py-2 rounded-lg border border-white/10 text-primary text-xs font-semibold hover:bg-white/5 transition-colors">Manage</button>
                    </div>
                </div>`;
        }).join("");
    } catch (err) { list.innerHTML = `<div class="col-span-full text-center text-sm text-on-surface-variant py-12">Failed to load platforms.</div>`; }
}

async function fetchTopContent(params) {
    const tbody = document.getElementById("content-table");
    tbody.innerHTML = `<tr><td colspan="4" class="px-5 py-8 text-center"><div class="spinner mx-auto"></div></td></tr>`;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/metrics/top-content${params}`);
        const data = await res.json();
        tbody.innerHTML = data.map(item => {
            const badge = platformBadge(item.platform);
            return `
                <tr class="content-row hover:bg-white/[0.03] transition-colors group cursor-pointer">
                    <td class="text-sm font-semibold group-hover:text-primary transition-colors">${item.title}</td>
                    <td>${badge}</td>
                    <td class="text-right text-sm font-semibold text-primary">${item.engagement}%</td>
                    <td class="text-right text-sm font-mono text-on-surface-variant">${Number(item.reach).toLocaleString()}</td>
                </tr>`;
        }).join("");
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="4" class="px-5 py-8 text-center text-sm text-on-surface-variant">Failed to load content.</td></tr>`;
    }
}

function platformBadge(platform) {
    const map = {
        youtube: { icon: "play_arrow", label: "YouTube", cls: "badge-youtube" },
        instagram: { icon: "photo_camera", label: "Instagram", cls: "badge-instagram" },
        tiktok: { icon: "tag", label: "TikTok", cls: "badge-tiktok" }
    };
    const m = map[platform] || { icon: "analytics", label: platform, cls: "" };
    return `<div class="platform-badge ${m.cls}"><span class="material-symbols-outlined">${m.icon}</span>${m.label}</div>`;
}

async function fetchAudience(params) {
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/audience/insights${params}`);
        const data = await res.json();
        renderAudienceCharts(data);

        const ageData = data.demographics.filter(d => d.category === "age");
        const primary = ageData.reduce((a, b) => (b.value > a.value ? b : a), ageData[0]);
        const primaryEl = document.getElementById("kpi-audience-primary");
        if (primaryEl && primary) primaryEl.textContent = primary.label;
    } catch (err) { console.error(err); }
}

// Charts
function renderReachChart(points) {
    const ctx = document.getElementById("reachChart").getContext("2d");
    if (state.charts.reach) state.charts.reach.destroy();

    const grad = ctx.createLinearGradient(0, 0, 0, 320);
    grad.addColorStop(0, "rgba(77, 142, 255, 0.35)");
    grad.addColorStop(1, "rgba(77, 142, 255, 0)");

    state.charts.reach = new Chart(ctx, {
        type: "line",
        data: {
            labels: points.map(p => p.date),
            datasets: [{
                label: "Reach",
                data: points.map(p => p.value),
                borderColor: "#4d8eff",
                borderWidth: 3,
                backgroundColor: grad,
                fill: true,
                tension: 0.4,
                pointRadius: 4,
                pointBackgroundColor: "#ffffff",
                pointBorderColor: "#4d8eff",
                pointBorderWidth: 2
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { grid: { color: "rgba(255,255,255,0.05)" }, ticks: { color: "#8c909f" } },
                x: { grid: { display: false }, ticks: { color: "#8c909f" } }
            }
        }
    });
}

function renderPlatformChart(labels, values, labelNames) {
    const ctx = document.getElementById("platformChart").getContext("2d");
    if (state.charts.platform) state.charts.platform.destroy();

    state.charts.platform = new Chart(ctx, {
        type: "doughnut",
        data: {
            labels: labelNames,
            datasets: [{
                data: values,
                backgroundColor: labels.map(l => DISTRIBUTION_COLORS[l] || "#adc6ff"),
                borderWidth: 2,
                borderColor: "#0c1321",
                hoverOffset: 10
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: "72%",
            plugins: { legend: { display: false } }
        }
    });
}

function renderAudienceCharts(data) {
    const ageData = data.demographics.filter(d => d.category === "age");
    const gender = data.demographics.filter(d => d.category === "gender");
    const malePct = (gender.find(g => g.label === "Male")?.value ?? 52) / 100;
    const femalePct = (gender.find(g => g.label === "Female")?.value ?? 48) / 100;

    const ctxAge = document.getElementById("audienceChart").getContext("2d");
    if (state.charts.audience) state.charts.audience.destroy();
    state.charts.audience = new Chart(ctxAge, {
        type: "bar",
        data: {
            labels: ageData.map(d => d.label),
            datasets: [
                {
                    label: "Female",
                    data: ageData.map(d => +(d.value * femalePct).toFixed(1)),
                    backgroundColor: "rgba(208, 188, 255, 0.85)",
                    borderColor: "#d0bcff",
                    borderWidth: 1,
                    borderRadius: 4
                },
                {
                    label: "Male",
                    data: ageData.map(d => +(d.value * malePct).toFixed(1)),
                    backgroundColor: "rgba(173, 198, 255, 0.85)",
                    borderColor: "#adc6ff",
                    borderWidth: 1,
                    borderRadius: 4
                }
            ]
        },
        options: {
            indexAxis: "y",
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: "top",
                    labels: { usePointStyle: true, boxWidth: 8, color: "#c2c6d6" }
                }
            },
            scales: {
                x: { stacked: true, grid: { color: "rgba(255,255,255,0.05)" }, ticks: { color: "#8c909f" } },
                y: { stacked: true, grid: { display: false }, ticks: { color: "#8c909f" } }
            }
        }
    });

    const ctxGeo = document.getElementById("geoChart").getContext("2d");
    if (state.charts.geo) state.charts.geo.destroy();
    state.charts.geo = new Chart(ctxGeo, {
        type: "polarArea",
        data: {
            labels: data.geography.map(d => d.label),
            datasets: [{
                data: data.geography.map(d => d.value),
                backgroundColor: ["rgba(173,198,255,0.7)", "rgba(208,188,255,0.7)", "rgba(78,222,163,0.7)", "rgba(77,142,255,0.7)", "rgba(87,27,193,0.7)"],
                borderColor: "#19202e",
                borderWidth: 2
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: "right",
                    labels: { usePointStyle: true, boxWidth: 8, color: "#c2c6d6" }
                }
            },
            scales: {
                r: {
                    ticks: { display: false },
                    grid: { color: "rgba(255,255,255,0.1)" },
                    angleLines: { color: "rgba(255,255,255,0.1)" }
                }
            }
        }
    });
}

async function checkHealth() {
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();
        if (data.status === "ok") {
            healthStatusEl.textContent = "Online";
        } else {
            healthStatusEl.textContent = "Offline";
        }
    } catch (err) {
        healthStatusEl.textContent = "Offline";
    }
}

init();
