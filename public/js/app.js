(function () {
    "use strict";

    var state = {
        gameMode: "author",
        period: "all"
    };

    var cache = {};

    var els = {
        body: document.getElementById("leaderboard-body"),
        loading: document.getElementById("loading"),
        error: document.getElementById("error"),
        table: document.getElementById("leaderboard"),
        empty: document.getElementById("empty-state"),
        periodToggle: document.getElementById("period-toggle"),
        modeToggle: document.getElementById("game-mode-toggle")
    };

    function formatScore(ms) {
        var totalSeconds = Math.floor(ms / 1000);
        var minutes = Math.floor(totalSeconds / 60);
        var seconds = totalSeconds % 60;
        var millis = ms % 1000;

        if (minutes > 0) {
            return minutes + ":" + pad(seconds) + "." + pad3(millis);
        }
        return seconds + "." + pad3(millis);
    }

    function pad(n) {
        return n < 10 ? "0" + n : "" + n;
    }

    function pad3(n) {
        if (n < 10) return "00" + n;
        if (n < 100) return "0" + n;
        return "" + n;
    }

    function formatDate(iso) {
        var d = new Date(iso);
        return d.toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric"
        });
    }

    function showLoading() {
        els.loading.style.display = "flex";
        els.table.style.display = "none";
        els.empty.style.display = "none";
        els.error.style.display = "none";
    }

    function showError(msg) {
        els.loading.style.display = "none";
        els.error.style.display = "block";
        els.error.textContent = msg;
    }

    function fetchLeaderboard() {
        var key = state.gameMode + ":" + state.period;

        if (cache[key]) {
            render(cache[key]);
            return;
        }

        showLoading();

        var params = new URLSearchParams();
        params.set("game_mode", state.gameMode);
        params.set("period", state.period);

        fetch("api/leaderboard?" + params.toString())
            .then(function (res) {
                if (!res.ok) {
                    throw new Error("request");
                }
                return res.json();
            })
            .then(function (data) {
                cache[key] = data;
                render(data);
            })
            .catch(function (err) {
                if (err.message === "request") {
                    showError("Failed to load leaderboard. Please try again later.");
                } else {
                    showError("Something went wrong while reading the response.");
                }
            });
    }

    function render(data) {
        els.loading.style.display = "none";
        els.error.style.display = "none";

        var scores = data.scores || [];

        if (scores.length === 0) {
            els.table.style.display = "none";
            els.empty.style.display = "flex";
            return;
        }

        els.table.style.display = "table";
        els.empty.style.display = "none";

        els.body.innerHTML = "";

        for (var i = 0; i < scores.length; i++) {
            var s = scores[i];
            var tr = document.createElement("tr");

            tr.innerHTML =
                '<td class="col-rank">' + escapeHtml(String(s.rank)) + "</td>" +
                '<td class="col-player"><a href="https://trackmania.io/#/player/' + encodeURIComponent(s.player.openplanet_id) + '" target="_blank" rel="noopener">' + escapeHtml(s.player.display_name) + "</a></td>" +
                '<td class="col-maps">' + escapeHtml(String(s.maps_completed)) + "</td>" +
                '<td class="col-skipped">' + (new Date(s.created_at) < new Date("2026-02-01") ? "" : escapeHtml(String(s.maps_skipped))) + "</td>" +
                '<td class="col-score">' + escapeHtml(formatScore(s.score)) + "</td>" +
                '<td class="col-date">' + escapeHtml(formatDate(s.created_at)) + "</td>";

            els.body.appendChild(tr);
        }
    }

    function escapeHtml(str) {
        var div = document.createElement("div");
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // Theme toggle
    var themeToggle = document.getElementById("theme-toggle");
    var themeCycle = ["dark", "light", "system"];

    function applyTheme(setting) {
        var resolved = setting;
        if (setting === "system") {
            resolved = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
        }
        document.documentElement.setAttribute("data-theme", resolved);

        var icons = themeToggle.querySelectorAll(".theme-icon");
        for (var i = 0; i < icons.length; i++) {
            icons[i].classList.toggle("active", icons[i].getAttribute("data-theme") === setting);
        }

        var labels = { system: "System theme", light: "Light mode", dark: "Dark mode" };
        themeToggle.title = labels[setting];
    }

    themeToggle.addEventListener("click", function () {
        var current = localStorage.getItem("theme") || "dark";
        var next = themeCycle[(themeCycle.indexOf(current) + 1) % themeCycle.length];
        localStorage.setItem("theme", next);
        applyTheme(next);
    });

    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () {
        if ((localStorage.getItem("theme") || "dark") === "system") {
            applyTheme("system");
        }
    });

    applyTheme(localStorage.getItem("theme") || "dark");

    // Event listeners
    els.modeToggle.addEventListener("click", function (e) {
        if (e.target.classList.contains("toggle-btn") && !e.target.classList.contains("active")) {
            var buttons = els.modeToggle.querySelectorAll(".toggle-btn");
            for (var i = 0; i < buttons.length; i++) {
                buttons[i].classList.remove("active");
            }
            e.target.classList.add("active");
            state.gameMode = e.target.getAttribute("data-value");
            fetchLeaderboard();
        }
    });

    els.periodToggle.addEventListener("click", function (e) {
        if (e.target.classList.contains("toggle-btn") && !e.target.classList.contains("active")) {
            var buttons = els.periodToggle.querySelectorAll(".toggle-btn");
            for (var i = 0; i < buttons.length; i++) {
                buttons[i].classList.remove("active");
            }
            e.target.classList.add("active");
            state.period = e.target.getAttribute("data-value");
            fetchLeaderboard();
        }
    });

    // Initial load
    fetchLeaderboard();
})();
