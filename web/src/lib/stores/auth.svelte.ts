import type { TokenResponse } from "$lib/api/types";

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";
const ADMIN_ACCESS_TOKEN_KEY = "admin_access_token";
const ADMIN_REFRESH_TOKEN_KEY = "admin_refresh_token";

function parseJwtPayload(token: string): Record<string, unknown> | null {
	try {
		const parts = token.split(".");
		if (parts.length !== 3) return null;
		let base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
		while (base64.length % 4) base64 += "=";
		const payload = atob(base64);
		return JSON.parse(payload);
	} catch {
		return null;
	}
}

function isTokenExpired(token: string): boolean {
	const payload = parseJwtPayload(token);
	if (!payload || typeof payload.exp !== "number") return true;
	return Date.now() >= payload.exp * 1000;
}

function createAuthStore() {
	let accessToken = $state<string | null>(null);
	let refreshToken = $state<string | null>(null);
	let username = $state<string | null>(null);
	let role = $state<string | null>(null);
	let campId = $state<string | null>(null);
	let impersonatedBy = $state<string | null>(null);

	function loadFromStorage() {
		if (typeof window === "undefined") return;
		const storedAccess = localStorage.getItem(ACCESS_TOKEN_KEY);
		const storedRefresh = localStorage.getItem(REFRESH_TOKEN_KEY);

		if (storedAccess && storedRefresh) {
			setTokens({ access_token: storedAccess, refresh_token: storedRefresh });
		}
	}

	function setTokens(tokens: TokenResponse) {
		accessToken = tokens.access_token;
		refreshToken = tokens.refresh_token;

		const payload = parseJwtPayload(tokens.access_token);
		username = (payload?.username as string) ?? null;
		role = (payload?.role as string) ?? null;
		campId = (payload?.camp_id as string) ?? null;
		impersonatedBy = (payload?.impersonated_by as string) ?? null;

		localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
		localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
	}

	function clear() {
		accessToken = null;
		refreshToken = null;
		username = null;
		role = null;
		campId = null;
		impersonatedBy = null;
		localStorage.removeItem(ACCESS_TOKEN_KEY);
		localStorage.removeItem(REFRESH_TOKEN_KEY);
		localStorage.removeItem(ADMIN_ACCESS_TOKEN_KEY);
		localStorage.removeItem(ADMIN_REFRESH_TOKEN_KEY);
	}

	function startImpersonation(tokens: TokenResponse) {
		if (accessToken) {
			localStorage.setItem(ADMIN_ACCESS_TOKEN_KEY, accessToken);
		}
		if (refreshToken) {
			localStorage.setItem(ADMIN_REFRESH_TOKEN_KEY, refreshToken);
		}
		setTokens(tokens);
	}

	function stopImpersonation() {
		const adminAccess = localStorage.getItem(ADMIN_ACCESS_TOKEN_KEY);
		const adminRefresh = localStorage.getItem(ADMIN_REFRESH_TOKEN_KEY);

		localStorage.removeItem(ADMIN_ACCESS_TOKEN_KEY);
		localStorage.removeItem(ADMIN_REFRESH_TOKEN_KEY);

		if (adminAccess && adminRefresh) {
			setTokens({ access_token: adminAccess, refresh_token: adminRefresh });
		} else {
			clear();
		}
	}

	loadFromStorage();

	return {
		get accessToken() {
			return accessToken;
		},
		get refreshToken() {
			return refreshToken;
		},
		get username() {
			return username;
		},
		get role() {
			return role;
		},
		get campId() {
			return campId;
		},
		get isAuthenticated() {
			return accessToken !== null && refreshToken !== null;
		},
		get isAccessExpired() {
			return accessToken !== null && isTokenExpired(accessToken);
		},
		get isImpersonating() {
			return impersonatedBy !== null && impersonatedBy !== "";
		},
		setTokens,
		clear,
		startImpersonation,
		stopImpersonation,
	};
}

export const auth = createAuthStore();
