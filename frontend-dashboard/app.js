const API_URL = "http://localhost:8080/health";

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