(function () {
    "use strict";

    // month: "" = all time, "current" = this month, "YYYY-MM" = specific archive month
    var state = {
        gameMode: "author",
        month: ""
    };

    var cache = {};
    var fetchGen = 0;

    var els = {
        body: document.getElementById("leaderboard-body"),
        loading: document.getElementById("loading"),
        error: document.getElementById("error"),
        table: document.getElementById("leaderboard"),
        empty: document.getElementById("empty-state"),
        periodToggle: document.getElementById("period-toggle"),
        modeToggle: document.getElementById("game-mode-toggle"),
        archiveBtn: document.getElementById("archive-btn"),
        archiveDropdown: document.getElementById("archive-dropdown")
    };

    function formatScore(ms) {
        var totalSeconds = Math.floor(ms / 1000);
        var minutes = Math.floor(totalSeconds / 60);
        var seconds = totalSeconds % 60;
        var hundredths = Math.floor((ms % 1000) / 10);

        if (minutes > 0) {
            return minutes + ":" + pad(seconds) + "." + pad(hundredths);
        }
        return seconds + "." + pad(hundredths);
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
        els.table.style.display = "none";
        els.error.style.display = "block";
        els.error.textContent = msg;
    }

    function getCurrentMonth() {
        var now = new Date();
        var y = now.getUTCFullYear();
        var m = now.getUTCMonth() + 1;
        return y + "-" + (m < 10 ? "0" + m : "" + m);
    }

    var MONTH_NAMES = ["Jan", "Feb", "Mar", "Apr", "May", "Jun",
                       "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

    function formatMonthLabel(y, m) {
        return MONTH_NAMES[m - 1] + " " + y;
    }

    function generateArchiveMonths() {
        var months = [];
        var now = new Date();
        var curY = now.getUTCFullYear();
        var curM = now.getUTCMonth() + 1;
        // Go back from previous month to Dec 2025
        var y = curY;
        var m = curM - 1;
        if (m === 0) { m = 12; y--; }
        while (y > 2025 || (y === 2025 && m >= 12)) {
            var key = y + "-" + (m < 10 ? "0" + m : "" + m);
            months.push({ key: key, label: formatMonthLabel(y, m) });
            m--;
            if (m === 0) { m = 12; y--; }
        }
        return months;
    }

    function populateArchiveDropdown() {
        var months = generateArchiveMonths();
        els.archiveDropdown.innerHTML = "";
        for (var i = 0; i < months.length; i++) {
            var btn = document.createElement("button");
            btn.setAttribute("data-month", months[i].key);
            btn.textContent = months[i].label;
            els.archiveDropdown.appendChild(btn);
        }
    }

    function resolveMonth() {
        if (state.month === "current") return getCurrentMonth();
        return state.month;
    }

    function closeArchiveDropdown() {
        els.archiveDropdown.classList.remove("open");
        els.archiveBtn.classList.remove("open");
    }

    function updateArchiveSelection() {
        var buttons = els.archiveDropdown.querySelectorAll("button");
        var resolved = resolveMonth();
        for (var i = 0; i < buttons.length; i++) {
            buttons[i].classList.toggle("selected", buttons[i].getAttribute("data-month") === resolved);
        }
    }

    function fetchLeaderboard() {
        var resolved = resolveMonth();
        var key = state.gameMode + ":" + resolved;

        if (cache[key]) {
            render(cache[key]);
            return;
        }

        showLoading();

        var gen = ++fetchGen;
        var params = new URLSearchParams();
        params.set("game_mode", state.gameMode);
        if (resolved) {
            params.set("month", resolved);
        }

        fetch("api/leaderboard?" + params.toString())
            .then(function (res) {
                if (!res.ok) {
                    throw new Error("request");
                }
                return res.json();
            })
            .then(function (data) {
                if (gen !== fetchGen) return;
                cache[key] = data;
                render(data);
            })
            .catch(function (err) {
                if (gen !== fetchGen) return;
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

        var statsCutoff = new Date("2026-02-01");
        for (var i = 0; i < scores.length; i++) {
            var s = scores[i];
            var tr = document.createElement("tr");
            var hasStats = new Date(s.created_at) >= statsCutoff;

            tr.innerHTML =
                '<td class="col-rank">' + escapeHtml(String(s.rank)) + "</td>" +
                '<td class="col-player"><a href="https://trackmania.io/#/player/' + encodeURIComponent(s.player.openplanet_id) + '" target="_blank" rel="noopener">' + escapeHtml(s.player.display_name) + "</a></td>" +
                '<td class="col-maps">' + (hasStats ? escapeHtml(String(s.maps_completed)) : "") + "</td>" +
                '<td class="col-skipped">' + (hasStats ? escapeHtml(String(s.maps_skipped)) : "") + "</td>" +
                '<td class="col-score">' + escapeHtml(formatScore(s.score)) + "</td>" +
                '<td class="col-date">' + escapeHtml(formatDate(s.created_at)) + "</td>";

            els.body.appendChild(tr);
        }
    }

    var escapeEl = document.createElement("div");
    function escapeHtml(str) {
        escapeEl.textContent = str;
        return escapeEl.innerHTML;
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

    function setActiveToggle(value) {
        var buttons = els.periodToggle.querySelectorAll(".toggle-btn");
        for (var i = 0; i < buttons.length; i++) {
            buttons[i].classList.toggle("active", buttons[i].getAttribute("data-value") === value);
        }
    }

    function resetArchiveLabel() {
        els.archiveBtn.querySelector(".archive-label").textContent = "Archive";
    }

    // --- Hash routing ---
    function pushHash() {
        var h = state.gameMode;
        if (state.month) {
            h += "/" + state.month;
        }
        history.replaceState(null, "", "#" + h);
    }

    function syncUI() {
        // Game mode toggles
        var modeBtns = els.modeToggle.querySelectorAll(".toggle-btn");
        for (var i = 0; i < modeBtns.length; i++) {
            modeBtns[i].classList.toggle("active", modeBtns[i].getAttribute("data-value") === state.gameMode);
        }

        // Period toggles + archive label
        if (state.month === "") {
            setActiveToggle("all");
            resetArchiveLabel();
        } else if (state.month === "current") {
            setActiveToggle("month");
            resetArchiveLabel();
        } else {
            setActiveToggle("archive");
            var parts = state.month.split("-");
            els.archiveBtn.querySelector(".archive-label").textContent = formatMonthLabel(parseInt(parts[0], 10), parseInt(parts[1], 10));
        }
    }

    function applyHash() {
        var hash = location.hash.replace(/^#/, "");
        if (!hash) {
            state.gameMode = "author";
            state.month = "";
        } else {
            var segments = hash.split("/");
            var mode = segments[0];
            if (mode === "author" || mode === "gold") {
                state.gameMode = mode;
            } else {
                state.gameMode = "author";
            }
            state.month = segments[1] || "";
        }
        syncUI();
        closeArchiveDropdown();
        fetchLeaderboard();
    }

    // Event listeners
    els.modeToggle.addEventListener("click", function (e) {
        if (e.target.classList.contains("toggle-btn") && !e.target.classList.contains("active")) {
            state.gameMode = e.target.getAttribute("data-value");
            closeArchiveDropdown();
            pushHash();
            syncUI();
            fetchLeaderboard();
        }
    });

    els.periodToggle.addEventListener("click", function (e) {
        var btn = e.target.closest(".toggle-btn");
        if (!btn) return;

        var value = btn.getAttribute("data-value");

        if (value === "archive") {
            els.archiveDropdown.classList.toggle("open");
            els.archiveBtn.classList.toggle("open");
            updateArchiveSelection();
            return;
        }

        closeArchiveDropdown();

        if (btn.classList.contains("active")) return;

        if (value === "all") {
            state.month = "";
        } else if (value === "month") {
            state.month = "current";
        }

        pushHash();
        syncUI();
        fetchLeaderboard();
    });

    els.archiveDropdown.addEventListener("click", function (e) {
        var btn = e.target.closest("button");
        if (!btn) return;

        state.month = btn.getAttribute("data-month");
        closeArchiveDropdown();
        pushHash();
        syncUI();
        updateArchiveSelection();
        fetchLeaderboard();
    });

    document.addEventListener("click", function (e) {
        if (!els.archiveDropdown.contains(e.target) && e.target !== els.archiveBtn) {
            closeArchiveDropdown();
        }
    });

    window.addEventListener("hashchange", applyHash);

    // Init
    populateArchiveDropdown();
    applyHash();
})();
