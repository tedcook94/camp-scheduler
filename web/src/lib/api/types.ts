export interface TokenResponse {
	access_token: string;
	refresh_token: string;
}

export interface Camp {
	id: string;
	name: string;
	location: string | null;
	enabled: boolean;
}

export interface User {
	id: string;
	camp_id: string | null;
	username: string;
	email: string;
	first_name: string;
	last_name: string;
	role: "admin" | "super_admin";
	created_at: string;
	updated_at: string;
}

export interface CreateCampRequest {
	name: string;
	location?: string | null;
}

export interface UpdateCampRequest {
	name: string;
	location?: string | null;
	enabled?: boolean;
}

export interface CreateUserRequest {
	camp_id?: string | null;
	username: string;
	email: string;
	password: string;
	first_name: string;
	last_name: string;
	role: "admin" | "super_admin";
}

export interface UpdateUserRequest {
	camp_id?: string | null;
	username: string;
	email: string;
	first_name: string;
	last_name: string;
	role: "admin" | "super_admin";
}

export interface UpdatePasswordRequest {
	password: string;
}

export interface Certification {
	id: string;
	camp_id: string;
	name: string;
}

export interface AgeGroup {
	id: string;
	camp_id: string;
	name: string;
}

export interface ApiError {
	error: string;
}

export interface Cabin {
	id: string;
	camp_id: string;
	default_age_group_id: string;
	default_age_group_name: string;
	name: string;
}

export interface CreateCabinRequest {
	name: string;
	default_age_group_id: string;
}

export interface UpdateCabinRequest {
	name: string;
	default_age_group_id: string;
}

export interface Season {
	id: string;
	camp_id: string;
	name: string;
	start_date: string;
	end_date: string;
}

export interface CreateSeasonRequest {
	name: string;
	start_date: string;
	end_date: string;
}

export interface UpdateSeasonRequest {
	name: string;
	start_date: string;
	end_date: string;
}

export interface Session {
	id: string;
	camp_id: string;
	season_id: string;
	name: string;
	previous_session_id: string | null;
}

export interface CreateSessionRequest {
	name: string;
	season_id: string;
	previous_session_id?: string | null;
}

export interface UpdateSessionRequest {
	name: string;
	season_id: string;
	previous_session_id?: string | null;
}

export interface Activity {
	id: string;
	camp_id: string;
	name: string;
}

export interface ActivityCertification {
	id: string;
	camp_id: string;
	activity_id: string;
	certification_id: string;
	certification_name?: string;
}

export interface TimeSlot {
	id: string;
	camp_id: string;
	name: string;
}

export interface SessionTimeSlot {
	id: string;
	camp_id: string;
	session_id: string;
	time_slot_id: string;
	sort_order: number;
}

export interface CreateSessionTimeSlotRequest {
	time_slot_id: string;
	sort_order: number;
}

export interface UpdateSessionTimeSlotRequest {
	time_slot_id: string;
	sort_order: number;
}

export interface SessionActivity {
	id: string;
	camp_id: string;
	session_time_slot_id: string;
	activity_id: string;
	capacity: number;
	required_counselors: number;
}

export interface CreateSessionActivityRequest {
	activity_id: string;
	capacity: number;
	required_counselors: number;
}

export interface UpdateSessionActivityRequest {
	activity_id: string;
	capacity: number;
	required_counselors: number;
}

export interface Counselor {
	id: string;
	camp_id: string;
	name: string;
	junior_counselor: boolean;
	enabled: boolean;
}

export interface CreateCounselorRequest {
	name: string;
	junior_counselor: boolean;
}

export interface UpdateCounselorRequest {
	name: string;
	junior_counselor: boolean;
	enabled: boolean;
}

export interface CounselorCertification {
	id: string;
	camp_id: string;
	counselor_id: string;
	certification_id: string;
	certification_name?: string;
}

export interface AddCounselorCertificationRequest {
	certification_id: string;
}

export interface SessionHistory {
	id: string;
	camp_id: string;
	counselor_id: string;
	session_id: string;
	age_group_id: string;
	cabin_id?: string;
}

export interface CreateSessionHistoryRequest {
	session_id: string;
	age_group_id: string;
	cabin_id?: string;
}

export interface UpdateSessionHistoryRequest {
	session_id: string;
	age_group_id: string;
	cabin_id?: string;
}

export interface HistorySummary {
	id: string;
	session_name: string;
	season_id: string;
	season_name: string;
	age_group_name: string;
	cabin_name?: string;
}

export interface AgeGroupPreference {
	id: string;
	camp_id: string;
	counselor_id: string;
	session_id: string;
	age_group_id: string;
	rank: number;
}

export interface AgeGroupPreferenceItem {
	age_group_id: string;
	rank: number;
}

export interface CocounselorPreference {
	id: string;
	camp_id: string;
	counselor_id: string;
	session_id: string;
	preferred_counselor_id: string;
	rank: number;
}

export interface CocounselorPreferenceItem {
	preferred_counselor_id: string;
	rank: number;
}

export interface ActivityPreference {
	id: string;
	camp_id: string;
	counselor_id: string;
	session_id: string;
	activity_id: string;
	rank: number;
}

export interface ActivityPreferenceItem {
	activity_id: string;
	rank: number;
}

export interface Camper {
	id: string;
	camp_id: string;
	name: string;
}

export interface CreateCamperRequest {
	name: string;
}

export interface UpdateCamperRequest {
	name: string;
}

export interface SessionAgeGroup {
	id: string;
	camp_id: string;
	session_id: string;
	age_group_id: string;
}

export interface CreateSessionAgeGroupRequest {
	age_group_id: string;
}

export interface UpdateSessionAgeGroupRequest {
	age_group_id: string;
}

export interface SessionCabin {
	id: string;
	camp_id: string;
	session_id: string;
	session_age_group_id: string;
	cabin_id: string;
	group_size: number | null;
	required_counselors: number | null;
}

export interface CreateSessionCabinRequest {
	session_age_group_id: string;
	cabin_id: string;
	group_size?: number | null;
	required_counselors?: number | null;
}

export interface UpdateSessionCabinRequest {
	cabin_id: string;
	group_size?: number | null;
	required_counselors?: number | null;
}

export interface Enrollment {
	id: string;
	camp_id: string;
	camper_id: string;
	session_age_group_id: string;
	camper_name: string;
	session_id: string;
	age_group_id: string;
}

export interface CreateEnrollmentRequest {
	camper_id: string;
	session_age_group_id: string;
}

export interface CamperFriendPreference {
	id: string;
	camp_id: string;
	camper_id: string;
	session_id: string;
	preferred_camper_id: string;
	rank: number;
}

export interface CamperFriendPreferenceItem {
	preferred_camper_id: string;
	rank: number;
}
