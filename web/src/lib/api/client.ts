import { auth } from "$lib/stores/auth.svelte";
import type { ApiError, TokenResponse } from "./types";

class ApiClientError extends Error {
	status: number;
	data?: Record<string, unknown>;

	constructor(status: number, message: string, data?: Record<string, unknown>) {
		super(message.charAt(0).toUpperCase() + message.slice(1));
		this.name = "ApiClientError";
		this.status = status;
		this.data = data;
	}
}

export { ApiClientError };

let refreshPromise: Promise<boolean> | null = null;

function handleAuthFailure() {
	if (auth.isImpersonating) {
		auth.stopImpersonation();
	} else {
		auth.clear();
	}
}

async function refreshTokens(): Promise<boolean> {
	if (!auth.refreshToken) return false;

	try {
		const res = await fetch("/api/v1/auth/refresh", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ refresh_token: auth.refreshToken }),
		});

		if (!res.ok) {
			handleAuthFailure();
			return false;
		}

		const tokens: TokenResponse = await res.json();
		auth.setTokens(tokens);
		return true;
	} catch {
		handleAuthFailure();
		return false;
	}
}

async function ensureValidToken(): Promise<boolean> {
	if (!auth.isAuthenticated) return false;

	if (!auth.isAccessExpired) return true;

	// Deduplicate concurrent refresh attempts
	if (!refreshPromise) {
		refreshPromise = refreshTokens().finally(() => {
			refreshPromise = null;
		});
	}

	return refreshPromise;
}

async function request<T>(
	path: string,
	options: RequestInit = {},
): Promise<T> {
	const hasAuth = await ensureValidToken();

	const headers = new Headers(options.headers);

	if (hasAuth && auth.accessToken) {
		headers.set("Authorization", `Bearer ${auth.accessToken}`);
	}

	if (options.body && !headers.has("Content-Type")) {
		headers.set("Content-Type", "application/json");
	}

	const res = await fetch(path, { ...options, headers });

	if (res.status === 401 && hasAuth) {
		// Token might have expired between check and request — try refresh once
		const refreshed = await refreshTokens();
		if (refreshed && auth.accessToken) {
			headers.set("Authorization", `Bearer ${auth.accessToken}`);
			const retryRes = await fetch(path, { ...options, headers });
			return handleResponse<T>(retryRes);
		}
	}

	return handleResponse<T>(res);
}

async function handleResponse<T>(res: Response): Promise<T> {
	if (res.ok) {
		const text = await res.text();
		if (!text) return undefined as T;
		return JSON.parse(text) as T;
	}

	let message = `request failed with status ${res.status}`;
	let data: Record<string, unknown> | undefined;
	try {
		const body: ApiError & Record<string, unknown> = await res.json();
		if (body.error) message = body.error;
		data = body;
	} catch {
		// Response wasn't JSON
	}

	throw new ApiClientError(res.status, message, data);
}

export const api = {
	get: <T>(path: string, options?: RequestInit) =>
		request<T>(path, options),

	post: <T>(path: string, body?: unknown, options?: RequestInit) =>
		request<T>(path, {
			...options,
			method: "POST",
			body: body !== undefined ? JSON.stringify(body) : undefined,
		}),

	put: <T>(path: string, body?: unknown, options?: RequestInit) =>
		request<T>(path, {
			...options,
			method: "PUT",
			body: body !== undefined ? JSON.stringify(body) : undefined,
		}),

	delete: <T>(path: string, options?: RequestInit) =>
		request<T>(path, { ...options, method: "DELETE" }),

	// download issues an authenticated GET and returns the response Blob along
	// with the suggested filename parsed from Content-Disposition (when present).
	download: async (
		path: string,
		options?: RequestInit,
	): Promise<{ blob: Blob; filename: string | null }> => {
		const hasAuth = await ensureValidToken();
		const headers = new Headers(options?.headers);
		if (hasAuth && auth.accessToken) {
			headers.set("Authorization", `Bearer ${auth.accessToken}`);
		}

		let res = await fetch(path, { ...options, method: "GET", headers });
		if (res.status === 401 && hasAuth) {
			const refreshed = await refreshTokens();
			if (refreshed && auth.accessToken) {
				headers.set("Authorization", `Bearer ${auth.accessToken}`);
				res = await fetch(path, { ...options, method: "GET", headers });
			}
		}
		if (!res.ok) {
			let message = `download failed with status ${res.status}`;
			try {
				const body = await res.json();
				if (body?.error) message = body.error;
			} catch {
				// not json
			}
			throw new ApiClientError(res.status, message);
		}
		const blob = await res.blob();
		const cd = res.headers.get("Content-Disposition") || "";
		let filename: string | null = null;
		// RFC 5987: filename*=charset'lang'percent-encoded-value (UTF-8 / escaped).
		const star = /filename\*=[^']*'[^']*'([^;]+)/i.exec(cd);
		if (star) {
			try {
				filename = decodeURIComponent(star[1].trim());
			} catch {
				filename = null;
			}
		}
		if (!filename) {
			const plain = /filename="?([^";]+)"?/i.exec(cd);
			if (plain) filename = plain[1];
		}
		return { blob, filename };
	},
};
