var adminStatusResetTimer = null;
var adminStatusResetDelayMs = 5000;
var adminBypassBeforeUnloadWarning = false;

function allowAdminPageExit() {
  adminBypassBeforeUnloadWarning = true;
}

function collectEditorState(container, fieldSelector) {
  var state = {};
  if (!container) {
    return state;
  }

  var fields = container.querySelectorAll(fieldSelector);
  var processedRadioNames = {};

  Array.prototype.forEach.call(fields, function (field) {
    if (!field) {
      return;
    }

    var key = field.getAttribute("data-user-field") || field.getAttribute("data-team-field") || field.name || field.id;
    if (!key) {
      return;
    }

    if (field.type === "radio") {
      if (processedRadioNames[field.name]) {
        return;
      }
      processedRadioNames[field.name] = true;

      var checked = container.querySelector('input[name="' + field.name + '"]:checked');
      state[key] = checked ? String(checked.value || "") : "";
      return;
    }

    state[key] = typeof field.value === "string" ? field.value : "";
  });

  return state;
}

function refreshEditorDirtyState(container, fieldSelector) {
  if (!container) {
    return false;
  }

  var initialState = container.dataset.initialState || "{}";
  var currentState = JSON.stringify(collectEditorState(container, fieldSelector));
  var isDirty = initialState !== currentState;

  container.dataset.unsavedChanges = isDirty ? "true" : "false";
  return isDirty;
}

function initializeEditorDirtyTracking(container, fieldSelector) {
  if (!container) {
    return;
  }

  var syncInitialState = function () {
    container.dataset.initialState = JSON.stringify(collectEditorState(container, fieldSelector));
    container.dataset.unsavedChanges = "false";
  };

  syncInitialState();

  container.addEventListener("input", function () {
    refreshEditorDirtyState(container, fieldSelector);
  });

  container.addEventListener("change", function () {
    refreshEditorDirtyState(container, fieldSelector);
  });

  return syncInitialState;
}

function initializeAdminFormDirtyTracking(form) {
  if (!form) {
    return;
  }

  return initializeEditorDirtyTracking(form, 'input, select, textarea');
}

function isAdminLogoImagePath(logoValue) {
  var logo = (logoValue || "").toString().trim();
  return /^\/static\/img\/team-logos\/badge-[a-z0-9-]+\.(gif|jpe?g|png|svg)$/i.test(logo);
}

var ADMIN_LOGO_UPLOAD_ACCEPT = ".svg,.gif,.png,.jpg,.jpeg,image/svg+xml,image/gif,image/png,image/jpeg";
var ADMIN_LOGO_UPLOAD_FORMATS = ["SVG", "GIF", "PNG", "JPG/JPEG"];
var ADMIN_LOGO_UPLOAD_MAX_BYTES = 512 * 1024;
var ADMIN_LOGO_UPLOAD_MAX_LABEL = "512 KB";
var ADMIN_LOGO_UPLOAD_RECOMMENDED_SIZE = "64 x 48 px";

function formatAdminLogoFileSize(bytes) {
  var size = Number(bytes) || 0;
  if (size < 1024) {
    return size + " B";
  }
  if (size < 1024 * 1024) {
    return Math.ceil(size / 1024) + " KB";
  }
  return (size / (1024 * 1024)).toFixed(1).replace(/\.0$/, "") + " MB";
}

function ensureAdminLogoUploadFormats(form) {
  if (!form) {
    return;
  }

  var logoFileInput = form.querySelector('input[name="logo_file"]');
  if (logoFileInput) {
    logoFileInput.setAttribute("accept", ADMIN_LOGO_UPLOAD_ACCEPT);
    logoFileInput.setAttribute("required", "required");
  }

  var uploadField = logoFileInput ? logoFileInput.parentNode : null;
  var uploadLabel = uploadField ? uploadField.querySelector('label[for="admin-add-logo-file"], label') : null;
  if (uploadLabel) {
    uploadLabel.textContent = "Upload Logo File";
  }

  var formats = form.querySelector(".admin-logo-upload-formats");
  if (!formats) {
    formats = document.createElement("div");
    formats.className = "admin-logo-upload-formats";
    formats.setAttribute("aria-label", "Accepted logo upload formats");
    if (uploadField && uploadField.parentNode) {
      uploadField.parentNode.insertBefore(formats, uploadField.nextSibling);
    } else {
      form.insertBefore(formats, form.firstChild);
    }
  }

  while (formats.firstChild) {
    formats.removeChild(formats.firstChild);
  }

  var formatCopy = document.createElement("div");
  formatCopy.className = "admin-logo-upload-formats-copy";

  var formatLabel = document.createElement("span");
  formatLabel.className = "admin-logo-upload-formats-label";
  formatLabel.textContent = "Accepted formats";
  formatCopy.appendChild(formatLabel);

  var formatNote = document.createElement("span");
  formatNote.className = "admin-logo-upload-formats-note";
  formatNote.textContent = "Upload an SVG, GIF, PNG, or JPG/JPEG file.";
  formatCopy.appendChild(formatNote);
  formats.appendChild(formatCopy);

  var limitChips = document.createElement("div");
  limitChips.className = "admin-logo-upload-limits";
  limitChips.setAttribute("aria-label", "Logo upload size limits");

  var maxChip = document.createElement("span");
  maxChip.className = "admin-logo-limit-chip";
  maxChip.textContent = "Max file size: " + ADMIN_LOGO_UPLOAD_MAX_LABEL;
  limitChips.appendChild(maxChip);

  var recommendedChip = document.createElement("span");
  recommendedChip.className = "admin-logo-limit-chip";
  recommendedChip.textContent = "Recommended size: " + ADMIN_LOGO_UPLOAD_RECOMMENDED_SIZE;
  limitChips.appendChild(recommendedChip);
  formats.appendChild(limitChips);

  var formatChips = document.createElement("div");
  formatChips.className = "admin-logo-format-chips";
  for (var i = 0; i < ADMIN_LOGO_UPLOAD_FORMATS.length; i += 1) {
    var chip = document.createElement("span");
    chip.className = "admin-logo-format-chip";
    chip.textContent = ADMIN_LOGO_UPLOAD_FORMATS[i];
    formatChips.appendChild(chip);
  }
  formats.appendChild(formatChips);

  var selectedFileSize = form.querySelector(".admin-logo-upload-file-size");
  if (!selectedFileSize) {
    selectedFileSize = document.createElement("p");
    selectedFileSize.className = "admin-logo-upload-file-size";
    selectedFileSize.setAttribute("aria-live", "polite");
    if (formats.parentNode) {
      formats.parentNode.insertBefore(selectedFileSize, formats.nextSibling);
    } else {
      form.appendChild(selectedFileSize);
    }
  }

  function updateSelectedFileSize() {
    if (!selectedFileSize) {
      return;
    }
    var logoFile = logoFileInput && logoFileInput.files && logoFileInput.files.length ? logoFileInput.files[0] : null;
    selectedFileSize.textContent = logoFile ? "Selected file: " + logoFile.name + " (" + formatAdminLogoFileSize(logoFile.size) + ")" : "No file selected";
    selectedFileSize.classList.toggle("is-over-limit", !!(logoFile && logoFile.size > ADMIN_LOGO_UPLOAD_MAX_BYTES));
  }

  updateSelectedFileSize();
  if (logoFileInput && !logoFileInput.dataset.logoSizeListenerBound) {
    logoFileInput.addEventListener("change", updateSelectedFileSize);
    logoFileInput.dataset.logoSizeListenerBound = "true";
  }

  var helperCopy = form.querySelector(".admin-copy--small");
  if (!helperCopy) {
    helperCopy = document.createElement("p");
    helperCopy.className = "admin-copy--small";
    if (formats.parentNode) {
      formats.parentNode.insertBefore(helperCopy, formats.nextSibling);
    } else {
      form.appendChild(helperCopy);
    }
  }
  helperCopy.textContent = "Use the file slug to control the stored logo name.";
}

