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
	archived: boolean;
}

export interface AgeGroup {
	id: string;
	camp_id: string;
	name: string;
	archived: boolean;
}

export interface ApiError {
	error: string;
}

export type Gender = "male" | "female";

export interface Cabin {
	id: string;
	camp_id: string;
	default_age_group_id: string;
	default_age_group_name: string;
	name: string;
	default_group_size: number;
	default_required_counselors: number;
	gender: Gender;
	archived: boolean;
}

export interface CreateCabinRequest {
	name: string;
	default_age_group_id: string;
	default_group_size: number;
	default_required_counselors: number;
	gender: Gender;
}

export interface UpdateCabinRequest {
	name: string;
	default_age_group_id: string;
	default_group_size: number;
	default_required_counselors: number;
	gender: Gender;
}

export interface Season {
	id: string;
	camp_id: string;
	name: string;
	start_date: string;
	end_date: string;
	archived: boolean;
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
	archived: boolean;
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

export interface CopySessionRequest {
	name: string;
	season_id: string;
	previous_session_id?: string | null;
}

export interface Activity {
	id: string;
	camp_id: string;
	name: string;
	archived: boolean;
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
	archived: boolean;
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
	first_name: string;
	last_name: string;
	name: string;
	junior_counselor: boolean;
	archived: boolean;
	gender: Gender;
}

export interface CreateCounselorRequest {
	first_name: string;
	last_name: string;
	junior_counselor: boolean;
	gender: Gender;
}

export interface UpdateCounselorRequest {
	first_name: string;
	last_name: string;
	junior_counselor: boolean;
	gender: Gender;
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
	first_name: string;
	last_name: string;
	name: string;
	gender: Gender;
}

export interface CreateCamperRequest {
	first_name: string;
	last_name: string;
	gender: Gender;
}

export interface UpdateCamperRequest {
	first_name: string;
	last_name: string;
	gender: Gender;
}

export interface SessionAgeGroup {
	id: string;
	camp_id: string;
	session_id: string;
	age_group_id: string;
}

export interface SessionCounselor {
	id: string;
	camp_id: string;
	session_id: string;
	counselor_id: string;
	counselor_first_name: string;
	counselor_last_name: string;
	counselor_name: string;
	junior_counselor: boolean;
	archived: boolean;
	gender: Gender;
}

export interface AddSessionCounselorRequest {
	counselor_id: string;
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
	group_size: number;
	required_counselors: number;
}

export interface CreateSessionCabinRequest {
	session_age_group_id: string;
	cabin_id: string;
	group_size: number;
	required_counselors: number;
}

export interface UpdateSessionCabinRequest {
	cabin_id: string;
	group_size: number;
	required_counselors: number;
}

export interface Enrollment {
	id: string;
	camp_id: string;
	camper_id: string;
	session_age_group_id: string;
	camper_first_name: string;
	camper_last_name: string;
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

export interface ScoreBreakdown {
	Constraint: string;
	Score: number;
	Message: string;
}

export interface AssignmentDetail {
	id: string;
	counselor_id?: string;
	counselor_first_name?: string;
	counselor_last_name?: string;
	counselor_name?: string;
	camper_id?: string;
	camper_first_name?: string;
	camper_last_name?: string;
	camper_name?: string;
	cabin_id?: string;
	cabin_name?: string;
	age_group_name?: string;
	session_activity_id?: string;
	activity_name?: string;
	time_slot_name?: string;
	sort_order?: number;
}

export interface ExplanationDetail {
	id: string;
	counselor_id?: string;
	counselor_first_name?: string;
	counselor_last_name?: string;
	counselor_name?: string;
	camper_id?: string;
	camper_first_name?: string;
	camper_last_name?: string;
	camper_name?: string;
	explanation_type: "reason" | "unmet_preference" | "ineligible_preference";
	constraint_name: string | null;
	rank?: number | null;
	message: string;
}

export interface UnassignedTimeSlotRef {
	session_time_slot_id: string;
	time_slot_name: string;
}

export interface UnassignedCounselor {
	counselor_id: string;
	counselor_first_name?: string;
	counselor_last_name?: string;
	counselor_name: string;
	missing_time_slots?: UnassignedTimeSlotRef[];
}

export interface SolutionDetailResponse {
	id: string;
	camper_solution_id?: string;
	assignment_run_id: string;
	solution_index: number;
	score: number;
	score_breakdown: ScoreBreakdown[];
	camper_score_breakdown?: ScoreBreakdown[];
	assignments: AssignmentDetail[];
	explanations: ExplanationDetail[];
	unassigned_counselors: UnassignedCounselor[];
}

export interface SolutionResponse {
	id: string;
	camper_solution_id?: string;
	assignment_run_id: string;
	solution_index: number;
	score: number;
	score_breakdown: ScoreBreakdown[];
	camper_score_breakdown?: ScoreBreakdown[];
}

export type RunType = "cabin" | "activity_schedule";

export interface RunResponse {
	id: string;
	camp_id: string;
	session_id: string;
	run_type: RunType;
	status: "completed" | "selected";
	is_stale: boolean;
	selected_solution_id: string | null;
	created_at: string;
}

export interface RunDetailResponse extends RunResponse {
	solutions: SolutionResponse[];
}

export interface TriggerRunRequest {
	run_type: RunType;
	max_solutions?: number;
	max_iterations?: number;
	weights?: {
		returning_age_group?: number;
		returning_cabin?: number;
		cocounselor_preference?: number;
		age_group_preference?: number;
		multiple_seniors?: number;
	};
	camper_weights?: {
		friend_preference?: number;
	};
	activity_weights?: {
		activity_preference?: number;
	};
}
