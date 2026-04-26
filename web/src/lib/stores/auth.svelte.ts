import { authClient } from "$lib/auth-client";

interface JwtPayload {
	sub?: string;
	username?: string;
	email?: string;
	role?: string;
	camp_id?: string;
	org_role?: string;
	impersonated_by?: string;
	first_name?: string;
	last_name?: string;
	exp?: number;
	[key: string]: unknown;
}

function parseJwtPayload(token: string): JwtPayload | null {
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
	// Treat as expired 30s before actual exp to avoid clock skew / in-flight races
	return Date.now() >= (payload.exp - 30) * 1000;
}

interface OrgSummary {
	id: string;
	name: string;
	slug: string;
}

function createAuthStore() {
	let token = $state<string | null>(null);
	let claims = $state<JwtPayload | null>(null);
	let sessionLoaded = $state(false);
	let hasSession = $state(false);
	let organizations = $state<OrgSummary[]>([]);

	let tokenPromise: Promise<string | null> | null = null;

	function applyToken(next: string | null) {
		token = next;
		claims = next ? parseJwtPayload(next) : null;
	}

	async function refreshToken(): Promise<string | null> {
		if (tokenPromise) return tokenPromise;
		tokenPromise = (async () => {
			try {
				// BetterAuth jwt plugin: GET /api/auth/token returns { token: string }
				const res = await fetch("/api/auth/token", {
					credentials: "include",
				});
				if (!res.ok) {
					applyToken(null);
					hasSession = false;
					return null;
				}
				const data = (await res.json()) as { token?: string };
				if (data.token) {
					applyToken(data.token);
					hasSession = true;
					return data.token;
				}
				applyToken(null);
				return null;
			} catch {
				applyToken(null);
				return null;
			} finally {
				tokenPromise = null;
			}
		})();
		return tokenPromise;
	}

	async function ensureToken(): Promise<string | null> {
		if (token && !isTokenExpired(token)) return token;
		return refreshToken();
	}

	async function loadOrganizations() {
		try {
			const result = await authClient.organization.list();
			const list = (result?.data ?? []) as OrgSummary[];
			organizations = list.map((o) => ({ id: o.id, name: o.name, slug: o.slug }));
		} catch {
			organizations = [];
		}
	}

	async function init() {
		if (sessionLoaded) return;
		try {
			const session = await authClient.getSession();
			if (session?.data?.session) {
				hasSession = true;
				await refreshToken();
				await loadOrganizations();
			} else {
				hasSession = false;
				applyToken(null);
			}
		} catch {
			hasSession = false;
			applyToken(null);
		} finally {
			sessionLoaded = true;
		}
	}

	async function onSignedIn() {
		hasSession = true;
		sessionLoaded = true;
		await refreshToken();
		await loadOrganizations();
	}

	async function signOut() {
		try {
			await authClient.signOut();
		} catch {
			// ignore — clear local state regardless
		}
		applyToken(null);
		hasSession = false;
		organizations = [];
	}

	async function setActiveOrganization(organizationId: string) {
		await authClient.organization.setActive({ organizationId });
		// Force a fresh JWT so camp_id reflects the new active org
		applyToken(null);
		await refreshToken();
	}

	async function stopImpersonating() {
		try {
			await authClient.admin.stopImpersonating();
		} catch {
			// ignore — refresh state regardless
		}
		applyToken(null);
		await refreshToken();
		await loadOrganizations();
	}

	if (typeof window !== "undefined") {
		void init();
	}

	return {
		get accessToken() {
			return token;
		},
		get username() {
			return (claims?.username as string) ?? null;
		},
		get email() {
			return (claims?.email as string) ?? null;
		},
		get role() {
			return (claims?.role as string) ?? null;
		},
		get campId() {
			return (claims?.camp_id as string) ?? null;
		},
		get orgRole() {
			return (claims?.org_role as string) ?? null;
		},
		get isAuthenticated() {
			return hasSession;
		},
		get isInitialized() {
			return sessionLoaded;
		},
		get isAccessExpired() {
			return token !== null && isTokenExpired(token);
		},
		get isImpersonating() {
			const v = claims?.impersonated_by;
			return typeof v === "string" && v.length > 0;
		},
		get organizations() {
			return organizations;
		},
		ensureToken,
		refreshToken,
		onSignedIn,
		signOut,
		setActiveOrganization,
		stopImpersonating,
		loadOrganizations,
	};
}

export const auth = createAuthStore();
