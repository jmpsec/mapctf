function getSelectedEmblem() {
  return $("#register_logo").val() || "";
}

function setSelectedEmblem(logoName) {
  if (!logoName) {
    return;
  }
  $("#register_logo").val(logoName);
  $("#register_emblem_list .emblem-item").removeClass("active");
  $('#register_emblem_list .emblem-item[data-logo="' + logoName + '"]').addClass("active");
}

function buildEmblemPicker() {
  var $sprite = $("#mctf-svg-sprite");
  var $list = $("#register_emblem_list");
  if ($list.length === 0 || $sprite.length === 0) {
    return;
  }

  var badges = [];
  $sprite.find('symbol[id^="icon--badge-"]').each(function () {
    var symbolID = this.id || "";
    var logoName = symbolID.replace("icon--badge-", "");
    if (logoName !== "") {
      badges.push(logoName);
    }
  });
  badges.sort();

  if (badges.length === 0) {
    return;
  }

  var currentLogo = getSelectedEmblem();
  var defaultLogo = currentLogo || badges[0];
  var html = "";
  for (var i = 0; i < badges.length; i++) {
    var logo = badges[i];
    var activeClass = logo === defaultLogo ? " active" : "";
    html +=
      '<li class="emblem-item' +
      activeClass +
      '" data-logo="' +
      logo +
      '" role="button" tabindex="0" aria-label="Choose emblem ' +
      logo +
      '">' +
      '<svg class="icon--badge"><use xlink:href="#icon--badge-' +
      logo +
      '"></use></svg>' +
      "</li>";
  }

  $list.html(html);
  $("#register_logo").val(defaultLogo);
}

function sendRegistration() {
  var _username = $("#register_username").val();
  var _password = $("#register_password").val();
  var _name = $("#register_name").val();
  var _email = $("#register_email").val();
  var _team_name = $("#register_team_name").val();
  var _logo = getSelectedEmblem();
  var _url = $("#register_url").val();

  var data = {
    username: _username,
    password: _password,
    name: _name,
    email: _email,
    logo: _logo,
    team: _team_name
  };
  sendPostRequest(data, _url, "", false, false);
}

$("#register_password").keyup(function (event) {
  if (event.keyCode === 13) {
    $("#register_button").click();
  }
});

$(document).ready(function () {
  buildEmblemPicker();
});

$("body").on("content-loaded", function () {
  buildEmblemPicker();
});

$("body").on("click", "#register_emblem_list .emblem-item", function (event) {
  event.preventDefault();
  setSelectedEmblem($(this).data("logo"));
});

$("body").on("keydown", "#register_emblem_list .emblem-item", function (event) {
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    setSelectedEmblem($(this).data("logo"));
  }
});

window.getSelectedEmblem = getSelectedEmblem;
