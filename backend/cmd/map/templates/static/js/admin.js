var adminStatusResetTimer = null;
var adminStatusResetDelayMs = 5000;
var adminCountriesCache = [];

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
      populateCountrySelects(adminCountriesCache);

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

      var logoSelect = form.querySelector('select[name="logo"]');
      var logoPreview = form.querySelector("#admin-add-category-logo-preview i");
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
        var description = (form.querySelector('input[name="description"]').value || "").trim();
        var logo = (form.querySelector('select[name="logo"]').value || "").trim();

        if (!name) {
          showTransientAdminStatus("error", "Category name is required");
          return;
        }

        form.dataset.submitting = "true";

        createAdminCategory(createURL, {
          name: name,
          description: description,
          logo: logo,
        })
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "Category created");
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to create category");
          })
          .finally(function () {
            delete form.dataset.submitting;
          });
      });
    });
  });
}

function extractCountryNamesFromListviewMarkup(markup) {
  if (!markup || typeof markup !== "string") {
    return [];
  }

  var parser = new DOMParser();
  var doc = parser.parseFromString(markup, "text/html");
  var rows = doc.querySelectorAll("tr[data-country]");
  var seen = {};
  var countries = [];

  rows.forEach(function (row) {
    var country = (row.getAttribute("data-country") || "").trim();
    if (!country || seen[country]) {
      return;
    }
    seen[country] = true;
    countries.push(country);
  });

  countries.sort(function (a, b) {
    return a.localeCompare(b);
  });

  if (countries.length === 0) {
    var matches = markup.match(/data-country="([^"]+)"/g) || [];
    matches.forEach(function (entry) {
      var country = entry.replace('data-country="', "").replace('"', "").trim();
      if (!country || seen[country]) {
        return;
      }
      seen[country] = true;
      countries.push(country);
    });
    countries.sort(function (a, b) {
      return a.localeCompare(b);
    });
  }

  return countries;
}

function getFallbackCountries() {
  return [
    "Argentina",
    "Australia",
    "Brazil",
    "Canada",
    "China",
    "France",
    "Germany",
    "India",
    "Italy",
    "Japan",
    "Mexico",
    "Netherlands",
    "South Africa",
    "Spain",
    "United Kingdom",
    "United States",
  ];
}

function populateCountrySelects(countries) {
  var countrySelects = document.querySelectorAll('select[name="country"]');
  if (!countrySelects.length) {
    return;
  }

  countrySelects.forEach(function (select) {
    var previousValue = select.value || "";
    var currentValue = (select.dataset && select.dataset.currentCountry) ? String(select.dataset.currentCountry).trim() : "";
    select.innerHTML = "";

    var placeholder = document.createElement("option");
    placeholder.value = "";
    placeholder.textContent = "Select country";
    select.appendChild(placeholder);

    if (countries && countries.length) {
      countries.forEach(function (country) {
        var option = document.createElement("option");
        option.value = country;
        option.textContent = country;
        select.appendChild(option);
      });
    }

    if (previousValue) {
      select.value = previousValue;
    } else if (currentValue) {
      select.value = currentValue;

      if (select.value !== currentValue) {
        var matchedOption = null;
        Array.prototype.slice.call(select.options).forEach(function (opt) {
          if (String(opt.value || "").toLowerCase() === currentValue.toLowerCase()) {
            matchedOption = opt;
          }
        });

        if (matchedOption) {
          select.value = matchedOption.value;
        } else {
          var existingOption = document.createElement("option");
          existingOption.value = currentValue;
          existingOption.textContent = currentValue;
          select.appendChild(existingOption);
          select.value = currentValue;
        }
      }
    }
  });
}

function initAdminChallengeCountryDropdowns() {
  var countrySelects = document.querySelectorAll('select[name="country"].admin-country-select');
  if (!countrySelects.length) {
    return;
  }

  fetch("/static/inc/gameboard/listview.html", {
    credentials: "same-origin",
    headers: { Accept: "text/html" },
  })
    .then(function (response) {
      if (!response.ok) {
        throw new Error("Failed to load country list");
      }
      return response.text();
    })
    .then(function (markup) {
      var countries = extractCountryNamesFromListviewMarkup(markup);
      if (!countries.length) {
        countries = getFallbackCountries();
      }
      adminCountriesCache = countries;
      populateCountrySelects(countries);
    })
    .catch(function () {
      adminCountriesCache = getFallbackCountries();
      populateCountrySelects(adminCountriesCache);
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
  initAdminChallengeSaveButtons();
  initAdminChallengeDeleteButtons();
  initAdminChallengeCountryDropdowns();
  initAdminAddChallengeModal();
  initAdminAddUserModal();
  initAdminAddTeamModal();
  initAdminAddCategoryModal();
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
