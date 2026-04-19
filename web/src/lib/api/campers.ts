import { api } from "./client";
import type {
	Camper,
	CamperFriendPreference,
	CamperFriendPreferenceItem,
	CreateCamperRequest,
	CreateEnrollmentRequest,
	CreateSessionAgeGroupRequest,
	CreateSessionCabinRequest,
	Enrollment,
	SessionAgeGroup,
	SessionCabin,
	UpdateCamperRequest,
	UpdateSessionAgeGroupRequest,
	UpdateSessionCabinRequest,
} from "./types";

export const camperApi = {
	list: () => api.get<Camper[]>("/api/v1/campers"),
	get: (id: string) => api.get<Camper>(`/api/v1/campers/${id}`),
	create: (data: CreateCamperRequest) =>
		api.post<Camper>("/api/v1/campers", data),
	update: (id: string, data: UpdateCamperRequest) =>
		api.put<Camper>(`/api/v1/campers/${id}`, data),
	delete: (id: string) => api.delete<void>(`/api/v1/campers/${id}`),
};

export const enrollmentApi = {
	list: (sessionId: string) =>
		api.get<Enrollment[]>(`/api/v1/sessions/${sessionId}/enrollments`),
	get: (sessionId: string, id: string) =>
		api.get<Enrollment>(`/api/v1/sessions/${sessionId}/enrollments/${id}`),
	create: (sessionId: string, data: CreateEnrollmentRequest) =>
		api.post<Enrollment>(`/api/v1/sessions/${sessionId}/enrollments`, data),
	delete: (sessionId: string, id: string) =>
		api.delete<void>(`/api/v1/sessions/${sessionId}/enrollments/${id}`),
};

export const sessionAgeGroupApi = {
	list: (sessionId: string, signal?: AbortSignal) =>
		api.get<SessionAgeGroup[]>(`/api/v1/sessions/${sessionId}/age-groups`, { signal }),
	get: (sessionId: string, id: string) =>
		api.get<SessionAgeGroup>(`/api/v1/sessions/${sessionId}/age-groups/${id}`),
	create: (sessionId: string, data: CreateSessionAgeGroupRequest) =>
		api.post<SessionAgeGroup>(`/api/v1/sessions/${sessionId}/age-groups`, data),
	update: (sessionId: string, id: string, data: UpdateSessionAgeGroupRequest) =>
		api.put<SessionAgeGroup>(`/api/v1/sessions/${sessionId}/age-groups/${id}`, data),
	delete: (sessionId: string, id: string) =>
		api.delete<void>(`/api/v1/sessions/${sessionId}/age-groups/${id}`),
};

export const sessionCabinApi = {
	list: (sessionId: string, signal?: AbortSignal) =>
		api.get<SessionCabin[]>(`/api/v1/sessions/${sessionId}/cabins`, { signal }),
	get: (sessionId: string, id: string) =>
		api.get<SessionCabin>(`/api/v1/sessions/${sessionId}/cabins/${id}`),
	create: (sessionId: string, data: CreateSessionCabinRequest) =>
		api.post<SessionCabin>(`/api/v1/sessions/${sessionId}/cabins`, data),
	update: (sessionId: string, id: string, data: UpdateSessionCabinRequest) =>
		api.put<SessionCabin>(`/api/v1/sessions/${sessionId}/cabins/${id}`, data),
	delete: (sessionId: string, id: string) =>
		api.delete<void>(`/api/v1/sessions/${sessionId}/cabins/${id}`),
};

export const camperFriendPreferenceApi = {
	list: (sessionId: string, camperId: string, signal?: AbortSignal) =>
		api.get<CamperFriendPreference[]>(
			`/api/v1/sessions/${sessionId}/campers/${camperId}/friend-preferences`,
			{ signal },
		),
	replaceAll: (sessionId: string, camperId: string, data: CamperFriendPreferenceItem[]) =>
		api.put<CamperFriendPreference[]>(
			`/api/v1/sessions/${sessionId}/campers/${camperId}/friend-preferences`,
			data,
		),
};
