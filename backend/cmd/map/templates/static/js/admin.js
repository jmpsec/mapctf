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

function createAdminChallenge(createURL, payload) {
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
          throw new Error(data.message || "Failed to create challenge");
        }
        return data;
      });
  });
}

function createAdminChallengeUpdate(updateURL, payload) {
  return fetch(updateURL, {
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
          throw new Error(data.message || "Failed to update challenge");
        }
        return data;
      });
  });
}

function deleteAdminChallenge(deleteURL) {
  return fetch(deleteURL, {
    method: "POST",
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    body: "{}",
  }).then(function (response) {
    return response
      .json()
      .catch(function () {
        return {};
      })
      .then(function (data) {
        if (!response.ok || data.success === false) {
          throw new Error(data.message || "Failed to delete challenge");
        }
        return data;
      });
  });
}

function createAdminCategory(createURL, payload) {
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
          throw new Error(data.message || "Failed to create category");
        }
        return data;
      });
  });
}

function updateAdminCategory(updateURL, payload) {
  return fetch(updateURL, {
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
          throw new Error(data.message || "Failed to update category");
        }
        return data;
      });
  });
}

function importAdminChallenges(importURL, formData) {
  return fetch(importURL, {
    method: "POST",
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    body: formData,
  }).then(function (response) {
    return response
      .json()
      .catch(function () {
        return {};
      })
      .then(function (data) {
        if (!response.ok || data.success === false) {
          throw new Error(data.message || "Failed to import challenges");
        }
        return data;
      });
  });
}

function postAdminActionJSON(actionURL) {
  return fetch(actionURL, {
    method: "POST",
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    body: "{}",
  }).then(function (response) {
    return response
      .json()
      .catch(function () {
        return {};
      })
      .then(function (data) {
        if (!response.ok || data.success === false) {
          throw new Error(data.message || "Request failed");
        }
        return data;
      });
  });
}

function initAdminImportChallengesButton() {
  var importBtn = document.querySelector('[data-action="import-all-challenges"]');
  if (!importBtn) {
    return;
  }

  var fileInput = document.getElementById("admin-challenges-import-file");
  if (!fileInput) {
    showTransientAdminStatus("error", "Import file input not found");
    return;
  }

  importBtn.addEventListener("click", function (event) {
    event.preventDefault();
    fileInput.click();
  });

  fileInput.addEventListener("change", function () {
    var importURL = importBtn.getAttribute("data-import-url");
    if (!importURL) {
      showTransientAdminStatus("error", "Missing import URL");
      fileInput.value = "";
      return;
    }

    if (importBtn.dataset.submitting === "true") {
      fileInput.value = "";
      return;
    }

    if (!fileInput.files || !fileInput.files.length) {
      return;
    }

    var importFile = fileInput.files[0];
    var formData = new FormData();
    formData.append("file", importFile);

    importBtn.dataset.submitting = "true";
    importBtn.disabled = true;

    importAdminChallenges(importURL, formData)
      .then(function (data) {
        showTransientAdminStatus(data.status || "ok", data.message || "Challenges imported");
        window.location.reload();
      })
      .catch(function (error) {
        showTransientAdminStatus("error", error.message || "Failed to import challenges");
      })
      .finally(function () {
        delete importBtn.dataset.submitting;
        importBtn.disabled = false;
        fileInput.value = "";
      });
  });
}

