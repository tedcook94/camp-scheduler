import { auth } from "$lib/stores/auth.svelte";
import type { ApiError, TokenResponse } from "./types";

class ApiClientError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = "ApiClientError";
		this.status = status;
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

	const headers: Record<string, string> = {
		...(options.headers as Record<string, string>),
	};

	if (hasAuth && auth.accessToken) {
		headers["Authorization"] = `Bearer ${auth.accessToken}`;
	}

	if (options.body && !headers["Content-Type"]) {
		headers["Content-Type"] = "application/json";
	}

	const res = await fetch(path, { ...options, headers });

	if (res.status === 401 && hasAuth) {
		// Token might have expired between check and request — try refresh once
		const refreshed = await refreshTokens();
		if (refreshed) {
			headers["Authorization"] = `Bearer ${auth.accessToken}`;
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
	try {
		const body: ApiError = await res.json();
		if (body.error) message = body.error;
	} catch {
		// Response wasn't JSON
	}

	throw new ApiClientError(res.status, message);
}

export const api = {
	get: <T>(path: string) => request<T>(path),

	post: <T>(path: string, body?: unknown) =>
		request<T>(path, {
			method: "POST",
			body: body !== undefined ? JSON.stringify(body) : undefined,
		}),

	put: <T>(path: string, body?: unknown) =>
		request<T>(path, {
			method: "PUT",
			body: body !== undefined ? JSON.stringify(body) : undefined,
		}),

	delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
