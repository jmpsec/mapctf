var adminStatusResetTimer = null;
var adminStatusResetDelayMs = 5000;

function setAdminStatus(status, message) {
  var statusEl = document.querySelector(".admin-section--status");
  if (!statusEl) {
    return;
  }

  var statusValueEl = statusEl.querySelector(".highlighted");
  var statusMessageEl = statusEl.querySelector(".admin-section--status-message");
  if (!statusValueEl || !statusMessageEl) {
    return;
  }

  var readyStatus = statusEl.dataset.readyStatus || "ready";
  var value = status && status.trim() ? status.trim() : readyStatus;
  var detail = message && message.trim() ? " - " + message.trim() : "";

  statusValueEl.textContent = value;
  statusMessageEl.textContent = detail;
}

function resetAdminStatusToReady() {
  setAdminStatus("", "");
}

function showTransientAdminStatus(status, message) {
  setAdminStatus(status, message);

  if (adminStatusResetTimer !== null) {
    window.clearTimeout(adminStatusResetTimer);
  }

  adminStatusResetTimer = window.setTimeout(function () {
    resetAdminStatusToReady();
    adminStatusResetTimer = null;
  }, adminStatusResetDelayMs);
}

function initAdminStatusFromServerState() {
  var statusEl = document.querySelector(".admin-section--status");
  if (!statusEl) {
    return;
  }

  var params = new URLSearchParams(window.location.search);
  var initialStatus = params.get("status") || "";
  var initialMessage = params.get("msg") || "";

  if (initialStatus || initialMessage) {
    showTransientAdminStatus(initialStatus, initialMessage);
    return;
  }

  resetAdminStatusToReady();
}

function doAdminLogout(logoutURL) {
  fetch(logoutURL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: "{}",
    credentials: "same-origin",
  })
    .then(function (response) {
      if (!response.ok) {
        throw new Error("logout failed");
      }
      return response.json();
    })
    .then(function (data) {
      if (data && data.redirect) {
        window.location.replace(data.redirect);
        return;
      }
      window.location.replace(logoutURL.replace("/logout", "/login"));
    })
    .catch(function () {
      window.location.replace(logoutURL.replace("/logout", "/login"));
    });
}

function initAdminLogoutModal() {
  var logoutLinks = document.querySelectorAll(".js-prompt-logout");
  logoutLinks.forEach(function (logoutLink) {
    logoutLink.addEventListener(
      "click",
      function (event) {
        event.preventDefault();
        event.stopImmediatePropagation();

        var logoutURL = logoutLink.getAttribute("data-logout-url") || logoutLink.getAttribute("href");
        if (!logoutURL) {
          return;
        }

        if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
          doAdminLogout(logoutURL);
          return;
        }

        MAP_CTF.modal.loadPopup("action-logout", function () {
          var confirmBtn = document.querySelector("#mctf-modal .js-confirm-logout");
          if (!confirmBtn) {
            return;
          }
          confirmBtn.addEventListener("click", function (confirmEvent) {
            confirmEvent.preventDefault();
            doAdminLogout(logoutURL);
          });
        });
      },
      true,
    );
  });
}

function submitAdminForm(form) {
  if (!form) {
    return;
  }

  if (form.dataset.submitting === "true") {
    return;
  }

  var action = form.getAttribute("action");
  if (!action) {
    return;
  }

  var method = (form.getAttribute("method") || "post").toUpperCase();
  var formData = new FormData(form);
  var payload = {};
  formData.forEach(function (value, key) {
    payload[key] = value;
  });

  form.dataset.submitting = "true";

  fetch(action, {
    method: method,
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    body: JSON.stringify(payload),
  })
    .then(function (response) {
      return response
        .json()
        .catch(function () {
          return {};
        })
        .then(function (data) {
          if (!response.ok || data.success === false) {
            throw new Error(data.message || "Request failed");
          }
          showTransientAdminStatus(data.status || "ok", data.message || "Updated");
        });
    })
    .catch(function (error) {
      showTransientAdminStatus("error", error.message || "Request failed");
    })
    .finally(function () {
      delete form.dataset.submitting;
    });
}

function initAdminAjaxForms() {
  var forms = document.querySelectorAll(".mctf-admin-main form");
  forms.forEach(function (form) {
    form.addEventListener("submit", function (event) {
      event.preventDefault();
      submitAdminForm(form);
    });
  });
}

document.addEventListener("DOMContentLoaded", function () {
  initAdminStatusFromServerState();
  initAdminLogoutModal();
  initAdminAjaxForms();
});

function saveSettingValue(input) {
  if (!input) {
    return;
  }

  var form = input.form;
  if (!form) {
    return;
  }

  if (typeof form.requestSubmit === "function") {
    form.requestSubmit();
    return;
  }

  submitAdminForm(form);
}
