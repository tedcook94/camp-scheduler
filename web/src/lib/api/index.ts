import { api } from "./client";
import type {
	Activity,
	ActivityCertification,
	ActivityPreference,
	ActivityPreferenceItem,
	AgeGroup,
	AgeGroupPreference,
	AgeGroupPreferenceItem,
	AddCounselorCertificationRequest,
	Cabin,
	Camp,
	Certification,
	CocounselorPreference,
	CocounselorPreferenceItem,
	Counselor,
	CounselorCertification,
	CreateCabinRequest,
	CreateCampRequest,
	CreateCounselorRequest,
	CreateSeasonRequest,
	CreateSessionActivityRequest,
	CreateSessionHistoryRequest,
	CreateSessionRequest,
	CreateSessionTimeSlotRequest,
	CreateUserRequest,
	HistorySummary,
	Season,
	Session,
	SessionActivity,
	SessionHistory,
	SessionTimeSlot,
	TimeSlot,
	TokenResponse,
	UpdateCabinRequest,
	UpdateCampRequest,
	UpdateCounselorRequest,
	UpdatePasswordRequest,
	UpdateSeasonRequest,
	UpdateSessionActivityRequest,
	UpdateSessionHistoryRequest,
	UpdateSessionRequest,
	UpdateSessionTimeSlotRequest,
	UpdateUserRequest,
	User,
} from "./types";

export {
	camperApi,
	enrollmentApi,
	sessionAgeGroupApi,
	sessionCabinApi,
	camperFriendPreferenceApi,
} from "./campers";

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

export const cabinApi = {
	list: () => api.get<Cabin[]>("/api/v1/cabins"),
	get: (id: string) => api.get<Cabin>(`/api/v1/cabins/${id}`),
	create: (data: CreateCabinRequest) =>
		api.post<Cabin>("/api/v1/cabins", data),
	update: (id: string, data: UpdateCabinRequest) =>
		api.put<Cabin>(`/api/v1/cabins/${id}`, data),
	delete: (id: string) => api.delete<void>(`/api/v1/cabins/${id}`),
};

export const seasonApi = {
	list: () => api.get<Season[]>("/api/v1/seasons"),
	get: (id: string) => api.get<Season>(`/api/v1/seasons/${id}`),
	create: (data: CreateSeasonRequest) =>
		api.post<Season>("/api/v1/seasons", data),
	update: (id: string, data: UpdateSeasonRequest) =>
		api.put<Season>(`/api/v1/seasons/${id}`, data),
	delete: (id: string) => api.delete<void>(`/api/v1/seasons/${id}`),
};

export const sessionApi = {
	list: () => api.get<Session[]>("/api/v1/sessions"),
	get: (id: string) => api.get<Session>(`/api/v1/sessions/${id}`),
	create: (data: CreateSessionRequest) =>
		api.post<Session>("/api/v1/sessions", data),
	update: (id: string, data: UpdateSessionRequest) =>
		api.put<Session>(`/api/v1/sessions/${id}`, data),
	delete: (id: string) => api.delete<void>(`/api/v1/sessions/${id}`),
};

export const activityApi = {
	list: () => api.get<Activity[]>("/api/v1/activities"),
	get: (id: string) => api.get<Activity>(`/api/v1/activities/${id}`),
	create: (name: string) => api.post<Activity>("/api/v1/activities", { name }),
	update: (id: string, name: string) =>
		api.put<Activity>(`/api/v1/activities/${id}`, { name }),
	delete: (id: string) => api.delete<void>(`/api/v1/activities/${id}`),
};

export const activityCertificationApi = {
	listAll: () =>
		api.get<ActivityCertification[]>("/api/v1/activity-certifications"),
	list: (activityId: string) =>
		api.get<ActivityCertification[]>(`/api/v1/activities/${activityId}/certifications`),
	add: (activityId: string, certificationId: string) =>
		api.post<ActivityCertification>(`/api/v1/activities/${activityId}/certifications`, { certification_id: certificationId }),
	remove: (activityId: string, id: string) =>
		api.delete<void>(`/api/v1/activities/${activityId}/certifications/${id}`),
};

export const timeSlotApi = {
	list: () => api.get<TimeSlot[]>("/api/v1/time-slots"),
	get: (id: string) => api.get<TimeSlot>(`/api/v1/time-slots/${id}`),
	create: (name: string) => api.post<TimeSlot>("/api/v1/time-slots", { name }),
	update: (id: string, name: string) =>
		api.put<TimeSlot>(`/api/v1/time-slots/${id}`, { name }),
	delete: (id: string) => api.delete<void>(`/api/v1/time-slots/${id}`),
};

export const sessionTimeSlotApi = {
	list: (sessionId: string) =>
		api.get<SessionTimeSlot[]>(`/api/v1/sessions/${sessionId}/time-slots`),
	get: (sessionId: string, id: string) =>
		api.get<SessionTimeSlot>(`/api/v1/sessions/${sessionId}/time-slots/${id}`),
	create: (sessionId: string, data: CreateSessionTimeSlotRequest) =>
		api.post<SessionTimeSlot>(`/api/v1/sessions/${sessionId}/time-slots`, data),
	update: (sessionId: string, id: string, data: UpdateSessionTimeSlotRequest) =>
		api.put<SessionTimeSlot>(`/api/v1/sessions/${sessionId}/time-slots/${id}`, data),
	delete: (sessionId: string, id: string) =>
		api.delete<void>(`/api/v1/sessions/${sessionId}/time-slots/${id}`),
	reorder: (sessionId: string, orderedIds: string[]) =>
		api.put<void>(`/api/v1/sessions/${sessionId}/time-slots/reorder`, { ordered_ids: orderedIds }),
};

