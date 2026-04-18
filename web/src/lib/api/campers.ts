import { api } from "./client";
import type {
	Camper,
	CamperFriendPreference,
	CamperFriendPreferenceItem,
	CreateCamperRequest,
	CreateEnrollmentRequest,
	Enrollment,
	SessionAgeGroup,
	UpdateCamperRequest,
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
