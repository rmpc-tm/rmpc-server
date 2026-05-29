(function () {
    "use strict";

    // month: "" = all time, "current" = this month, "YYYY-MM" = archive, "hof" = hall of fame
    var state = {
        gameMode: "author",
        month: ""
    };

    var cache = {};
    var fetchGen = 0;
    var hofCache = {};
    var hofFetchGen = 0;

    var els = {
        body: document.getElementById("leaderboard-body"),
        loading: document.getElementById("loading"),
        error: document.getElementById("error"),
        table: document.getElementById("leaderboard"),
        tableWrap: document.getElementById("leaderboard-wrap"),
        empty: document.getElementById("empty-state"),
        periodToggle: document.getElementById("period-toggle"),
        modeToggle: document.getElementById("game-mode-toggle"),
        archiveBtn: document.getElementById("archive-btn"),
        archiveDropdown: document.getElementById("archive-dropdown"),
        hofBtn: document.getElementById("hof-btn"),
        hofWrap: document.getElementById("hof-wrap"),
        hofBody: document.getElementById("hof-body"),
        hofEmpty: document.getElementById("hof-empty"),
        hofDescription: document.getElementById("hof-description")
    };

    // --- Activity chart ---
    (function renderActivityChart() {
        var container = document.getElementById("activity-bars");
        if (!container) return;

        var maxHeight = 18;

        fetch("api/activity")
            .then(function (res) { return res.ok ? res.json() : null; })
            .then(function (data) {
                if (!data || !data.medals || data.medals.length === 0) return;

                var points = data.medals;

                // Create bars at 0 height
                var bars = [];
                for (var i = 0; i < points.length; i++) {
                    var bar = document.createElement("div");
                    bar.className = "activity-bar";
                    bar.style.height = "0px";
                    bar.style.opacity = "0";
                    container.appendChild(bar);
                    bars.push(bar);
                }

                var max = 1;
                for (var i = 0; i < points.length; i++) {
                    if (points[i] > max) max = points[i];
                }

                // Double rAF ensures the browser paints the 0-height state first
                requestAnimationFrame(function () { requestAnimationFrame(function () {
                    for (var i = 0; i < points.length; i++) {
                        var count = points[i];
                        var h = count === 0 ? 2 : Math.round((count / max) * maxHeight);
                        var opacity = count === 0 ? 0.04 : 0.06 + (count / max) * 0.14;
                        var duration = 0.4 + Math.random() * 0.6;

                        bars[i].style.transition = "height " + duration + "s ease-out, opacity " + duration + "s ease-out";
                        bars[i].style.height = h + "px";
                        bars[i].style.opacity = opacity;
                    }
                }); });
            })
            .catch(function () {});
    })();

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

    function hideAllViews() {
        els.tableWrap.style.display = "none";
        els.empty.style.display = "none";
        els.hofWrap.style.display = "none";
        els.hofEmpty.style.display = "none";
        els.error.style.display = "none";
    }

    function showLoading() {
        els.loading.querySelector("span").textContent =
            state.month === "hof" ? "Loading hall of fame..." : "Loading scores...";
        els.loading.style.display = "flex";
        hideAllViews();
    }

    function showError(msg) {
        els.loading.style.display = "none";
        hideAllViews();
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
        // Current month first, then previous months back to Dec 2025
        months.push({ key: "current", label: formatMonthLabel(curY, curM), current: true });
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
            if (months[i].current) {
                btn.innerHTML = escapeHtml(months[i].label) + ' <span class="month-tag">Current</span>';
            } else {
                btn.textContent = months[i].label;
            }
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

    function fetchData() {
        els.hofDescription.style.display = state.month === "hof" ? "" : "none";
        if (state.month === "hof") {
            fetchHallOfFame();
        } else {
            fetchLeaderboard();
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
        els.hofWrap.style.display = "none";
        els.hofEmpty.style.display = "none";

        var scores = data.scores || [];

        if (scores.length === 0) {
            els.tableWrap.style.display = "none";
            els.empty.style.display = "flex";
            return;
        }

        els.tableWrap.style.display = "";
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
                '<td class="col-date" title="' + escapeHtml(new Date(s.created_at).toLocaleString()) + '">' + escapeHtml(formatDate(s.created_at)) + "</td>";

            els.body.appendChild(tr);
        }
    }

    var escapeEl = document.createElement("div");
    function escapeHtml(str) {
        escapeEl.textContent = str;
        return escapeEl.innerHTML;
    }

    // --- Hall of Fame ---
    function fetchHallOfFame() {
        var mode = state.gameMode;
        if (hofCache[mode]) {
            renderHof(hofCache[mode]);
            return;
        }

        showLoading();

        var gen = ++hofFetchGen;
        fetch("api/halloffame?game_mode=" + encodeURIComponent(mode))
            .then(function (res) {
                if (!res.ok) {
                    throw new Error("request");
                }
                return res.json();
            })
            .then(function (data) {
                if (gen !== hofFetchGen) return;
                hofCache[mode] = data;
                renderHof(data);
            })
            .catch(function (err) {
                if (gen !== hofFetchGen) return;
                if (err.message === "request") {
                    showError("Failed to load hall of fame. Please try again later.");
                } else {
                    showError("Something went wrong while reading the response.");
                }
            });
    }

    function renderHof(data) {
        els.loading.style.display = "none";
        els.error.style.display = "none";
        els.tableWrap.style.display = "none";
        els.empty.style.display = "none";

        var entries = (data && data.entries) || [];
        els.hofBody.innerHTML = "";

        if (entries.length === 0) {
            els.hofWrap.style.display = "none";
            els.hofEmpty.style.display = "flex";
            return;
        }

        els.hofWrap.style.display = "";
        els.hofEmpty.style.display = "none";

        for (var i = 0; i < entries.length; i++) {
            var e = entries[i];
            var trophies =
                repeat("🥇", e.gold) +   // 🥇
                repeat("🥈", e.silver) + // 🥈
                repeat("🥉", e.bronze);  // 🥉
            var tr = document.createElement("tr");
            tr.innerHTML =
                '<td class="col-rank">' + escapeHtml(String(e.rank)) + "</td>" +
                '<td class="col-player"><a href="https://trackmania.io/#/player/' + encodeURIComponent(e.player.openplanet_id) + '" target="_blank" rel="noopener">' + escapeHtml(e.player.display_name) + "</a></td>" +
                '<td class="col-trophies">' + trophies + "</td>";
            els.hofBody.appendChild(tr);
        }
    }

    function repeat(s, n) {
        var out = "";
        for (var i = 0; i < n; i++) out += s;
        return out;
    }

    function setActiveToggle(value) {
        var buttons = els.periodToggle.querySelectorAll(".toggle-btn");
        for (var i = 0; i < buttons.length; i++) {
            buttons[i].classList.toggle("active", buttons[i].getAttribute("data-value") === value);
        }
        els.hofBtn.classList.toggle("active", value === "hof");
    }

    function resetArchiveLabel() {
        els.archiveBtn.querySelector(".archive-label").textContent = "Month";
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
        } else if (state.month === "hof") {
            setActiveToggle("hof");
            resetArchiveLabel();
        } else {
            setActiveToggle("archive");
            var monthKey = state.month === "current" ? getCurrentMonth() : state.month;
            var parts = monthKey.split("-");
            els.archiveBtn.querySelector(".archive-label").textContent = formatMonthLabel(parseInt(parts[0], 10), parseInt(parts[1], 10));
        }
    }

    function applyHash() {
        var hash = location.hash.replace(/^#/, "");
        closeArchiveDropdown();

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
        fetchData();
    }

    // Event listeners
    els.modeToggle.addEventListener("click", function (e) {
        if (e.target.classList.contains("toggle-btn") && !e.target.classList.contains("active")) {
            state.gameMode = e.target.getAttribute("data-value");
            closeArchiveDropdown();
            pushHash();
            syncUI();
            fetchData();
        }
    });

    function onPeriodClick(e) {
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
        } else if (value === "hof") {
            state.month = "hof";
        }

        pushHash();
        syncUI();
        fetchData();
    }

    els.periodToggle.addEventListener("click", onPeriodClick);
    els.hofBtn.addEventListener("click", onPeriodClick);

    els.archiveDropdown.addEventListener("click", function (e) {
        var btn = e.target.closest("button");
        if (!btn) return;

        state.month = btn.getAttribute("data-month");
        closeArchiveDropdown();
        pushHash();
        syncUI();
        updateArchiveSelection();
        fetchData();
    });

    document.addEventListener("click", function (e) {
        if (!els.archiveDropdown.contains(e.target) && !els.archiveBtn.contains(e.target)) {
            closeArchiveDropdown();
        }
    });

    window.addEventListener("hashchange", applyHash);

    // Init
    populateArchiveDropdown();
    applyHash();
})();
