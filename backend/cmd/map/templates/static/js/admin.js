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

function initAdminUsersSearch() {
  var searchInput = document.getElementById("admin-users-search");
  if (!searchInput) {
    return;
  }

  var userCards = Array.prototype.slice.call(document.querySelectorAll("#users section.admin-box[data-user-search]"));
  if (!userCards.length) {
    return;
  }

  function filterUsers() {
    var query = (searchInput.value || "").trim().toLowerCase();
    userCards.forEach(function (card) {
      if (!query) {
        card.style.display = "";
        return;
      }

      var haystack = (card.getAttribute("data-user-search") || "").toLowerCase();
      card.style.display = haystack.indexOf(query) !== -1 ? "" : "none";
    });
  }

  searchInput.addEventListener("input", filterUsers);
}

function initAdminTeamsSearch() {
  var searchInput = document.getElementById("admin-teams-search");
  if (!searchInput) {
    return;
  }

  var teamCards = Array.prototype.slice.call(document.querySelectorAll("#teams section.admin-box[data-team-search]"));
  if (!teamCards.length) {
    return;
  }

  function filterTeams() {
    var query = (searchInput.value || "").trim().toLowerCase();
    teamCards.forEach(function (card) {
      if (!query) {
        card.style.display = "";
        return;
      }

      var haystack = (card.getAttribute("data-team-search") || "").toLowerCase();
      card.style.display = haystack.indexOf(query) !== -1 ? "" : "none";
    });
  }

  searchInput.addEventListener("input", filterTeams);
}

function createAdminUser(createURL, payload) {
  return fetch(createURL, {
    method: "POST",
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    body: JSON.stringify(payload),
  }).then(function (response) {
    return response
      .json()
      .catch(function () {
        return {};
      })
      .then(function (data) {
        if (!response.ok || data.success === false) {
          throw new Error(data.message || "Failed to create user");
        }
        return data;
      });
  });
}

function createAdminTeam(createURL, payload) {
  return fetch(createURL, {
    method: "POST",
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    body: JSON.stringify(payload),
  }).then(function (response) {
    return response
      .json()
      .catch(function () {
        return {};
      })
      .then(function (data) {
        if (!response.ok || data.success === false) {
          throw new Error(data.message || "Failed to create team");
        }
        return data;
      });
  });
}

function initAdminAddUserModal() {
  var addUserBtn = document.querySelector('[data-action="add-new-user"]');
  if (!addUserBtn) {
    return;
  }

  addUserBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var createURL = addUserBtn.getAttribute("data-create-url");
    if (!createURL) {
      showTransientAdminStatus("error", "Missing user creation URL");
      return;
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      showTransientAdminStatus("error", "Modal system unavailable");
      return;
    }

    MAP_CTF.modal.loadPopup("add-user", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var form = modal.querySelector("#admin-add-user-form");
      if (!form) {
        return;
      }

      var teamSelect = form.querySelector('select[name="team_id"]');
      var teamsTemplate = document.getElementById("admin-users-team-options-template");
      if (teamSelect && teamsTemplate) {
        teamSelect.innerHTML = teamsTemplate.innerHTML;
      }

      var usernameInput = form.querySelector('input[name="username"]');
      if (usernameInput) {
        usernameInput.focus();
      }

      form.addEventListener("submit", function (submitEvent) {
        submitEvent.preventDefault();

        if (form.dataset.submitting === "true") {
          return;
        }

        var username = (form.querySelector('input[name="username"]').value || "").trim();
        var password = form.querySelector('input[name="password"]').value || "";
        var name = (form.querySelector('input[name="name"]').value || "").trim();
        var email = (form.querySelector('input[name="email"]').value || "").trim();
        var teamField = form.querySelector('[name="team_id"]');
        var teamID = teamField && typeof teamField.value === "string" ? teamField.value.trim() : "";

        if (!username || !password) {
          showTransientAdminStatus("error", "Username and password are required");
          return;
        }

        form.dataset.submitting = "true";

        createAdminUser(createURL, {
          username: username,
          password: password,
          name: name,
          email: email,
          team_id: teamID,
        })
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "User created");
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to create user");
          })
          .finally(function () {
            delete form.dataset.submitting;
          });
      });
    });
  });
}

function initAdminAddTeamModal() {
  var addTeamBtn = document.querySelector('[data-action="add-new-team"]');
  if (!addTeamBtn) {
    return;
  }

  addTeamBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var createURL = addTeamBtn.getAttribute("data-create-url");
    if (!createURL) {
      showTransientAdminStatus("error", "Missing team creation URL");
      return;
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      showTransientAdminStatus("error", "Modal system unavailable");
      return;
    }

    MAP_CTF.modal.loadPopup("add-team", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var form = modal.querySelector("#admin-add-team-form");
      if (!form) {
        return;
      }

      var logoSelect = form.querySelector('select[name="logo"]');
      var logosTemplate = document.getElementById("admin-teams-logo-options-template");
      if (logoSelect && logosTemplate) {
        logoSelect.innerHTML = logosTemplate.innerHTML;
      }

      var nameInput = form.querySelector('input[name="name"]');
      if (nameInput) {
        nameInput.focus();
      }

      form.addEventListener("submit", function (submitEvent) {
        submitEvent.preventDefault();

        if (form.dataset.submitting === "true") {
          return;
        }

        var name = (form.querySelector('input[name="name"]').value || "").trim();
        var logoField = form.querySelector('[name="logo"]');
        var logo = logoField && typeof logoField.value === "string" ? logoField.value.trim() : "";

        if (!name) {
          showTransientAdminStatus("error", "Team name is required");
          return;
        }

        form.dataset.submitting = "true";

        createAdminTeam(createURL, {
          name: name,
          logo: logo,
        })
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "Team created");
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to create team");
          })
          .finally(function () {
            delete form.dataset.submitting;
          });
      });
    });
  });
}

document.addEventListener("DOMContentLoaded", function () {
  initAdminStatusFromServerState();
  initAdminLogoutModal();
  initAdminAjaxForms();
  initAdminUsersSearch();
  initAdminTeamsSearch();
  initAdminAddUserModal();
  initAdminAddTeamModal();
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
