function getAjaxMessageBox() {
  return $("#ajax-message-box");
}

function clearAjaxMessage() {
  var $messageBox = getAjaxMessageBox();
  if ($messageBox.length === 0) {
    return;
  }
  $messageBox.removeClass("ajax-message--error ajax-message--success").text("").hide();
}

function showAjaxMessage(message, messageType) {
  var $messageBox = getAjaxMessageBox();
  if ($messageBox.length === 0) {
    return;
  }
  var cssClass = messageType === "success" ? "ajax-message--success" : "ajax-message--error";
  $messageBox.removeClass("ajax-message--error ajax-message--success").addClass(cssClass).text(message).show();
}

function sendGetRequest(req_url, _modal, _callback) {
  clearAjaxMessage();
  $.ajax({
    url: req_url,
    dataType: "json",
    type: "GET",
    contentType: "application/json",
    success: function (data, textStatus, jQxhr) {
      console.log("OK");
      console.log(data);
      if (_modal) {
        $("#successModalMessage").text(data.message);
        $("#successModal").modal();
      }
      if (_callback) {
        _callback(data);
      }
    },
    error: function (jqXhr, textStatus, errorThrown) {
      var _serverMessage = "Request failed. Please try again.";
      if (jqXhr.responseJSON && (jqXhr.responseJSON.error || jqXhr.responseJSON.message)) {
        _serverMessage = jqXhr.responseJSON.error || jqXhr.responseJSON.message;
      } else if (jqXhr.responseText) {
        try {
          var _serverJSON = $.parseJSON(jqXhr.responseText);
          _serverMessage = _serverJSON.error || _serverJSON.message || _serverMessage;
        } catch (_parseErr) {
          _serverMessage = jqXhr.responseText;
        }
      } else if (errorThrown) {
        _serverMessage = errorThrown;
      }
      showAjaxMessage(_serverMessage, "error");
      console.log("Client: " + textStatus);
    },
  });
}

function sendPostRequest(req_data, req_url, _redir, _modal, _callback) {
  clearAjaxMessage();
  $.ajax({
    url: req_url,
    dataType: "json",
    type: "POST",
    contentType: "application/json",
    data: JSON.stringify(req_data),
    processData: false,
    success: function (data, textStatus, jQxhr) {
      console.log("OK");
      console.log(data);
      if (_modal) {
        $("#successModalMessage").text(data.message);
        $("#successModal").modal();
      }
      var _redirectUrl = _redir;
      if (_redirectUrl === "" && data.redirect) {
        _redirectUrl = data.redirect;
      }
      if (_redirectUrl !== "") {
        showAjaxMessage(data.message || "Success. Redirecting...", "success");
        setTimeout(function () {
          window.location.replace(_redirectUrl);
        }, 900);
      }
      if (_callback) {
        _callback(data);
      }
    },
    error: function (jqXhr, textStatus, errorThrown) {
      var _serverMessage = "Request failed. Please try again.";
      if (jqXhr.responseJSON && (jqXhr.responseJSON.error || jqXhr.responseJSON.message)) {
        _serverMessage = jqXhr.responseJSON.error || jqXhr.responseJSON.message;
      } else if (jqXhr.responseText) {
        try {
          var _serverJSON = $.parseJSON(jqXhr.responseText);
          _serverMessage = _serverJSON.error || _serverJSON.message || _serverMessage;
        } catch (_parseErr) {
          _serverMessage = jqXhr.responseText;
        }
      } else if (errorThrown) {
        _serverMessage = errorThrown;
      }
      showAjaxMessage(_serverMessage, "error");
      console.log("Client: " + textStatus);
    },
  });
}
