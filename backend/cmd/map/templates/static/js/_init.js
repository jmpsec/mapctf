//
// the javascript for all the initkit-related
//  functioning, including the page router
//
(function (_initkit, $, undefined) {
  var $loadTarget, $body, MCTF_SECTION;

  /* --------------------------------------------
   * --PRIVATE
   * -------------------------------------------- */

  /**
   * set up and build the page, then run the init sequence
   */
  function _initkitBuildPage() {
    //
    // ok, everything is ready. lets do this
    //
    _initkit.init();
  }

  /* --------------------------------------------
   * --PUBLIC
   * -------------------------------------------- */

  /**
   * demo for the viewmode, since there's no interaction there.
   */
  _initkit.viewModeDemo = function () {
    $("body").on("gameboard-loaded", function (event) {
      setTimeout(function () {
        MAP_CTF.gameboard.captureCountry("Algeria");

        setTimeout(function () {
          MAP_CTF.gameboard.captureCountry("Chile");
        }, 10000);
      }, 4000);
    });
  };

  /**
   * enable the active state on the main nav. This function gets
   *  called since the pages are loaded via ajax.
   */
  _initkit.enableNavActiveState = function () {
    var hash = window.location.hash.replace("#", "");

    $(".mctf-main-nav a")
      .removeClass("active")
      .filter(function () {
        var href = $(this).data("active");

        if (href === undefined || !href.indexOf || hash === "") {
          return false;
        }
        return href.indexOf(hash) > -1;
      })
      .addClass("active");
  };

  /**
   * enable the active state on the admin nav. This function gets
   *  called since the pages are loaded via ajax.
   */
  _initkit.enableAdminActiveState = function () {
    var hash = window.location.hash.replace("#", "");

    $("#mctf-admin-nav li")
      .removeClass("active")
      .filter(function () {
        var href = $("a", this).attr("href").replace("#", "");

        if (href === undefined || !href.indexOf || hash === "") {
          return false;
        }
        return href.indexOf(hash) > -1;
      })
      .addClass("active");
  };

  /**
   * loads the content in the initkit
   */
  _initkit.load = function (file) {
    var loadDir = "/static/inc",
      loadSection = MCTF_SECTION,
      fileExt = ".html",
      loadPath = loadDir + "/" + loadSection + "/" + file + fileExt;

    //
    // if we're on mobile, show the mobile screen
    //
    if (window.innerWidth < 960) {
      loadPath = loadDir + "/components/mobile" + fileExt;
      $("body").addClass("mobile-device");
    }

    $loadTarget.load(loadPath, function (response, status, jqxhr) {
      if (status === "error") {
        console.error("There was a problem loading the content");
        console.log("loadPath: " + loadPath);
        console.log(response);
        console.error("/end error");
      } else {
        if (MAP_CTF !== undefined) {
          // Build emblem pickers from SVG sprite for any placeholder containers.
          $(".emblem-carousel").each(function () {
            MAP_CTF.buildEmblemCarousel($(this));
          });

          MAP_CTF.init();

          if (MCTF_SECTION === "gameboard" || MCTF_SECTION === "viewer-mode") {
            MAP_CTF.gameboard.init();

            if (MAP_CTF.gameboard.isViewMode()) {
              _initkit.viewModeDemo();
            }
          } else if (MCTF_SECTION === "admin") {
            MAP_CTF.admin.init();
          }

          $("body").trigger("content-loaded", { page: file });
        }
      }
    });
  };

  /**
   * router for the different initkit pages
   */
  _initkit.router = function (hash) {
    var fallback = "main",
      loadFile = !hash || hash == "" ? fallback : hash;

    _initkit.enableNavActiveState();
    _initkit.enableAdminActiveState();

    _initkit.load(loadFile);
  };

  _initkit.init = function () {
    $(window)
      .on("hashchange", function (event) {
        event.preventDefault();
        var hash = window.location.hash.replace("#", "");
        _initkit.router(hash);
      })
      .trigger("hashchange");
  };

  /**
   * set up stuff on document ready:
   *   - set global variables
   */
  $(document).ready(function () {
    $body = $("body");
    $loadTarget = $("#mctf-initkit");
    MCTF_SECTION = $body.data("section");

    //
    // route the sequence - if we're on the pages side of things,
    //  load the router and loader so we can navigate the
    //  different pages. Otherwise, if we're on the gameboard,
    //  init all the MAP_CTF things and load the gameboard
    //
    if (MCTF_SECTION === "pages") {
      _initkitBuildPage();
    } else if (MCTF_SECTION === "gameboard" || MCTF_SECTION === "viewer-mode") {
      _initkit.init();
    } else if (MCTF_SECTION === "admin") {
      _initkitBuildAdmin();
    }
  });
})((window._initkit = window._initkit || {}), jQuery);
