(function () {
  async function refreshDashboard() {
    try {
      if (document.hidden) return;
      var response = await fetch(window.location.href, { cache: "no-store", signal: AbortSignal.timeout(10000) });
      if (!response.ok || response.redirected) throw new Error("Dashboard unavailable");
      var page = new DOMParser().parseFromString(await response.text(), "text/html");
      var dashboard = page.getElementById("admin-dashboard");
      if (!dashboard) throw new Error("Dashboard missing");
      document.getElementById("admin-dashboard").replaceWith(dashboard);
      document.getElementById("dashboard-refresh-error").hidden = true;
    } catch (error) {
      document.getElementById("dashboard-refresh-error").hidden = false;
    } finally {
      window.setTimeout(refreshDashboard, 15000);
    }
  }
  window.setTimeout(refreshDashboard, 15000);
})();