function initAdminChallengeActionsButtons() {
  var enableAllChallengesBtn = document.querySelector('[data-action="enable-all-challenges"]');
  if (enableAllChallengesBtn) {
    enableAllChallengesBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var enableAllURL = enableAllChallengesBtn.getAttribute("data-enable-all-url");
      if (!enableAllURL) {
        showTransientAdminStatus("error", "Missing enable all challenges URL");
        return;
      }
      if (enableAllChallengesBtn.dataset.submitting === "true") {
        return;
      }

      enableAllChallengesBtn.dataset.submitting = "true";
      enableAllChallengesBtn.disabled = true;

      postAdminActionJSON(enableAllURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All challenges enabled");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to enable all challenges");
        })
        .finally(function () {
          delete enableAllChallengesBtn.dataset.submitting;
          enableAllChallengesBtn.disabled = false;
        });
    });
  }

  var disableAllChallengesBtn = document.querySelector('[data-action="disable-all-challenges"]');
  if (disableAllChallengesBtn) {
    disableAllChallengesBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var disableAllURL = disableAllChallengesBtn.getAttribute("data-disable-all-url");
      if (!disableAllURL) {
        showTransientAdminStatus("error", "Missing disable all challenges URL");
        return;
      }
      if (disableAllChallengesBtn.dataset.submitting === "true") {
        return;
      }

      disableAllChallengesBtn.dataset.submitting = "true";
      disableAllChallengesBtn.disabled = true;

      postAdminActionJSON(disableAllURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All challenges disabled");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to disable all challenges");
        })
        .finally(function () {
          delete disableAllChallengesBtn.dataset.submitting;
          disableAllChallengesBtn.disabled = false;
        });
    });
  }

  var deleteAllChallengesBtn = document.querySelector('[data-action="delete-all-challenges"]');
  if (deleteAllChallengesBtn) {
    deleteAllChallengesBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var deleteAllURL = deleteAllChallengesBtn.getAttribute("data-delete-all-url");
      if (!deleteAllURL) {
        showTransientAdminStatus("error", "Missing delete all challenges URL");
        return;
      }
      if (deleteAllChallengesBtn.dataset.submitting === "true") {
        return;
      }

      function runDeleteAllChallenges() {
        deleteAllChallengesBtn.dataset.submitting = "true";
        deleteAllChallengesBtn.disabled = true;

        postAdminActionJSON(deleteAllURL)
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "All challenges deleted");
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to delete all challenges");
          })
          .finally(function () {
            delete deleteAllChallengesBtn.dataset.submitting;
            deleteAllChallengesBtn.disabled = false;
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
          });
      }

      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        if (window.confirm("Delete all challenges? This cannot be undone.")) {
          runDeleteAllChallenges();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-delete-all-challenges", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var confirmBtn = modal.querySelector(".js-confirm-delete-all-challenges");
        if (!confirmBtn) {
          return;
        }
        confirmBtn.addEventListener("click", function (confirmEvent) {
          confirmEvent.preventDefault();
          runDeleteAllChallenges();
        });
      });
    });
  }

  var deleteAllCategoriesBtn = document.querySelector('[data-action="delete-all-categories"]');
  if (!deleteAllCategoriesBtn) {
    return;
  }

  deleteAllCategoriesBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var deleteAllCategoriesURL = deleteAllCategoriesBtn.getAttribute("data-delete-all-categories-url");
    if (!deleteAllCategoriesURL) {
      showTransientAdminStatus("error", "Missing delete all categories URL");
      return;
    }
    if (deleteAllCategoriesBtn.dataset.submitting === "true") {
      return;
    }

    function runDeleteAllCategories() {
      deleteAllCategoriesBtn.dataset.submitting = "true";
      deleteAllCategoriesBtn.disabled = true;

      postAdminActionJSON(deleteAllCategoriesURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All categories deleted");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to delete all categories");
        })
        .finally(function () {
          delete deleteAllCategoriesBtn.dataset.submitting;
          deleteAllCategoriesBtn.disabled = false;
          if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
            MAP_CTF.modal.close();
          }
        });
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      if (window.confirm("Delete all categories? Challenges must be deleted first.")) {
        runDeleteAllCategories();
      }
      return;
    }

    MAP_CTF.modal.loadPopup("action-delete-all-categories", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var confirmBtn = modal.querySelector(".js-confirm-delete-all-categories");
      if (!confirmBtn) {
        return;
      }
      confirmBtn.addEventListener("click", function (confirmEvent) {
        confirmEvent.preventDefault();
        runDeleteAllCategories();
      });
    });
  });
}

