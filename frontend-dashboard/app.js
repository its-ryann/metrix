// Locally, frontend and backend run as separate containers, so hit localhost:8080 directly.
// In production, Vercel rewrites route /health to the backend service on the same origin.
const isLocal = ["localhost", "127.0.0.1"].includes(window.location.hostname);
const API_BASE = isLocal ? "http://localhost:8080" : "";
const API_URL = `${API_BASE}/health`;

async function checkHealth() {
    const statusEl = document.getElementById("health-status");
    try {
        const response = await fetch(API_URL);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const data = await response.json();
        statusEl.textContent = `${data.status.toUpperCase()} (${data.service})`;
        statusEl.style.color = "#4ade80"; // Green
    } catch (err) {
        statusEl.textContent = "OFFLINE";
        statusEl.style.color = "#f87171"; // Red
    }
}

checkHealth();