export const sessionActivityApi = {
	listAll: (sessionId: string) =>
		api.get<SessionActivity[]>(`/api/v1/sessions/${sessionId}/activities`),
	list: (sessionId: string, timeSlotId: string) =>
		api.get<SessionActivity[]>(`/api/v1/sessions/${sessionId}/time-slots/${timeSlotId}/activities`),
	get: (sessionId: string, timeSlotId: string, id: string) =>
		api.get<SessionActivity>(`/api/v1/sessions/${sessionId}/time-slots/${timeSlotId}/activities/${id}`),
	create: (sessionId: string, timeSlotId: string, data: CreateSessionActivityRequest) =>
		api.post<SessionActivity>(`/api/v1/sessions/${sessionId}/time-slots/${timeSlotId}/activities`, data),
	update: (sessionId: string, timeSlotId: string, id: string, data: UpdateSessionActivityRequest) =>
		api.put<SessionActivity>(`/api/v1/sessions/${sessionId}/time-slots/${timeSlotId}/activities/${id}`, data),
	delete: (sessionId: string, timeSlotId: string, id: string) =>
		api.delete<void>(`/api/v1/sessions/${sessionId}/time-slots/${timeSlotId}/activities/${id}`),
	copyFromTimeSlot: (sessionId: string, targetTimeSlotId: string, sourceTimeSlotId: string) =>
		api.post<SessionActivity[]>(`/api/v1/sessions/${sessionId}/time-slots/${targetTimeSlotId}/copy-activities`, {
			source_session_time_slot_id: sourceTimeSlotId,
		}),
};

export const counselorApi = {
	list: () => api.get<Counselor[]>("/api/v1/counselors"),
	get: (id: string) => api.get<Counselor>(`/api/v1/counselors/${id}`),
	create: (data: CreateCounselorRequest) =>
		api.post<Counselor>("/api/v1/counselors", data),
	update: (id: string, data: UpdateCounselorRequest) =>
		api.put<Counselor>(`/api/v1/counselors/${id}`, data),
	delete: (id: string) => api.delete<void>(`/api/v1/counselors/${id}`),
};

export const counselorCertificationApi = {
	list: (counselorId: string) =>
		api.get<CounselorCertification[]>(`/api/v1/counselors/${counselorId}/certifications`),
	add: (counselorId: string, data: AddCounselorCertificationRequest) =>
		api.post<CounselorCertification>(`/api/v1/counselors/${counselorId}/certifications`, data),
	remove: (counselorId: string, id: string) =>
		api.delete<void>(`/api/v1/counselors/${counselorId}/certifications/${id}`),
};

export const sessionHistoryApi = {
	list: (counselorId: string) =>
		api.get<SessionHistory[]>(`/api/v1/counselors/${counselorId}/session-history`),
	get: (counselorId: string, id: string) =>
		api.get<SessionHistory>(`/api/v1/counselors/${counselorId}/session-history/${id}`),
	create: (counselorId: string, data: CreateSessionHistoryRequest) =>
		api.post<SessionHistory>(`/api/v1/counselors/${counselorId}/session-history`, data),
	update: (counselorId: string, id: string, data: UpdateSessionHistoryRequest) =>
		api.put<SessionHistory>(`/api/v1/counselors/${counselorId}/session-history/${id}`, data),
	delete: (counselorId: string, id: string) =>
		api.delete<void>(`/api/v1/counselors/${counselorId}/session-history/${id}`),
	summary: (counselorId: string, seasonId?: string) => {
		const params = seasonId ? `?season_id=${seasonId}` : "";
		return api.get<HistorySummary[]>(`/api/v1/counselors/${counselorId}/history${params}`);
	},
};

export const ageGroupPreferenceApi = {
	list: (sessionId: string, counselorId: string, signal?: AbortSignal) =>
		api.get<AgeGroupPreference[]>(`/api/v1/sessions/${sessionId}/counselors/${counselorId}/age-group-preferences`, { signal }),
	replaceAll: (sessionId: string, counselorId: string, data: AgeGroupPreferenceItem[]) =>
		api.put<AgeGroupPreference[]>(`/api/v1/sessions/${sessionId}/counselors/${counselorId}/age-group-preferences`, data),
};

export const cocounselorPreferenceApi = {
	list: (sessionId: string, counselorId: string, signal?: AbortSignal) =>
		api.get<CocounselorPreference[]>(`/api/v1/sessions/${sessionId}/counselors/${counselorId}/cocounselor-preferences`, { signal }),
	replaceAll: (sessionId: string, counselorId: string, data: CocounselorPreferenceItem[]) =>
		api.put<CocounselorPreference[]>(`/api/v1/sessions/${sessionId}/counselors/${counselorId}/cocounselor-preferences`, data),
};

export const activityPreferenceApi = {
	list: (sessionId: string, counselorId: string, signal?: AbortSignal) =>
		api.get<ActivityPreference[]>(`/api/v1/sessions/${sessionId}/counselors/${counselorId}/activity-preferences`, { signal }),
	replaceAll: (sessionId: string, counselorId: string, data: ActivityPreferenceItem[]) =>
		api.put<ActivityPreference[]>(`/api/v1/sessions/${sessionId}/counselors/${counselorId}/activity-preferences`, data),
};
