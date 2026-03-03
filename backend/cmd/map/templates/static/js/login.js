function sendLogin() {
  var _user = $("#login_username").val();
  var _password = $("#login_password").val();
  var _url = $("#login_url").val();

  var data = {
    username: _user,
    password: _password,
  };
  sendPostRequest(data, _url, "", false, false);
}

function sendLogout() {
  var _url = "/logout";
  var data = {
    csrftoken: _csrf,
  };
  sendPostRequest(data, _url, "/logout", false);
}

$("#login_password").keyup(function (event) {
  if (event.keyCode === 13) {
    $("#login_button").click();
  }
});