function normalizeAdminLogoSymbol(logoValue) {
  var logo = (logoValue || "").toString().trim();
  if (!logo) {
    return "invader";
  }
  if (logo.indexOf("/") > -1) {
    logo = logo.substring(logo.lastIndexOf("/") + 1);
  }
  logo = logo.replace(/^#icon--badge-/, "");
  logo = logo.replace(/^icon--badge-/, "");
  logo = logo.replace(/^badge-/, "");
  logo = logo.replace(/\.svg$/i, "");
  return logo || "invader";
}

function setAdminLogoMedia(container, logoValue) {
  if (!container) {
    return;
  }

  while (container.firstChild) {
    container.removeChild(container.firstChild);
  }

  if (isAdminLogoImagePath(logoValue)) {
    var img = document.createElement("img");
    img.className = "icon icon--badge admin-logo-img";
    img.src = logoValue;
    img.alt = "";
    container.appendChild(img);
    return;
  }

  var svgNS = "http://www.w3.org/2000/svg";
  var svg = document.createElementNS(svgNS, "svg");
  var use = document.createElementNS(svgNS, "use");
  var symbol = "#icon--badge-" + normalizeAdminLogoSymbol(logoValue);
  svg.setAttribute("class", "icon icon--badge");
  use.setAttributeNS("http://www.w3.org/1999/xlink", "xlink:href", symbol);
  use.setAttribute("href", symbol);
  svg.appendChild(use);
  container.appendChild(svg);
}

function hasUnsavedAdminEditorChanges() {
  return document.querySelector('[data-unsaved-changes="true"]') !== null;
}

function showDiscardUnsavedAdminChangesModal(onConfirm) {
  if (typeof onConfirm !== "function") {
    return;
  }

  if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
    if (window.confirm("You have unsaved changes. Leave this page and discard them?")) {
      allowAdminPageExit();
      onConfirm();
    }
    return;
  }

  MAP_CTF.modal.loadPopup("action-cancel", function () {
    var modal = document.getElementById("mctf-modal");
    if (!modal) {
      return;
    }

    var confirmBtn = modal.querySelector(".js-confirm-discard-changes");
    if (!confirmBtn) {
      return;
    }

    confirmBtn.addEventListener("click", function (event) {
      event.preventDefault();
      if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
        MAP_CTF.modal.close();
      }
      allowAdminPageExit();
      onConfirm();
    });
  });
}

function confirmDiscardUnsavedAdminChanges() {
  if (!hasUnsavedAdminEditorChanges()) {
    return true;
  }

  return window.confirm("You have unsaved changes. Leave this page and discard them?");
}

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
  allowAdminPageExit();
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

        var proceedWithLogout = function () {
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
        };

        if (hasUnsavedAdminEditorChanges()) {
          showDiscardUnsavedAdminChangesModal(proceedWithLogout);
          return;
        }

        proceedWithLogout();
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
          if (typeof form._syncInitialState === "function") {
            form._syncInitialState();
          }
          showTransientAdminStatus(data.status || "ok", data.message || "Updated");
          if (form.dataset.reloadOnSuccess === "true" || form.dataset.adminReloadOnSuccess === "true") {
            window.location.reload();
            return;
          }
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
    form._syncInitialState = initializeAdminFormDirtyTracking(form);
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

function initAdminLogosSearch() {
  var searchInput = document.getElementById("admin-logos-search");
  if (!searchInput) {
    return;
  }

  var logoRows = Array.prototype.slice.call(document.querySelectorAll("#team-logos .admin-setting-row[data-logo-search]"));
  if (!logoRows.length) {
    return;
  }

  function filterLogos() {
    var query = (searchInput.value || "").trim().toLowerCase();
    logoRows.forEach(function (row) {
      if (!query) {
        row.style.display = "";
        return;
      }

      var haystack = (row.getAttribute("data-logo-search") || "").toLowerCase();
      row.style.display = haystack.indexOf(query) !== -1 ? "" : "none";
    });
  }

  searchInput.addEventListener("input", filterLogos);
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

function createAdminLogo(createURL, payload) {
  var headers = {
    Accept: "application/json",
    "X-Requested-With": "XMLHttpRequest",
  };
  var body = payload;

  if (!(payload instanceof FormData)) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(payload);
  }

  return fetch(createURL, {
    method: "POST",
    credentials: "same-origin",
    headers: headers,
    body: body,
  }).then(function (response) {
    return response
      .json()
      .catch(function () {
        return {};
      })
      .then(function (data) {
        if (!response.ok || data.success === false) {
          throw new Error(data.message || "Failed to create logo");
        }
        return data;
      });
  });
}

function updateAdminLogo(updateURL, payload) {
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
          throw new Error(data.message || "Failed to update logo");
        }
        return data;
      });
  });
}

function updateAdminTeam(updateURL, payload) {
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
          throw new Error(data.message || "Failed to update team");
        }
        return data;
      });
  });
}

function updateAdminUser(updateURL, payload) {
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
          throw new Error(data.message || "Failed to update user");
        }
        return data;
      });
  });
}

