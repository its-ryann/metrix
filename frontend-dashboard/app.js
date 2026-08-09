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
    isRegistering: false,
    isResettingPassword: false,
    connectedPlatforms: [],
    lastContentData: []
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
const refreshBtn = document.getElementById("refresh-btn");
const notificationsBtn = document.getElementById("notifications-btn");
const upgradeProBtn = document.getElementById("upgrade-pro-btn");
const upgradeProCta = document.getElementById("upgrade-pro-cta");
const openConnectBtn = document.getElementById("open-connect-btn");
const connectModal = document.getElementById("connect-modal");
const consentAuthorizeBtn = document.getElementById("consent-authorize-btn");
const consentCancelBtn = document.getElementById("consent-cancel-btn");
const modalCloseBtn = document.getElementById("modal-close-btn");
const connectDisplayName = document.getElementById("connect-display-name");
const profileForm = document.getElementById("profile-form");
const passwordForm = document.getElementById("password-form");
const deleteAccountBtn = document.getElementById("delete-account-btn");
const gotoPlatforms = document.getElementById("goto-platforms");
const exportBtn = document.getElementById("export-btn");
const globalSearch = document.getElementById("global-search");
const platformList = document.getElementById("platform-list");

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
    authToggleBtn.addEventListener("click", () => {
        if (state.isResettingPassword) {
            showLoginForm();
        } else {
            toggleAuthMode();
        }
    });
    if (forgotLink) {
        forgotLink.addEventListener("click", (e) => {
            e.preventDefault();
            showForgotPassword();
        });
    }

    if (refreshBtn) refreshBtn.addEventListener("click", () => {
        loadDashboardData();
        toast("Dashboard refreshed", "success");
    });
    if (notificationsBtn) notificationsBtn.addEventListener("click", () => {
        toast("You're all caught up — no new notifications.", "info");
    });
    if (upgradeProBtn) upgradeProBtn.addEventListener("click", () => {
        toast("Metrix Pro is coming soon. Stay tuned!", "info");
    });
    if (upgradeProCta) upgradeProCta.addEventListener("click", () => {
        toast("Metrix Pro is coming soon. Stay tuned!", "info");
    });

    if (openConnectBtn) openConnectBtn.addEventListener("click", openConnectModal);
    if (connectModal) connectModal.addEventListener("click", (e) => {
        if (e.target.dataset.closeModal !== undefined) closeConnectModal();
    });
    if (modalCloseBtn) modalCloseBtn.addEventListener("click", closeConnectModal);
    if (consentCancelBtn) consentCancelBtn.addEventListener("click", closeConnectModal);
    if (consentAuthorizeBtn) consentAuthorizeBtn.addEventListener("click", authorizePlatform);
    document.querySelectorAll(".connect-option").forEach(btn => {
        btn.addEventListener("click", () => selectPlatform(btn.dataset.platform));
    });
    if (platformList) platformList.addEventListener("click", handlePlatformAction);

    if (forgotLink) {
        forgotLink.addEventListener("click", (e) => {
            e.preventDefault();
            showForgotPassword();
        });
    }
    if (profileForm) profileForm.addEventListener("submit", handleProfileSubmit);
    if (passwordForm) passwordForm.addEventListener("submit", handlePasswordSubmit);
    if (deleteAccountBtn) deleteAccountBtn.addEventListener("click", handleDeleteAccount);
    if (gotoPlatforms) gotoPlatforms.addEventListener("click", () => switchView("platforms"));
    if (exportBtn) exportBtn.addEventListener("click", exportContentCSV);
    if (globalSearch) globalSearch.addEventListener("input", filterContent);
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
    state.isResettingPassword = false;
    if (state.isRegistering) {
        if (authTitle) authTitle.textContent = "Create Account";
        if (authBtnLabel) authBtnLabel.textContent = "CREATE ACCOUNT";
        authToggleBtn.innerHTML = `Already have an account? <span class="text-primary font-semibold">Sign In</span>`;
        authToggleBtn.onclick = showLoginForm;
        nameGroup.classList.remove("hidden");
    } else {
        if (authTitle) authTitle.textContent = "Sign In";
        if (authBtnLabel) authBtnLabel.textContent = "SIGN IN";
        authToggleBtn.innerHTML = `Don't have an account? <span class="text-primary font-semibold">Create Account</span>`;
        authToggleBtn.onclick = toggleAuthMode;
        nameGroup.classList.add("hidden");
    }
    const authNotice = document.getElementById("auth-notice");
    if (authNotice) authNotice.classList.add("hidden");
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
    state.isResettingPassword = false;
    loginOverlay.classList.remove("hidden");
    appContent.classList.add("hidden");
}

