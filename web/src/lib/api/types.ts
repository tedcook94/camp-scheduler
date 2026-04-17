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