function setAdminUserPassword(updateURL, payload) {
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
          throw new Error(data.message || "Failed to update password");
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

function createAdminActivity(createURL, payload) {
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
          throw new Error(data.message || "Failed to create activity entry");
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

function initAdminGameActionsButtons() {
  var importBtn = document.querySelector('[data-action="import-full-game"]');
  var importInput = document.getElementById("admin-game-import-file");
  if (!importBtn || !importInput) {
    return;
  }

  importBtn.addEventListener("click", function (event) {
    event.preventDefault();
    importInput.click();
  });

  importInput.addEventListener("change", function () {
    var importURL = importBtn.getAttribute("data-import-url");
    if (!importURL) {
      showTransientAdminStatus("error", "Missing full-game import URL");
      importInput.value = "";
      return;
    }
    if (importBtn.dataset.submitting === "true") {
      importInput.value = "";
      return;
    }

    if (!importInput.files || !importInput.files.length) {
      return;
    }

    var importFile = importInput.files[0];
    var formData = new FormData();
    formData.append("file", importFile);

    importBtn.dataset.submitting = "true";
    importBtn.disabled = true;

    importAdminChallenges(importURL, formData)
      .then(function (data) {
        showTransientAdminStatus(data.status || "ok", data.message || "Full game imported");
        window.location.reload();
      })
      .catch(function (error) {
        showTransientAdminStatus("error", error.message || "Failed to import full game");
      })
      .finally(function () {
        delete importBtn.dataset.submitting;
        importBtn.disabled = false;
        importInput.value = "";
      });
  });
}

function initAdminSettingsActionsButtons() {
  var importBtn = document.querySelector('[data-action="import-all-settings"]');
  var resetDefaultsBtn = document.querySelector('[data-action="reset-settings-defaults"]');
  if (!importBtn && !resetDefaultsBtn) {
    return;
  }

  if (importBtn) {
    var fileInput = document.getElementById("admin-settings-import-file");
    if (!fileInput) {
      showTransientAdminStatus("error", "Settings import file input not found");
      return;
    }

    importBtn.addEventListener("click", function (event) {
      event.preventDefault();
      fileInput.click();
    });

    fileInput.addEventListener("change", function () {
      var importURL = importBtn.getAttribute("data-import-url");
      if (!importURL) {
        showTransientAdminStatus("error", "Missing settings import URL");
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
          showTransientAdminStatus(data.status || "ok", data.message || "Settings imported");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to import settings");
        })
        .finally(function () {
          delete importBtn.dataset.submitting;
          importBtn.disabled = false;
          fileInput.value = "";
        });
    });
  }

  if (resetDefaultsBtn) {
    resetDefaultsBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var resetDefaultsURL = resetDefaultsBtn.getAttribute("data-reset-defaults-url");
      if (!resetDefaultsURL) {
        showTransientAdminStatus("error", "Missing reset defaults URL");
        return;
      }
      if (resetDefaultsBtn.dataset.submitting === "true") {
        return;
      }

      function runResetDefaults() {
        resetDefaultsBtn.dataset.submitting = "true";
        resetDefaultsBtn.disabled = true;

        postAdminActionJSON(resetDefaultsURL)
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "Settings reset to defaults");
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to reset settings");
          })
          .finally(function () {
            delete resetDefaultsBtn.dataset.submitting;
            resetDefaultsBtn.disabled = false;
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
          });
      }

      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        if (window.confirm("Reset all settings to defaults?")) {
          runResetDefaults();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-reset-settings-defaults", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var confirmBtn = modal.querySelector(".js-confirm-reset-settings-defaults");
        if (!confirmBtn) {
          return;
        }
        confirmBtn.addEventListener("click", function (confirmEvent) {
          confirmEvent.preventDefault();
          runResetDefaults();
        });
      });
    });
  }
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

function initAdminTeamActionsButtons() {
  function bindImportAction(buttonSelector, inputID, missingURLMsg, successMsg, errorMsg) {
    var importBtn = document.querySelector(buttonSelector);
    var importInput = document.getElementById(inputID);
    if (!importBtn || !importInput) {
      return;
    }

    importBtn.addEventListener("click", function (event) {
      event.preventDefault();
      importInput.click();
    });

    importInput.addEventListener("change", function () {
      var importURL = importBtn.getAttribute("data-import-url");
      if (!importURL) {
        showTransientAdminStatus("error", missingURLMsg);
        return;
      }

      var file = importInput.files && importInput.files[0];
      if (!file) {
        return;
      }

      if (importBtn.dataset.submitting === "true") {
        return;
      }

      var formData = new FormData();
      formData.append("file", file);

      importBtn.dataset.submitting = "true";
      importBtn.disabled = true;

      importAdminChallenges(importURL, formData)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || successMsg);
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || errorMsg);
        })
        .finally(function () {
          delete importBtn.dataset.submitting;
          importBtn.disabled = false;
          importInput.value = "";
        });
    });
  }

  bindImportAction('[data-action="import-all-teams"]', "admin-teams-import-file", "Missing teams import URL", "Teams imported", "Failed to import teams");
  bindImportAction(
    '[data-action="import-all-team-logos"]',
    "admin-team-logos-import-file",
    "Missing logos import URL",
    "Logos imported",
    "Failed to import logos"
  );

  function bindSimpleAction(actionSelector, urlAttribute, successMessage, errorMessage) {
    var btn = document.querySelector(actionSelector);
    if (!btn) {
      return;
    }
    btn.addEventListener("click", function (event) {
      event.preventDefault();

      var actionURL = btn.getAttribute(urlAttribute);
      if (!actionURL) {
        showTransientAdminStatus("error", "Missing action URL");
        return;
      }
      if (btn.dataset.submitting === "true") {
        return;
      }

      btn.dataset.submitting = "true";
      btn.disabled = true;

      postAdminActionJSON(actionURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || successMessage);
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || errorMessage);
        })
        .finally(function () {
          delete btn.dataset.submitting;
          btn.disabled = false;
        });
    });
  }

  bindSimpleAction('[data-action="enable-all-teams"]', "data-enable-all-teams-url", "All teams enabled", "Failed to enable all teams");
  bindSimpleAction('[data-action="disable-all-teams"]', "data-disable-all-teams-url", "All teams disabled", "Failed to disable all teams");
  bindSimpleAction('[data-action="visible-all-teams"]', "data-visible-all-teams-url", "All teams set visible", "Failed to set all teams visible");
  bindSimpleAction('[data-action="invisible-all-teams"]', "data-invisible-all-teams-url", "All teams set invisible", "Failed to set all teams invisible");
  bindSimpleAction('[data-action="enable-all-logos"]', "data-enable-all-logos-url", "All logos enabled", "Failed to enable all logos");
  bindSimpleAction('[data-action="disable-all-logos"]', "data-disable-all-logos-url", "All logos disabled", "Failed to disable all logos");

  var deleteAllLogosBtn = document.querySelector('[data-action="delete-all-logos"]');
  if (deleteAllLogosBtn) {
    deleteAllLogosBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var deleteAllLogosURL = deleteAllLogosBtn.getAttribute("data-delete-all-logos-url");
      if (!deleteAllLogosURL) {
        showTransientAdminStatus("error", "Missing delete all logos URL");
        return;
      }
      if (deleteAllLogosBtn.dataset.submitting === "true") {
        return;
      }

      function runDeleteAllLogos() {
        deleteAllLogosBtn.dataset.submitting = "true";
        deleteAllLogosBtn.disabled = true;

        postAdminActionJSON(deleteAllLogosURL)
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "All logos deleted");
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to delete all logos");
          })
          .finally(function () {
            delete deleteAllLogosBtn.dataset.submitting;
            deleteAllLogosBtn.disabled = false;
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
          });
      }

      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        if (
          window.confirm(
            "Delete all custom logos? Platform badges stay in the catalog. This cannot be undone."
          )
        ) {
          runDeleteAllLogos();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-delete-all-logos", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var confirmBtn = modal.querySelector(".js-confirm-delete-all-logos");
        if (!confirmBtn) {
          return;
        }
        confirmBtn.addEventListener("click", function (confirmEvent) {
          confirmEvent.preventDefault();
          runDeleteAllLogos();
        });
      });
    });
  }

  var deleteAllTeamsBtn = document.querySelector('[data-action="delete-all-teams"]');
  if (deleteAllTeamsBtn) {
  deleteAllTeamsBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var deleteAllTeamsURL = deleteAllTeamsBtn.getAttribute("data-delete-all-teams-url");
    if (!deleteAllTeamsURL) {
      showTransientAdminStatus("error", "Missing delete all teams URL");
      return;
    }
    if (deleteAllTeamsBtn.dataset.submitting === "true") {
      return;
    }

    function runDeleteAllTeams() {
      deleteAllTeamsBtn.dataset.submitting = "true";
      deleteAllTeamsBtn.disabled = true;

      postAdminActionJSON(deleteAllTeamsURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All teams deleted");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to delete all teams");
        })
        .finally(function () {
          delete deleteAllTeamsBtn.dataset.submitting;
          deleteAllTeamsBtn.disabled = false;
          if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
            MAP_CTF.modal.close();
          }
        });
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      if (window.confirm("Delete all teams? This cannot be undone.")) {
        runDeleteAllTeams();
      }
      return;
    }

    MAP_CTF.modal.loadPopup("action-delete-all-teams", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var confirmBtn = modal.querySelector(".js-confirm-delete-all-teams");
      if (!confirmBtn) {
        return;
      }
      confirmBtn.addEventListener("click", function (confirmEvent) {
        confirmEvent.preventDefault();
        runDeleteAllTeams();
      });
    });
  });
  }
}