function showForgotPassword() {
    state.isResettingPassword = true;
    const authTitle = document.getElementById("auth-title");
    const authBtnLabel = document.getElementById("auth-btn-label");
    const authNotice = document.getElementById("auth-notice");
    if (authTitle) authTitle.textContent = "Reset Password";
    if (authBtnLabel) authBtnLabel.textContent = "SEND RESET LINK";
    if (nameGroup) nameGroup.classList.add("hidden");
    if (authToggleBtn) {
        authToggleBtn.innerHTML = `<span class="text-primary font-semibold">Back to Sign In</span>`;
    }
    if (authNotice) {
        authNotice.classList.remove("hidden");
        authNotice.innerHTML = "Enter your email. If an account exists, we'll send a reset link.";
        authNotice.className = "text-xs text-info bg-info-container/10 border border-info-container/20 rounded-lg px-3 py-2";
    }
}

function showLoginForm() {
    state.isRegistering = false;
    state.isResettingPassword = false;
    const authTitle = document.getElementById("auth-title");
    const authBtnLabel = document.getElementById("auth-btn-label");
    const authNotice = document.getElementById("auth-notice");
    if (authTitle) authTitle.textContent = "Sign In";
    if (authBtnLabel) authBtnLabel.textContent = "SIGN IN";
    if (nameGroup) nameGroup.classList.add("hidden");
    if (authToggleBtn) authToggleBtn.innerHTML = `Don't have an account? <span class="text-primary font-semibold">Create Account</span>`;
    if (authNotice) authNotice.classList.add("hidden");
}

function showApp() {
    loginOverlay.classList.add("hidden");
    appContent.classList.remove("hidden");
    if (state.user) {
        userGreeting.textContent = `Welcome back, ${state.user.name}`;
        userAvatar.textContent = initials(state.user.name);
    }
    refreshConnectedPlatforms();
}

function initials(name) {
    return (name || "M").split(" ").map(w => w[0]).filter(Boolean).slice(0, 2).join("").toUpperCase() || "M";
}

function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

function timeAgo(iso) {
    const then = new Date(iso);
    const secs = Math.floor((Date.now() - then.getTime()) / 1000);
    if (isNaN(secs) || secs < 60) return "Just now";
    const mins = Math.floor(secs / 60);
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
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
    const authTitle = document.getElementById("auth-title");
    if (authTitle && authTitle.textContent === "Reset Password") {
        await handleResetRequest();
        return;
    }

    // Registration or login flow
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

async function handleResetRequest() {
    setAuthLoading(true);
    const email = document.getElementById("email").value;
    if (!email) {
        alert("Please enter your email address.");
        setAuthLoading(false);
        return;
    }
    try {
        const res = await fetch(`${API_BASE}/api/v1/auth/reset-password`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email })
        });
        const data = await res.json();
        if (!res.ok) {
            alert(data.error || "Failed to send reset link.");
            return;
        }
        showLoginForm();
        toast(data.message || "Reset instructions sent to your email.", "success");
    } catch (err) {
        showInlineNotice("Can't reach the backend. Check that the API is running.");
    } finally {
        setAuthLoading(false);
    }
}

