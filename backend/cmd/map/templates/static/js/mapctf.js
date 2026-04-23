//
// MAPCTF javascript
//
(function (MAP_CTF, $, undefined) {
  var $body;

  // colors
  var COLOR_LIGHT_BLUE = "#cff8fa",
    COLOR_TEAL_BLUE = "#5cf0f6",
    COLOR_MAIN_BLUE = "#13242b";

  var COUNTRY_POLL_INTERVAL_MS = 15000,
    TEAM_POLL_INTERVAL_MS = 15000;

  // checks
  var ua = navigator.userAgent.toLowerCase(),
    is_firefox = ua.indexOf("firefox") > -1,
    is_ie = ua.indexOf("msie") > -1;

  MAP_CTF.debug = false;

  /* --------------------------------------------
   * --util
   * -------------------------------------------- */

  /**
   * get the given parameter value
   */
  function getUrlParameter(sParam) {
    var sPageURL = decodeURIComponent(window.location.search.substring(1)),
      sURLVariables = sPageURL.split("&"),
      sParameterName,
      i;

    for (i = 0; i < sURLVariables.length; i++) {
      sParameterName = sURLVariables[i].split("=");

      if (sParameterName[0] === sParam) {
        return sParameterName[1] === undefined ? true : sParameterName[1];
      }
    }
  }

  /* --------------------------------------------
   * --modules
   * -------------------------------------------- */

  /**
   * --gameboard
   *
   * handles all the loading of the modules and the map, as well as
   *  all map-related event listeners and animations
   */
  MAP_CTF.gameboard = (function () {
    var GAMEBOARD_LOADED = false,
      LIST_VIEW = false,
      VIEW_ONLY = false,
      CURRENT_ZOOM = 1,
      PRE_CAPTURE_TRANSFORM = null,
      COUNTRY_DATA,
      LAST_COUNTRY_ACTIVE_STATE = null,
      ACTIVE_COUNTRY_FILTER = null,
      COUNTRY_POLL_IN_FLIGHT = false,
      COUNTRY_POLL_TIMER = null,
      ACTIVITY_DATA,
      TEAM_DATA,
      TEAM_POLL_IN_FLIGHT = false,
      TEAM_POLL_TIMER = null,
      $gameboard,
      $listview,
      $mapSvg,
      $map,
      $countryHover;

    /**
     * enable click and drag capabilities on the map
     */
    var enableClickAndDrag = (function () {
      if (typeof d3 === "undefined") {
        return;
      }

      var $window = $(window);

      var zoom = d3.zoom().scaleExtent([1, 10]).on("zoom", zoomed);

      var svgMap, container;

      function init() {
        svgMap = d3.select("#mctf-gameboard-map").call(zoom);
        container = svgMap.select(".view-controller").call(zoom);

        $window.on("keyup", function (event) {
          var key = event.which,
            ww = $window.width(),
            wh = $window.height(),
            zoomin = is_firefox ? 61 : 187,
            zoomout = is_firefox ? 173 : 189;

          // the plus (zoom in) or the minus
          if (key === zoomin || key === zoomout) {
            var multiplier = key === zoomin ? 1 : -1;

            // Get current transform
            var currentTransform = d3.zoomTransform(svgMap.node());

            // Record the coordinates (in data space) of the center (in screen space).
            var center0 = [504, 325],
              translate0 = [currentTransform.x, currentTransform.y],
              coordinates0 = coordinates(center0),
              newScale = currentTransform.k * Math.pow(2, multiplier);

            // Clamp scale
            if (newScale > 10) {
              newScale = 10;
            } else if (newScale < 1) {
              newScale = 1;
            }

            // Translate back to the center.
            var center1 = point(coordinates0);
            var newTransform = d3.zoomIdentity.translate(translate0[0] + center0[0] - center1[0], translate0[1] + center0[1] - center1[1]).scale(newScale);

            svgMap.transition().duration(400).call(zoom.transform, newTransform);
          }
        });
      }

      /**
       * zoom and pan to the given coordinates
       *
       * @param latFocus (number) // default = 0
       *   - the x coordinate of the indicator
       *
       * @param lngFocus (number) // default = 0
       *   - the y coordinate of the indicator
       *
       * @param newScale (number) // default = 1
       *   - the zoom level
       */
      function zoomToPoint(latFocus, lngFocus, newScale) {
        // default parameters
        if (latFocus === undefined) latFocus = 0;

        if (lngFocus === undefined) lngFocus = 0;

        if (newScale === undefined) newScale = 1;

        var latModifier = -1 * (1 + (latFocus - 620) / 504);
        var lngModifier = VIEW_ONLY ? -1 * (1 + (lngFocus - 400) / 400) : -1 * (1 + (lngFocus - 325) / 325);

        var newTransform = d3.zoomIdentity.translate(latFocus * latModifier, lngFocus * lngModifier).scale(newScale);

        svgMap.transition().duration(400).call(zoom.transform, newTransform);
      }

      /**
       * function to call when the svg is zoomed
       */
      function zoomed(event, scale) {
        var transform = event ? event.transform : d3.zoomTransform(svgMap.node());
        var zoomScale = scale ? scale : transform.k,
          xyOffset = zoomScale,
          translateVal = scale ? [0, 0] : [transform.x, transform.y],
          indicatorRatio = 1 / zoomScale,
          transformRatio = 5.6 - 5.6 * indicatorRatio,
          panX = translateVal[0],
          panY = translateVal[1];

        MAP_CTF.modal.closeHoverPopup();

        container.attr("style", "transform: translate(" + panX + "px," + panY + "px) scale(" + zoomScale + ")");

        $(".countries .map-indicator path", $mapSvg).css({
          transform: "translate(" + transformRatio + "px," + transformRatio + "px) scale(" + indicatorRatio + ")",
        });

        $(".countries .land", $mapSvg).each(function () {
          var $self = $(this),
            modifier = $self.attr("class").indexOf("active") > -1 ? 1 : 1.5;

          $(this).css({
            "stroke-width": indicatorRatio * modifier,
          });
        });
      }

      /**
       * get svg coordinates
       */
      function coordinates(point) {
        var transform = d3.zoomTransform(svgMap.node());
        var scale = transform.k,
          translate = [transform.x, transform.y];
        return [(point[0] - translate[0]) / scale, (point[1] - translate[1]) / scale];
      }

      /**
       * get a point
       */
      function point(coordinates) {
        var transform = d3.zoomTransform(svgMap.node());
        var scale = transform.k,
          translate = [transform.x, transform.y];
        return [coordinates[0] * scale + translate[0], coordinates[1] * scale + translate[1]];
      }

      /**
       * get the zoom
       */
      function getZoom() {
        return d3.zoomTransform(svgMap.node()).k;
      }

      /**
       * get the active zoom/pan transform
       */
      function getTransform() {
        var transform = d3.zoomTransform(svgMap.node());
        return {
          x: transform.x,
          y: transform.y,
          k: transform.k,
        };
      }

      /**
       * restore a previously captured zoom/pan transform
       */
      function setTransform(transform) {
        if (!transform) {
          return;
        }

        var newTransform = d3.zoomIdentity.translate(transform.x, transform.y).scale(transform.k);

        svgMap.transition().duration(400).call(zoom.transform, newTransform);
      }

      return {
        init: init,
        zoom: zoom,
        getZoom: getZoom,
        getTransform: getTransform,
        setTransform: setTransform,
        zoomToPoint: zoomToPoint,
        zoomed: zoomed,
      };
    })(); // enableClickAndDrag

    //
    // PRIVATE
    //

    /**
     * build the gameboard
     */
    function build() {
      var countryDataLoaded = getCountryData();

      //
      // add the modules and the map
      //
      var modulesLoaded = loadModules(),
        mapLoaded = loadMap(),
        listViewLoaded = loadListView(),
        activityDataLoaded = loadActivityData(),
        teamDataLoaded = loadTeamData();

      $.when(mapLoaded, listViewLoaded, countryDataLoaded).done(function () {
        renderCountryData();
      });

      $.when(modulesLoaded, countryDataLoaded).done(function () {
        renderFilterOptions();
      });

      // do stuff when the map and modules are loaded
      $.when(modulesLoaded, mapLoaded, listViewLoaded, teamDataLoaded, activityDataLoaded).done(function () {
        console.log("modules, map, list view, team data, and activity data are loaded");

        // trigger an event for the gameboard loaded, so
        //  external things know that everything has been
        //  loaded (like the initkit js).
        $("body").trigger("gameboard-loaded");

        // check off that the gameboard is loaded (or in the
        //  process of being loaded)
        GAMEBOARD_LOADED = true;

        //
        // set up the listeners
        //
        if (!VIEW_ONLY) {
          gameEventListeners();
        }

        //
        // init some other stuff
        //

        // load the command line
        MAP_CTF.command_line.init();

        // populate the leaderboard module
        setupLeaderboard();

        // popuplate the team module
        setupTeams();
        setupActivity();
        startCountryPolling();
        startTeamPolling();
      });
    }

    /* --------------------------------------------
     * --setup
     * -------------------------------------------- */

    /**
     * set up the team module in the gameboard with the list of
     *  active teams (from the team.json data) and event
     *  listeners for the team modal
     */
    function setupTeams() {
      if (TEAM_DATA === undefined) {
        console.error("No team data available.");
        return;
      }

      var $teamgrid = $('aside[data-module="teams"] .grid-list');
      var showTeamMembers = shouldShowTeamMembers();
      $teamgrid.empty();

      //
      // build the team module in the gameboard, which will
      //  list the active teams
      //
      $.each(TEAM_DATA, function (teamName, teamData) {
        var $item = $("<li></li>");
        var $link = $('<a href="#" class="team-card"></a>').attr("data-team", teamName);
        var $header = $('<div class="team-card-header"></div>');
        var $identity = $('<div class="team-card-identity"></div>');
        var $badge = $('<svg class="icon--badge"><use xlink:href=""></use></svg>');
        var $text = $('<div class="team-card-text"></div>');
        var $name = $('<div class="team-card-name"></div>').text(teamName);
        var $points = $('<div class="team-card-points"></div>');
        var $members = $('<ul class="team-card-members"></ul>');
        var $footer = $('<div class="team-card-footer"></div>');
        var members = $.isArray(teamData.team_members) ? teamData.team_members : [];

        if (teamData.has_alert) {
          $item.addClass("alert");
        }

        $("use", $badge).attr("xlink:href", "#icon--badge-" + teamData.badge);
        $points.append('<span class="team-card-points-value mctf-numbers"></span>');
        $(".team-card-points-value", $points).text(teamData.points);
        $points.append('<span class="team-card-points-label">Points</span>');

        $text.append($name);
        if (showTeamMembers && members.length) {
          $.each(members, function (_, memberName) {
            $members.append($("<li></li>").text(memberName));
          });
          $text.append($members);
        }

        $identity.append($badge, $text);
        $header.append($identity, $points);
        $footer.append($('<span class="team-card-last-score-label">Last score</span>'));
        $footer.append($('<span class="team-card-last-score-value"></span>').text(teamData.last_score_label));

        $link.append($header, $footer);
        $item.append($link);
        $teamgrid.append($item);
      });

      //
      // launch the team modals
      //
      $teamgrid.off("click", "a[data-team]");
      $teamgrid.on("click", "a[data-team]", function (event) {
        event.preventDefault();
        var team = $(this).data("team");

        //
        // @note - here, let's check to see if the team is
        //  set in the markup. Since this is a initkit,
        //  we'll set a default for demo purposes.
        //
        if (team === undefined || team === "") {
          team = "Tank SF";
        }

        var teamData = TEAM_DATA[team];

        if (teamData === undefined) {
          console.error("Invalid team name in markup");
          return;
        }

        MAP_CTF.modal.loadPopup("team", function () {
          var $modal = $("#mctf-modal"),
            $teamMembers = $(".team-members", $modal);

          // team name
          $(".team-name", $modal).text(team);

          // team badge
          $(".icon--badge use", $modal).attr("xlink:href", "#icon--badge-" + teamData.badge);

          // team members
          $teamMembers.empty();
          if (showTeamMembers) {
            $.each(teamData.team_members, function () {
              $teamMembers.append("<li>" + this + "</li>");
            });
          }
          if (showTeamMembers && teamData.team_members.length) {
            $(".team-members-section", $modal).show();
          } else {
            $(".team-members-section", $modal).hide();
          }

          $(".team-points-value", $modal).text(teamData.points);
          $(".last-score", $modal).text(teamData.last_score_label);
        });
      });
    }

    function setupLeaderboard() {
      if (TEAM_DATA === undefined) {
        console.error("No team data available for leaderboard.");
        return;
      }

      var $leaderboard = $('aside[data-module="leaderboard"]');
      var $leaderboardList = $(".leaderboard-list", $leaderboard);
      var currentTeam = getCurrentTeamName();
      var currentTeamData = currentTeam && TEAM_DATA[currentTeam] ? TEAM_DATA[currentTeam] : null;
      var sortedTeams = [];

      $leaderboardList.empty();

      $.each(TEAM_DATA, function (teamName, teamData) {
        sortedTeams.push({
          name: teamName,
          data: teamData,
        });
      });

      sortedTeams.sort(function (a, b) {
        return a.data.rank - b.data.rank;
      });

      if (currentTeamData) {
        $(".player-name", $leaderboard).text(currentTeam);
        $(".module-top .player-rank .stat-value", $leaderboard).text(currentTeamData.rank);
        $(".module-top .player-score .stat-value", $leaderboard).text(currentTeamData.points);
      } else {
        $(".player-name", $leaderboard).text("No Team");
        $(".module-top .player-rank .stat-value", $leaderboard).text("--");
        $(".module-top .player-score .stat-value", $leaderboard).text("0");
      }

      if (!sortedTeams.length) {
        $leaderboardList.append('<li class="leaderboard-empty">No teams yet</li>');
        return;
      }

      $.each(sortedTeams, function (_, teamEntry) {
        var teamName = teamEntry.name;
        var teamData = teamEntry.data;
        var markup =
          '<li class="mctf-user-card">' +
          '<div class="user-avatar">' +
          '<svg class="icon--badge"><use xlink:href="#icon--badge-' +
          teamData.badge +
          '"></use></svg>' +
          "</div>" +
          '<div class="player-info">' +
          "<h6>" +
          teamName +
          "</h6>" +
          '<span class="player-rank"><span class="stat-label">Rank</span><span class="stat-value">' +
          teamData.rank +
          "</span></span>" +
          '<span class="player-score"><span class="stat-label">Points</span><span class="stat-value">' +
          teamData.points +
          "</span></span>" +
          "</div>" +
          "</li>";

        $leaderboardList.append(markup);
      });
    }

    function getCurrentTeamName() {
      if (MAP_CTF.data && MAP_CTF.data.CONF && MAP_CTF.data.CONF.currentTeam) {
        return MAP_CTF.data.CONF.currentTeam;
      }

      return "";
    }

    function buildActivityText(entry) {
      var parts = [];

      if (entry && entry.Message) {
        parts.push(entry.Message);
      }
      if (entry && entry.Action) {
        parts.push(entry.Action);
      }
      if (entry && entry.Arguments) {
        parts.push(entry.Arguments);
      }

      return $.trim(parts.join(" "));
    }

    function setupActivity() {
      var $activityStream = $('aside[data-module="activity"] .activity-stream');
      var currentTeam = getCurrentTeamName();

      $activityStream.empty();
      $activityStream.append('<li class="activity-empty" style="display: none">No activity yet</li>');

      if ($.isArray(ACTIVITY_DATA) && ACTIVITY_DATA.length) {
        $.each(ACTIVITY_DATA, function (_, entry) {
          var subject = entry && entry.Subject ? entry.Subject.toString() : "";
          var messageText = buildActivityText(entry);
          var isYourTeam = currentTeam && subject === currentTeam;
          var itemClass = isYourTeam ? "your-team" : "opponent-team";
          var subjectClass = isYourTeam ? "your-name" : "opponent-name";
          var $item = $("<li></li>").addClass(itemClass + " activity-entry");

          if (subject) {
            $item.append($("<span></span>").addClass(subjectClass).text(subject));
            if (messageText) {
              $item.append(document.createTextNode(" " + messageText));
            }
          } else if (messageText) {
            $item.text(messageText);
          }

          if (!subject && !messageText) {
            return;
          }

          $activityStream.append($item);
        });
      }

      updateActivityEmptyState($activityStream);
    }

    function updateActivityEmptyState($activityStream) {
      var $emptyState = $activityStream.find(".activity-empty");
      var hasVisibleEntries = $activityStream.find(".activity-entry:visible").length > 0;

      $emptyState.toggle(!hasVisibleEntries);
    }

    function getCurrentUUID() {
      var path = (window.location.pathname || "").toString();
      var segments = path.split("/");

      if (segments.length > 1 && segments[1] !== "") {
        return segments[1];
      }

      return "";
    }

    function normalizeBadgeName(logoValue) {
      var badge = (logoValue || "").toString().trim();

      if (!badge) {
        return "invader";
      }

      if (badge.indexOf("/") > -1) {
        badge = badge.substring(badge.lastIndexOf("/") + 1);
      }

      badge = badge.replace(/^#icon--badge-/, "");
      badge = badge.replace(/^icon--badge-/, "");
      badge = badge.replace(/^badge-/, "");
      badge = badge.replace(/\.svg$/i, "");

      if (!badge) {
        return "invader";
      }

      return badge;
    }

    function normalizeTeamMembers(team) {
      if ($.isArray(team && team.TeamMembers)) {
        return team.TeamMembers;
      }

      if ($.isArray(team && team.team_members)) {
        return team.team_members;
      }

      return [];
    }

    function shouldShowTeamMembers() {
      return $body && $body.attr("data-show-team-members") === "true";
    }

    function formatLastScoreLabel(lastScoreValue) {
      if (!lastScoreValue) {
        return "No score yet";
      }

      var lastScoreDate = new Date(lastScoreValue);

      if (isNaN(lastScoreDate.getTime())) {
        return "No score yet";
      }

      return lastScoreDate.toLocaleString([], {
        year: "numeric",
        month: "short",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      });
    }

    function mapServerTeamsToTeamData(serverTeams) {
      if (!$.isArray(serverTeams)) {
        return serverTeams || {};
      }

      var mapped = {};
      var sortedTeams = serverTeams.slice(0);

      sortedTeams.sort(function (a, b) {
        var pointsA = parseInt(a && (a.Points !== undefined ? a.Points : a.points), 10);
        var pointsB = parseInt(b && (b.Points !== undefined ? b.Points : b.points), 10);
        var safePointsA = isNaN(pointsA) ? 0 : pointsA;
        var safePointsB = isNaN(pointsB) ? 0 : pointsB;

        if (safePointsB !== safePointsA) {
          return safePointsB - safePointsA;
        }

        var nameA = (a && (a.Name || a.name) ? a.Name || a.name : "").toString().toLowerCase();
        var nameB = (b && (b.Name || b.name) ? b.Name || b.name : "").toString().toLowerCase();
        return nameA.localeCompare(nameB);
      });

      var rank = 1;
      $.each(sortedTeams, function (_, team) {
        var teamName = team && (team.Name || team.name);

        if (!team || !teamName) {
          return;
        }

        if (team.Active === false || team.Visible === false) {
          return;
        }

        teamName = teamName.toString();
        var totalPoints = parseInt(team.Points !== undefined ? team.Points : team.points, 10);
        var safeTotalPoints = isNaN(totalPoints) ? 0 : totalPoints;
        var teamMembers = normalizeTeamMembers(team);
        var lastScoreValue = team.LastScore || team.last_score || "";

        mapped[teamName] = {
          badge: normalizeBadgeName(team.Logo || team.logo),
          team_members: teamMembers,
          rank: rank,
          last_score: lastScoreValue,
          last_score_label: formatLastScoreLabel(lastScoreValue),
          points: safeTotalPoints,
        };
        rank += 1;
      });

      return mapped;
    }

    /* --------------------------------------------
     * --
     * -------------------------------------------- */

    /**
     * get the owner of the given country, and return the markup
     *  for rendering somewhere
     *
     * @param capturedBy (string)
     *   - the capturing team
     */
    function getCapturedByMarkup(capturedBy) {
      if (capturedBy === undefined) {
        return "Uncaptured";
      }

      var capturedClass = capturedBy === MAP_CTF.currentUser ? "your-name" : "opponent-name";

      return '<span class="' + capturedClass + '">' + capturedBy + "</span>";
    }

    /**
     * automatically scroll through the content on the sidebar
     *  modules. This happens in the "view only" mode
     */
    function autoScrollModules() {
      var $modules = $('aside[data-module="under-attack"], aside[data-module="leaderboard-viewmode"]');

      $modules.each(function () {
        var $scrollable = $(".module-scrollable", this),
          scrollHeight = $scrollable.children("ul").height() - $scrollable.height(),
          scrollTime = scrollHeight / 0.05,
          scrollInterval;

        $scrollable.on("mouseover", function (event) {
          event.preventDefault();
          clearInterval(scrollInterval);
        });
        $scrollable.on("mouseout", startScroll);

        /**
         * start the scrolling interval
         */
        function startScroll() {
          scrollInterval = setInterval(function () {
            var st = $scrollable.scrollTop(),
              scrollLeft = $("ul", $scrollable).height() - $scrollable.height();

            if (st >= scrollLeft) {
              $scrollable.scrollTop(0);
            } else {
              $scrollable.scrollTop(st + 1);
            }
          }, 20);
        }

        startScroll();
      });
    }

    /**
     * the event listeners for the game
     */
    function gameEventListeners() {
      var $svgCountries = $(".countries > g", $mapSvg);

      /* --------------------------------------------
       * --modules
       * -------------------------------------------- */

      //
      // open the module
      //
      $("aside[data-module]").on("click", ".module-header", function (event) {
        event.preventDefault();
        $(this).closest("aside").toggleClass("active");

        $body.trigger("module-changestate");
      });

      //
      // check to see if a module has changed its state - it it
      //  has, check the listview to rearrange it
      //
      $body.on("module-changestate", function () {
        var rightModules = false,
          bottomModules = false,
          margin = "25%",
          $listviewContainer = $(".listview-container", $listview);

        $("aside[data-module].active").each(function () {
          var $container = $(this).closest(".mctf-module-container");

          if ($container.hasClass("column-right")) {
            rightModules = true;

            if ($container.width() > 350) {
              margin = "390px";
            }
          } else if ($container.hasClass("container--row")) {
            bottomModules = true;
          }
        });

        if (rightModules) {
          $listviewContainer.css("right", margin);
        } else {
          $listviewContainer.css("right", "10px");
        }
        if (bottomModules) {
          $listviewContainer.css("bottom", "34vh");
        } else {
          $listviewContainer.css("bottom", "100px");
        }
      });

      /* --------------------------------------------
       * --alerts
       * -------------------------------------------- */

      //
      // create new individual chat
      //
      $body.on("click", ".add-new-help-chat", function (event) {
        event.preventDefault();
        var country = $(this).data("country"),
          $alerts = $('aside[data-module="world-chat"] .alerts'),
          $chat = $("li.alert-placeholder", $alerts).eq(0).clone();

        $chat.removeClass("alert-placeholder").attr("data-help", country).appendTo($alerts).find(".alert .highlighted").text(country);
        $(".js-expand-individual-chat", $chat).trigger("click");
      });

      //
      // delete alert
      //
      $body.on("click", ".js-delete-alert", function (event) {
        event.preventDefault();
        var $self = $(this),
          $li = $self.closest("li").addClass("removing");

        $self.closest(".alerts").removeClass("individual-chat-enabled").closest("aside").removeClass("individual-chat-active");

        setTimeout(function () {
          $li.remove();
        }, 400);
      });

      //
      // expand individual chat
      //
      $body.on("click", ".js-expand-individual-chat", function (event) {
        event.preventDefault();

        $(this).closest("li").toggleClass("active").closest(".alerts").toggleClass("individual-chat-enabled").closest("aside").toggleClass("individual-chat-active");

        $body.trigger("module-changestate");
      });

      /* --------------------------------------------
       * --inputs
       * -------------------------------------------- */

      //
      // filter the map and list view based on category / point value
      //
      $body.on('change', 'input[name="mctf--module--filter--category"], input[name="mctf--module--filter--point-value"]', function (event) {
        event.preventDefault();
        ACTIVE_COUNTRY_FILTER = {
          name: this.name,
          value: $(this).val(),
        };
        applyCountryFilter(this.name, $(this).val());
      });

      //
      // filter the map based on status
      //
      $('input[name="mctf--map-select"]').on("change", function (event) {
        event.preventDefault();
        var select = $(this).val();

        $svgCountries.each(function () {
          var countryGroup = d3.select(this),
            captureTeam = countryGroup.attr("data-captured");

          countryGroup.classed("inactive", false);
          countryGroup.classed("highlighted", false);

          if (select !== "all") {
            if ((select === "your-team" && captureTeam === MAP_CTF.data.CONF.currentTeam) || (select === "opponent-team" && captureTeam && captureTeam !== MAP_CTF.data.CONF.currentTeam)) {
              countryGroup.classed("highlighted", true);
            } else {
              countryGroup.classed("inactive", true);
            }
          }
        });

        $("tr", $listview).each(function () {
          var $tr = $(this),
            $self = $tr.removeClass("inactive highlighted"),
            captureTeam = $self.data("captured");

          if (select !== "all") {
            if ((select === "your-team" && captureTeam === MAP_CTF.data.CONF.currentTeam) || (select === "opponent-team" && captureTeam && captureTeam !== MAP_CTF.data.CONF.currentTeam) || (select === "give-help" && $(".status--give-help", $tr).length > 0) || (select === "need-help" && $(".status--incoming-help", $tr).length > 0)) {
              $self.addClass("highlighted");
            } else {
              $self.addClass("inactive");
            }
          }
        });
      });

      //
      // activity module filter
      //
      $('input[name="mctf--module--activity"]').on("change", function (event) {
        event.preventDefault();
        var $self = $(this),
          select = $self.val(),
          $activityStream = $self.closest(".module-content").find(".activity-stream"),
          $li = $activityStream.find(".activity-entry").show();

        if (select !== "all") {
          $li
            .filter(function () {
              return $(this).attr("class") !== select;
            })
            .hide();
        }

        updateActivityEmptyState($activityStream);
      });

      /* --------------------------------------------
       * --country interaction
       * -------------------------------------------- */

      //
      // on country click, open the "capture country" modal
      //
      $map.on("click", ".country-hover g", function (event) {
        event.preventDefault();

        var country = $('[class~="land"]', this).attr("title");

        CURRENT_ZOOM = enableClickAndDrag.getZoom();
        captureCountry(country);
      });

      //
      // hover on a country
      //
      $map.hoverIntent({
        over: countryHover,
        out: function () {},
        selector: ".countries > g",
      });

      $countryHover.on("mouseleave", function (event) {
        event.preventDefault();
        MAP_CTF.modal.closeHoverPopup();
        $countryHover.empty();
      });
    } // function gameEventListeners()

    /* --------------------------------------------
     * --svg interactions
     * -------------------------------------------- */

    /**
     * when a country is clicked on:
     *   - add the crosshairs interaction
     *   - launch the "capture_country" modal
     *
     * @param country (string)
     *   - the country that is being captured
     *
     * @param capturingTeam (string)
     *   - an optional parameter for the team that is attempting
     *      to capture the given country
     */
    function captureCountry(country, capturingTeam) {
      var $selectCountry = $('.countries .land[title="' + country + '"]', $mapSvg),
        data = COUNTRY_DATA ? COUNTRY_DATA[country] : null,
        capturedBy = getCapturedByMarkup($selectCountry.closest("g").data("captured")),
        showAnimation = !(is_ie || LIST_VIEW),
        animationDuration = !showAnimation ? 0 : 600;

      // make sure there's a country node
      if ($selectCountry.length === 0) {
        console.error(country + " is not a valid country");
        return;
      }

      if (!data || !data.active) {
        return;
      }

      if ($countryHover.has("g").length === 0) {
        var $hoveredCountry = $('.countries .land[title="' + country + '"]', $mapSvg)
          .closest("g")
          .clone();
        $countryHover.data($hoveredCountry.data());
        $countryHover.empty().append($hoveredCountry);
      }

      // close the hover popup
      MAP_CTF.modal.closeHoverPopup();

      PRE_CAPTURE_TRANSFORM = enableClickAndDrag.getTransform();

      // engage the crosshairs animation
      if (showAnimation) {
        engageCrosshairs();
      }

      // if there's no data, don't continue
      if (!COUNTRY_DATA) {
        return;
      }

      if (VIEW_ONLY) {
        setTimeout(function () {
          captureViewOnly(country, capturedBy, capturingTeam);
        }, animationDuration);
      } else {
        setTimeout(function () {
          launchCaptureModal(country, capturedBy);
        }, animationDuration);
      }
    } // function countryClick();

    /**
     * the animation that plays when the user is in view-only mode
     *
     * @param country (string)
     *   - the country being captured
     *
     * @param capturedBy (string)
     *   - the user or team who has captured this country
     *
     * @param capturingTeam (string)
     *   - an optional parameter for the team that is attempting
     *      to capture the given country
     */
    function captureViewOnly(country, capturedBy, capturingTeam) {
      if (capturingTeam === undefined) {
        capturingTeam = MAP_CTF.data.CONF.currentTeam;
      }

      MAP_CTF.modal.viewmodePopup(function () {
        var $container = $("#mctf-country-popup"),
          positionX = $(".longitude-focus").position().left + 60,
          positionY = $(".latitude-focus").position().top - $container.height() - 60,
          points = COUNTRY_DATA && COUNTRY_DATA[country] ? COUNTRY_DATA[country].points : 0;

        $(".capturing-team-name", $container).text(capturingTeam);
        $(".points-value", $container).text("+ " + points + " Pts");
        $(".country-owner", $container).html(capturedBy);
        $(".country-name", $container).text(formatCountryLabel(country));

        $container.css({
          left: positionX + "px",
          top: positionY + "px",
        });

        setTimeout(function () {
          removeCaptured();
          MAP_CTF.modal.closeHoverPopup();

          enableClickAndDrag.zoomToPoint();
        }, 5000);
      });
    }

    /**
     * remove all the captured/hovered states from the map
     *
     * @param event (object)
     *   - if this is called from an event listener, it comes with
     *      an event object
     */
    function removeCaptured(event) {
      if (event) event.preventDefault();

      if (PRE_CAPTURE_TRANSFORM) {
        enableClickAndDrag.setTransform(PRE_CAPTURE_TRANSFORM);
        PRE_CAPTURE_TRANSFORM = null;
      } else if (CURRENT_ZOOM > 1.1) {
        enableClickAndDrag.zoomToPoint();
      }

      $('[class~="country-clicked"]', $mapSvg).fadeOut(function () {
        $(this).remove();
      });
      $countryHover.empty();
    }

    /**
     * launch the "capture country" modal
     *
     * @param country (string)
     *   - the country that is being captured
     *
     * @param capturedBy (string)
     *   - the user or team who has captured this country
     */
    function launchCaptureModal(country, capturedBy) {
      var data = COUNTRY_DATA[country];

      if (!data || !data.active) {
        MAP_CTF.modal.closeHoverPopup();
        $countryHover.empty();
        return;
      }

      MAP_CTF.modal.loadPopup("country-capture", function () {
        var $container = $(".mctf-modal-content"),
          intro = data ? data.intro : "",
          hint = data ? data.hint : "",
          points = data ? data.points : "",
          category = data ? data.category : "",
          completed = data ? data.completed : "";

        $(".country-name", $container).text(formatCountryLabel(country));
        $(".capture-text", $container).text(intro);
        $(".capture-hint div", $container).text(hint);
        $(".points-number", $container).text(points);
        $(".country-category", $container).text(category);
        $(".country-owner", $container).html(capturedBy);

        if (completed instanceof Array) {
          $.each(completed, function () {
            $(".completed-list", $container).append("<li>" + this + "</li>");
          });
        }

        //
        // event listeners
        //
        $(".js-trigger-hint", $container).on("click", function (event) {
          event.preventDefault();
          $(this).onlySiblingWithClass("active").closest(".mctf-modal-content").removeClass("help-enabled").addClass("hint-enabled");
        });
        $(".js-trigger-help", $container).on("click", function (event) {
          event.preventDefault();
          $(this).onlySiblingWithClass("active").closest(".mctf-modal-content").removeClass("hint-enabled").addClass("help-enabled");
        });

        $(".js-close-modal", $container).on("click", removeCaptured);
      });
    } // function launchCaptureModal();

    /**
     * on hover of a country in the svg
     */
    function countryHover(event) {
      if (event) event.preventDefault();

      var $self = $(this),
        country = $('[class~="land"]', $self).attr("title"),
        capturedBy = getCapturedByMarkup($self.data("captured")),
        mouse_x = event.pageX,
        mouse_y = event.pageY;

      if (!COUNTRY_DATA) {
        return;
      }

      var data = COUNTRY_DATA[country];

      MAP_CTF.modal.countryHoverPopup(function () {
        var $container = $("#mctf-country-popup").css({
            left: mouse_x + "px",
            top: mouse_y + "px",
          }),
          points = data ? data.points : "",
          category = data ? data.category : "";

        $(".country-name", $container).text(formatCountryLabel(country));
        $(".points-number", $container).text(points);
        $(".country-category", $container).text(category);
        $(".country-owner", $container).html(capturedBy);
      });

      //
      // add the country path to the hover group so we can
      //  see the outline
      //
      var clone = $self.clone();
      $countryHover.data($self.data()).empty().append(clone);
    } // function countryHover();

    /**
     * the crosshairs interstitial animation that takes place
     *  before the "capture_country" modal appears
     */
    function engageCrosshairs() {
      var svgMap = d3.select("#mctf-gameboard-map"),
        hoveredCountry = svgMap.select(".country-hover .land"),
        indicator = svgMap.select(".country-hover .map-indicator");

      //
      // make sure the indicator exists.
      //
      if (indicator.empty()) {
        return;
      }

      var indicatorStyle = indicator.attr("transform"),
        // the view controller
        viewController = svgMap.select(".view-controller"),
        country_clicked = viewController.append("g").attr("class", "country-clicked"),
        $country_clicked = $('[class~="country-clicked"]', $mapSvg),
        // setting the explicit height and width of the indictator
        //  seem to be the most effecttive way to position the
        //  crosshair stuff. The sizes are based on the size of the
        //  indicator when the svg is it's natural size
        indicatorWidth = indicatorStyle.indexOf("scale(") > -1 ? 5.82 : 9.7,
        indicatorHeight = indicatorStyle.indexOf("scale(") > -1 ? 5.46 : 9.1,
        translateString = indicatorStyle.replace(" scale(0.6)", "").substring(indicatorStyle.lastIndexOf("translate(") + 10, indicatorStyle.lastIndexOf(")")),
        translateArray = translateString.replace(new RegExp("px", "g"), "").replace(" ", ",").split(","),
        latFocus = parseFloat(translateArray[0]) + indicatorWidth / 2,
        lngFocus = parseFloat(translateArray[1]) + indicatorHeight / 2 + 2,
        mapHeight = $mapSvg.height(),
        mapWidth = $mapSvg.width(),
        zoom = enableClickAndDrag.getZoom();

      zoom = zoom < 2 ? 2 : zoom;

      var zoomRatio = 1 / zoom,
        focusAdjusted = 10 * zoomRatio,
        latLngAdjusted = 1 / (5.6 * zoomRatio),
        xRatio = 1 / zoomRatio / 10;

      enableClickAndDrag.zoomToPoint(latFocus, lngFocus, 2);

      // add a slightly transparent background to overlay
      //  the rest of the map
      country_clicked.append("rect").attr("x", 0).attr("y", 0).attr("width", mapWidth).attr("height", mapHeight).attr("fill", "#13242b").attr("style", "opacity:.6");

      // add the hovered country path
      $country_clicked.append(hoveredCountry.node());

      // add the crosshairs
      var crosshairs_xy_translate = "translate(" + (latFocus + xRatio - 100) + "px," + (lngFocus - xRatio - 100) + "px) scale(" + zoomRatio + ")";
      var crosshairs_xy = country_clicked
        .append("g")
        .attr("class", "crosshairs")
        .attr("style", "transform:" + crosshairs_xy_translate + "; -webkit-transform:" + crosshairs_xy_translate + ";-moz-transform:" + crosshairs_xy_translate + ";");

      var crosshairs = crosshairs_xy.append("g").attr("class", "crosshairs-rotate");

      // x
      var latLines_translate = "translate(" + xRatio + "px," + (lngFocus - xRatio - 100) + "px)";
      var latLines = country_clicked
        .append("g")
        .attr("class", "latitude-focus")
        .attr("stroke", COLOR_TEAL_BLUE)
        .attr("stroke-width", zoomRatio)
        .attr("style", "transform:" + latLines_translate + ";-webkit-transform:" + latLines_translate + ";-moz-transform:" + latLines_translate + ";");
      latLines.append("path").attr("d", "M0,0L" + (latFocus - focusAdjusted) + ",0");
      latLines.append("path").attr("d", "M" + (latFocus + focusAdjusted) + ",0L" + mapWidth + ",0");

      // y
      var lngLines_translate = "translate(" + (latFocus + xRatio - 100) + "px,-" + xRatio + "px)";
      var lngLines = country_clicked
        .append("g")
        .attr("class", "longitude-focus")
        .attr("stroke", COLOR_TEAL_BLUE)
        .attr("stroke-width", zoomRatio)
        .attr("style", "transform:" + lngLines_translate + ";-webkit-transform:" + lngLines_translate + ";-moz-transform:" + lngLines_translate + ";");

      lngLines.append("path").attr("d", "M0,0L0," + (lngFocus - focusAdjusted));
      lngLines.append("path").attr("d", "M0," + (lngFocus + focusAdjusted) + "L0," + mapHeight);

      // circles
      crosshairs.append("circle").attr("cx", 0).attr("cy", 0).attr("r", 30).attr("fill", "none").attr("stroke-dasharray", "31.4 15.7").attr("style", "transform:rotate(-75deg);-webkit-transform:rotate(-75deg);-moz-transform:rotate(-75deg)").attr("stroke", COLOR_TEAL_BLUE);
      crosshairs.append("circle").attr("cx", 0).attr("cy", 0).attr("r", 35).attr("fill", "none").attr("stroke-dasharray", "9.158333333 45.79167").attr("style", "transform:rotate(-53deg);-webkit-transform:rotate(-53deg);-moz-transform:rotate(-53deg)").attr("stroke-width", 2).attr("stroke", COLOR_TEAL_BLUE);

      setTimeout(function () {
        var latLines_active = "translate(" + xRatio + "px," + (lngFocus - xRatio) + "px)";
        latLines.attr("style", "transform:" + latLines_active + ";-webkit-transform:" + latLines_active + ";-moz-transform:" + latLines_active + ";");

        var latLines_active = "translate(" + (latFocus + xRatio) + "px,-" + xRatio + "px)";
        lngLines.attr("style", "transform:" + latLines_active + ";-webkit-transform:" + latLines_active + ";-moz-transform:" + latLines_active + ";");

        var crosshairs_xy_active = "translate(" + (latFocus + xRatio) + "px," + (lngFocus - xRatio) + "px) scale(" + zoomRatio + ")";
        crosshairs_xy.attr("style", "transform:" + crosshairs_xy_active + ";-webkit-transform:" + crosshairs_xy_active + ";-moz-transform:" + crosshairs_xy_active + ";");
      }, 10);

      // add the indicator
      $country_clicked.append(indicator.node());
    }

    /* --------------------------------------------
     * --loading
     * -------------------------------------------- */

    /**
     * load up all the modules
     *
     * @return Promise
     *   - when all the modules are loaded, return a promise
     */
    function loadModules() {
      var $modules = $("aside[data-module]", $gameboard),
        df = $.Deferred(),
        deferredArray = [];

      $modules.each(function () {
        var $self = $(this),
          module = $self.data("module"),
          modulePath = "/static/inc/gameboard/modules/" + module + ".html";

        var get = $.get(modulePath, function (data, status, jqxhr) {
          $self.html(data);
        }).fail(function () {
          console.error("There was a problem retrieving the module.");
          console.log(modulePath);
          console.error("/error");
        });

        deferredArray.push(get);
      });

      $.when.apply($, deferredArray).done(function () {
        console.log("modules loaded");

        if (VIEW_ONLY) {
          autoScrollModules();
        }

        df.resolve();
      });

      return df;
    }

    /**
     * load the svg map. This inserts the svg into the page and
     *  then sets up some jquery object variables for use
     *  elsewhere in this module.
     */
    function loadMap() {
      var df = $.Deferred();
      $map = $(".mctf-map");
      $mapSvg = $("#mctf-gameboard-map", $map);

      if ($mapSvg.length) {
        console.log("map loaded");
        $countryHover = $('[class~="country-hover"]', $mapSvg);
        enableClickAndDrag.init();
        return df.resolve().promise();
      }

      var mapPath = "/static/svg/map/worldLow.svg";

      return $.get(
        mapPath,
        function (data, status, jqxhr) {
          console.log("map loaded");

          $map = $(".mctf-map");
          $map.html(data);

          $mapSvg = $("#mctf-gameboard-map");
          $countryHover = $('[class~="country-hover"]', $mapSvg);

          enableClickAndDrag.init();
        },
        "html",
      ).fail(function () {
        console.error("There was a problem loading the svg map");
        console.error("/error");
      });
    }

    /**
     * load the list view for the game
     */
    function loadListView() {
      var df = $.Deferred();

      console.log("List view loaded");
      $listview = $(".mctf-listview");

      if (!$listview.length) {
        console.error("There was a problem loading the List View");
        console.error("/error");
        return df.reject().promise();
      }

      $listview.html('<div class="listview-container"><table><tbody></tbody></table></div>');
      return df.resolve().promise();
    }

    /**
     * load the team data
     */
    function loadTeamData(forceRefresh) {
      if (!forceRefresh && TEAM_DATA) {
        return $.Deferred().resolve(TEAM_DATA).promise();
      }

      var df = $.Deferred();
      var uuid = getCurrentUUID();
      var serverPath = uuid ? "/" + encodeURIComponent(uuid) + "/json/teams" : "";
      var fallbackPath = "/static/data/teams.json";

      function loadFallbackData() {
        return $.get(
          fallbackPath,
          function (data) {
            TEAM_DATA = mapServerTeamsToTeamData(data);
            df.resolve(TEAM_DATA);
          },
          "json",
        ).fail(function (jqhxr, status, error) {
          console.error("There was a problem retrieving the team data.");
          console.log(fallbackPath);
          console.log(status);
          console.log(error);
          console.error("/error");
          df.reject(jqhxr, status, error);
        });
      }

      if (!serverPath) {
        loadFallbackData();
        return df.promise();
      }

      $.get(
        serverPath,
        function (data, status, jqxhr) {
          TEAM_DATA = mapServerTeamsToTeamData(data);
          df.resolve(TEAM_DATA);
        },
        "json",
      ).fail(function () {
        loadFallbackData();
      });

      return df.promise();
    }

    function refreshTeamData() {
      if (TEAM_POLL_IN_FLIGHT) {
        return;
      }

      TEAM_POLL_IN_FLIGHT = true;

      loadTeamData(true)
        .done(function () {
          setupTeams();
          setupLeaderboard();
        })
        .always(function () {
          TEAM_POLL_IN_FLIGHT = false;
        });
    }

    function startTeamPolling() {
      if (TEAM_POLL_TIMER) {
        clearInterval(TEAM_POLL_TIMER);
      }

      TEAM_POLL_TIMER = setInterval(function () {
        refreshTeamData();
      }, TEAM_POLL_INTERVAL_MS);
    }

    function loadActivityData() {
      if (ACTIVITY_DATA) {
        return $.Deferred().resolve(ACTIVITY_DATA).promise();
      }

      var df = $.Deferred();
      var uuid = getCurrentUUID();

      if (!uuid) {
        ACTIVITY_DATA = [];
        return df.resolve(ACTIVITY_DATA).promise();
      }

      $.get(
        "/" + encodeURIComponent(uuid) + "/json/activity",
        function (data) {
          var activityEntries = $.isArray(data) ? data.slice(0) : [];

          activityEntries.sort(function (a, b) {
            var dateA = new Date(a && a.CreatedAt ? a.CreatedAt : 0).getTime();
            var dateB = new Date(b && b.CreatedAt ? b.CreatedAt : 0).getTime();

            return dateB - dateA;
          });

          ACTIVITY_DATA = activityEntries;
          df.resolve(ACTIVITY_DATA);
        },
        "json",
      ).fail(function () {
        ACTIVITY_DATA = [];
        df.resolve(ACTIVITY_DATA);
      });

      return df.promise();
    }

    /**
     * get the game data, which is stored as json
     *
     * @return Deferred
     *   - indicate that this jqxhr request is all done
     */
    function getCountryData(forceRefresh) {
      if (!forceRefresh && COUNTRY_DATA) {
        return $.Deferred().resolve(COUNTRY_DATA).promise();
      }

      var uuid = getCurrentUUID();
      if (!uuid) {
        COUNTRY_DATA = {};
        return $.Deferred().resolve(COUNTRY_DATA).promise();
      }

      var loadPath = "/" + encodeURIComponent(uuid) + "/json/countries";

      return $.get(
        loadPath,
        function (data, status, jqxhr) {
          COUNTRY_DATA = data;
        },
        "json",
      ).fail(function (jqxhr, status, error) {
        console.error("There was a problem retrieving the game data.");
        console.log(loadPath);
        console.log(status);
        console.log(error);
        if (!COUNTRY_DATA) {
          COUNTRY_DATA = {};
        }
      });
    }

    function refreshCountryData() {
      if (COUNTRY_POLL_IN_FLIGHT) {
        return;
      }

      COUNTRY_POLL_IN_FLIGHT = true;

      getCountryData(true)
        .done(function () {
          renderCountryData();
        })
        .always(function () {
          COUNTRY_POLL_IN_FLIGHT = false;
        });
    }

    function startCountryPolling() {
      if (COUNTRY_POLL_TIMER) {
        clearInterval(COUNTRY_POLL_TIMER);
      }

      COUNTRY_POLL_TIMER = setInterval(function () {
        refreshCountryData();
      }, COUNTRY_POLL_INTERVAL_MS);
    }

    function formatCountryLabel(country) {
      var data = COUNTRY_DATA && COUNTRY_DATA[country] ? COUNTRY_DATA[country] : null;
      var flagEmoji = data && data.flag_emoji ? String(data.flag_emoji).trim() : "";
      return flagEmoji ? country + " " + flagEmoji : country;
    }

    /**
     * since a lot of the data is in an external file, go through
     *  that data and markup the svg so we can use in
     */
    function renderLiveListView() {
      if (!$listview || !$listview.length) {
        return;
      }

      var tbody = $("tbody", $listview).first();
      if (!tbody.length) {
        return;
      }

      tbody.empty();

      Object.keys(COUNTRY_DATA)
        .sort(function (a, b) {
          return String(a).localeCompare(String(b));
        })
        .forEach(function (countryName) {
          var data = COUNTRY_DATA[countryName];
          if (!data) {
            return;
          }

          var isActive = !!data.active;
          var $row = $("<tr></tr>").attr("data-country", countryName);
          var $name = $("<td></td>").text(formatCountryLabel(countryName));
          var $points = $("<td></td>").text(isActive ? (data.points || 0) + " Pts" : "");
          var $category = $("<td></td>").text(isActive ? data.category || "" : "");
          var $status = $("<td></td>");

          $row.attr("data-active", isActive ? "true" : "false");
          $row.toggleClass("country-disabled", !isActive);

          if (isActive) {
            $status.append('<span class="mctf-status status--open">Open</span>');
          } else {
            $status.text("Unavailable");
          }

          $row.append($name, $points, $category, $status);
          tbody.append($row);
        });

      listviewEventListeners($listview);
    }

    function renderCountryData() {
      if (!COUNTRY_DATA) {
        return;
      }

      renderFilterOptions();

      var previousActiveState = LAST_COUNTRY_ACTIVE_STATE || {};
      var nextActiveState = {};
      var newlyActiveCountries = [];

      $(".countries .land", $mapSvg).each(function () {
        var $countryPath = $(this),
          $group = $countryPath.closest("g"),
          country = $countryPath.attr("title"),
          data = COUNTRY_DATA[country];

        nextActiveState[country] = !!(data && data.active);

        if (data && data.active) {
          $countryPath.addClass("active");
          $group.removeClass("country-disabled");
          $group.attr("data-category", data.category);
          $group.attr("data-points", data.points);
        } else {
          $countryPath.removeClass("active");
          $group.removeAttr("data-category");
          $group.removeAttr("data-points");
          $group.addClass("country-disabled");
        }
      });

      $.each(nextActiveState, function (country, isActive) {
        if (isActive && !previousActiveState[country]) {
          newlyActiveCountries.push(country);
        }
      });

      LAST_COUNTRY_ACTIVE_STATE = nextActiveState;

      if ($listview && $listview.length > 0) {
        renderLiveListView();

        $('tr[data-country]', $listview).each(function () {
          var $row = $(this),
            country = $row.data("country"),
            data = COUNTRY_DATA[country];

          if (data) {
            if (data.active) {
              $row.attr("data-category", data.category);
              $row.attr("data-points", data.points);
              $("td:nth-child(2)", $row).text(data.points + " Pts");
              $("td:nth-child(3)", $row).text(data.category);
            } else {
              $row.removeAttr("data-category");
              $row.removeAttr("data-points");
              $("td:nth-child(2)", $row).text("");
              $("td:nth-child(3)", $row).text("");
            }
          }
        });
      }

      reapplyActiveCountryFilter();

      $.each(newlyActiveCountries, function (_, country) {
        animateCountryActivation(country);
      });
    }

    function animateCountryActivation(country) {
      if (!$mapSvg || !$mapSvg.length || !$map || !$map.length) {
        return;
      }

      var $countryGroup = $('.countries .land[title="' + country + '"]', $mapSvg).closest("g");
      if (!$countryGroup.length) {
        return;
      }

      var groupNode = $countryGroup.get(0);
      if (!groupNode || typeof groupNode.getBoundingClientRect !== "function") {
        return;
      }

      var groupRect = groupNode.getBoundingClientRect();
      if (!groupRect.width && !groupRect.height) {
        return;
      }

      var mapRect = $map.get(0).getBoundingClientRect();
      var centerX = groupRect.left + groupRect.width / 2 - mapRect.left;
      var centerY = groupRect.top + groupRect.height / 2 - mapRect.top;

      $countryGroup.removeClass("country-just-activated");
      void groupNode.offsetWidth;
      $countryGroup.addClass("country-just-activated");

      setTimeout(function () {
        $countryGroup.removeClass("country-just-activated");
      }, 1400);

      var $wave = $('<div class="country-activation-wave"></div>');
      $wave.css({
        left: centerX + "px",
        top: centerY + "px",
      });

      $map.append($wave);

      setTimeout(function () {
        $wave.remove();
      }, 1500);
    }

    function slugifyFilterValue(value) {
      return String(value)
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
    }

    function getUniqueFilterValues(key) {
      if (!COUNTRY_DATA) {
        return [];
      }

      var values = $.map(COUNTRY_DATA, function (data) {
        if (!data || data[key] === undefined || data[key] === null || data[key] === "") {
          return null;
        }

        return data[key];
      });

      values = $.grep(values, function (value, index) {
        return values.indexOf(value) === index;
      });

      values.sort(function (a, b) {
        if (key === "points") {
          return Number(a) - Number(b);
        }

        return String(a).localeCompare(String(b));
      });

      return values;
    }

    function renderFilterOptions() {
      if (!COUNTRY_DATA) {
        return;
      }

      var selectedValues = {
        "mctf--module--filter--category": ACTIVE_COUNTRY_FILTER && ACTIVE_COUNTRY_FILTER.name === "mctf--module--filter--category" ? ACTIVE_COUNTRY_FILTER.value : ($('input[name="mctf--module--filter--category"]:checked').val() || "All"),
        "mctf--module--filter--point-value": ACTIVE_COUNTRY_FILTER && ACTIVE_COUNTRY_FILTER.name === "mctf--module--filter--point-value" ? ACTIVE_COUNTRY_FILTER.value : ($('input[name="mctf--module--filter--point-value"]:checked').val() || "All"),
      };

      var filterConfigs = [
        {
          type: "category",
          name: "mctf--module--filter--category",
          values: getUniqueFilterValues("category"),
          label: function (value) {
            return value;
          },
        },
        {
          type: "point-value",
          name: "mctf--module--filter--point-value",
          values: getUniqueFilterValues("points"),
          label: function (value) {
            return value + " Pts";
          },
        },
      ];

      $.each(filterConfigs, function (_, config) {
        $('[data-filter-options="' + config.type + '"]').each(function (index) {
          var $list = $(this);

          $list.empty();

          $.each(config.values, function (_, value) {
            var slug = slugifyFilterValue(value),
              optionId = config.name + "--" + slug + "--" + index,
              label = config.label(value),
              checked = selectedValues[config.name] === String(value) ? ' checked=""' : "";

            $list.append(
              '<li><input type="radio" name="' +
                config.name +
                '" value="' +
                value +
                '" id="' +
                optionId +
                '"' +
                checked +
                ' /><label for="' +
                optionId +
                '" class="click-effect"><span>' +
                label +
                "</span></label></li>",
            );
          });

          var allChecked = selectedValues[config.name] === "All" || $.inArray(selectedValues[config.name], $.map(config.values, function (value) {
            return String(value);
          })) === -1 ? ' checked=""' : "";

          $list.append(
            '<li><input type="radio" name="' +
              config.name +
              '" value="All" id="' +
              config.name +
              '--all--' +
              index +
              '"' +
              allChecked +
              ' /><label for="' +
              config.name +
              '--all--' +
              index +
              '" class="click-effect"><span>All</span></label></li>',
          );
        });
      });
    }

    function applyCountryFilter(filterName, value) {
      var $svgCountries = $(".countries > g", $mapSvg),
        attribute = filterName === "mctf--module--filter--point-value" ? "data-points" : "data-category";

      $svgCountries.each(function () {
        var countryGroup = d3.select(this),
          attributeValue = countryGroup.attr(attribute),
          matches = value === "All" || attributeValue === String(value);

        countryGroup.classed("inactive", !matches);
        countryGroup.classed("highlighted", value !== "All" && matches);
      });

      $("tr[data-country]", $listview).each(function () {
        var $row = $(this),
          attributeValue = String($row.attr(attribute) || ""),
          matches = value === "All" || attributeValue === String(value);

        $row.removeClass("inactive highlighted");

        if (value !== "All") {
          $row.addClass(matches ? "highlighted" : "inactive");
        }
      });
    }

    function reapplyActiveCountryFilter() {
      if (!ACTIVE_COUNTRY_FILTER || !ACTIVE_COUNTRY_FILTER.name) {
        return;
      }

      var $selected = $('input[name="' + ACTIVE_COUNTRY_FILTER.name + '"][value="' + ACTIVE_COUNTRY_FILTER.value + '"]');
      if (!$selected.length) {
        ACTIVE_COUNTRY_FILTER.value = "All";
        $selected = $('input[name="' + ACTIVE_COUNTRY_FILTER.name + '"][value="All"]');
      }

      if ($selected.length) {
        $selected.prop("checked", true);
      }

      applyCountryFilter(ACTIVE_COUNTRY_FILTER.name, ACTIVE_COUNTRY_FILTER.value);
    }

    /* --------------------------------------------
     * --list view
     * -------------------------------------------- */

    /**
     * toggle the list view on/off
     */
    function toggleListView(enabled) {
      var activeClass = "listview-enabled",
        toggle = enabled === undefined ? !LIST_VIEW : enabled ? true : false;

      // the containers (for moving the modules around)
      (($containerLeft = $(".mctf-module-container.container--column.column-left")),
        ($containerRight = $(".mctf-module-container.container--column.column-right")),
        ($containerRow = $(".mctf-module-container.container--row")),
        // the modules
        ($module_activity = $('aside[data-module="activity"]')),
        ($module_leaderboard = $('aside[data-module="leaderboard"]')),
        ($module_domination = $('aside[data-module="world-domination"]')));

      if (toggle) {
        $gameboard.addClass(activeClass);
        LIST_VIEW = true;
        $module_activity.prependTo($containerRow).addClass("module--outer-left");
        $module_domination.appendTo($containerLeft);
        $module_leaderboard.prependTo($containerRight);
      } else {
        $gameboard.removeClass(activeClass);
        LIST_VIEW = false;
        $module_leaderboard.appendTo($containerLeft);
        $module_activity.appendTo($containerLeft).removeClass("module--outer-left");
        $module_domination.prependTo($containerRow);
      }
    }

    /**
     * the event listeners for the list view mode of the gameboard
     *
     * @param $listview (jquery object)
     *   - the listview
     */
    function listviewEventListeners($listview) {
      //
      // click on the row, engage the "capture_country"
      //  modal, except for in a couple situations
      //
      $("tr", $listview).on("click", function (event) {
        event.preventDefault();
        var $tr = $(this).closest("tr"),
          country = $tr.data("country");

        //
        // try to capture the country if:
        //   - the country is not using help
        //   - the country is NOT captured
        //
        if (!$tr.hasClass("help-enabled") && !$tr.hasClass("country-disabled") && $tr.data("captured") === undefined) {
          captureCountry(country);
        }
      });

      //
      // the incoming help status
      //
      $("tr .status--incoming-help", $listview).on("click", function (event) {
        event.preventDefault();
        event.stopPropagation();
        var country = $(this).closest("tr").data("country");

        MAP_CTF.modal.loadPopup("country-help", function () {
          $("#mctf-modal .add-new-help-chat").data("country", country);
          $("#mctf-modal .country-name").text(formatCountryLabel(country));
        });
      });

      //
      // offer to give help to someone who needs it
      //
      $("tr .status--give-help", $listview).on("click", function (event) {
        event.preventDefault();
        event.stopPropagation();
        var country = $(this).closest("tr").data("country");
        MAP_CTF.modal.loadPopup("country-help-opponent", function () {
          $("#mctf-modal .add-new-help-chat").data("country", country);
        });
      });

      //
      // click on the timer to launch the individual chat
      //
      $("tr .status--timer", $listview).on("click", function (event) {
        event.preventDefault();
        event.stopPropagation();
        var country = $(this).closest("tr").data("country");
        $('.alerts li[data-help="' + country + '"] .js-expand-individual-chat').trigger("click");
      });
    }

    /* --------------------------------------------
     * --init
     * -------------------------------------------- */

    /**
     * init the gameboard
     */
    function init() {
      // init the jquery object variables
      $gameboard = $("#mctf-gameboard");

      VIEW_ONLY = $body.data("section") === "viewer-mode";

      if (GAMEBOARD_LOADED === false) {
        build();
      }
    }

    /**
     * utility function to check if the game is currently in
     *  view-only mode
     */
    function isViewMode() {
      return VIEW_ONLY;
    }

    return {
      init: init,
      data: getCountryData,
      captureCountry: captureCountry,
      resetCapture: removeCaptured,
      toggleListView: toggleListView,

      // enable the zoomable stuff from console
      enableClickAndDrag: enableClickAndDrag,

      // check to see if we're in view mode (for external use)
      isViewMode: isViewMode,
    };
  })(); // gameboard

  /**
   * --modal
   */
  MAP_CTF.modal = (function () {
    var LOAD_EXT = ".html",
      ACTIVE_CLASS = "visible",
      POPUP_CLASSES = "mctf-modal-wrapper modal--popup",
      DEFAULT_CLASSES = "mctf-modal-wrapper modal--default",
      MODAL_DIR = "/static/inc/modals/",
      $modalContainer,
      $modal,
      $countryHover;

    /**
     * initialize the modal, including grabbing the modal div and
     *  setting up event listeners
     */
    function init() {
      $modal = $("#mctf-modal");
      $modalContainer = $("#mctf-initkit");
      if ($modalContainer.length === 0) {
        $modalContainer = $("body");
      }
      $countryHover = $("#mctf-country-popup");

      //
      // trigger the launch of a modal
      //
      $body.on("click", ".js-launch-modal", function (event) {
        event.preventDefault();
        var modal = $(this).data("modal"),
          cb;

        //
        // if we're launching the login modal, add the active
        //  class to the nav item
        //
        if (modal === "login") {
          $(".mctf-main-nav a").removeClass("active");
          $(this).addClass("active");
        }

        load(modal, cb);
      });

      //
      // close the modal
      //
      $body.on("click", ".js-close-modal", close);

      // Esc should close the active modal via its own close flow.
      $(window).on("keyup", function (event) {
        var key = event.which || 0,
          eventKey = event.key || "";
        if (key === 27 || eventKey === "Escape") {
          closeActive(event);
        }
      });
    }

    /**
     * close the modal
     *
     * @param event (object)
     *   - if this function call comes from an event listener,
     *      prevent the default action
     */
    function close(event) {
      if (event) event.preventDefault();

      $('div[id^="mctf-modal"]').removeClass(ACTIVE_CLASS);

      //
      // @NOTICE
      // this is here to re-enable the active state on the nav
      //  in case it was altered when the modal was launched.
      //
      if (typeof _initkit !== "undefined") {
        _initkit.enableNavActiveState();
      }
    }

    /**
     * close only the active modal, preferring its own close button
     * so modal-specific cleanup handlers run.
     */
    function closeActive(event) {
      if (event) event.preventDefault();

      var $activeModal = $('div[id^="mctf-modal"].' + ACTIVE_CLASS).last();
      if ($activeModal.length > 0) {
        if ($activeModal.hasClass("modal--country-capture") && MAP_CTF.gameboard && typeof MAP_CTF.gameboard.resetCapture === "function") {
          MAP_CTF.gameboard.resetCapture();
        }

        var $closeBtn = $(".js-close-modal", $activeModal).first();
        if ($closeBtn.length > 0) {
          $closeBtn.trigger("click");
          return;
        }
      }

      close();
    }

    /**
     * close the poup
     *
     * @param event (object)
     *   - if this function call comes from an event listener,
     *      prevent the default action
     */
    function closeHoverPopup(event) {
      if (event) event.preventDefault();

      $countryHover.removeClass(ACTIVE_CLASS);
    }

    /**
     * call MAP_CTF.loadComponent for the modal content
     *
     * @param $modal (jquery object)
     *   - the modal jquery object to load the content into
     *
     * @param loadPath (string)
     *   - the path to the modal file
     *
     * @param cb (function)
     *   - the callback
     */
    function openAndLoad($modal, loadPath, cb) {
      MAP_CTF.loadComponent($modal, loadPath, function () {
        if (typeof cb === "function") {
          cb();
        }
        $modal.addClass(ACTIVE_CLASS);
      });
    }

    /* --------------------------------------------
     * --the modal rendering
     * -------------------------------------------- */

    /**
     * there are two types of modals - default and popup. The
     *  default modal takes up a full page, while the poup modal
     *  creates a popup box for content. Both of these wrapper
     *  functions take the same parameters.
     *
     * @param modalName (string)
     *   - the filename of the modal content you're looking to
     *      load up, without the trailing .html
     *
     * @param cb (function)
     *   - a callback function for after the modal content loads
     */
    function loadPopup(modalName, cb) {
      _load(modalName, POPUP_CLASSES, MODAL_DIR, cb);
    }
    function load(modalName, cb) {
      _load(modalName, DEFAULT_CLASSES, MODAL_DIR, cb);
    }

    /**
     * load and create
     *
     * @param modalName (string)
     *   - the filename of the modal content you're looking to
     *      load up
     *
     * @param modalClasses (string)
     *   - the classes for the modal
     *
     * @param loadDir (string)
     *   - the location in the filesystem where we're looking for
     *      the file
     *
     * @param cb (function)
     *   - a callback function for after the modal content loads
     *
     */
    function _load(modalName, modalClasses, loadDir, cb) {
      var loadPath = loadDir + modalName + LOAD_EXT;

      closeHoverPopup();

      modalClasses += " modal--" + modalName;

      if ($modal.length === 0) {
        $modal = $('<div id="mctf-modal" class="' + modalClasses + '" />').appendTo($modalContainer);
      } else {
        $modal.removeAttr("class").addClass(modalClasses);
      }

      openAndLoad($modal, loadPath, cb);
    }

    /**
     * create a persistent modal. This is used to build a modal
     *  that is very specific, and should be loaded as quickly
     *  as possible after initially loaded, like the command line
     *  modal.
     *
     * @param modalName (string)
     *   - the name of the module being loaded
     *
     * @param cb (function)
     *   - callback funtion for after the persistent modal is loaded
     */
    function loadPersistent(modalName, cb) {
      var loadPath = MODAL_DIR + modalName + LOAD_EXT,
        modalId = "mctf-modal-persistent--" + modalName,
        $modal = $(modalId);

      if ($modal.length === 0) {
        $modal = $('<div id="' + modalId + '" class="' + POPUP_CLASSES + '" />').appendTo($modalContainer);
      }

      MAP_CTF.loadComponent($modal, loadPath, cb);
    }

    /**
     * open the persistent modal
     *
     * @param modalName (string)
     *   - the name of the modal to open
     */
    function openPersistent(modalName) {
      var modalId = "#mctf-modal-persistent--" + modalName;

      $(modalId).addClass(ACTIVE_CLASS);
    }

    /**
     * a specific function for rendering the country hover popup
     *
     * @param cb (function)
     *   - a callback to render the country data in the popup
     */
    function countryHoverPopup(cb) {
      var loadPath = "/static/inc/modals/country-popup.html";
      if ($countryHover.length === 0) {
        $countryHover = $('<div id="mctf-country-popup" class="mctf-popup-content popup--hover mctf-section-border" />').appendTo($modalContainer);
      }

      openAndLoad($countryHover, loadPath, cb);
    }

    /**
     * the code for the popup in the view-only mode
     *
     * @param cb (function)
     *   - a callback to render the country data in the popup
     */
    function viewmodePopup(cb) {
      var loadPath = "/static/inc/modals/country-popup--viewmode.html";
      if ($countryHover.length === 0) {
        $countryHover = $('<div id="mctf-country-popup" class="mctf-popup-content popup--view-only" />').appendTo($modalContainer);
      }

      openAndLoad($countryHover, loadPath, cb);
    }

    return {
      init: init,

      // loads the basic modal
      load: load,

      // loads a persistent modal
      loadPersistent: loadPersistent,

      // open a persistent modal
      openPersistent: openPersistent,

      // load a popup modal
      loadPopup: loadPopup,

      // load and show the popup modal for a country hover
      countryHoverPopup: countryHoverPopup,

      // load and show the view only country info
      viewmodePopup: viewmodePopup,

      // close the popup modal for a country hover
      closeHoverPopup: closeHoverPopup,

      // close the regular modal
      close: close,
      closeActive: closeActive,
    };
  })(); // modal

  /**
   * --graphics
   * build graphics (like the scorecard) using d3
   */
  MAP_CTF.graphics = (function () {
    /**
     * set up event listeners
     */
    function init() {
      $(".mctf-graphic").each(function () {
        var $graphic = $(this),
          datafile = $graphic.data("file");

        if ($graphic.hasClass("initialized")) {
          return;
        }

        $graphic.addClass("initialized");

        build(this, datafile);
      });

      //
      // scoreboard filter
      //
      $('input[name="mctf-scoreboard-filter"]').on("change", function (event) {
        event.preventDefault();
        var team = $(this).val(),
          $teamLine = $('.scoreboard-graphic-container .team-score-line[data-team="' + team + '"]');

        if (this.checked) {
          $teamLine.show();
        } else {
          $teamLine.hide();
        }
      });
    }

    /**
     * build the graph
     *
     * @param svgEl (object)
     *   - the svg element to attach the graphic to
     *
     * @param datafile (string)
     *   - the file to load, which contains the data
     */
    function build(svgEl, datafile) {
      if (datafile === undefined) {
        return;
      }

      var $container = $(svgEl).closest(".scoreboard-graphic-container");

      $.get(
        datafile,
        function (data, status, jqxhr) {
          var scores = data;

          var minMaxArray = [];

          $.each(scores, function () {
            $.merge(minMaxArray, this.values);
          });

          var graphic = d3.select(svgEl),
            MARGIN = { left: 60, right: 20, bottom: 40 },
            WIDTH = $container.length > 0 ? $container.width() - MARGIN.left - MARGIN.right : 820 - MARGIN.left - MARGIN.right,
            HEIGHT = 220 - MARGIN.bottom,
            X_START = 1,
            X_LENGTH = 10,
            xRange = d3.scaleLinear().range([0, WIDTH]).domain([X_START, X_LENGTH]),
            yRange = d3
              .scaleLinear()
              .range([HEIGHT - 0, 0])
              .domain([
                d3.min(minMaxArray, function (d) {
                  return d.score;
                }),
                d3.max(minMaxArray, function (d) {
                  return d.score + 10;
                }),
              ]),
            xAxis = d3.axisBottom(xRange).ticks(X_LENGTH);

          yAxis = d3.axisLeft(yRange).ticks(6);

          graphic
            .append("svg:g")
            .attr("class", "x axis")
            .attr("transform", "translate(" + MARGIN.left + "," + HEIGHT + ")")
            .call(xAxis)
            .selectAll("line")
            .attr("transform", "translate(0, -6)");
          graphic
            .append("svg:g")
            .attr("class", "y axis")
            .attr("transform", "translate(" + MARGIN.left + ",0)")
            .call(yAxis)
            .selectAll("line")
            .attr("transform", "translate(6,0)");

          // Add the text label for the Y axis
          graphic
            .append("text")
            .attr("transform", "rotate(-90)")
            .attr("y", 0)
            .attr("x", 0 - HEIGHT / 2)
            .attr("dy", "1em")
            .attr("stroke", "#fff")
            .style("text-anchor", "middle")
            .text("Score");

          var lineFunc = d3
            .line()
            .x(function (d) {
              return xRange(d.time) + MARGIN.left;
            })
            .y(function (d) {
              return yRange(d.score);
            })
            .curve(d3.curveLinear);

          graphic.append("svg:rect").attr("width", WIDTH).attr("height", HEIGHT).attr("x", MARGIN.left).attr("fill", "#142e35");

          var graphLine = graphic.append("svg:g").attr("class", "mouseline").attr("opacity", "0");

          graphLine
            .append("svg:path")
            .attr("stroke", COLOR_LIGHT_BLUE)
            .attr("stroke-width", 2)
            .attr("d", "M0,0L0," + HEIGHT);

          graphLine.append("circle").attr("cx", 0).attr("cy", 5).attr("r", 5).attr("stroke", COLOR_LIGHT_BLUE).attr("fill", "black").attr("stroke-width", 2);

          scores.forEach(function (d, i) {
            graphic.append("svg:path").attr("d", lineFunc(d.values)).attr("class", "team-score-line").attr("stroke", d.color).attr("data-team", d.team).attr("stroke-width", 2).attr("fill", "none");
          });

          graphic
            .append("svg:rect")
            .attr("width", WIDTH)
            .attr("height", HEIGHT)
            .attr("x", MARGIN.left)
            .attr("fill", "none")
            .attr("pointer-events", "all")
            .on("mouseout", function () {
              d3.select(".mouseline").attr("opacity", "0");
            })
            .on("mouseover", function () {
              d3.select(".mouseline").attr("opacity", "1");
            })
            .on("mousemove", function (event) {
              var xCoor = d3.pointer(event, this)[0];
              d3.select(".mouseline").attr("transform", "translate(" + xCoor + ",0)");
            });
        },
        "json",
      ).fail(function (jqxhr, status, error) {
        console.error("There was a problem retrieving the game scores.");
        console.log(status);
        console.log(error);
        console.error("/error");
      });
    }

    return {
      init: init,
      build: build,
    };
  })(); // graphics

  /**
   * --command line
   */
  MAP_CTF.command_line = (function () {
    var loadPath = "/static/data/command-line.json",
      modalName = "command-line",
      $cmdPromptList,
      $cmdResultsList,
      COMMAND_DATA;

    /**
     * event listeners for the command line
     *  - the keyup on the window to trigger the command line
     *  - the keyup on the input prompt
     */
    function eventListeners() {
      var $promptInput = $("#command-prompt--input"),
        $filterResultsInput = $("#command-prompt--filter-results"),
        // since the modal is fading in, we need to delay the
        //  focus so that it actually focuses when the modal
        //  appears
        animDelay = 400,
        numCommands = $("li", $cmdPromptList).length;

      //
      // get the command line up
      //
      $(window).on("keyup", function (event) {
        var key = event.which,
          eventKey = event.key,
          eventCode = event.code,
          isSlash = key === 191 || eventKey === "/" || eventCode === "Slash";

        // if the forward slash has been pressed
        if (isSlash) {
          event.preventDefault();

          MAP_CTF.modal.close();
          MAP_CTF.modal.closeHoverPopup();
          MAP_CTF.modal.openPersistent(modalName);
          if ($("li", $cmdPromptList).length % 2 === 0) {
            $cmdPromptList.addClass("offset");
          }
          setTimeout(function () {
            $promptInput.focus();
          }, animDelay);
        }
        // esc closes the command prompt
        else if (key === 27) {
          clearCommandPrompt();
          MAP_CTF.modal.closeActive();
        }
      }); // window.on('keyup')

      //
      // type in the command line prompt
      //
      $promptInput.on("keyup", function (event) {
        event.preventDefault();

        var $self = $(this),
          key = event.which,
          cmd = $self.val(),
          $autocomplete = $self.siblings(".autocomplete");

        // if the "enter" key has been pressed
        if (key === 13) {
          var autocompleteCmd = $autocomplete.text(),
            selectedCmd = COMMAND_DATA.commands[cmd];

          if (selectedCmd === undefined) {
            selectedCmd = COMMAND_DATA.commands[autocompleteCmd];

            if (selectedCmd) {
              $promptInput.val(autocompleteCmd);
            }
          }

          if (selectedCmd) {
            chooseCommand(selectedCmd);
            $promptInput.trigger("blur");
            $filterResultsInput.trigger("focus");
          } else {
            $cmdResultsList.append("<li>Invalid command</li>");
          }
        }
        // up arrow goes up in results box
        else if (key === 38) {
          var $active = $cmdPromptList.find("li.selected");

          if ($active.prevAll(":not(.hidden)").length > 0) {
            $active = $active.removeClass("selected").prevAll(":not(.hidden)").eq(0).addClass("selected");
          }

          $promptInput.val($active.text());
          $autocomplete.empty();
          checkSelectedVisible();
        }
        // down arrow goes down in results box
        else if (key === 40) {
          var $active = $cmdPromptList.find("li.selected");

          if ($active.nextAll(":not(.hidden)").length > 0) {
            $active = $active.removeClass("selected").nextAll(":not(.hidden)").eq(0).addClass("selected");
          }

          if ($active.length === 0) {
            $active = $cmdPromptList.find("li:not(.hidden)").eq(0).addClass("selected");
          }

          $promptInput.val($active.text());
          $autocomplete.empty();
          checkSelectedVisible();
        }
        // the user is actually typing
        else {
          $("li", $cmdPromptList)
            .removeClass("hidden selected")
            .filter(function () {
              var text = $(this).text();
              return text.indexOf(cmd) !== 0;
            })
            .addClass("hidden");

          if (cmd !== "") {
            var first = $cmdPromptList.find("li:not(.hidden)").eq(0).text();
            $autocomplete.text(first);
          } else {
            $autocomplete.empty();
          }

          $cmdResultsList.empty();
        }
      }); // $promptInput.on('keyup')

      //
      // filter the results from the selected command
      //
      $filterResultsInput.on("keyup", function (event) {
        event.preventDefault();

        var $self = $(this),
          key = event.which,
          search = $self.val(),
          $autocomplete = $self.siblings(".autocomplete"),
          $selected = $cmdResultsList.find("li.selected");

        if (key === 13) {
          $("body").trigger("command-option-selected", {
            selected: $selected.text(),
          });
        }
        // up arrow goes up in results box
        else if (key === 38) {
          var $active = $cmdResultsList.find("li.selected");

          if ($active.prevAll(":not(.hidden)").length > 0) {
            $active = $active.removeClass("selected").prevAll(":not(.hidden)").eq(0).addClass("selected");
          }

          $filterResultsInput.val($active.text());
          $autocomplete.empty();
          checkSelectedVisible();
        }
        // down arrow goes down in results box
        else if (key === 40) {
          var $active = $cmdResultsList.find("li.selected");

          if ($active.nextAll(":not(.hidden)").length > 0) {
            $active = $active.removeClass("selected").nextAll(":not(.hidden)").eq(0).addClass("selected");
          }

          if ($active.length === 0) {
            $active = $cmdResultsList.find("li:not(.hidden)").eq(0).addClass("selected");
          }

          $filterResultsInput.val($active.text());
          $autocomplete.empty();
          checkSelectedVisible();
        }
        // else if the use is actually typing
        else {
          var $selected = $cmdResultsList
            .find("li")
            .removeClass("hidden selected")
            .filter(function () {
              var val = $(this).text().toLowerCase();
              return val.indexOf(search.toLowerCase()) !== 0;
            })
            .addClass("hidden");

          if ($selected.hasClass("hidden")) {
            $selected.removeClass("selected");

            if ($selected.nextAll(":not(.hidden)").length > 0) {
              $selected.nextAll(":not(.hidden)").eq(0).addClass("selected");
            } else if ($selected.prevAll(":not(.hidden)").length > 0) {
              $selected.prevAll(":not(.hidden)").eq(0).addClass("selected");
            }
          }

          if (search !== "") {
            var first = $cmdResultsList.find("li:not(.hidden)").eq(0).text();

            if (search.charAt(0) === search.charAt(0).toLowerCase()) {
              first = first.toLowerCase();
            }

            $autocomplete.text(first);
          } else {
            $autocomplete.empty();
          }

          if ($selected.length === 0) {
            $cmdResultsList.find("li").eq(0).addClass("selected");
          }
        }
      });
    } // event listeners

    /**
     * clear all the command prompt stuff
     */
    function clearCommandPrompt() {
      var $promptInput = $("#command-prompt--input"),
        $filterResultsInput = $("#command-prompt--filter-results");

      $(".mctf-command-line .autocomplete").empty();
      $promptInput.val("");
      $filterResultsInput.val("");
      $cmdResultsList.empty();
      $cmdPromptList.find("li").removeClass("hidden selected");
    }

    /**
     * check to see if the selected option is visible in
     *  the container
     */
    function checkSelectedVisible() {
      var $selected = $cmdResultsList.find("li.selected");

      if ($selected.length === 0) {
        return;
      }

      var resultsListScroll = $cmdResultsList.scrollTop(),
        resultListHeight = $cmdResultsList.height(),
        selectedPos = $selected.position();

      if (selectedPos.top > resultListHeight) {
        $cmdResultsList.animate({
          scrollTop: resultsListScroll + selectedPos.top + "px",
        });
      } else if (selectedPos.top < 0) {
        $cmdResultsList.animate({
          scrollTop: resultsListScroll + selectedPos.top + "px",
        });
      }
    }

    /**
     * a command has been chosen. Now do stuff, depending on the
     *  command and the data in the command
     *
     * @param cmdData (object)
     *   - an object with data for the selected command
     */
    function chooseCommand(cmdData) {
      var results = cmdData.results,
        cmdFunction = cmdData.function;

      if (results) {
        if (typeof results === "string") {
          var list = COMMAND_DATA.results_library[results];

          if (list) {
            $.each(list, function (index, listItem) {
              $cmdResultsList.append("<li>" + listItem + "</li>");
            });
          }
        } else {
          $.each(results, function (index, listItem) {
            $cmdResultsList.append("<li>" + listItem + "</li>");
          });
        }

        $cmdResultsList.find("li:first-child").addClass("selected");
      }

      if (cmdFunction) {
        var funcName = cmdFunction.name,
          param = cmdFunction.param;

        switch (funcName) {
          case "change-radio":
            cmd_changeRadio(param);
            break;
          case "capture-country":
            cmd_captureCountry();
            break;
          case "close-module":
            cmd_closeModule();
            break;
          case "open-module":
            cmd_openModule();
            break;
          case "open-listview":
            cmd_toggleListView();
            break;
          default:
            console.error("That command's associated function is undefined.");
            break;
        }
      }
    }

    /* --------------------------------------------
     * --cmd functions
     *
     * functions for the commands
     * -------------------------------------------- */

    /**
     * change a radio button selection
     *
     * @param inputName (string)
     *   - the input name we're looking to change
     */
    function cmd_changeRadio(inputName) {
      $("body").on("command-option-selected", function (event, data) {
        if (inputName === "mctf--module--filter--category") {
          $('aside[data-module="filter"]').addClass("active");
          $body.trigger("module-changestate");
        }

        $('input[name="' + inputName + '"]').prop("checked", false);

        $('input[name="' + inputName + '"][value="' + data.selected + '"]')
          .trigger("change")
          .prop("checked", true);

        MAP_CTF.modal.close();
        clearCommandPrompt();

        $("body").off("command-option-selected");
      });
    }

    /**
     * capture a country
     */
    function cmd_captureCountry() {
      $("body").on("command-option-selected", function (event, data) {
        var country = data.selected;

        MAP_CTF.modal.close();
        clearCommandPrompt();

        MAP_CTF.gameboard.captureCountry(country);

        $("body").off("command-option-selected");
      });
    }

    /**
     * close a module
     */
    function cmd_closeModule() {
      $("body").on("command-option-selected", function (event, data) {
        var module = data.selected;

        if (module === "All") {
          $("aside").removeClass("active");
        } else {
          $('aside[data-name="' + module + '"]').removeClass("active");
        }
        $body.trigger("module-changestate");

        MAP_CTF.modal.close();
        clearCommandPrompt();

        $("body").off("command-option-selected");
      });
    }

    /**
     * open a module
     */
    function cmd_openModule() {
      $("body").on("command-option-selected", function (event, data) {
        var module = data.selected;

        if (module === "All") {
          $("aside").addClass("active");
        } else {
          $('aside[data-name="' + module + '"]').addClass("active");
        }
        $body.trigger("module-changestate");

        MAP_CTF.modal.close();
        clearCommandPrompt();

        $("body").off("command-option-selected");
      });
    }

    /**
     * toggle the list view
     */
    function cmd_toggleListView() {
      $("body").on("command-option-selected", function (event, data) {
        var enable = data.selected === "On" ? true : false;

        MAP_CTF.gameboard.toggleListView(enable);

        MAP_CTF.modal.close();
        clearCommandPrompt();

        $("body").off("command-option-selected");
      });
    }

    /* --------------------------------------------
     * --init
     * -------------------------------------------- */

    /**
     * init the command line functionality
     */
    function init() {
      MAP_CTF.modal.loadPersistent(modalName, function () {
        $.get(
          loadPath,
          function (data, status, jqxhr) {
            $cmdPromptList = $(".mctf-command-line .command-list ul");
            $cmdResultsList = $(".mctf-command-line .command-results ul");

            COMMAND_DATA = data;

            if (COMMAND_DATA && COMMAND_DATA.commands) {
              $.each(COMMAND_DATA.commands, function (command, cmdData) {
                $cmdPromptList.append("<li>" + command + "</li>");
              });

              eventListeners();
            }
          },
          "json",
        ).fail(function () {
          console.error("There was a problem retrieving the commands.");
          console.error("/error");
        });
      });
    }

    return {
      init: init,
    };
  })(); // command line

  /**
   * --admin
   */
  MAP_CTF.admin = (function () {
    var PLAYERS_PER_TEAM = 1;

    /**
     * check the admin forms for errors
     *
     * @param $clicked (jquery object)
     *   - the clicked element. From this, we'll find the form
     *      elements we're looking to validate
     *
     * @return Boolean
     *   - whether or not the form is valud
     */
    function validateAdminForm($clicked) {
      var valid = true,
        $validateForm = $clicked.closest(".validate-form");
      (($required = $(".form-el--required", $validateForm)), (errorClass = "form-error"));

      if ($validateForm.length === 0) {
        $validateForm = $clicked.closest(".mctf-admin-main");
      }

      $(".error-msg", $validateForm).remove();

      $required.removeClass(errorClass).each(function () {
        var $self = $(this),
          $requiredEl = $('input[type="text"], input[type="password"]', $self),
          $logoName = $(".logo-name", $self);

        //
        // all the conditions that would make this element
        //  trigger an error
        //
        if ($requiredEl.val() === "" || ($logoName.length > 0 && $logoName.text() === "")) {
          $self.addClass(errorClass);
          valid = false;

          if ($(".error-msg", $validateForm).length === 0) {
            $(".admin-box-header h3", $validateForm).after('<span class="error-msg">Please fix the errors in red</span>');
          }

          return;
        }
      });

      return valid;
    }

    /**
     * add a new section
     *
     * @param $clicked (jquery object)
     *   - the clicked button
     */
    function addNewSection($clicked) {
      var $sectionContainer = $clicked.closest(".admin-buttons").siblings(".admin-sections"),
        $lastSection = $(".admin-box", $sectionContainer).last(),
        $newSection = $lastSection.clone(),
        // +1 for the 0-based index, +1 for the new section
        //  being added
        sectionIndex = $lastSection.index() + 2;

      //
      // update some stuff in the cloned section
      //
      var $title = $(".admin-box-header h3", $newSection),
        titleText = $title.text().toLowerCase(),
        switchName = $('input[type="radio"]', $newSection).first().attr("name");

      if (switchName) {
        newSwitchName = switchName.substr(0, switchName.lastIndexOf("--")) + "--" + sectionIndex;

        $("#" + switchName + "--on", $newSection).attr("id", newSwitchName + "--on");
        $('label[for="' + switchName + '--on"]', $newSection).attr("for", newSwitchName + "--on");
        $("#" + switchName + "--off", $newSection).attr("id", newSwitchName + "--off");
        $('label[for="' + switchName + '--off"]', $newSection).attr("for", newSwitchName + "--off");
        $('input[type="radio"]', $newSection).attr("name", newSwitchName);
      }

      $newSection.removeClass("section-locked");
      $(".emblem-carousel .emblem-item.active", $newSection).removeClass("active");
      $(".form-error", $newSection).removeClass("form-error");
      $(".post-avatar, .logo-name", $newSection).removeClass("has-avatar").empty();
      $(".error-msg", $newSection).remove();
      $('input[type="text"], input[type="password"]', $newSection).prop("disabled", false);

      $(".dk-select", $newSection).remove();

      $("select", $newSection).dropkick();

      if (titleText.indexOf("team") > -1) {
        $title.text("Team " + sectionIndex);
      } else if (titleText.indexOf("quiz level") > -1) {
        $title.text("Quiz Level " + sectionIndex);
      } else if (titleText.indexOf("base level") > -1) {
        $title.text("Base Level " + sectionIndex);
      } else if (titleText.indexOf("flag level") > -1) {
        $title.text("Flag Level " + sectionIndex);
      } else if (titleText.indexOf("player") > -1) {
        $title.text("Player " + sectionIndex);
      }

      $('input[type="text"], input[type="password"]', $newSection).val("");

      $sectionContainer.append($newSection);
    }

    /**
     * render the registration page, updating text and values
     *  based on the number of players that have been set
     */
    function renderRegistrationPage() {
      var $sections = $("#mctf-initkit .admin-sections");

      if (PLAYERS_PER_TEAM > 1) {
        var $playerList = $(".player-list"),
          $playerInfo = $("li", $playerList);
        $(".admin-box-header h3", $sections).text("Team 1");
        $sections.addClass("team-registration");

        for (var i = 2; i <= PLAYERS_PER_TEAM; i++) {
          var $newRow = $playerInfo.clone();
          $(".player-list--label", $newRow).text("Player " + i + " Name");

          $playerList.append($newRow);
        }
      }
    }

    /* --------------------------------------------
     * --init
     * -------------------------------------------- */

    /**
     * init the admin stuff
     */
    function init() {
      $body.off("content-loaded").on("content-loaded", function (event, data) {
        if (data && data.page && data.page === "registration") {
          renderRegistrationPage();
        }
      });

      //
      // actionable buttons
      //
      $(".mctf-admin-main")
        .off("click")
        .on("click", "[data-action]", function (event) {
          event.preventDefault();
          var $self = $(this),
            $section = $self.closest(".admin-box"),
            action = $self.data("action"),
            actionModal = $self.data("actionModal"),
            lockClass = "section-locked",
            sectionTitle = $self.closest("#mctf-initkit").find(".admin-page-header h3").text().replace(" ", "_");

          //
          // route the actions
          //
          if (action === "save") {
            var valid = validateAdminForm($self);

            if (actionModal && valid === false) {
              actionModal = "error";
            }

            if (valid) {
              $section.addClass(lockClass);
              $('input[type="text"], input[type="password"]', $section).prop("disabled", true);
            }
          } else if (action === "add-new") {
            addNewSection($self);
          } else if (action === "edit") {
            $section.removeClass(lockClass);
            $('input[type="text"], input[type="password"]', $section).prop("disabled", false);
          } else if (action === "delete") {
            $section.remove();

            // rename the section boxes
            $(".admin-box").each(function (i, el) {
              var $titleObj = $(".admin-box-header h3", el),
                title = $titleObj.text(),
                newTitle = title.substring(0, title.lastIndexOf(" ") + 1) + (i + 1);

              $titleObj.text(newTitle);
            });
          }

          //
          // if there's a modal
          //
          if (actionModal) {
            MAP_CTF.modal.loadPopup("action-" + actionModal, function () {
              $("#mctf-modal .admin-section-name").text(sectionTitle);
            });
          }
        });

      //
      // modal actionable
      //
      $body.on("click", ".js-confirm-save", function (event) {
        var $status = $(".admin-section--status .highlighted");
        $status.text("Saved");

        setTimeout(function () {
          $status.fadeOut(function () {
            $status.text("").removeAttr("style");
          });
        }, 5000);
      });

      //
      // select a logo
      //
      $body.on("click", ".js-choose-logo", function (event) {
        event.preventDefault();

        var $self = $(this),
          $container = $self.closest(".mctf-column-container");

        MAP_CTF.modal.loadPopup("choose-logo", function () {
          var $modal = $("#mctf-modal");

          MAP_CTF.buildEmblemCarousel($(".emblem-carousel", $modal));

          $(".js-store-logo", $modal).on("click", function (event) {
            event.preventDefault();
            var $active = $(".emblem-item.active", $modal),
              logoName = ($active.data("logo") || "").toString(),
              logo = '<svg class="icon icon--badge"><use xlink:href="#icon--badge-' + logoName + '"></use></svg>';

            $(".post-avatar", $container).addClass("has-avatar").html(logo);
            $(".logo-name", $container).text(logoName);
          });
        });
      });

      //
      // change the players per team
      //
      $("#mctf-admin--players-per-team").on("change", function (event) {
        event.preventDefault();
        var val = $(this).val();

        PLAYERS_PER_TEAM = val;
      });

      //
      // prompt logout
      //
      $(".js-prompt-logout").on("click", function (event) {
        event.preventDefault();
        MAP_CTF.modal.loadPopup("action-logout");
      });
    }

    return {
      init: init,
    };
  })(); // admin

  /* --------------------------------------------
   * --public
   * -------------------------------------------- */

  MAP_CTF.init = function () {
    //
    // set up global variables
    //

    $body = $("body");

    //
    // load the svg sprite. This is in the MAP_CTF namespace
    //  rather than the initkit as this is the recommended
    //  method of loading the sprite through a purely front-end
    //  solution. This can be removed if the sprite is included
    //  via some server-side solution.
    //
    MAP_CTF.loadComponent("#mctf-svg-sprite", "/static/svg/icons/icons.svg");

    // load the modal
    MAP_CTF.modal.init();

    //
    // any modules that does stuff based on loaded content (for
    //  example, modals or svg grahics) should get fired when the
    //  "content-loaded" event gets fired. The modules that load
    //  content should trigger this event when the content is
    //  done loading
    //
    $body
      .on("content-loaded", function (event) {
        // load the graphics
        MAP_CTF.graphics.init();
      })
      .trigger("content-loaded");

    /* --------------------------------------------
     * --more generic stuff
     * -------------------------------------------- */

    //
    // dropkick - for select form elements
    //
    $("select").dropkick();

    /* --------------------------------------------
     * --global event listeners
     * -------------------------------------------- */

    //
    // radio tabs
    //
    $body.on("change", ".radio-tabs input[type='radio']", function (event) {
      var $tabs = $(this).closest(".radio-tabs"),
        $tabContent = $tabs.parent().next(".module-scrollable, .tab-content-container").find(".tab-content-container").addBack(".tab-content-container").eq(0);

      if ($tabContent.length > 0 && this.value) {
        event.preventDefault();
        $('.radio-tab-content[data-tab="' + this.value + '"]', $tabContent).onlySiblingWithClass("active");
      }
    });

    //
    // trending list filtering
    //
    var filteredCategories = [];
    $('.trending-list input[type="checkbox"]').on("change", function (event) {
      event.preventDefault();
      var $posts = $(".post-list--main .mctf-post:not(.pinned-post)").removeClass("hidden"),
        isChecked = this.checked,
        category = $(this).val();

      if (isChecked) {
        filteredCategories.push(category);
      } else {
        var index = filteredCategories.indexOf(category);
        filteredCategories.splice(index, 1);
      }

      if (filteredCategories.length === 0) {
        return;
      }

      $posts
        .filter(function () {
          var categories = $(this).data("categories"),
            hasCat = true;

          $.each(filteredCategories, function (i, cat) {
            if (categories.indexOf(cat) > -1) {
              hasCat = false;
              return;
            }
          });

          return hasCat;
        })
        .addClass("hidden");
    });

    //
    // click events
    //
    $body.on("click", ".click-effect", function () {
      var $self = $(this).addClass("clicked");

      $self.find("span").on("animationend", function () {
        $self.removeClass("clicked");

        $self.off("animationend");
      });
    });

    //
    // read more posts
    //
    $(".post-readmore").on("click", function (event) {
      event.preventDefault();
      var $self = $(this),
        $content = $self.closest(".post-content").toggleClass("show-full");

      if ($content.hasClass("show-full")) {
        $self.text("Close Post");
      } else {
        $self.text("Read More");
      }
    });

    //
    // rules table of contents
    //
    $(".rules--table-of-contents li a").on("click", function (event) {
      event.preventDefault();

      var $rules = $(".mctf-rules section"),
        index = $(this).parent().index(),
        offset = $(".page--rules .mctf-section-header").innerHeight(),
        $selectedRule = $rules.eq(index),
        rulesScrollTop = $selectedRule.position().top + offset;

      $(".page--rules").animate({
        scrollTop: rulesScrollTop + "px",
      });
    });

    //
    // choose a logo
    //
    $body.on("click", ".emblem-carousel .emblem-item", function (event) {
      event.preventDefault();
      $(this).onlySiblingWithClass("active");
    });
    $body.on("keydown", ".emblem-carousel .emblem-item", function (event) {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        $(this).onlySiblingWithClass("active");
      }
    });
  };

  /**
   * Build emblem picker from loaded SVG sprite symbols.
   */
  MAP_CTF.buildEmblemCarousel = function (target, selectedLogo) {
    var $target = typeof target === "object" ? target : $(target);
    if ($target.length === 0) {
      return;
    }

    var badges = [];
    $("#mctf-svg-sprite symbol[id^='icon--badge-']").each(function () {
      var symbolID = this.id || "";
      var logoName = symbolID.replace("icon--badge-", "");
      if (logoName) {
        badges.push(logoName);
      }
    });
    badges.sort();

    if (badges.length === 0) {
      $target.empty();
      return;
    }

    var activeLogo = selectedLogo && badges.indexOf(selectedLogo) >= 0 ? selectedLogo : badges[0];
    var html = '<ul class="emblem-grid" aria-label="Emblem options">';
    for (var i = 0; i < badges.length; i++) {
      var logo = badges[i];
      var activeClass = logo === activeLogo ? " active" : "";
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
    html += "</ul>";
    $target.html(html);
  };

  /**
   * load a component into a target on the site
   *
   * @param target (string)
   *   - selector to select where to load the content
   *
   * @param component (string)
   *   - the name of the component to load
   *
   * @param cb (function)
   *   - callback function for when the load is successful
   */
  MAP_CTF.loadComponent = function (target, component, cb) {
    var $target = typeof target === "object" ? target : $(target);

    $target.load(component, function (response, status, jqxhr) {
      if (status === "error") {
        console.error("There was a problem loading the component:");
        console.log("target: " + target);
        console.log("component: " + component);
        console.error("/end error");
      } else {
        //
        // fire the "content-loaded" event to initialize any
        //  dynamic content that is in the loaded content
        //
        $("body").trigger("content-loaded", { component: component });

        if (typeof cb === "function") {
          cb();
        }
      }
    });
  };

  /**
   * Bootstrap section-specific modules when initkit is not in use.
   */
  function bootstrapWithoutInitkit() {
    var section = ($("body").data("section") || "").toString();

    MAP_CTF.init();

    if (section === "gameboard" || section === "viewer-mode") {
      MAP_CTF.gameboard.init();
    } else if (section === "admin") {
      MAP_CTF.admin.init();
    }
  }

  function normalizePath(path) {
    var normalized = (path || "").toString().split("?")[0].split("#")[0];
    if (normalized.length > 1 && normalized.charAt(normalized.length - 1) === "/") {
      normalized = normalized.slice(0, -1);
    }
    return normalized;
  }

  function updateMainNavActiveLink() {
    var currentPath = normalizePath(window.location.pathname);
    var $links = $("#mctf-main-nav a[data-active]");

    $links.removeClass("active");

    $links.each(function () {
      var href = $(this).attr("href");
      if (!href || href.charAt(0) === "#") {
        return;
      }
      var linkPath = normalizePath(href);
      if (linkPath === currentPath) {
        $(this).addClass("active");
      }
    });
  }

  /**
   * set up stuff on document ready
   */
  $(document).ready(function () {
    updateMainNavActiveLink();

    //
    // check to make sure that we're not using the initkit, and
    //  then initialize the Capture the Flag scripts. This
    //  prevents the init() function from running multiple times
    //  if we are using the initkit.
    //
    if (typeof _initkit === "undefined") {
      bootstrapWithoutInitkit();
    }
  });
})((window.MAP_CTF = window.MAP_CTF || {}), jQuery);