function initAdminUserActionsButtons() {
  var importBtn = document.querySelector('[data-action="import-all-users"]');
  var importInput = document.getElementById("admin-users-import-file");
  if (importBtn && importInput) {
    importBtn.addEventListener("click", function (event) {
      event.preventDefault();
      importInput.click();
    });

    importInput.addEventListener("change", function () {
      var importURL = importBtn.getAttribute("data-import-url");
      if (!importURL) {
        showTransientAdminStatus("error", "Missing users import URL");
        return;
      }

      var file = importInput.files && importInput.files[0];
      if (!file) {
        return;
      }

      if (importBtn.dataset.submitting === "true") {
        return;
      }

      var formData = new FormData();
      formData.append("file", file);

      importBtn.dataset.submitting = "true";
      importBtn.disabled = true;

      importAdminChallenges(importURL, formData)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "Users imported");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to import users");
        })
        .finally(function () {
          delete importBtn.dataset.submitting;
          importBtn.disabled = false;
          importInput.value = "";
        });
    });
  }

  var disableAllUsersBtn = document.querySelector('[data-action="disable-all-users"]');
  var enableAllUsersBtn = document.querySelector('[data-action="enable-all-users"]');
  if (enableAllUsersBtn) {
    enableAllUsersBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var enableAllUsersURL = enableAllUsersBtn.getAttribute("data-enable-all-users-url");
      if (!enableAllUsersURL) {
        showTransientAdminStatus("error", "Missing enable all users URL");
        return;
      }
      if (enableAllUsersBtn.dataset.submitting === "true") {
        return;
      }

      enableAllUsersBtn.dataset.submitting = "true";
      enableAllUsersBtn.disabled = true;

      postAdminActionJSON(enableAllUsersURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All users enabled");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to enable all users");
        })
        .finally(function () {
          delete enableAllUsersBtn.dataset.submitting;
          enableAllUsersBtn.disabled = false;
        });
    });
  }

  if (disableAllUsersBtn) {
    disableAllUsersBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var disableAllUsersURL = disableAllUsersBtn.getAttribute("data-disable-all-users-url");
      if (!disableAllUsersURL) {
        showTransientAdminStatus("error", "Missing disable all users URL");
        return;
      }
      if (disableAllUsersBtn.dataset.submitting === "true") {
        return;
      }

      function runDisableAllUsers() {
        disableAllUsersBtn.dataset.submitting = "true";
        disableAllUsersBtn.disabled = true;

        postAdminActionJSON(disableAllUsersURL)
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "All users disabled");
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to disable all users");
          })
          .finally(function () {
            delete disableAllUsersBtn.dataset.submitting;
            disableAllUsersBtn.disabled = false;
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
          });
      }

      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        if (window.confirm("Disable all users?")) {
          runDisableAllUsers();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-disable-all-users", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var confirmBtn = modal.querySelector(".js-confirm-disable-all-users");
        if (!confirmBtn) {
          return;
        }
        confirmBtn.addEventListener("click", function (confirmEvent) {
          confirmEvent.preventDefault();
          runDisableAllUsers();
        });
      });
    });
  }

  var deleteAllUsersBtn = document.querySelector('[data-action="delete-all-users"]');
  if (!deleteAllUsersBtn) {
    return;
  }

  deleteAllUsersBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var deleteAllUsersURL = deleteAllUsersBtn.getAttribute("data-delete-all-users-url");
    if (!deleteAllUsersURL) {
      showTransientAdminStatus("error", "Missing delete all users URL");
      return;
    }
    if (deleteAllUsersBtn.dataset.submitting === "true") {
      return;
    }

    function runDeleteAllUsers() {
      deleteAllUsersBtn.dataset.submitting = "true";
      deleteAllUsersBtn.disabled = true;

      postAdminActionJSON(deleteAllUsersURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All users deleted");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to delete all users");
        })
        .finally(function () {
          delete deleteAllUsersBtn.dataset.submitting;
          deleteAllUsersBtn.disabled = false;
          if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
            MAP_CTF.modal.close();
          }
        });
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      if (window.confirm("Delete all users? This cannot be undone.")) {
        runDeleteAllUsers();
      }
      return;
    }

    MAP_CTF.modal.loadPopup("action-delete-all-users", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var confirmBtn = modal.querySelector(".js-confirm-delete-all-users");
      if (!confirmBtn) {
        return;
      }
      confirmBtn.addEventListener("click", function (confirmEvent) {
        confirmEvent.preventDefault();
        runDeleteAllUsers();
      });
    });
  });
}