function handleLogout() {
    state.token = null;
    state.user = null;
    state.connectedPlatforms = [];
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

    if (viewId === "settings") loadSettings();
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
    const legend = document.getElementById("platform-legend");
    const platforms = state.connectedPlatforms.length ? state.connectedPlatforms : ["youtube", "instagram", "tiktok"];
    try {
        const sums = {};
        for (const p of platforms) {
            const res = await fetchWithAuth(`${API_BASE}/api/v1/metrics/timeseries?platform=${p}&timeframe=${state.timeframe}`);
            const data = await res.json();
            sums[p] = data.data.reduce((a, b) => a + b.value, 0);
        }
        const total = platforms.reduce((a, p) => a + sums[p], 0) || 1;
        const values = platforms.map(p => sums[p]);

        renderPlatformChart(platforms, values, platforms.map(p => (PLATFORM_META[p] || { label: p }).label));

        const totalEl = document.getElementById("platform-distribution-total");
        if (totalEl) totalEl.textContent = total.toLocaleString();

        if (legend) {
            legend.innerHTML = platforms.map((p, i) => {
                const meta = PLATFORM_META[p] || { label: p };
                const color = DISTRIBUTION_COLORS[p] || "#adc6ff";
                const pct = Math.round((values[i] / total) * 100);
                return `<div class="legend-row flex items-center justify-between">
                    <div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:${color}"></span><span class="text-sm text-on-surface">${meta.label}</span></div>
                    <span class="legend-pct text-sm text-on-surface-variant font-mono">${pct}%</span>
                </div>`;
            }).join("");
        }
    } catch (err) { console.error(err); }
}

async function refreshConnectedPlatforms() {
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/platform-accounts`);
        const data = await res.json();
        state.connectedPlatforms = data.filter(a => a.status === "connected").map(a => a.platform);
    } catch (err) { /* silent */ }
}

async function fetchPlatforms(params) {
    const list = document.getElementById("platform-list");
    list.innerHTML = `<div class="col-span-full flex justify-center py-12"><div class="spinner"></div></div>`;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/platform-accounts${params}`);
        const data = await res.json();
        state.connectedPlatforms = data.filter(a => a.status === "connected").map(a => a.platform);
        if (!data.length) {
            list.innerHTML = `
                <div class="col-span-full text-center py-16 px-6">
                    <div class="w-16 h-16 mx-auto rounded-full bg-surface-variant flex items-center justify-center mb-4">
                        <span class="material-symbols-outlined text-on-surface-variant text-[30px]">hub</span>
                    </div>
                    <h3 class="text-lg font-semibold">No platforms connected</h3>
                    <p class="text-sm text-on-surface-variant mt-1 max-w-sm mx-auto">Connect your first data source to start tracking your cross-platform performance.</p>
                </div>`;
            return;
        }
        list.innerHTML = data.map(acc => {
            const meta = PLATFORM_META[acc.platform] || { label: acc.platform, icon: "hub", className: "" };
            const isConnected = acc.status === "connected";
            const followers = isConnected ? Number(acc.followers).toLocaleString() : "—";
            const lastSync = acc.last_synced ? timeAgo(acc.last_synced) : (isConnected ? "Syncing..." : "Never synced");
            const actionBtn = isConnected
                ? `<button class="px-4 py-2 rounded-lg border border-red-500/30 text-red-300 text-xs font-semibold hover:bg-red-500/10 transition-colors" data-action="disconnect" data-id="${acc.id}">Disconnect</button>`
                : `<button class="px-4 py-2 rounded-lg bg-primary-container text-on-primary-container text-xs font-bold tracking-wider hover:opacity-90 transition-opacity" data-action="reconnect" data-id="${acc.id}">Reconnect</button>`;
            return `
                <div class="glass-card rounded-xl p-5 flex flex-col gap-4 hover:bg-white/5 transition-colors ${meta.className}">
                    <div class="flex justify-between items-start">
                        <div class="flex items-center gap-3">
                            <div class="platform-icon"><span class="material-symbols-outlined">${meta.icon}</span></div>
                            <div>
                                <h3 class="font-semibold">${escapeHtml(meta.label)}</h3>
                                <p class="text-sm text-on-surface-variant mt-0.5">${escapeHtml(acc.display_name)}</p>
                            </div>
                        </div>
                        <span class="status-badge ${isConnected ? "connected" : "warn"}">${isConnected ? "Connected" : "Disconnected"}</span>
                    </div>
                    <div class="grid grid-cols-2 gap-3 text-center">
                        <div class="rounded-lg bg-surface-container-lowest/60 border border-white/5 py-2">
                            <div class="text-sm font-bold">${followers}</div>
                            <div class="text-[10px] uppercase tracking-widest text-outline mt-0.5">14d Reach</div>
                        </div>
                        <div class="rounded-lg bg-surface-container-lowest/60 border border-white/5 py-2">
                            <div class="text-sm font-bold">${escapeHtml(lastSync)}</div>
                            <div class="text-[10px] uppercase tracking-widest text-outline mt-0.5">Last Sync</div>
                        </div>
                    </div>
                    <div class="mt-auto flex gap-2">${actionBtn}</div>
                </div>`;
        }).join("");
    } catch (err) { list.innerHTML = `<div class="col-span-full text-center text-sm text-on-surface-variant py-12">Failed to load platforms.</div>`; }
}

