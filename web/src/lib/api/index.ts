import { api } from "./client";
import type {
	AgeGroup,
	Camp,
	Certification,
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
	// Camp-scoped (requires camp_id in JWT)
	get: () => api.get<Camp>("/api/v1/camp"),
	update: (data: UpdateCampRequest) =>
		api.put<Camp>("/api/v1/camp", data),

	// Super-admin only
	list: () => api.get<Camp[]>("/api/v1/admin/camps"),
	getById: (id: string) => api.get<Camp>(`/api/v1/admin/camps/${id}`),
	create: (data: CreateCampRequest) =>
		api.post<Camp>("/api/v1/admin/camps", data),
	updateById: (id: string, data: UpdateCampRequest) =>
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

	impersonate: (id: string) =>
		api.post<TokenResponse>(`/api/v1/admin/users/${id}/impersonate`),
};

export const certificationApi = {
	list: () => api.get<Certification[]>("/api/v1/certifications"),
	get: (id: string) => api.get<Certification>(`/api/v1/certifications/${id}`),
	create: (name: string) => api.post<Certification>("/api/v1/certifications", { name }),
	update: (id: string, name: string) =>
		api.put<Certification>(`/api/v1/certifications/${id}`, { name }),
	delete: (id: string) => api.delete<void>(`/api/v1/certifications/${id}`),
};

export const ageGroupApi = {
	list: () => api.get<AgeGroup[]>("/api/v1/age-groups"),
	get: (id: string) => api.get<AgeGroup>(`/api/v1/age-groups/${id}`),
	create: (name: string) => api.post<AgeGroup>("/api/v1/age-groups", { name }),
	update: (id: string, name: string) =>
		api.put<AgeGroup>(`/api/v1/age-groups/${id}`, { name }),
	delete: (id: string) => api.delete<void>(`/api/v1/age-groups/${id}`),
};