function initAdminCountryActionsButtons() {
  var deleteAllCountriesBtn = document.querySelector('[data-action="delete-all-countries"]');
  if (!deleteAllCountriesBtn) {
    return;
  }

  deleteAllCountriesBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var deleteAllCountriesURL = deleteAllCountriesBtn.getAttribute("data-delete-all-countries-url");
    if (!deleteAllCountriesURL) {
      showTransientAdminStatus("error", "Missing delete all countries URL");
      return;
    }
    if (deleteAllCountriesBtn.dataset.submitting === "true") {
      return;
    }

    function runDeleteAllCountries() {
      deleteAllCountriesBtn.dataset.submitting = "true";
      deleteAllCountriesBtn.disabled = true;

      postAdminActionJSON(deleteAllCountriesURL)
        .then(function (data) {
          showTransientAdminStatus(data.status || "ok", data.message || "All countries deleted");
          window.location.reload();
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to delete all countries");
        })
        .finally(function () {
          delete deleteAllCountriesBtn.dataset.submitting;
          deleteAllCountriesBtn.disabled = false;
          if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
            MAP_CTF.modal.close();
          }
        });
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      if (window.confirm("Delete all countries? This cannot be undone and will clear country assignments from challenges.")) {
        runDeleteAllCountries();
      }
      return;
    }

    MAP_CTF.modal.loadPopup("action-delete-all-countries", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var confirmBtn = modal.querySelector(".js-confirm-delete-all-countries");
      if (!confirmBtn) {
        return;
      }
      confirmBtn.addEventListener("click", function (confirmEvent) {
        confirmEvent.preventDefault();
        runDeleteAllCountries();
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
        var description = (form.querySelector('[name="description"]').value || "").trim();
        var url = (form.querySelector('[name="url"]').value || "").trim();
        var categoryID = (form.querySelector('select[name="category_id"]').value || "").trim();
        var country = (form.querySelector('[name="country"]').value || "").trim();
        var flag = (form.querySelector('input[name="flag"]').value || "").trim();
        var hint = (form.querySelector('[name="hint"]').value || "").trim();
        var points = String(form.querySelector('input[name="points"]').value || "0").trim();
        var bonus = String(form.querySelector('input[name="bonus"]').value || "0").trim();
        var bonusDecay = String(form.querySelector('input[name="bonus_decay"]').value || "0").trim();
        var hintPenalty = String(form.querySelector('input[name="hint_penalty"]').value || "0").trim();
        var helpPenalty = String(form.querySelector('input[name="help_penalty"]').value || "0").trim();

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
          url: url,
          category_id: categoryID,
          country: country,
          active: "false",
          points: points,
          bonus: bonus,
          bonus_decay: bonusDecay,
          hint_penalty: hintPenalty,
          help_penalty: helpPenalty,
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

function initAdminAddActivityModal() {
  var addActivityBtn = document.querySelector('[data-action="add-custom-activity"]');
  if (!addActivityBtn) {
    return;
  }

  addActivityBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var createURL = addActivityBtn.getAttribute("data-create-url");
    if (!createURL) {
      showTransientAdminStatus("error", "Missing activity creation URL");
      return;
    }

    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      showTransientAdminStatus("error", "Modal system unavailable");
      return;
    }

    MAP_CTF.modal.loadPopup("add-activity", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var form = modal.querySelector("#admin-add-activity-form");
      if (!form) {
        return;
      }

      var subjectInput = form.querySelector('input[name="subject"]');
      if (subjectInput) {
        subjectInput.focus();
      }

      form.addEventListener("submit", function (submitEvent) {
        submitEvent.preventDefault();

        if (form.dataset.submitting === "true") {
          return;
        }

        var subject = (form.querySelector('input[name="subject"]').value || "").trim();
        var action = (form.querySelector('select[name="action"]').value || "").trim();
        var visible = (form.querySelector('select[name="visible"]').value || "true").trim() !== "false";
        var message = (form.querySelector('input[name="message"]').value || "").trim();
        if (!subject && !message) {
          showTransientAdminStatus("error", "Subject or message is required");
          return;
        }

        form.dataset.submitting = "true";

        createAdminActivity(createURL, {
          subject: subject,
          action: action,
          visible: visible,
          message: message,
        })
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "Activity entry created");
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to create activity entry");
          })
          .finally(function () {
            delete form.dataset.submitting;
          });
      });
    });
  });
}