async function handlePlatformAction(e) {
    const btn = e.target.closest("[data-action]");
    if (!btn) return;
    const action = btn.dataset.action;
    const id = btn.dataset.id;
    if (action === "disconnect") await disconnectPlatform(id, btn);
    else if (action === "reconnect") await reconnectPlatform(id, btn);
}

async function disconnectPlatform(id, btn) {
    if (!confirm("Disconnect this platform? It will stop syncing (historical data is kept).")) return;
    btn.disabled = true;
    btn.textContent = "Disconnecting...";
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/platform-accounts/${id}`, { method: "DELETE" });
        if (!res.ok) throw new Error("Failed to disconnect");
        toast("Platform disconnected", "info");
        fetchPlatforms();
    } catch (err) {
        btn.disabled = false;
        btn.textContent = "Disconnect";
        toast(err.message, "error");
    }
}

async function reconnectPlatform(id, btn) {
    btn.disabled = true;
    btn.textContent = "Reconnecting...";
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/platform-accounts/${id}/reconnect`, { method: "POST" });
        if (!res.ok) throw new Error("Failed to reconnect");
        toast("Platform reconnected", "success");
        fetchPlatforms();
    } catch (err) {
        btn.disabled = false;
        btn.textContent = "Reconnect";
        toast(err.message, "error");
    }
}

