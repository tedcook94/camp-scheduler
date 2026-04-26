import { api } from "./client";
import type {
	CamperCabinOverride,
	CounselorActivityOverride,
	CounselorCabinOverride,
	CreateCamperCabinOverrideRequest,
	CreateCounselorActivityOverrideRequest,
	CreateCounselorCabinOverrideRequest,
	SessionOverrides,
} from "./types";

export const overrideApi = {
	list: (sessionId: string): Promise<SessionOverrides> =>
		api.get<SessionOverrides>(`/api/v1/sessions/${sessionId}/overrides`),

	createCounselorCabin: (
		sessionId: string,
		data: CreateCounselorCabinOverrideRequest,
	): Promise<CounselorCabinOverride> =>
		api.post<CounselorCabinOverride>(
			`/api/v1/sessions/${sessionId}/overrides/counselor-cabin`,
			data,
		),

	deleteCounselorCabin: (sessionId: string, overrideId: string): Promise<void> =>
		api.delete<void>(
			`/api/v1/sessions/${sessionId}/overrides/counselor-cabin/${overrideId}`,
		),

	createCamperCabin: (
		sessionId: string,
		data: CreateCamperCabinOverrideRequest,
	): Promise<CamperCabinOverride> =>
		api.post<CamperCabinOverride>(
			`/api/v1/sessions/${sessionId}/overrides/camper-cabin`,
			data,
		),

	deleteCamperCabin: (sessionId: string, overrideId: string): Promise<void> =>
		api.delete<void>(
			`/api/v1/sessions/${sessionId}/overrides/camper-cabin/${overrideId}`,
		),

	createCounselorActivity: (
		sessionId: string,
		data: CreateCounselorActivityOverrideRequest,
	): Promise<CounselorActivityOverride> =>
		api.post<CounselorActivityOverride>(
			`/api/v1/sessions/${sessionId}/overrides/counselor-activity`,
			data,
		),

	deleteCounselorActivity: (sessionId: string, overrideId: string): Promise<void> =>
		api.delete<void>(
			`/api/v1/sessions/${sessionId}/overrides/counselor-activity/${overrideId}`,
		),
};