function initAdminActivityDeleteButtons() {
  var deleteButtons = document.querySelectorAll('[data-action="delete-activity"]');
  if (!deleteButtons.length) {
    return;
  }

  deleteButtons.forEach(function (deleteBtn) {
    deleteBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var deleteURL = deleteBtn.getAttribute("data-delete-url");
      if (!deleteURL) {
        showTransientAdminStatus("error", "Missing activity delete URL");
        return;
      }

      var item = deleteBtn.closest(".admin-activity-item");
      if (!item) {
        showTransientAdminStatus("error", "Unable to locate activity entry");
        return;
      }

      if (deleteBtn.dataset.submitting === "true") {
        return;
      }

      var activityLine = item.querySelector(".admin-activity-line");
      var activitySummary = activityLine ? String(activityLine.textContent || "").replace(/\s+/g, " ").trim() : "";

      function runDelete() {
        deleteBtn.dataset.submitting = "true";
        deleteBtn.disabled = true;

        fetch(deleteURL, {
          method: "POST",
          headers: {
            "Content-Type": "application/json; charset=utf-8",
          },
        })
          .then(function (response) {
            return response
              .json()
              .catch(function () {
                return {};
              })
              .then(function (data) {
                if (!response.ok || data.success === false) {
                  throw new Error(data.message || "Failed to delete activity entry");
                }
                return data;
              });
          })
          .then(function (data) {
            item.remove();
            showTransientAdminStatus(data.status || "ok", data.message || "Activity entry deleted");
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to delete activity entry");
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
        if (window.confirm("Delete this activity entry?")) {
          runDelete();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-delete-activity", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var summaryPlaceholder = modal.querySelector(".js-delete-activity-summary");
        if (summaryPlaceholder) {
          summaryPlaceholder.textContent = activitySummary;
        }

        var confirmBtn = modal.querySelector(".js-confirm-delete-activity");
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
      var logoPreviewContainer = form.querySelector(".admin-add-team-logo-preview-wrap");

      function updateAddTeamLogoPreview() {
        if (!logoSelect || !logoPreviewContainer) {
          return;
        }

        var logo = (logoSelect.value || "").trim();
        if (!logo || logo === "random") {
          var firstLogoOption = form.querySelector('select[name="logo"] option[value]:not([value="random"])');
          logo = firstLogoOption ? (firstLogoOption.value || "").trim() : "invader";
        }

        if (!logo) {
          logo = "invader";
        }

        setAdminLogoMedia(logoPreviewContainer, logo);
      }
      if (logoSelect) {
        logoSelect.addEventListener("change", updateAddTeamLogoPreview);
      }
      updateAddTeamLogoPreview();

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

function initAdminUserSettingsEditors() {
  var userCards = Array.prototype.slice.call(document.querySelectorAll("#users section.admin-box[data-user-update-url]"));
  if (!userCards.length) {
    return;
  }

  userCards.forEach(function (card) {
    var saveBtn = card.querySelector('[data-action="save"]');
    var setPasswordBtn = card.querySelector('[data-action="set-password"]');
    var updateURL = card.getAttribute("data-user-update-url");
    var nameInput = card.querySelector('input[data-user-field="name"]');
    var emailInput = card.querySelector('input[data-user-field="email"]');
    var teamSelect = card.querySelector('select[data-user-field="team_id"]');
    var syncInitialState = initializeEditorDirtyTracking(card, "[data-user-field]");

    if (setPasswordBtn) {
      setPasswordBtn.addEventListener("click", function (event) {
        event.preventDefault();

        var passwordURL = setPasswordBtn.getAttribute("data-user-password-url");
        if (!passwordURL) {
          showTransientAdminStatus("error", "Missing user password URL");
          return;
        }
        if (setPasswordBtn.dataset.submitting === "true") {
          return;
        }
        if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
          showTransientAdminStatus("error", "Password modal is unavailable");
          return;
        }

        MAP_CTF.modal.loadPopup("set-user-password", function () {
          var modal = document.getElementById("mctf-modal");
          if (!modal) {
            return;
          }

          var form = modal.querySelector("#admin-set-user-password-form");
          var passwordInput = form ? form.querySelector('input[name="new_password"]') : null;
          var usernameEl = modal.querySelector(".js-set-user-password-username");
          if (!form || !passwordInput) {
            return;
          }

          var username = setPasswordBtn.getAttribute("data-user-username") || (card.querySelector("h3") ? card.querySelector("h3").textContent : "this user");
          if (usernameEl) {
            usernameEl.textContent = username;
          }
          passwordInput.focus();

          form.addEventListener("submit", function (submitEvent) {
            submitEvent.preventDefault();

            if (form.dataset.submitting === "true") {
              return;
            }

            var newPassword = passwordInput.value || "";
            if (!newPassword.trim()) {
              showTransientAdminStatus("error", "New password is required");
              return;
            }
            if (typeof passwordInput.checkValidity === "function" && !passwordInput.checkValidity()) {
              if (typeof passwordInput.reportValidity === "function") {
                passwordInput.reportValidity();
              } else {
                showTransientAdminStatus("error", "New password is required");
              }
              return;
            }

            form.dataset.submitting = "true";
            setPasswordBtn.dataset.submitting = "true";
            setPasswordBtn.disabled = true;

            setAdminUserPassword(passwordURL, {
              new_password: newPassword,
            })
              .then(function (data) {
                showTransientAdminStatus(data.status || "ok", data.message || "Password updated");
                if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
                  MAP_CTF.modal.close();
                }
              })
              .catch(function (error) {
                showTransientAdminStatus("error", error.message || "Failed to update password");
              })
              .finally(function () {
                delete form.dataset.submitting;
                delete setPasswordBtn.dataset.submitting;
                setPasswordBtn.disabled = false;
                passwordInput.value = "";
              });
          });
        });
      });
    }

    if (!saveBtn || !teamSelect) {
      return;
    }

    saveBtn.addEventListener("click", function (event) {
      event.preventDefault();

      if (!updateURL) {
        showTransientAdminStatus("error", "Missing user update URL");
        return;
      }
      if (saveBtn.dataset.submitting === "true") {
        return;
      }

      var nameValue = nameInput && typeof nameInput.value === "string" ? nameInput.value.trim() : "";
      var emailValue = emailInput && typeof emailInput.value === "string" ? emailInput.value.trim() : "";
      if (emailValue && emailInput && typeof emailInput.checkValidity === "function" && !emailInput.checkValidity()) {
        showTransientAdminStatus("error", "Email is invalid");
        return;
      }

      var teamValue = typeof teamSelect.value === "string" ? teamSelect.value.trim() : "";
      if (teamValue === "") {
        teamValue = "0";
      }
      var adminInput = card.querySelector('input[data-user-field="admin"]:checked');
      var serviceInput = card.querySelector('input[data-user-field="service"]:checked');
      var activeInput = card.querySelector('input[data-user-field="active"]:checked');
      var adminValue = adminInput ? String(adminInput.value).trim() : "";
      var serviceValue = serviceInput ? String(serviceInput.value).trim() : "";
      var activeValue = activeInput ? String(activeInput.value).trim() : "";

      if (!adminValue || !serviceValue || !activeValue) {
        showTransientAdminStatus("error", "Admin, service and active values are required");
        return;
      }

      saveBtn.dataset.submitting = "true";
      saveBtn.disabled = true;

      updateAdminUser(updateURL, {
        name: nameValue,
        email: emailValue,
        team_id: teamValue,
        admin: adminValue,
        service: serviceValue,
        active: activeValue,
      })
        .then(function (data) {
          if (nameInput) {
            nameInput.value = nameValue;
          }
          if (emailInput) {
            emailInput.value = emailValue;
          }
          var sessionSearch = Array.prototype.map
            .call(card.querySelectorAll("input:not([data-user-field]), textarea"), function (field) {
              return field.value || field.textContent || "";
            })
            .join(" ");
          card.setAttribute("data-user-search", [card.querySelector("h3") ? card.querySelector("h3").textContent : "", nameValue, emailValue, teamValue, sessionSearch].join(" "));
          if (typeof syncInitialState === "function") {
            syncInitialState();
          }
          showTransientAdminStatus(data.status || "ok", data.message || "User updated");
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to update user");
        })
        .finally(function () {
          delete saveBtn.dataset.submitting;
          saveBtn.disabled = false;
        });
    });
  });
}

function initAdminAddLogoButton() {
  var addLogoBtn = document.querySelector('[data-action="add-new-logo"]');
  if (!addLogoBtn) {
    return;
  }

  addLogoBtn.addEventListener("click", function (event) {
    event.preventDefault();

    var createURL = addLogoBtn.getAttribute("data-create-url");
    if (!createURL) {
      showTransientAdminStatus("error", "Missing logo creation URL");
      return;
    }
    if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
      showTransientAdminStatus("error", "Modal system unavailable");
      return;
    }

    MAP_CTF.modal.loadPopup("add-logo", function () {
      var modal = document.getElementById("mctf-modal");
      if (!modal) {
        return;
      }

      var form = modal.querySelector("#admin-add-logo-form");
      if (!form) {
        return;
      }

      var nameInput = form.querySelector('input[name="name"]');
      if (nameInput) {
        nameInput.focus();
      }
      var logoFileInput = form.querySelector('input[name="logo_file"]');
      ensureAdminLogoUploadFormats(form);

      form.addEventListener("submit", function (submitEvent) {
        submitEvent.preventDefault();

        if (form.dataset.submitting === "true") {
          return;
        }

        var logoName = (form.querySelector('input[name="name"]').value || "").trim();
        var logoSymbol = (form.querySelector('input[name="logo"]').value || "").trim();
        var logoFile = logoFileInput && logoFileInput.files && logoFileInput.files.length ? logoFileInput.files[0] : null;

        if (!logoName) {
          showTransientAdminStatus("error", "Logo name is required");
          return;
        }
        if (!logoFile) {
          showTransientAdminStatus("error", "Upload a logo file before creating a custom logo");
          return;
        }
        if (logoFile.size > ADMIN_LOGO_UPLOAD_MAX_BYTES) {
          showTransientAdminStatus("error", "Maximum logo upload size is " + ADMIN_LOGO_UPLOAD_MAX_LABEL);
          return;
        }

        var formData = new FormData();
        formData.append("name", logoName);
        formData.append("logo", logoSymbol);
        formData.append("logo_file", logoFile);
        formData.append("logo_slug", logoSymbol);

        form.dataset.submitting = "true";

        createAdminLogo(createURL, formData)
          .then(function (data) {
            showTransientAdminStatus(data.status || "ok", data.message || "Logo created");
            if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
              MAP_CTF.modal.close();
            }
            window.location.reload();
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to create logo");
          })
          .finally(function () {
            delete form.dataset.submitting;
          });
      });
    });
  });
}

function initAdminEditLogoGrid() {
  var logoButtons = Array.prototype.slice.call(document.querySelectorAll("#team-logos .js-edit-logo[data-logo-update-url]"));
  if (!logoButtons.length) {
    return;
  }

  logoButtons.forEach(function (logoBtn) {
    logoBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var updateURL = logoBtn.getAttribute("data-logo-update-url");
      if (!updateURL) {
        showTransientAdminStatus("error", "Missing logo update URL");
        return;
      }
      if (typeof MAP_CTF === "undefined" || !MAP_CTF.modal || typeof MAP_CTF.modal.loadPopup !== "function") {
        showTransientAdminStatus("error", "Modal system unavailable");
        return;
      }

      MAP_CTF.modal.loadPopup("edit-logo", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }
        var form = modal.querySelector("#admin-edit-logo-form");
        if (!form) {
          return;
        }

        var nameInput = form.querySelector('input[name="name"]');
        var fileInput = form.querySelector('input[name="file"]');
        var previewContainer = form.querySelector(".edit-logo-preview");
        var enabledOn = form.querySelector('input[name="enabled"][value="true"]');
        var enabledOff = form.querySelector('input[name="enabled"][value="false"]');
        var protectedOn = form.querySelector('input[name="protected"][value="true"]');
        var protectedOff = form.querySelector('input[name="protected"][value="false"]');

        var logoName = (logoBtn.getAttribute("data-logo-name") || "").trim();
        var logoSymbol = (logoBtn.getAttribute("data-logo-symbol") || "").trim();
        var logoFile = (logoBtn.getAttribute("data-logo-file") || "").trim();
        var logoEnabled = (logoBtn.getAttribute("data-logo-enabled") || "false").toLowerCase() === "true";
        var logoProtected = (logoBtn.getAttribute("data-logo-protected") || "false").toLowerCase() === "true";
        var isPlatform = (logoBtn.getAttribute("data-logo-custom") || "true").toLowerCase() === "false";

        if (nameInput) {
          nameInput.value = logoName;
          nameInput.readOnly = isPlatform;
          if (isPlatform) {
            nameInput.setAttribute("title", "Platform logo name cannot be changed");
          } else {
            nameInput.removeAttribute("title");
          }
          nameInput.focus();
        }
        if (fileInput) {
          fileInput.value = logoFile;
        }
        setAdminLogoMedia(previewContainer, logoSymbol || "invader");
        if (enabledOn && enabledOff) {
          enabledOn.checked = logoEnabled;
          enabledOff.checked = !logoEnabled;
        }
        if (protectedOn && protectedOff) {
          protectedOn.checked = logoProtected;
          protectedOff.checked = !logoProtected;
        }

        form.addEventListener("submit", function (submitEvent) {
          submitEvent.preventDefault();

          if (form.dataset.submitting === "true") {
            return;
          }

          var nameValue = isPlatform
            ? (logoBtn.getAttribute("data-logo-name") || "").trim()
            : (form.querySelector('input[name="name"]').value || "").trim();
          var enabledInput = form.querySelector('input[name="enabled"]:checked');
          var protectedInput = form.querySelector('input[name="protected"]:checked');
          var enabledValue = enabledInput ? String(enabledInput.value).trim() : "";
          var protectedValue = protectedInput ? String(protectedInput.value).trim() : "";

          if (!nameValue) {
            showTransientAdminStatus("error", "Logo name is required");
            return;
          }
          if (!enabledValue || !protectedValue) {
            showTransientAdminStatus("error", "Enabled and protected values are required");
            return;
          }

          form.dataset.submitting = "true";
          updateAdminLogo(updateURL, {
            name: nameValue,
            enabled: enabledValue,
            protected: protectedValue,
          })
            .then(function (data) {
              showTransientAdminStatus(data.status || "ok", data.message || "Logo updated");
              if (MAP_CTF.modal && typeof MAP_CTF.modal.close === "function") {
                MAP_CTF.modal.close();
              }
              window.location.reload();
            })
            .catch(function (error) {
              showTransientAdminStatus("error", error.message || "Failed to update logo");
            })
            .finally(function () {
              delete form.dataset.submitting;
            });
        });
      });
    });
  });
}