async function fetchTopContent(params) {
    const tbody = document.getElementById("content-table");
    tbody.innerHTML = `<tr><td colspan="4" class="px-5 py-8 text-center"><div class="spinner mx-auto"></div></td></tr>`;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/metrics/top-content${params}`);
        const data = await res.json();
        state.lastContentData = data;
        if (!data.length || (data.length === 1 && !data[0].platform)) {
            tbody.innerHTML = `<tr><td colspan="4" class="px-5 py-10 text-center text-sm text-on-surface-variant">No content yet — connect a platform to get started.</td></tr>`;
            return;
        }
        tbody.innerHTML = data.map(item => {
            const badge = platformBadge(item.platform);
            return `
                <tr class="content-row hover:bg-white/[0.03] transition-colors group cursor-pointer">
                    <td class="text-sm font-semibold group-hover:text-primary transition-colors">${escapeHtml(item.title)}</td>
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

// Toasts
function toast(message, type = "success") {
    const container = document.getElementById("toast-container");
    if (!container) return;
    const icons = { success: "check", error: "error", info: "info" };
    const el = document.createElement("div");
    el.className = `toast ${type}`;
    el.innerHTML = `
        <div class="toast-icon"><span class="material-symbols-outlined">${icons[type] || "info"}</span></div>
        <div class="toast-message">${escapeHtml(message)}</div>`;
    container.appendChild(el);
    setTimeout(() => {
        el.classList.add("hide");
        setTimeout(() => el.remove(), 320);
    }, 4500);
}

// Connect Platform Modal
let selectedPlatform = null;

function openConnectModal() {
    if (!connectModal) return;
    connectModal.classList.remove("hidden");
    document.getElementById("modal-step-pick").classList.remove("hidden");
    document.getElementById("modal-step-consent").classList.add("hidden");
    connectDisplayName.value = "";
    document.querySelectorAll(".connect-option").forEach(btn => {
        btn.disabled = state.connectedPlatforms.includes(btn.dataset.platform);
    });
}

function closeConnectModal() {
    if (connectModal) connectModal.classList.add("hidden");
    selectedPlatform = null;
}

function selectPlatform(platform) {
    selectedPlatform = platform;
    const meta = PLATFORM_META[platform] || { label: platform, icon: "hub" };
    document.getElementById("consent-platform-name").textContent = meta.label;
    document.getElementById("consent-platform-icon").innerHTML = `<span class="material-symbols-outlined">${meta.icon}</span>`;
    document.getElementById("modal-step-pick").classList.add("hidden");
    document.getElementById("modal-step-consent").classList.remove("hidden");
}

async function authorizePlatform() {
    if (!selectedPlatform) return;
    const btn = consentAuthorizeBtn;
    const original = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = `<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 0 1 8-8v4a4 4 0 0 0-4 4H4z"></path></svg> AUTHORIZING...`;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/oauth/connect`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ platform: selectedPlatform })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Failed to connect platform");
        closeConnectModal();
        toast("Redirecting to OAuth provider...", "info");
        window.open(data.url, "_blank");
        setTimeout(() => {
            refreshConnectedPlatforms();
            if (state.currentView === "platforms") fetchPlatforms();
        }, 3000);
    } catch (err) {
        toast(err.message, "error");
    } finally {
        btn.disabled = false;
        btn.innerHTML = original;
    }
}

// Settings
async function loadSettings() {
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/auth/me`);
        const user = await res.json();
        if (!res.ok) throw new Error(user.error || "Failed to load profile");
        state.user = user;
        const nameInput = document.getElementById("profile-name");
        const emailInput = document.getElementById("profile-email");
        const preview = document.getElementById("settings-name-preview");
        const avatar = document.getElementById("settings-avatar");
        if (nameInput) nameInput.value = user.name;
        if (emailInput) emailInput.value = user.email;
        if (preview) preview.textContent = user.name;
        if (avatar) avatar.textContent = initials(user.name);
        userGreeting.textContent = `Welcome back, ${user.name}`;
        userAvatar.textContent = initials(user.name);
    } catch (err) { /* fetchWithAuth already handles 401 */ }

    const apiList = document.getElementById("settings-api-list");
    if (apiList) {
        try {
            const res = await fetchWithAuth(`${API_BASE}/api/v1/platform-accounts`);
            const accounts = await res.json();
            if (!accounts.length) {
                apiList.innerHTML = `<div class="col-span-full text-sm text-on-surface-variant">No connected platforms.</div>`;
            } else {
                apiList.innerHTML = accounts.map(a => {
                    const meta = PLATFORM_META[a.platform] || { label: a.platform, icon: "hub" };
                    const ok = a.status === "connected";
                    return `
                        <div class="rounded-xl bg-surface-container-low border ${ok ? "border-white/10" : "border-red-500/20"} p-4 flex items-center gap-3">
                            <div class="platform-icon"><span class="material-symbols-outlined">${meta.icon}</span></div>
                            <div class="flex-1 min-w-0">
                                <div class="text-sm font-semibold truncate">${escapeHtml(meta.label)}</div>
                                <div class="text-xs text-on-surface-variant truncate">${escapeHtml(a.display_name)}</div>
                            </div>
                            <span class="status-badge ${ok ? "connected" : "warn"}">${ok ? "Active" : "Disconnected"}</span>
                        </div>`;
                }).join("");
            }
        } catch (err) {
            apiList.innerHTML = `<div class="col-span-full text-sm text-on-surface-variant">Failed to load connections.</div>`;
        }
    }
}

