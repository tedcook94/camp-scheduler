import type { AssignmentDetail } from "$lib/api/types";

export interface ActivityGroup {
	timeSlotName: string;
	activities: { activityName: string; counselors: string[] }[];
}

export interface CabinGroup {
	ageGroupName: string;
	cabins: { cabinName: string; counselors: string[]; campers: string[] }[];
}

// groupByTimeSlotActivity collapses a flat assignment list into nested
// time-slot → activity → counselor groups. Assumes assignments are already
// ordered by (time_slot.sort_order, activity_name) — which the backend
// query guarantees — so we can build groups in a single linear pass.
export function groupByTimeSlotActivity(
	assignments: AssignmentDetail[],
): ActivityGroup[] {
	const groups: ActivityGroup[] = [];
	let currentSlot: ActivityGroup | null = null;
	let currentActivity: { activityName: string; counselors: string[] } | null = null;

	for (const a of assignments) {
		const slotName = a.time_slot_name || "Unknown";
		const actName = a.activity_name || "Unknown";
		const person = a.counselor_name || a.counselor_id?.slice(0, 8) || "Unknown";

		if (!currentSlot || currentSlot.timeSlotName !== slotName) {
			currentSlot = { timeSlotName: slotName, activities: [] };
			currentActivity = null;
			groups.push(currentSlot);
		}

		if (!currentActivity || currentActivity.activityName !== actName) {
			currentActivity = { activityName: actName, counselors: [] };
			currentSlot.activities.push(currentActivity);
		}

		currentActivity.counselors.push(person);
	}

	return groups;
}

// groupByAgeGroupCabin collapses a flat assignment list (mixed counselor +
// camper rows) into nested age-group → cabin → people groups.
export function groupByAgeGroupCabin(assignments: AssignmentDetail[]): CabinGroup[] {
	const byAge = new Map<
		string,
		Map<string, { cabinName: string; counselors: string[]; campers: string[] }>
	>();

	for (const a of assignments) {
		const ageGroupName = a.age_group_name || "Unknown";
		const cabinName = a.cabin_name || "Unknown";
		const cabinKey = a.cabin_id || cabinName;

		let cabinMap = byAge.get(ageGroupName);
		if (!cabinMap) {
			cabinMap = new Map();
			byAge.set(ageGroupName, cabinMap);
		}
		let cabin = cabinMap.get(cabinKey);
		if (!cabin) {
			cabin = { cabinName, counselors: [], campers: [] };
			cabinMap.set(cabinKey, cabin);
		}

		if (a.counselor_id || a.counselor_name) {
			cabin.counselors.push(
				a.counselor_name || a.counselor_id?.slice(0, 8) || "Unknown",
			);
		} else if (a.camper_id || a.camper_name) {
			cabin.campers.push(a.camper_name || a.camper_id?.slice(0, 8) || "Unknown");
		}
	}

	const groups: CabinGroup[] = [];
	for (const [ageGroupName, cabinMap] of byAge) {
		groups.push({ ageGroupName, cabins: Array.from(cabinMap.values()) });
	}
	return groups;
}