function initAdminTeamSettingsEditors() {
  var teamCards = Array.prototype.slice.call(document.querySelectorAll("#teams section.admin-box[data-team-update-url]"));
  if (!teamCards.length) {
    return;
  }

  teamCards.forEach(function (card) {
    var editBtn = card.querySelector('[data-action="edit"]');
    var saveBtn = card.querySelector('[data-action="save"]');
    var updateURL = card.getAttribute("data-team-update-url");
    var editableFields = Array.prototype.slice.call(card.querySelectorAll("[data-team-field]"));
    var nameInput = card.querySelector('input[data-team-field="name"]');
    var logoSelect = card.querySelector('select[data-team-field="logo"]');
    var iconWrap = card.querySelector(".admin-team-icon");
    var nameDisplay = card.querySelector(".js-team-name-display");
    var syncInitialState = initializeEditorDirtyTracking(card, "[data-team-field]");

    function setEditing(editing) {
      editableFields.forEach(function (field) {
        field.disabled = !editing;
      });
      card.dataset.editing = editing ? "true" : "false";
    }

    function updateLogoPreview() {
      if (!logoSelect || !iconWrap) {
        return;
      }
      var logo = (logoSelect.value || "").trim();
      if (!logo || logo === "random") {
        return;
      }
      setAdminLogoMedia(iconWrap, logo);
    }

    setEditing(true);
    if (logoSelect) {
      logoSelect.addEventListener("change", updateLogoPreview);
    }

    if (editBtn) {
      editBtn.addEventListener("click", function (event) {
        event.preventDefault();
        setEditing(true);
      });
    }

    if (!saveBtn) {
      return;
    }

    saveBtn.addEventListener("click", function (event) {
      event.preventDefault();

      if (!updateURL) {
        showTransientAdminStatus("error", "Missing team update URL");
        return;
      }
      if (saveBtn.dataset.submitting === "true") {
        return;
      }

      var nameValue = nameInput && typeof nameInput.value === "string" ? nameInput.value.trim() : "";
      var logoValue = logoSelect && typeof logoSelect.value === "string" ? logoSelect.value.trim() : "";
      var activeInput = card.querySelector('input[data-team-field="active"]:checked');
      var visibleInput = card.querySelector('input[data-team-field="visible"]:checked');
      var protectedInput = card.querySelector('input[data-team-field="protected"]:checked');
      var activeValue = activeInput ? String(activeInput.value).trim() : "";
      var visibleValue = visibleInput ? String(visibleInput.value).trim() : "";
      var protectedValue = protectedInput ? String(protectedInput.value).trim() : "";

      if (!nameValue || !logoValue || !activeValue || !visibleValue || !protectedValue) {
        showTransientAdminStatus("error", "Team name, logo, active, visible and protected values are required");
        return;
      }

      saveBtn.dataset.submitting = "true";
      saveBtn.disabled = true;

      updateAdminTeam(updateURL, {
        name: nameValue,
        logo: logoValue,
        active: activeValue,
        visible: visibleValue,
        protected: protectedValue,
      })
        .then(function (data) {
          updateLogoPreview();
          if (nameDisplay) {
            nameDisplay.textContent = nameValue;
          }
          card.setAttribute("data-team-search", nameValue);
          if (typeof syncInitialState === "function") {
            syncInitialState();
          }
          setEditing(true);
          showTransientAdminStatus(data.status || "ok", data.message || "Team updated");
        })
        .catch(function (error) {
          showTransientAdminStatus("error", error.message || "Failed to update team");
        })
        .finally(function () {
          delete saveBtn.dataset.submitting;
          saveBtn.disabled = false;
        });
    });
  });
}