async function handleProfileSubmit(e) {
    e.preventDefault();
    const name = document.getElementById("profile-name").value.trim();
    if (!name) { toast("Name cannot be empty", "error"); return; }
    const btn = e.target.querySelector("button[type=submit]");
    btn.disabled = true;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/settings/profile`, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Failed to save profile");
        state.user = data;
        toast("Profile updated", "success");
        loadSettings();
    } catch (err) {
        toast(err.message, "error");
    } finally {
        btn.disabled = false;
    }
}

async function handlePasswordSubmit(e) {
    e.preventDefault();
    const cur = document.getElementById("current-password").value;
    const next = document.getElementById("new-password").value;
    const confirm = document.getElementById("confirm-password").value;
    if (next !== confirm) { toast("New passwords do not match", "error"); return; }
    if (next.length < 8) { toast("Password must be at least 8 characters", "error"); return; }
    const btn = e.target.querySelector("button[type=submit]");
    btn.disabled = true;
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/auth/change-password`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ current_password: cur, new_password: next })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Failed to update password");
        e.target.reset();
        toast("Password updated", "success");
    } catch (err) {
        toast(err.message, "error");
    } finally {
        btn.disabled = false;
    }
}

async function handleDeleteAccount() {
    if (!deleteAccountBtn.dataset.armed) {
        deleteAccountBtn.dataset.armed = "1";
        const original = deleteAccountBtn.textContent;
        deleteAccountBtn.textContent = "CLICK AGAIN TO CONFIRM";
        deleteAccountBtn.classList.add("bg-red-500/30");
        clearTimeout(handleDeleteAccount._t);
        handleDeleteAccount._t = setTimeout(() => {
            delete deleteAccountBtn.dataset.armed;
            deleteAccountBtn.textContent = original;
            deleteAccountBtn.classList.remove("bg-red-500/30");
        }, 4000);
        return;
    }
    delete deleteAccountBtn.dataset.armed;
    deleteAccountBtn.disabled = true;
    deleteAccountBtn.textContent = "DELETING...";
    try {
        const res = await fetchWithAuth(`${API_BASE}/api/v1/auth/account`, { method: "DELETE" });
        if (!res.ok) throw new Error("Failed to delete account");
        localStorage.removeItem("metrix_token");
        state.token = null;
        state.user = null;
        toast("Account deleted. Sorry to see you go.", "info");
        setTimeout(() => showLogin(), 900);
    } catch (err) {
        deleteAccountBtn.disabled = false;
        deleteAccountBtn.textContent = "DELETE ACCOUNT";
        toast(err.message, "error");
    }
}

// Export + Search
function exportContentCSV() {
    if (!state.lastContentData.length) { toast("No content to export", "info"); return; }
    const header = "Title,Platform,Engagement (%),Reach\n";
    const rows = state.lastContentData.map(item => {
        const label = (PLATFORM_META[item.platform] || { label: item.platform || "—" }).label;
        return `"${item.title.replace(/"/g, '""')}",${label},${item.engagement},${item.reach}`;
    }).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "metrix-top-content.csv";
    a.click();
    URL.revokeObjectURL(url);
    toast("Content exported as CSV", "success");
}

function filterContent(e) {
    const term = e.target.value.trim().toLowerCase();
    document.querySelectorAll("#content-table tr").forEach(row => {
        const title = row.querySelector("td")?.textContent.toLowerCase() || "";
        row.style.display = !term || title.includes(term) ? "" : "none";
    });
}

// Auto refresh while the dashboard is visible
setInterval(() => {
    if (state.token && !appContent.classList.contains("hidden")) loadDashboardData();
}, 60000);

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