function initAdminAddChallengeModal() {
  var addChallengeBtn = document.querySelector('[data-action="add-new-challenge"]');
  if (!addChallengeBtn) {
    return;
  }

  addChallengeBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var createURL = addChallengeBtn.getAttribute("data-create-url");
    if (!createURL) {
      showTransientAdminStatus("error", "Missing challenge creation URL");
      return;
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      showTransientAdminStatus("error", "Modal system unavailable");
      return;
    }

    MAP_CTF.modal.loadPopup("add-challenge", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var form = modal.querySelector("#admin-add-challenge-form");
      if (!form) {
        return;
      }

      var countrySelect = form.querySelector('select[name="country"]');
      var availableCountriesTemplate = document.getElementById("admin-challenges-available-country-options-template");
      if (countrySelect && availableCountriesTemplate) {
        countrySelect.innerHTML = availableCountriesTemplate.innerHTML;
      }

      var categorySelect = form.querySelector('select[name="category_id"]');
      var categoriesTemplate = document.getElementById("admin-challenges-category-options-template");
      if (categorySelect && categoriesTemplate) {
        categorySelect.innerHTML = categoriesTemplate.innerHTML;
      }

      var titleInput = form.querySelector('input[name="title"]');
      if (titleInput) {
        titleInput.focus();
      }

      form.addEventListener("submit", function (submitEvent) {
        submitEvent.preventDefault();

        if (form.dataset.submitting === "true") {
          return;
        }

        var title = (form.querySelector('input[name="title"]').value || "").trim();
        var description = (form.querySelector('input[name="description"]').value || "").trim();
        var categoryID = (form.querySelector('select[name="category_id"]').value || "").trim();
        var country = (form.querySelector('[name="country"]').value || "").trim();
        var flag = (form.querySelector('input[name="flag"]').value || "").trim();
        var hint = (form.querySelector('input[name="hint"]').value || "").trim();
        var points = String(form.querySelector('input[name="points"]').value || "0").trim();
        var bonus = String(form.querySelector('input[name="bonus"]').value || "0").trim();
        var bonusDecay = String(form.querySelector('input[name="bonus_decay"]').value || "0").trim();
        var penalty = String(form.querySelector('input[name="penalty"]').value || "0").trim();
        var active = String(form.querySelector('select[name="active"]').value || "true").trim();

        if (!title || !flag) {
          showTransientAdminStatus("error", "Title and flag are required");
          return;
        }
        if (!categoryID) {
          showTransientAdminStatus("error", "Category is required");
          return;
        }

        form.dataset.submitting = "true";

        createAdminChallenge(createURL, {
          title: title,
          description: description,
          category_id: categoryID,
          country: country,
          active: active,
          points: points,
          bonus: bonus,
          bonus_decay: bonusDecay,
          penalty: penalty,
          flag: flag,
          hint: hint,
        })
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "Challenge created");
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to create challenge");
          })
          .finally(function () {
            delete form.dataset.submitting;
          });
      });
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

function openAdminCategoryModal(options) {
  if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
    showTransientAdminStatus("error", "Modal system unavailable");
    return;
  }

  MAP_CTF.modal.loadPopup("add-category", function () {
    var modal = document.getElementById("mctf-modal");
    if (!modal) {
      return;
    }

    var form = modal.querySelector("#admin-add-category-form");
    if (!form) {
      return;
    }

    var mode = options && options.mode === "edit" ? "edit" : "create";
    var submitURL = (options && options.url) || "";
    if (!submitURL) {
      showTransientAdminStatus("error", "Missing category action URL");
      return;
    }

    var modalTitle = modal.querySelector(".modal-title .highlighted");
    if (modalTitle) {
      modalTitle.textContent = mode === "edit" ? "Edit Category" : "Add Category";
    }

    var submitBtn = form.querySelector('button[type="submit"]');
    if (submitBtn) {
      submitBtn.textContent = mode === "edit" ? "Save Changes" : "Create Category";
    }

    var initialName = (options && options.name) || "";
    var initialDescription = (options && options.description) || "";
    var initialLogo = (options && options.logo) || "fa-solid fa-globe";

    var nameInput = form.querySelector('input[name="name"]');
    var descriptionInput = form.querySelector('input[name="description"]');
    var logoSelect = form.querySelector('select[name="logo"]');
    var logoPreview = form.querySelector("#admin-add-category-logo-preview i");

    if (nameInput) {
      nameInput.value = initialName;
      nameInput.focus();
    }
    if (descriptionInput) {
      descriptionInput.value = initialDescription;
    }
    if (logoSelect) {
      logoSelect.value = initialLogo;
    }

    var updateLogoPreview = function () {
      if (!logoSelect || !logoPreview) {
        return;
      }
      logoPreview.className = logoSelect.value || "fa-solid fa-globe";
    };
    if (logoSelect) {
      logoSelect.addEventListener("change", updateLogoPreview);
    }
    updateLogoPreview();

    form.addEventListener("submit", function (submitEvent) {
      submitEvent.preventDefault();

      if (form.dataset.submitting === "true") {
        return;
      }

      var name = (form.querySelector('input[name="name"]').value || "").trim();
      var description = (form.querySelector('input[name="description"]').value || "").trim();
      var logo = (form.querySelector('select[name="logo"]').value || "").trim();

      if (!name) {
        showTransientAdminStatus("error", "Category name is required");
        return;
      }

      form.dataset.submitting = "true";

      var request = mode === "edit" ? updateAdminCategory(submitURL, { name: name, description: description, logo: logo }) : createAdminCategory(submitURL, { name: name, description: description, logo: logo });

      request
        .then(function (data) {
          var fallbackMessage = mode === "edit" ? "Category updated" : "Category created";
          showTransientAdminStatus(data.status || "ok", data.message || fallbackMessage);
          if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
            MAP_CTF.modal.close();
          }
          window.location.reload();
        })
        .catch(function (error) {
          var fallbackError = mode === "edit" ? "Failed to update category" : "Failed to create category";
          showTransientAdminStatus("error", error.message || fallbackError);
        })
        .finally(function () {
          delete form.dataset.submitting;
        });
    });
  });
}

