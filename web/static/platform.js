(function () {
	"use strict";

	document.documentElement.classList.add("pf-js");

	var shell = document.querySelector("[data-app-shell]");
	var sidebar = document.getElementById("app-sidebar");
	var mobileToggle = document.querySelector("[data-mobile-nav-toggle]");
	var collapseToggle = document.querySelector("[data-sidebar-collapse]");
	var backdrop = document.querySelector(".pf-mobile-nav-backdrop");
	var collapseKey = "tockrplatform.sidebarCollapsed";
	var mobileQuery = window.matchMedia("(max-width: 920px)");

	function setMobileNavigation(open, returnFocus) {
		if (!shell || !mobileToggle) return;
		shell.classList.toggle("nav-open", open);
		document.body.classList.toggle("nav-open", open);
		mobileToggle.setAttribute("aria-expanded", open ? "true" : "false");
		if (backdrop) backdrop.hidden = !open;
		if (open && sidebar) {
			var firstControl = sidebar.querySelector("a, button");
			if (firstControl) firstControl.focus();
		} else if (returnFocus) {
			mobileToggle.focus();
		}
	}

	function setCollapsed(collapsed, persist) {
		if (!shell) return;
		shell.classList.toggle("sidebar-collapsed", collapsed);
		if (collapseToggle) {
			collapseToggle.setAttribute("aria-expanded", collapsed ? "false" : "true");
			collapseToggle.setAttribute("aria-label", collapsed ? "Expand navigation" : "Collapse navigation");
		}
		if (persist) {
			try {
				localStorage.setItem(collapseKey, collapsed ? "true" : "false");
			} catch (_) {
				// Navigation remains usable when browser storage is unavailable.
			}
		}
	}

	if (shell && mobileToggle) {
		mobileToggle.addEventListener("click", function () {
			setMobileNavigation(mobileToggle.getAttribute("aria-expanded") !== "true", true);
		});
		document.querySelectorAll("[data-mobile-nav-close]").forEach(function (control) {
			control.addEventListener("click", function () {
				setMobileNavigation(false, true);
			});
		});
		if (sidebar) {
			sidebar.addEventListener("click", function (event) {
				if (event.target.closest("a") && mobileQuery.matches) setMobileNavigation(false, false);
			});
		}
		document.addEventListener("keydown", function (event) {
			if (event.key === "Escape" && shell.classList.contains("nav-open")) setMobileNavigation(false, true);
		});
	}

	if (shell && collapseToggle) {
		try {
			setCollapsed(localStorage.getItem(collapseKey) === "true", false);
		} catch (_) {
			setCollapsed(false, false);
		}
		collapseToggle.addEventListener("click", function () {
			setCollapsed(!shell.classList.contains("sidebar-collapsed"), true);
		});
	}
})();
