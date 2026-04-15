import { api } from "./client";
import type {
	Camp,
	CreateCampRequest,
	CreateUserRequest,
	TokenResponse,
	UpdateCampRequest,
	UpdatePasswordRequest,
	UpdateUserRequest,
	User,
} from "./types";

export const authApi = {
	login: (username: string, password: string) =>
		api.post<TokenResponse>("/api/v1/auth/login", { username, password }),

	refresh: (refreshToken: string) =>
		api.post<TokenResponse>("/api/v1/auth/refresh", {
			refresh_token: refreshToken,
		}),
};

export const campApi = {
	list: () => api.get<Camp[]>("/api/v1/admin/camps"),

	get: (id: string) => api.get<Camp>(`/api/v1/admin/camps/${id}`),

	create: (data: CreateCampRequest) =>
		api.post<Camp>("/api/v1/admin/camps", data),

	update: (id: string, data: UpdateCampRequest) =>
		api.put<Camp>(`/api/v1/admin/camps/${id}`, data),

	delete: (id: string) => api.delete<void>(`/api/v1/admin/camps/${id}`),
};

export const userApi = {
	list: (campId?: string) => {
		const params = campId ? `?camp_id=${campId}` : "";
		return api.get<User[]>(`/api/v1/admin/users${params}`);
	},

	get: (id: string) => api.get<User>(`/api/v1/admin/users/${id}`),

	create: (data: CreateUserRequest) =>
		api.post<User>("/api/v1/admin/users", data),

	update: (id: string, data: UpdateUserRequest) =>
		api.put<User>(`/api/v1/admin/users/${id}`, data),

	updatePassword: (id: string, data: UpdatePasswordRequest) =>
		api.put<void>(`/api/v1/admin/users/${id}/password`, data),

	delete: (id: string) => api.delete<void>(`/api/v1/admin/users/${id}`),
};