function initAdminTeamDeleteButtons() {
  var deleteButtons = document.querySelectorAll('[data-action="delete-team"]');
  if (!deleteButtons.length) {
    return;
  }

  deleteButtons.forEach(function (deleteBtn) {
    deleteBtn.addEventListener("click", function (event) {
      event.preventDefault();

      var deleteURL = deleteBtn.getAttribute("data-delete-url");
      if (!deleteURL) {
        showTransientAdminStatus("error", "Missing team delete URL");
        return;
      }
      if (deleteBtn.dataset.submitting === "true") {
        return;
      }

      var teamCard = deleteBtn.closest("#teams section.admin-box[data-team-update-url]");
      var teamName = (deleteBtn.getAttribute("data-team-name") || "").trim();

      function runDelete() {
        deleteBtn.dataset.submitting = "true";
        deleteBtn.disabled = true;

        postAdminActionJSON(deleteURL)
          .then(function (data) {
            if (teamCard) {
              teamCard.remove();
            }
            showTransientAdminStatus(data.status || "ok", data.message || "Team deleted");
          })
          .catch(function (error) {
            showTransientAdminStatus("error", error.message || "Failed to delete team");
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
        if (window.confirm("Delete team" + (teamName ? " '" + teamName + "'" : "") + "?")) {
          runDelete();
        }
        return;
      }

      MAP_CTF.modal.loadPopup("action-delete-team", function () {
        var modal = document.getElementById("mctf-modal");
        if (!modal) {
          return;
        }

        var titlePlaceholder = modal.querySelector(".js-delete-team-title");
        if (titlePlaceholder) {
          titlePlaceholder.textContent = teamName ? " " + teamName : "";
        }

        var confirmBtn = modal.querySelector(".js-confirm-delete-team");
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
    var challengeRow = saveBtn.closest(".admin-challenge-row");
    if (challengeRow && typeof challengeRow._syncInitialState !== "function") {
      challengeRow._syncInitialState = initializeEditorDirtyTracking(challengeRow, '.admin-challenge-form input, .admin-challenge-form select, .admin-challenge-form textarea, .admin-activity-status-toggle input[type="radio"]');
    }

    saveBtn.addEventListener("click", function (event) {
      event.preventDefault();
      submitAdminChallengeRow(saveBtn);
    });
  });
}

function submitAdminChallengeRow(triggerEl) {
  var challengeRow = triggerEl && triggerEl.closest ? triggerEl.closest(".admin-challenge-row") : null;
  if (!challengeRow) {
    showTransientAdminStatus("error", "Unable to locate challenge row");
    return;
  }

  var saveBtn = challengeRow.querySelector('[data-action="save-challenge"]');
  var updateURL = saveBtn ? saveBtn.getAttribute("data-update-url") : "";
  if (!updateURL) {
    showTransientAdminStatus("error", "Missing challenge update URL");
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
  var url = (form.querySelector('input[name="url"]').value || "").trim();
  var categoryID = (form.querySelector('select[name="category_id"]').value || "").trim();
  var country = (form.querySelector('select[name="country"]').value || "").trim();
  var flag = (form.querySelector('input[name="flag"]').value || "").trim();
  var hint = (form.querySelector('textarea[name="hint"]').value || "").trim();
  var points = String(form.querySelector('input[name="points"]').value || "0").trim();
  var bonus = String(form.querySelector('input[name="bonus"]').value || "0").trim();
  var bonusDecay = String(form.querySelector('input[name="bonus_decay"]').value || "0").trim();
  var hintPenalty = String(form.querySelector('input[name="hint_penalty"]').value || "0").trim();
  var helpPenalty = String(form.querySelector('input[name="help_penalty"]').value || "0").trim();
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
  if (saveBtn) {
    saveBtn.disabled = true;
  }

  createAdminChallengeUpdate(updateURL, {
    title: title,
    description: description,
    url: url,
    category_id: categoryID,
    country: country,
    active: active,
    points: points,
    bonus: bonus,
    bonus_decay: bonusDecay,
    hint_penalty: hintPenalty,
    help_penalty: helpPenalty,
    flag: flag,
    hint: hint,
  })
    .then(function (data) {
      if (challengeRow && typeof challengeRow._syncInitialState === "function") {
        challengeRow._syncInitialState();
      }
      showTransientAdminStatus(data.status || "ok", data.message || "Challenge updated");
    })
    .catch(function (error) {
      showTransientAdminStatus("error", error.message || "Failed to update challenge");
    })
    .finally(function () {
      delete form.dataset.submitting;
      if (saveBtn) {
        saveBtn.disabled = false;
      }
    });
}

function initAdminChallengeStatusAutoSave() {
  var radios = document.querySelectorAll(".admin-activity-status-toggle input[type='radio']");
  if (!radios.length) {
    return;
  }

  radios.forEach(function (radio) {
    radio.addEventListener("change", function () {
      if (!radio.checked) {
        return;
      }
      submitAdminChallengeRow(radio);
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
  window.addEventListener("beforeunload", function (event) {
    if (adminBypassBeforeUnloadWarning || !hasUnsavedAdminEditorChanges()) {
      return;
    }

    event.preventDefault();
    event.returnValue = "";
  });

  document.addEventListener("click", function (event) {
    var link = event.target.closest("a[href]");
    if (!link) {
      return;
    }

    var href = (link.getAttribute("href") || "").trim();
    if (!href || href === "#" || href.indexOf("javascript:") === 0 || link.classList.contains("js-prompt-logout")) {
      return;
    }

    if (!hasUnsavedAdminEditorChanges()) {
      return;
    }

    event.preventDefault();
    showDiscardUnsavedAdminChangesModal(function () {
      window.location.assign(href);
    });
  });

  initAdminStatusFromServerState();
  initAdminLogoutModal();
  initAdminAjaxForms();
  initAdminGameActionsButtons();
  initAdminSettingsActionsButtons();
  initAdminUsersSearch();
  initAdminUserActionsButtons();
  initAdminCountryActionsButtons();
  initAdminTeamsSearch();
  initAdminLogosSearch();
  initAdminTeamActionsButtons();
  initAdminImportChallengesButton();
  initAdminChallengeActionsButtons();
  initAdminChallengeSaveButtons();
  initAdminChallengeStatusAutoSave();
  initAdminChallengeDeleteButtons();
  initAdminAddChallengeModal();
  initAdminAddActivityModal();
  initAdminActivityDeleteButtons();
  initAdminAddUserModal();
  initAdminUserSettingsEditors();
  initAdminAddTeamModal();
  initAdminTeamSettingsEditors();
  initAdminTeamDeleteButtons();
  initAdminAddLogoButton();
  initAdminEditLogoGrid();
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