function initAdminAddCategoryModal() {
  var addCategoryBtn = document.querySelector('[data-action="add-new-category"]');
  if (!addCategoryBtn) {
    return;
  }

  addCategoryBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var createURL = addCategoryBtn.getAttribute("data-create-url");
    if (!createURL) {
      showTransientAdminStatus("error", "Missing category creation URL");
      return;
    }

    openAdminCategoryModal({
      mode: "create",
      url: createURL,
      name: "",
      description: "",
      logo: "fa-solid fa-globe",
    });
  });
}

function initAdminEditCategoryButtons() {
  var editButtons = document.querySelectorAll('[data-action="edit-category"]');
  if (!editButtons.length) {
    return;
  }

  editButtons.forEach(function (editBtn) {
    editBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var updateURL = editBtn.getAttribute("data-update-url");
      if (!updateURL) {
        showTransientAdminStatus("error", "Missing category update URL");
        return;
      }

      openAdminCategoryModal({
        mode: "edit",
        url: updateURL,
        name: editBtn.getAttribute("data-category-name") || "",
        description: editBtn.getAttribute("data-category-description") || "",
        logo: editBtn.getAttribute("data-category-logo") || "fa-solid fa-globe",
      });
    });
  });
}

function initAdminDeleteCategoryButtons() {
  var deleteButtons = document.querySelectorAll('[data-action="delete-category"]');
  if (!deleteButtons.length) {
    return;
  }

  deleteButtons.forEach(function (deleteBtn) {
    deleteBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var deleteURL = deleteBtn.getAttribute("data-delete-url");
      if (!deleteURL) {
        showTransientAdminStatus("error", "Missing category delete URL");
        return;
      }
      if (deleteBtn.dataset.submitting === "true") {
        return;
      }

      var categoryName = (deleteBtn.getAttribute("data-category-name") || "").trim();
      function runDelete() {
        deleteBtn.dataset.submitting = "true";
        deleteBtn.disabled = true;

        postAdminActionJSON(deleteURL)
          .then(function (data) {
            var row = deleteBtn.closest(".admin-setting-row");
            if (row) {
              row.remove();
            }
            showTransientAdminStatus(data.status || "ok", data.message || "Category deleted");
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to delete category");
          })
          .finally(function () {
            delete deleteBtn.dataset.submitting;
            deleteBtn.disabled = false;
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
          });
      }

      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        var promptMessage = "Delete category" + (categoryName ? " '" + categoryName + "'" : "") + "?";
        if (window.confirm(promptMessage)) {
          runDelete();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-delete-category", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var titlePlaceholder = modal.querySelector(".js-delete-category-title");
        if (titlePlaceholder) {
          titlePlaceholder.textContent = categoryName ? " " + categoryName : "";
        }

        var confirmBtn = modal.querySelector(".js-confirm-delete-category");
        if (!confirmBtn) {
          return;
        }
        confirmBtn.addEventListener("click", function (confirmEvent) {
          confirmEvent.preventDefault();
          runDelete();
        });
      });
    });
  });
}

