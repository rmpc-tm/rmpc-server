(function () {
    "use strict";

    // view: "main" = leaderboard/HoF, "player" = single-player detail
    // month: "all" = all time, "current" = this month, "YYYY-MM" = archive, "hof" = hall of fame
    var state = {
        view: "main",
        gameMode: "author",
        month: "current",
        playerID: "",
        playerSig: ""
    };

    var cache = {};
    var hofCache = {};
    var playerCache = {};
    var fetchGen = 0;

    var els = {
        playerModal: document.getElementById("player-modal"),
        playerClose: document.getElementById("player-close"),
        body: document.getElementById("leaderboard-body"),
        loading: document.getElementById("loading"),
        error: document.getElementById("error"),
        table: document.getElementById("leaderboard"),
        tableWrap: document.getElementById("leaderboard-wrap"),
        showMore: document.getElementById("show-more"),
        showMoreLabel: document.getElementById("show-more-label"),
        empty: document.getElementById("empty-state"),
        periodToggle: document.getElementById("period-toggle"),
        modeToggle: document.getElementById("game-mode-toggle"),
        archiveBtn: document.getElementById("archive-btn"),
        archiveDropdown: document.getElementById("archive-dropdown"),
        hofBtn: document.getElementById("hof-btn"),
        hofWrap: document.getElementById("hof-wrap"),
        hofBody: document.getElementById("hof-body"),
        hofEmpty: document.getElementById("hof-empty"),
        hofDescription: document.getElementById("hof-description"),
        boardTitle: document.getElementById("board-title"),
        boardTitleText: document.getElementById("board-title-text"),
        boardTitleTag: document.getElementById("board-title-tag"),
        boardTitleMode: document.getElementById("board-title-mode"),
        playerLoading: document.getElementById("player-loading"),
        playerError: document.getElementById("player-error"),
        playerContent: document.getElementById("player-content"),
        playerName: document.getElementById("player-name"),
        playerSummary: document.getElementById("player-summary"),
        playerStatsAuthor: document.getElementById("player-stats-author"),
        playerStatsGold: document.getElementById("player-stats-gold"),
        playerBodyAuthor: document.getElementById("player-body-author"),
        playerBodyGold: document.getElementById("player-body-gold"),
        playerEmptyAuthor: document.getElementById("player-empty-author"),
        playerEmptyGold: document.getElementById("player-empty-gold")
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
        els.showMore.style.display = "none";
        els.empty.style.display = "none";
        els.hofWrap.style.display = "none";
        els.hofEmpty.style.display = "none";
        els.error.style.display = "none";
    }

    function openPlayerModal() {
        els.playerModal.style.display = "";
        document.body.classList.add("modal-open");
    }

    function closePlayerModal() {
        els.playerModal.style.display = "none";
        document.body.classList.remove("modal-open");
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
    var FULL_MONTH_NAMES = ["January", "February", "March", "April", "May", "June",
                            "July", "August", "September", "October", "November", "December"];

    function formatMonthLabel(y, m) {
        return MONTH_NAMES[m - 1] + " " + y;
    }

    // "2026-01" -> "January 2026"; unparsable keys pass through.
    function formatFullMonth(key) {
        var parts = String(key).split("-");
        var m = parseInt(parts[1], 10);
        if (!parts[0] || !m || m < 1 || m > 12) return String(key);
        return FULL_MONTH_NAMES[m - 1] + " " + parts[0];
    }

    function generateArchiveMonths() {
        var months = [];
        var now = new Date();
        var curY = now.getUTCFullYear();
        var curM = now.getUTCMonth() + 1;
        // Current month first, then previous months back to Dec 2025
        months.push({ key: "current", label: formatMonthLabel(curY, curM) });
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
        if (state.month === "all") return "";
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
        if (state.view === "player") {
            openPlayerModal();
            fetchPlayer();
            return;
        }
        closePlayerModal();
        els.hofDescription.style.display = state.month === "hof" ? "" : "none";
        els.boardTitle.style.display = state.month === "hof" ? "none" : "";
        if (state.month === "hof") {
            fetchHallOfFame();
        } else {
            fetchLeaderboard();
        }
    }

    // Shared fetch + cache + stale-response guard. `onLoading` / `onError` let
    // each endpoint choose where its loading and error UI lives.
    function fetchCached(opts) {
        var cached = opts.cacheStore[opts.cacheKey];
        if (cached) {
            opts.onSuccess(cached);
            return;
        }
        opts.onLoading();
        var gen = ++fetchGen;
        fetch(opts.url)
            .then(function (res) {
                if (res.status === 404) throw new Error("notfound");
                if (!res.ok) throw new Error("request");
                return res.json();
            })
            .then(function (data) {
                if (gen !== fetchGen) return;
                opts.cacheStore[opts.cacheKey] = data;
                opts.onSuccess(data);
            })
            .catch(function (err) {
                if (gen !== fetchGen) return;
                opts.onError(err.message);
            });
    }

    function fetchLeaderboard() {
        var resolved = resolveMonth();
        var params = new URLSearchParams();
        params.set("game_mode", state.gameMode);
        if (resolved) params.set("month", resolved);

        fetchCached({
            url: "api/leaderboard?" + params.toString(),
            cacheStore: cache,
            cacheKey: state.gameMode + ":" + resolved,
            onLoading: showLoading,
            onSuccess: render,
            onError: function (kind) {
                showError(kind === "request"
                    ? "Failed to load leaderboard. Please try again later."
                    : "Something went wrong while reading the response.");
            }
        });
    }

    // Rows are revealed PAGE_SIZE at a time; a tail shorter than MIN_TAIL is
    // revealed with the previous page instead of leaving a near-empty click.
    var PAGE_SIZE = 50;
    var MIN_TAIL = 15;
    var STATS_CUTOFF = new Date("2026-02-01");

    var shownScores = [];
    var shownCount = 0;

    function render(data) {
        els.loading.style.display = "none";
        els.error.style.display = "none";
        els.hofWrap.style.display = "none";
        els.hofEmpty.style.display = "none";

        var scores = data.scores || [];

        if (scores.length === 0) {
            els.tableWrap.style.display = "none";
            els.showMore.style.display = "none";
            els.empty.style.display = "flex";
            shownScores = [];
            shownCount = 0;
            return;
        }

        els.tableWrap.style.display = "";
        els.empty.style.display = "none";

        els.body.innerHTML = "";
        shownScores = scores;
        shownCount = 0;
        revealMore();
    }

    function revealMore() {
        var total = shownScores.length;
        var target = shownCount + PAGE_SIZE;
        if (total - target < MIN_TAIL) target = total;

        for (var i = shownCount; i < target; i++) {
            els.body.appendChild(scoreRow(shownScores[i]));
        }
        shownCount = target;

        var remaining = total - shownCount;
        if (remaining <= 0) {
            els.showMore.style.display = "none";
        } else {
            els.showMore.style.display = "";
            els.showMoreLabel.textContent = "Show more (" + remaining + ")";
        }
    }

    function scoreRow(s) {
        var tr = document.createElement("tr");
        var hasStats = new Date(s.created_at) >= STATS_CUTOFF;

        tr.innerHTML =
            '<td class="col-rank">' + s.rank + "</td>" +
            '<td class="col-player">' + playerLink(s.player) + "</td>" +
            '<td class="col-maps">' + (hasStats ? s.maps_completed : "") + "</td>" +
            '<td class="col-skipped">' + (hasStats ? s.maps_skipped : "") + "</td>" +
            '<td class="col-score">' + formatScore(s.score) + "</td>" +
            '<td class="col-date" title="' + escapeHtml(new Date(s.created_at).toLocaleString()) + '">' + formatDate(s.created_at) + "</td>";

        return tr;
    }

    var escapeEl = document.createElement("div");
    function escapeHtml(str) {
        escapeEl.textContent = str;
        return escapeEl.innerHTML;
    }

    function playerLink(p) {
        // p.t is the HMAC token. Without it the player page is unreachable,
        // so fall back to a plain span.
        var label = escapeHtml(p.display_name);
        if (!p || !p.t) return label;
        var href = "#player/" + encodeURIComponent(p.openplanet_id) + "/" + encodeURIComponent(p.t);
        return '<a href="' + href + '">' + label + "</a>";
    }

    // --- Hall of Fame ---
    function fetchHallOfFame() {
        var mode = state.gameMode;
        fetchCached({
            url: "api/halloffame?game_mode=" + encodeURIComponent(mode),
            cacheStore: hofCache,
            cacheKey: mode,
            onLoading: showLoading,
            onSuccess: renderHof,
            onError: function (kind) {
                showError(kind === "request"
                    ? "Failed to load hall of fame. Please try again later."
                    : "Something went wrong while reading the response.");
            }
        });
    }

    function renderHof(data) {
        els.loading.style.display = "none";
        els.error.style.display = "none";
        els.tableWrap.style.display = "none";
        els.showMore.style.display = "none";
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
                trophyIcons("gold", e.gold) +
                trophyIcons("silver", e.silver) +
                trophyIcons("bronze", e.bronze);
            var tr = document.createElement("tr");
            tr.innerHTML =
                '<td class="col-rank">' + e.rank + "</td>" +
                '<td class="col-player">' + playerLink(e.player) + "</td>" +
                '<td class="col-trophies">' + trophies + "</td>";
            els.hofBody.appendChild(tr);
        }
    }

    // Trophy icons are <use> references into a sprite injected once, so each
    // tier's gradients are defined a single time.
    var TROPHY_TIERS = {
        // highlight, base tone, shade, deep shade
        gold:   { rank: 1, label: "Gold",   colors: ["#FFF4B8", "#F2C230", "#B7860E", "#6E4F05"] },
        silver: { rank: 2, label: "Silver", colors: ["#FFFFFF", "#D3D9DF", "#939CA6", "#545B63"] },
        bronze: { rank: 3, label: "Bronze", colors: ["#FFD8B0", "#D48A48", "#95582A", "#55301A"] }
    };

    function trophySymbol(tier) {
        var c = TROPHY_TIERS[tier].colors;
        var n = TROPHY_TIERS[tier].rank;
        var id = "trophy-" + tier;
        return '<symbol id="' + id + '" viewBox="0 0 32 32">' +
            '<linearGradient id="' + id + '-h" x1="0" x2="1" y1="0" y2="0">' +
                '<stop offset="0" stop-color="' + c[2] + '"/><stop offset=".3" stop-color="' + c[0] + '"/>' +
                '<stop offset=".55" stop-color="' + c[1] + '"/><stop offset="1" stop-color="' + c[3] + '"/>' +
            "</linearGradient>" +
            '<linearGradient id="' + id + '-v" x1="0" x2="0" y1="0" y2="1">' +
                '<stop offset="0" stop-color="' + c[0] + '"/><stop offset="1" stop-color="' + c[2] + '"/>' +
            "</linearGradient>" +
            '<linearGradient id="' + id + '-base" x1="0" x2="0" y1="0" y2="1">' +
                '<stop offset="0" stop-color="#6A5344"/><stop offset="1" stop-color="#2E211B"/>' +
            "</linearGradient>" +
            // handles
            '<g fill="none" stroke="url(#' + id + '-v)" stroke-width="2" stroke-linecap="round">' +
                '<path d="M8.5 7.5H6Q4.2 7.5 4.2 9.6Q4.4 13.6 10.5 15.8"/>' +
                '<path d="M23.5 7.5H26Q27.8 7.5 27.8 9.6Q27.6 13.6 21.5 15.8"/>' +
            "</g>" +
            // stem, cup, rim, shine
            '<path d="M14.6 18.5H17.4L17 22.3H15Z" fill="url(#' + id + '-h)"/>' +
            '<path d="M7.8 5.2H24.2V9.5Q24.2 17.2 16 19.4Q7.8 17.2 7.8 9.5Z" fill="url(#' + id + '-h)"/>' +
            '<rect x="7.2" y="4" width="17.6" height="2.2" rx="1" fill="url(#' + id + '-v)"/>' +
            '<path d="M10.6 7.2Q10.6 13.3 13.6 16.4" stroke="#FFF" stroke-opacity=".55" stroke-width="1" fill="none" stroke-linecap="round"/>' +
            // engraved rank number
            '<g text-anchor="middle" font-family="Outfit, system-ui, sans-serif" font-weight="700" font-size="10">' +
                '<text x="16" y="15.7" fill="#FFF" fill-opacity=".25">' + n + "</text>" +
                '<text x="16" y="15.4" fill="' + c[3] + '" fill-opacity=".5">' + n + "</text>" +
            "</g>" +
            // collar, plinth, plaque
            '<rect x="12.4" y="21.8" width="7.2" height="2" rx=".6" fill="url(#' + id + '-h)"/>' +
            '<rect x="9.4" y="23.6" width="13.2" height="5.8" rx=".9" fill="url(#' + id + '-base)"/>' +
            '<rect x="9.9" y="23.6" width="12.2" height=".6" rx=".3" fill="#FFF" fill-opacity=".22"/>' +
            '<rect x="12.6" y="25.2" width="6.8" height="2.4" rx=".4" fill="url(#' + id + '-h)"/>' +
            "</symbol>";
    }

    function injectTrophySprite() {
        var holder = document.createElement("div");
        holder.innerHTML = '<svg class="trophy-sprite" aria-hidden="true">' +
            trophySymbol("gold") + trophySymbol("silver") + trophySymbol("bronze") + "</svg>";
        document.body.appendChild(holder.firstChild);
    }

    // months: ["2026-01", ...], one per trophy won in that tier.
    function trophyIcons(tier, months) {
        var out = "";
        for (var i = 0; i < (months || []).length; i++) {
            var label = formatFullMonth(months[i]) + " (" + tier + ")";
            out += '<svg class="trophy" role="img" aria-label="' + escapeHtml(label) + '">' +
                "<title>" + escapeHtml(label) + "</title>" +
                '<use href="#trophy-' + tier + '"/></svg>';
        }
        return out;
    }

    // --- Player detail ---
    function showPlayerLoading() {
        els.playerLoading.style.display = "flex";
        els.playerError.style.display = "none";
        els.playerContent.style.display = "none";
    }

    function showPlayerError(msg) {
        els.playerLoading.style.display = "none";
        els.playerError.style.display = "block";
        els.playerError.textContent = msg;
        els.playerContent.style.display = "none";
    }

    function fetchPlayer() {
        var params = new URLSearchParams();
        params.set("id", state.playerID);
        params.set("t", state.playerSig);

        fetchCached({
            url: "api/player?" + params.toString(),
            cacheStore: playerCache,
            cacheKey: state.playerID + ":" + state.playerSig,
            onLoading: showPlayerLoading,
            onSuccess: renderPlayer,
            onError: function (kind) {
                if (kind === "notfound") {
                    showPlayerError("Player not found or link expired.");
                } else if (kind === "request") {
                    showPlayerError("Failed to load player. Please try again later.");
                } else {
                    showPlayerError("Something went wrong while reading the response.");
                }
            }
        });
    }

    function renderPlayer(data) {
        els.playerLoading.style.display = "none";
        els.playerError.style.display = "none";
        els.playerContent.style.display = "";

        var tmioHref = "https://trackmania.io/#/player/" + encodeURIComponent(data.player.openplanet_id);
        els.playerName.innerHTML = '<a href="' + tmioHref + '" target="_blank" rel="noopener">' + escapeHtml(data.player.display_name) + "</a>";

        var byMode = {};
        for (var i = 0; i < data.modes.length; i++) {
            byMode[data.modes[i].game_mode] = data.modes[i];
        }
        var authorStats = computeModeStats(byMode.author);
        var goldStats = computeModeStats(byMode.gold);

        var totalRuns = authorStats.runs + goldStats.runs;
        var totalMedals = authorStats.medals + goldStats.medals;
        var totalSkips = authorStats.skips + goldStats.skips;

        els.playerSummary.innerHTML =
            summaryItem("Runs", totalRuns) +
            summaryItem("Medals", totalMedals) +
            summaryItem("Skipped", totalSkips);

        renderPlayerMode(byMode.author, authorStats, els.playerBodyAuthor, els.playerEmptyAuthor, els.playerStatsAuthor);
        renderPlayerMode(byMode.gold, goldStats, els.playerBodyGold, els.playerEmptyGold, els.playerStatsGold);
    }

    function computeModeStats(mode) {
        var stats = { runs: 0, best: 0, medals: 0, skips: 0 };
        if (!mode || !mode.stats) return stats;
        stats.runs = mode.stats.runs || 0;
        stats.best = mode.stats.best || 0;
        stats.medals = mode.stats.maps_completed || 0;
        stats.skips = mode.stats.maps_skipped || 0;
        return stats;
    }

    function summaryItem(label, value) {
        // label is a static string; value is a number we control.
        return '<li><span class="summary-label">' + label + '</span><span class="summary-value">' + value + '</span></li>';
    }

    function renderPlayerMode(mode, stats, tbody, empty, statsEl) {
        tbody.innerHTML = "";
        if (!mode || stats.runs === 0) {
            statsEl.textContent = "no runs";
            tbody.parentElement.style.display = "none";
            empty.style.display = "block";
            return;
        }
        tbody.parentElement.style.display = "";
        empty.style.display = "none";

        // mode.scores is capped; stats covers every run.
        var shown = mode.scores.length;
        statsEl.textContent = stats.runs + " run" + (stats.runs === 1 ? "" : "s")
            + (shown < stats.runs ? " (latest " + shown + ")" : "");

        // Tag the top 3 runs (by score, ties broken by date order) so CSS can
        // show the same medal accents as the leaderboard podium.
        var podiumClass = new Array(mode.scores.length);
        var ranked = mode.scores.map(function (s, i) { return { i: i, score: s.score }; });
        ranked.sort(function (a, b) { return b.score - a.score; });
        for (var r = 0; r < Math.min(3, ranked.length); r++) {
            podiumClass[ranked[r].i] = "podium-" + (r + 1);
        }

        for (var i = 0; i < mode.scores.length; i++) {
            var s = mode.scores[i];
            var tr = document.createElement("tr");
            if (podiumClass[i]) tr.className = podiumClass[i];
            tr.innerHTML =
                '<td class="col-date" title="' + escapeHtml(new Date(s.created_at).toLocaleString()) + '">' + formatDate(s.created_at) + "</td>" +
                '<td class="col-score">' + formatScore(s.score) + "</td>" +
                '<td class="col-maps">' + s.maps_completed + "</td>" +
                '<td class="col-skipped">' + s.maps_skipped + "</td>";
            tbody.appendChild(tr);
        }
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
        var h;
        if (state.view === "player") {
            h = "player/" + encodeURIComponent(state.playerID) + "/" + encodeURIComponent(state.playerSig);
        } else {
            h = state.gameMode;
            if (state.month) {
                h += "/" + state.month;
            }
        }
        history.replaceState(null, "", "#" + h);
    }

    function syncUI() {
        // Game mode toggles
        var modeBtns = els.modeToggle.querySelectorAll(".toggle-btn");
        for (var i = 0; i < modeBtns.length; i++) {
            modeBtns[i].classList.toggle("active", modeBtns[i].getAttribute("data-value") === state.gameMode);
        }

        // Period toggles + archive label + board title
        els.boardTitleMode.textContent = state.gameMode === "gold" ? "Gold" : "Author";
        els.boardTitleTag.style.display = state.month === "current" ? "" : "none";
        if (state.month === "all") {
            setActiveToggle("all");
            resetArchiveLabel();
            els.boardTitleText.textContent = "All Time";
        } else if (state.month === "hof") {
            setActiveToggle("hof");
            resetArchiveLabel();
        } else {
            setActiveToggle("archive");
            var monthKey = state.month === "current" ? getCurrentMonth() : state.month;
            var parts = monthKey.split("-");
            els.archiveBtn.querySelector(".archive-label").textContent =
                formatMonthLabel(parseInt(parts[0], 10), parseInt(parts[1], 10));
            els.boardTitleText.textContent = formatFullMonth(monthKey);
        }
    }

    function applyHash() {
        var hash = location.hash.replace(/^#/, "");
        closeArchiveDropdown();

        var segments = hash ? hash.split("/") : [];
        if (segments[0] === "player" && segments.length >= 3) {
            state.view = "player";
            state.playerID = decodeURIComponent(segments[1]);
            state.playerSig = decodeURIComponent(segments[2]);
        } else {
            state.view = "main";
            if (!hash) {
                state.gameMode = "author";
                state.month = "current";
            } else {
                var mode = segments[0];
                if (mode === "author" || mode === "gold") {
                    state.gameMode = mode;
                } else {
                    state.gameMode = "author";
                }
                state.month = segments[1] || "current";
            }
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
            state.month = "all";
        } else if (value === "hof") {
            state.month = "hof";
        }

        pushHash();
        syncUI();
        fetchData();
    }

    els.showMore.addEventListener("click", revealMore);

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

    function dismissPlayer() {
        if (state.view !== "player") return;
        // Navigate forward to the main view rather than walking history back —
        // history.back() can land on another player URL (e.g. after editing
        // the hash directly) and reopen the modal.
        var h = state.gameMode || "author";
        if (state.month) h += "/" + state.month;
        location.hash = "#" + h;
    }

    els.playerClose.addEventListener("click", dismissPlayer);
    els.playerModal.addEventListener("click", function (e) {
        if (e.target.hasAttribute("data-modal-close")) dismissPlayer();
    });
    document.addEventListener("keydown", function (e) {
        if (e.key === "Escape") dismissPlayer();
    });

    window.addEventListener("hashchange", applyHash);

    // Init
    injectTrophySprite();
    populateArchiveDropdown();
    applyHash();
})();