function initAdminChallengeSaveButtons() {
  var saveButtons = document.querySelectorAll('[data-action="save-challenge"]');
  if (!saveButtons.length) {
    return;
  }

  saveButtons.forEach(function (saveBtn) {
    saveBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var updateURL = saveBtn.getAttribute("data-update-url");
      if (!updateURL) {
        showTransientAdminStatus("error", "Missing challenge update URL");
        return;
      }

      var challengeRow = saveBtn.closest(".admin-challenge-row");
      if (!challengeRow) {
        showTransientAdminStatus("error", "Unable to locate challenge row");
        return;
      }

      var form = challengeRow.querySelector(".admin-challenge-form");
      if (!form) {
        showTransientAdminStatus("error", "Unable to locate challenge form");
        return;
      }

      if (form.dataset.submitting === "true") {
        return;
      }

      var title = (form.querySelector('input[name="title"]').value || "").trim();
      var description = (form.querySelector('textarea[name="description"]').value || "").trim();
      var categoryID = (form.querySelector('select[name="category_id"]').value || "").trim();
      var country = (form.querySelector('select[name="country"]').value || "").trim();
      var flag = (form.querySelector('input[name="flag"]').value || "").trim();
      var hint = (form.querySelector('textarea[name="hint"]').value || "").trim();
      var points = String(form.querySelector('input[name="points"]').value || "0").trim();
      var bonus = String(form.querySelector('input[name="bonus"]').value || "0").trim();
      var bonusDecay = String(form.querySelector('input[name="bonus_decay"]').value || "0").trim();
      var penalty = String(form.querySelector('input[name="penalty"]').value || "0").trim();
      var activeRadio = challengeRow.querySelector('.admin-activity-status-toggle input[type="radio"]:checked');
      var active = activeRadio ? String(activeRadio.value || "true").trim() : "true";

      if (!title || !flag) {
        showTransientAdminStatus("error", "Title and flag are required");
        return;
      }
      if (!categoryID) {
        showTransientAdminStatus("error", "Category is required");
        return;
      }

      form.dataset.submitting = "true";
      saveBtn.disabled = true;

      createAdminChallengeUpdate(updateURL, {
        title: title,
        description: description,
        category_id: categoryID,
        country: country,
        active: active,
        points: points,
        bonus: bonus,
        bonus_decay: bonusDecay,
        penalty: penalty,
        flag: flag,
        hint: hint,
      })
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "Challenge updated");
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to update challenge");
        })
        .finally(function () {
          delete form.dataset.submitting;
          saveBtn.disabled = false;
        });
    });
  });
}

function initAdminChallengeDeleteButtons() {
  var deleteButtons = document.querySelectorAll('[data-action="delete-challenge"]');
  if (!deleteButtons.length) {
    return;
  }

  deleteButtons.forEach(function (deleteBtn) {
    deleteBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var deleteURL = deleteBtn.getAttribute("data-delete-url");
      if (!deleteURL) {
        showTransientAdminStatus("error", "Missing challenge delete URL");
        return;
      }

      var challengeRow = deleteBtn.closest(".admin-challenge-row");
      if (!challengeRow) {
        showTransientAdminStatus("error", "Unable to locate challenge row");
        return;
      }

      if (deleteBtn.dataset.submitting === "true") {
        return;
      }
      var challengeTitleEl = challengeRow.querySelector(".admin-challenges-editor > label");
      var challengeTitle = challengeTitleEl ? String(challengeTitleEl.textContent || "").trim() : "";

      function runDelete() {
        deleteBtn.dataset.submitting = "true";
        deleteBtn.disabled = true;

        deleteAdminChallenge(deleteURL)
          .then(function (data) {
            challengeRow.remove();
            showTransientAdminStatus(data.status || "ok", data.message || "Challenge deleted");
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to delete challenge");
          })
          .finally(function () {
            delete deleteBtn.dataset.submitting;
            deleteBtn.disabled = false;
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
          });
      }

      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        if (window.confirm("Delete challenge" + (challengeTitle ? " '" + challengeTitle + "'" : "") + "?")) {
          runDelete();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-delete-challenge", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var titlePlaceholder = modal.querySelector(".js-delete-challenge-title");
        if (titlePlaceholder) {
          titlePlaceholder.textContent = challengeTitle ? " " + challengeTitle : "";
        }

        var confirmBtn = modal.querySelector(".js-confirm-delete-challenge");
        if (!confirmBtn) {
          return;
        }
        confirmBtn.addEventListener("click", function (confirmEvent) {
          confirmEvent.preventDefault();
          runDelete();
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
  initAdminImportChallengesButton();
  initAdminChallengeActionsButtons();
  initAdminChallengeSaveButtons();
  initAdminChallengeDeleteButtons();
  initAdminAddChallengeModal();
  initAdminAddUserModal();
  initAdminAddTeamModal();
  initAdminAddCategoryModal();
  initAdminEditCategoryButtons();
  initAdminDeleteCategoryButtons();
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
