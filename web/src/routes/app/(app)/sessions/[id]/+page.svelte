<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { page } from "$app/state";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import {
		sessionApi,
		sessionTimeSlotApi,
		sessionActivityApi,
		sessionAgeGroupApi,
		sessionCabinApi,
		timeSlotApi,
		activityApi,
		ageGroupApi,
		cabinApi,
	} from "$lib/api";
	import type {
		Session,
		SessionTimeSlot,
		SessionActivity,
		SessionAgeGroup,
		SessionCabin,
		TimeSlot,
		Activity,
		AgeGroup,
		Cabin,
	} from "$lib/api/types";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import * as Tabs from "$lib/components/ui/tabs";
	import { parsePositiveInt } from "$lib/utils";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import CopyIcon from "@lucide/svelte/icons/copy";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
	import GripVerticalIcon from "@lucide/svelte/icons/grip-vertical";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	let disabled = $derived.by(() => getCampDisabled());

	const sessionId = page.params.id!;

	const TAB_VALUES = ["cabins", "activities"] as const;
	type TabValue = (typeof TAB_VALUES)[number];

	let activeTab = $derived.by<TabValue>(() => {
		const t = page.url.searchParams.get("tab");
		return TAB_VALUES.includes(t as TabValue) ? (t as TabValue) : "cabins";
	});

	function setActiveTab(value: string) {
		const url = new URL(page.url);
		if (value === "cabins") {
			url.searchParams.delete("tab");
		} else {
			url.searchParams.set("tab", value);
		}
		goto(url, { replaceState: true, keepFocus: true, noScroll: true });
	}

	let session = $state<Session | null>(null);
	let sessionTimeSlots = $state<SessionTimeSlot[]>([]);
	let allTimeSlots = $state<TimeSlot[]>([]);
	let allActivities = $state<Activity[]>([]);
	// Map from session_time_slot id to its activities
	let activitiesByTimeSlot = $state<Map<string, SessionActivity[]>>(new Map());
	let sessionAgeGroups = $state<SessionAgeGroup[]>([]);
	let sessionCabins = $state<SessionCabin[]>([]);
	let allAgeGroups = $state<AgeGroup[]>([]);
	let allCabins = $state<Cabin[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	// Lookup maps
	let timeSlotMap = $derived(new Map(allTimeSlots.map((t) => [t.id, t.name])));
	let activityMap = $derived(new Map(allActivities.map((a) => [a.id, a.name])));
	let ageGroupMap = $derived(new Map(allAgeGroups.map((a) => [a.id, a.name])));
	let cabinMap = $derived(new Map(allCabins.map((c) => [c.id, c])));

	// Sorted session time slots by DB sort_order
	let sortedSessionTimeSlots = $derived(
		[...sessionTimeSlots].sort((a, b) => a.sort_order - b.sort_order)
	);

	// Available time slots (not yet assigned to this session), sorted alphabetically
	let availableTimeSlots = $derived(
		allTimeSlots
			.filter((t) => !sessionTimeSlots.some((st) => st.time_slot_id === t.id))
			.sort((a, b) => a.name.localeCompare(b.name))
	);

	// Add time slot dialog
	let addTimeSlotOpen = $state(false);
	let addTimeSlotId = $state("");
	let addingTimeSlot = $state(false);

	// Remove time slot confirmation
	let removeTimeSlotOpen = $state(false);
	let removeTimeSlotTarget = $state<SessionTimeSlot | null>(null);
	let removingTimeSlot = $state(false);

	// Add activity dialog
	let addActivityOpen = $state(false);
	let addActivityTimeSlotId = $state(""); // the session_time_slot id
	let addActivityId = $state("");
	let addActivityCapacity = $state("");
	let addActivityCounselors = $state("");
	let addingActivity = $state(false);
	let activityError = $state("");
	let capacityError = $state("");
	let counselorsError = $state("");

	// Remove activity confirmation
	let removeActivityOpen = $state(false);
	let removeActivityTarget = $state<{
		sessionTimeSlot: SessionTimeSlot;
		activity: SessionActivity;
	} | null>(null);
	let removingActivity = $state(false);

	// Edit activity dialog
	let editActivityOpen = $state(false);
	let editActivityTarget = $state<{
		sessionTimeSlot: SessionTimeSlot;
		activity: SessionActivity;
	} | null>(null);
	let editActivityCapacity = $state("");
	let editActivityCounselors = $state("");
	let editingActivity = $state(false);
	let editCapacityError = $state("");
	let editCounselorsError = $state("");

	// Copy activities dialog
	let copyActivitiesOpen = $state(false);
	let copySourceTimeSlotId = $state("");
	// Target can be an existing session time slot ID (prefixed "existing:") or an unassigned time slot ID (prefixed "new:")
	let copyTargetValue = $state("");
	let copyingActivities = $state(false);

	// --- Cabins state ---

	// session_age_groups sorted by underlying age-group name
	let sortedSessionAgeGroups = $derived(
		[...sessionAgeGroups].sort((a, b) =>
			(ageGroupMap.get(a.age_group_id) ?? "").localeCompare(
				ageGroupMap.get(b.age_group_id) ?? ""
			)
		)
	);

	// Cabins grouped by session_age_group_id, sorted alphabetically by cabin name
	let cabinsByAgeGroup = $derived.by(() => {
		const m = new Map<string, SessionCabin[]>();
		for (const sag of sessionAgeGroups) {
			m.set(sag.id, []);
		}
		for (const sc of sessionCabins) {
			const list = m.get(sc.session_age_group_id);
			if (list) list.push(sc);
		}
		for (const list of m.values()) {
			list.sort((a, b) =>
				(cabinMap.get(a.cabin_id)?.name ?? "").localeCompare(
					cabinMap.get(b.cabin_id)?.name ?? ""
				)
			);
		}
		return m;
	});

	// Only render age-group cards that have at least one cabin assigned
	let nonEmptyAgeGroups = $derived(
		sortedSessionAgeGroups.filter(
			(sag) => (cabinsByAgeGroup.get(sag.id) ?? []).length > 0
		)
	);

	// Cabins not yet assigned anywhere in this session
	let availableCabinsForSession = $derived.by(() => {
		const assigned = new Set(sessionCabins.map((sc) => sc.cabin_id));
		return allCabins
			.filter((c) => !assigned.has(c.id))
			.sort((a, b) => a.name.localeCompare(b.name));
	});

	// Collapsible age-group cards (only those with cabins)
	let expandedAgeGroups = $state<Set<string>>(new Set());
	let allAgeGroupsExpanded = $derived(
		nonEmptyAgeGroups.length > 0 &&
			nonEmptyAgeGroups.every((sag) => expandedAgeGroups.has(sag.id))
	);

	function toggleExpandAllAgeGroups() {
		if (allAgeGroupsExpanded) {
			expandedAgeGroups = new Set();
		} else {
			expandedAgeGroups = new Set(nonEmptyAgeGroups.map((sag) => sag.id));
		}
	}

	// Add cabin dialog
	let addCabinOpen = $state(false);
	let addCabinId = $state("");
	let addCabinAgeGroupId = $state("");
	let addCabinGroupSize = $state("");
	let addCabinCounselors = $state("");
	let addingCabin = $state(false);
	let addCabinError = $state("");
	let addCabinAgeGroupError = $state("");
	let addCabinGroupSizeError = $state("");
	let addCabinCounselorsError = $state("");

	// Edit cabin dialog
	let editCabinOpen = $state(false);
	let editCabinTarget = $state<SessionCabin | null>(null);
	let editCabinGroupSize = $state("");
	let editCabinCounselors = $state("");
	let editingCabin = $state(false);
	let editCabinGroupSizeError = $state("");
	let editCabinCounselorsError = $state("");

	// Remove cabin confirmation
	let removeCabinOpen = $state(false);
	let removeCabinTarget = $state<SessionCabin | null>(null);
	let removingCabin = $state(false);

	// Collapsible time slot cards — all collapsed by default
	let expandedTimeSlots = $state<Set<string>>(new Set());
	let allExpanded = $derived(
		sortedSessionTimeSlots.length > 0 &&
			sortedSessionTimeSlots.every((st) => expandedTimeSlots.has(st.id))
	);

	function toggleExpandAll() {
		if (allExpanded) {
			expandedTimeSlots = new Set();
		} else {
			expandedTimeSlots = new Set(sessionTimeSlots.map((st) => st.id));
		}
	}

	// Drag-and-drop reordering
	let draggedId = $state<string | null>(null);
	let dropInsert = $state<{ id: string; position: "before" | "after" } | null>(null);
	let reordering = $state(false);

	function handleDragStart(e: DragEvent, id: string) {
		draggedId = id;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = "move";
		}
	}

	function handleDragOver(e: DragEvent, id: string) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = "move";
		}
		if (!draggedId || draggedId === id) return;
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const midY = rect.top + rect.height / 2;
		const position = e.clientY < midY ? "before" : "after";
		dropInsert = { id, position };
	}

	function handleDragLeave(e: DragEvent) {
		const related = e.relatedTarget as Node | null;
		if (related && (e.currentTarget as HTMLElement).contains(related)) return;
		dropInsert = null;
	}

	async function handleDrop(e: DragEvent, targetId: string) {
		e.preventDefault();
		const insert = dropInsert;
		dropInsert = null;
		if (!draggedId || draggedId === targetId || reordering || !insert) return;

		const sorted = [...sortedSessionTimeSlots];
		const fromIdx = sorted.findIndex((st) => st.id === draggedId);
		let toIdx = sorted.findIndex((st) => st.id === targetId);
		if (fromIdx === -1 || toIdx === -1) return;

		const [moved] = sorted.splice(fromIdx, 1);
		// Adjust target index after removal
		if (insert.position === "after") {
			toIdx = sorted.findIndex((st) => st.id === targetId) + 1;
		} else {
			toIdx = sorted.findIndex((st) => st.id === targetId);
		}
		sorted.splice(toIdx, 0, moved);

		const updated = sorted.map((st, i) => ({ ...st, sort_order: i + 1 }));
		const snapshot = sessionTimeSlots;
		sessionTimeSlots = updated;
		draggedId = null;

		reordering = true;
		try {
			await sessionTimeSlotApi.reorder(
				sessionId,
				updated.map((st) => st.id)
			);
		} catch (err) {
			sessionTimeSlots = snapshot;
			const message =
				err instanceof ApiClientError ? err.message : "Failed to reorder time slots";
			toast.error(message);
		} finally {
			reordering = false;
		}
	}

	function handleDragEnd() {
		draggedId = null;
		dropInsert = null;
	}

	// Available activities for a given session time slot (not yet assigned)
	function getAvailableActivities(sessionTimeSlotId: string): Activity[] {
		const assigned = activitiesByTimeSlot.get(sessionTimeSlotId) ?? [];
		const assignedIds = new Set(assigned.map((a) => a.activity_id));
		return allActivities.filter((a) => !assignedIds.has(a.id));
	}

	onMount(async () => {
		try {
			const [s, stSlots, ts, acts, allSessionActs, sAgeGroups, sCabins, ags, cbs] =
				await Promise.all([
					sessionApi.get(sessionId),
					sessionTimeSlotApi.list(sessionId),
					timeSlotApi.list(),
					activityApi.list(),
					sessionActivityApi.listAll(sessionId),
					sessionAgeGroupApi.list(sessionId),
					sessionCabinApi.list(sessionId),
					ageGroupApi.list(),
					cabinApi.list(),
				]);
			session = s;
			sessionTimeSlots = stSlots;
			allTimeSlots = ts;
			allActivities = acts;
			sessionAgeGroups = sAgeGroups;
			sessionCabins = sCabins;
			allAgeGroups = ags;
			allCabins = cbs;

			// Group activities by session time slot
			const activitiesByTimeSlotMap = new Map<string, SessionActivity[]>();
			for (const st of stSlots) {
				activitiesByTimeSlotMap.set(st.id, []);
			}
			for (const sa of allSessionActs) {
				const list = activitiesByTimeSlotMap.get(sa.session_time_slot_id);
				if (list) {
					list.push(sa);
				}
			}
			activitiesByTimeSlot = activitiesByTimeSlotMap;
		} catch (err) {
			const message =
				err instanceof ApiClientError
					? err.message
					: "Failed to load session details";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	// --- Time slot management ---

	function openAddTimeSlot() {
		addTimeSlotId = "";
		addTimeSlotOpen = true;
	}

	async function handleAddTimeSlot() {
		if (!addTimeSlotId) return;
		addingTimeSlot = true;

		try {
			const nextOrder =
				sessionTimeSlots.length > 0
					? Math.max(...sessionTimeSlots.map((st) => st.sort_order)) + 1
					: 1;
			const created = await sessionTimeSlotApi.create(sessionId, {
				time_slot_id: addTimeSlotId,
				sort_order: nextOrder,
			});
			sessionTimeSlots = [...sessionTimeSlots, created];
			activitiesByTimeSlot = new Map([
				...activitiesByTimeSlot,
				[created.id, []],
			]);
			expandedTimeSlots = new Set([...expandedTimeSlots, created.id]);
			toast.success("Time slot added");
			addTimeSlotOpen = false;
		} catch (err) {
			const message =
				err instanceof ApiClientError
					? err.message
					: "Failed to add time slot";
			toast.error(message);
		} finally {
			addingTimeSlot = false;
		}
	}

	function confirmRemoveTimeSlot(st: SessionTimeSlot) {
		removeTimeSlotTarget = st;
		removeTimeSlotOpen = true;
	}

	async function handleRemoveTimeSlot() {
		if (!removeTimeSlotTarget) return;
		removingTimeSlot = true;

		try {
			await sessionTimeSlotApi.delete(sessionId, removeTimeSlotTarget.id);
			sessionTimeSlots = sessionTimeSlots.filter(
				(st) => st.id !== removeTimeSlotTarget!.id
			);
			const updated = new Map(activitiesByTimeSlot);
			updated.delete(removeTimeSlotTarget.id);
			activitiesByTimeSlot = updated;
			const updatedExpanded = new Set(expandedTimeSlots);
			updatedExpanded.delete(removeTimeSlotTarget.id);
			expandedTimeSlots = updatedExpanded;
			toast.success("Time slot removed");
			removeTimeSlotOpen = false;
			removeTimeSlotTarget = null;
		} catch (err) {
			const message =
				err instanceof ApiClientError
					? err.message
					: "Failed to remove time slot";
			toast.error(message);
		} finally {
			removingTimeSlot = false;
		}
	}

	// --- Activity management ---

	function openAddActivity(sessionTimeSlotId: string) {
		addActivityTimeSlotId = sessionTimeSlotId;
		addActivityId = "";
		addActivityCapacity = "";
		addActivityCounselors = "";
		activityError = "";
		capacityError = "";
		counselorsError = "";
		addActivityOpen = true;
	}

	async function handleAddActivity(e: SubmitEvent) {
		e.preventDefault();
		activityError = "";
		capacityError = "";
		counselorsError = "";

		let valid = true;
		if (!addActivityId) {
			activityError = "Activity is required.";
			valid = false;
		}
		const capacityResult = parsePositiveInt(addActivityCapacity);
		if (!capacityResult.ok) {
			capacityError = "Capacity must be at least 1.";
			valid = false;
		}
		const counselorsResult = parsePositiveInt(addActivityCounselors);
		if (!counselorsResult.ok) {
			counselorsError = "Required counselors must be at least 1.";
			valid = false;
		}
		if (capacityResult.ok && counselorsResult.ok && counselorsResult.value > capacityResult.value) {
			counselorsError = "Required counselors cannot exceed capacity.";
			valid = false;
		}
		if (!valid) return;

		const capacity = capacityResult.ok ? capacityResult.value : 0;
		const counselors = counselorsResult.ok ? counselorsResult.value : 0;

		addingActivity = true;

		try {
			const created = await sessionActivityApi.create(
				sessionId,
				addActivityTimeSlotId,
				{
					activity_id: addActivityId,
					capacity,
					required_counselors: counselors,
				}
			);
			const current = activitiesByTimeSlot.get(addActivityTimeSlotId) ?? [];
			activitiesByTimeSlot = new Map([
				...activitiesByTimeSlot,
				[addActivityTimeSlotId, [...current, created]],
			]);
			toast.success("Activity added");
			addActivityOpen = false;
		} catch (err) {
			const message =
				err instanceof ApiClientError
					? err.message
					: "Failed to add activity";
			toast.error(message);
		} finally {
			addingActivity = false;
		}
	}

	function confirmRemoveActivity(
		sessionTimeSlot: SessionTimeSlot,
		activity: SessionActivity
	) {
		removeActivityTarget = { sessionTimeSlot, activity };
		removeActivityOpen = true;
	}

	async function handleRemoveActivity() {
		if (!removeActivityTarget) return;
		removingActivity = true;
		const { sessionTimeSlot, activity } = removeActivityTarget;

		try {
			await sessionActivityApi.delete(
				sessionId,
				sessionTimeSlot.id,
				activity.id
			);
			const current =
				activitiesByTimeSlot.get(sessionTimeSlot.id) ?? [];
			activitiesByTimeSlot = new Map([
				...activitiesByTimeSlot,
				[
					sessionTimeSlot.id,
					current.filter((a) => a.id !== activity.id),
				],
			]);
			toast.success("Activity removed");
			removeActivityOpen = false;
			removeActivityTarget = null;
		} catch (err) {
			const message =
				err instanceof ApiClientError
					? err.message
					: "Failed to remove activity";
			toast.error(message);
		} finally {
			removingActivity = false;
		}
	}
	// --- Edit activity ---

	function openEditActivity(sessionTimeSlot: SessionTimeSlot, activity: SessionActivity) {
		editActivityTarget = { sessionTimeSlot, activity };
		editActivityCapacity = String(activity.capacity);
		editActivityCounselors = String(activity.required_counselors);
		editCapacityError = "";
		editCounselorsError = "";
		editActivityOpen = true;
	}

	async function handleEditActivity(e: SubmitEvent) {
		e.preventDefault();
		if (!editActivityTarget) return;
		editCapacityError = "";
		editCounselorsError = "";

		let valid = true;
		const capacityResult = parsePositiveInt(editActivityCapacity);
		if (!capacityResult.ok) {
			editCapacityError = "Capacity must be at least 1.";
			valid = false;
		}
		const counselorsResult = parsePositiveInt(editActivityCounselors);
		if (!counselorsResult.ok) {
			editCounselorsError = "Required counselors must be at least 1.";
			valid = false;
		}
		if (capacityResult.ok && counselorsResult.ok && counselorsResult.value > capacityResult.value) {
			editCounselorsError = "Required counselors cannot exceed capacity.";
			valid = false;
		}
		if (!valid) return;

		const capacity = capacityResult.ok ? capacityResult.value : 0;
		const counselors = counselorsResult.ok ? counselorsResult.value : 0;

		editingActivity = true;
		const { sessionTimeSlot, activity } = editActivityTarget;

		try {
			const updated = await sessionActivityApi.update(
				sessionId,
				sessionTimeSlot.id,
				activity.id,
				{
					activity_id: activity.activity_id,
					capacity,
					required_counselors: counselors,
				}
			);
			const current = activitiesByTimeSlot.get(sessionTimeSlot.id) ?? [];
			activitiesByTimeSlot = new Map([
				...activitiesByTimeSlot,
				[sessionTimeSlot.id, current.map((a) => (a.id === activity.id ? updated : a))],
			]);
			toast.success("Activity updated");
			editActivityOpen = false;
			editActivityTarget = null;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to update activity";
			toast.error(message);
		} finally {
			editingActivity = false;
		}
	}

	// --- Copy activities ---

	function openCopyActivities(sourceSessionTimeSlotId: string) {
		copySourceTimeSlotId = sourceSessionTimeSlotId;
		copyTargetValue = "";
		copyActivitiesOpen = true;
	}

	function getCopyTargetActivityCount(): number {
		if (!copyTargetValue || !copyTargetValue.startsWith("existing:")) return 0;
		const sessionTimeSlotId = copyTargetValue.slice("existing:".length);
		return (activitiesByTimeSlot.get(sessionTimeSlotId) ?? []).length;
	}

	async function handleCopyActivities() {
		if (!copySourceTimeSlotId || !copyTargetValue) return;
		copyingActivities = true;

		let createdTimeSlotId: string | null = null;

		try {
			let targetSessionTimeSlotId: string;

			if (copyTargetValue.startsWith("new:")) {
				const timeSlotId = copyTargetValue.slice("new:".length);
				const nextOrder =
					sessionTimeSlots.length > 0
						? Math.max(...sessionTimeSlots.map((st) => st.sort_order)) + 1
						: 1;
				const created = await sessionTimeSlotApi.create(sessionId, {
					time_slot_id: timeSlotId,
					sort_order: nextOrder,
				});
				createdTimeSlotId = created.id;
				sessionTimeSlots = [...sessionTimeSlots, created];
				expandedTimeSlots = new Set([...expandedTimeSlots, created.id]);
				targetSessionTimeSlotId = created.id;
			} else {
				targetSessionTimeSlotId = copyTargetValue.slice("existing:".length);
			}

			const newActivities = await sessionActivityApi.copyFromTimeSlot(
				sessionId,
				targetSessionTimeSlotId,
				copySourceTimeSlotId
			);
			activitiesByTimeSlot = new Map([
				...activitiesByTimeSlot,
				[targetSessionTimeSlotId, newActivities],
			]);
			toast.success("Activities copied");
			copyActivitiesOpen = false;
		} catch (err) {
			// Roll back the newly created time slot if the copy failed
			if (createdTimeSlotId) {
				try {
					await sessionTimeSlotApi.delete(sessionId, createdTimeSlotId);
					sessionTimeSlots = sessionTimeSlots.filter((st) => st.id !== createdTimeSlotId);
					const updated = new Map(activitiesByTimeSlot);
					updated.delete(createdTimeSlotId);
					activitiesByTimeSlot = updated;
					const updatedExpanded = new Set(expandedTimeSlots);
					updatedExpanded.delete(createdTimeSlotId);
					expandedTimeSlots = updatedExpanded;
				} catch {
					// Rollback failed — the empty time slot remains; user can remove it manually
				}
			}
			const message =
				err instanceof ApiClientError ? err.message : "Failed to copy activities";
			toast.error(message);
		} finally {
			copyingActivities = false;
		}
	}

	// --- Cabin management ---

	function openAddCabin() {
		addCabinId = "";
		addCabinAgeGroupId = "";
		addCabinGroupSize = "";
		addCabinCounselors = "";
		addCabinError = "";
		addCabinAgeGroupError = "";
		addCabinGroupSizeError = "";
		addCabinCounselorsError = "";
		addCabinOpen = true;
	}

	// When the cabin selection changes, default the age-group select to the
	// chosen cabin's default_age_group_id and seed group size + required
	// counselors from the cabin defaults.
	function onAddCabinSelected(cabinId: string) {
		addCabinId = cabinId;
		addCabinError = "";
		const cabin = cabinMap.get(cabinId);
		if (cabin) {
			addCabinAgeGroupId = cabin.default_age_group_id;
			addCabinAgeGroupError = "";
			addCabinGroupSize = String(cabin.default_group_size);
			addCabinCounselors = String(cabin.default_required_counselors);
			addCabinGroupSizeError = "";
			addCabinCounselorsError = "";
		}
	}

	async function handleAddCabin(e: SubmitEvent) {
		e.preventDefault();
		addCabinError = "";
		addCabinAgeGroupError = "";
		addCabinGroupSizeError = "";
		addCabinCounselorsError = "";

		let valid = true;
		if (!addCabinId) {
			addCabinError = "Cabin is required.";
			valid = false;
		}
		if (!addCabinAgeGroupId) {
			addCabinAgeGroupError = "Age group is required.";
			valid = false;
		}
		const sizeResult = parsePositiveInt(addCabinGroupSize);
		if (!sizeResult.ok) {
			addCabinGroupSizeError = sizeResult.error;
			valid = false;
		}
		const counselorsResult = parsePositiveInt(addCabinCounselors);
		if (!counselorsResult.ok) {
			addCabinCounselorsError = counselorsResult.error;
			valid = false;
		}
		if (!valid || !sizeResult.ok || !counselorsResult.ok) return;

		addingCabin = true;
		// Track a session_age_group we create so we can roll it back on cabin failure
		let createdSessionAgeGroupId: string | null = null;
		try {
			let sag = sessionAgeGroups.find((s) => s.age_group_id === addCabinAgeGroupId);
			if (!sag) {
				sag = await sessionAgeGroupApi.create(sessionId, {
					age_group_id: addCabinAgeGroupId,
				});
				createdSessionAgeGroupId = sag.id;
				sessionAgeGroups = [...sessionAgeGroups, sag];
			}

			const created = await sessionCabinApi.create(sessionId, {
				session_age_group_id: sag.id,
				cabin_id: addCabinId,
				group_size: sizeResult.value,
				required_counselors: counselorsResult.value,
			});
			sessionCabins = [...sessionCabins, created];
			expandedAgeGroups = new Set([...expandedAgeGroups, sag.id]);
			toast.success("Cabin added");
			addCabinOpen = false;
		} catch (err) {
			// Roll back the session_age_group we just created (if any)
			if (createdSessionAgeGroupId) {
				try {
					await sessionAgeGroupApi.delete(sessionId, createdSessionAgeGroupId);
					sessionAgeGroups = sessionAgeGroups.filter(
						(s) => s.id !== createdSessionAgeGroupId
					);
				} catch {
					// Rollback failed — leave the empty session_age_group; harmless
				}
			}
			const message =
				err instanceof ApiClientError ? err.message : "Failed to add cabin";
			toast.error(message);
		} finally {
			addingCabin = false;
		}
	}

	function openEditCabin(sc: SessionCabin) {
		editCabinTarget = sc;
		editCabinGroupSize = String(sc.group_size);
		editCabinCounselors = String(sc.required_counselors);
		editCabinGroupSizeError = "";
		editCabinCounselorsError = "";
		editCabinOpen = true;
	}

	async function handleEditCabin(e: SubmitEvent) {
		e.preventDefault();
		if (!editCabinTarget) return;
		editCabinGroupSizeError = "";
		editCabinCounselorsError = "";

		const sizeResult = parsePositiveInt(editCabinGroupSize);
		if (!sizeResult.ok) {
			editCabinGroupSizeError = sizeResult.error;
		}
		const counselorsResult = parsePositiveInt(editCabinCounselors);
		if (!counselorsResult.ok) {
			editCabinCounselorsError = counselorsResult.error;
		}
		if (!sizeResult.ok || !counselorsResult.ok) return;

		editingCabin = true;
		const target = editCabinTarget;
		try {
			const updated = await sessionCabinApi.update(sessionId, target.id, {
				cabin_id: target.cabin_id,
				group_size: sizeResult.value,
				required_counselors: counselorsResult.value,
			});
			sessionCabins = sessionCabins.map((sc) => (sc.id === target.id ? updated : sc));
			toast.success("Cabin updated");
			editCabinOpen = false;
			editCabinTarget = null;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to update cabin";
			toast.error(message);
		} finally {
			editingCabin = false;
		}
	}

	function confirmRemoveCabin(sc: SessionCabin) {
		removeCabinTarget = sc;
		removeCabinOpen = true;
	}

	async function handleRemoveCabin() {
		if (!removeCabinTarget) return;
		removingCabin = true;
		const target = removeCabinTarget;
		try {
			await sessionCabinApi.delete(sessionId, target.id);
			const remainingCabins = sessionCabins.filter((sc) => sc.id !== target.id);
			sessionCabins = remainingCabins;

			// If this was the last cabin under its session_age_group, delete the
			// session_age_group too — the user no longer manages those directly.
			const ageGroupStillUsed = remainingCabins.some(
				(sc) => sc.session_age_group_id === target.session_age_group_id
			);
			if (!ageGroupStillUsed) {
				try {
					await sessionAgeGroupApi.delete(sessionId, target.session_age_group_id);
					sessionAgeGroups = sessionAgeGroups.filter(
						(sag) => sag.id !== target.session_age_group_id
					);
					const next = new Set(expandedAgeGroups);
					next.delete(target.session_age_group_id);
					expandedAgeGroups = next;
				} catch {
					// Best-effort: empty session_age_group remains; harmless.
				}
			}

			toast.success("Cabin removed");
			removeCabinOpen = false;
			removeCabinTarget = null;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to remove cabin";
			toast.error(message);
		} finally {
			removingCabin = false;
		}
	}
</script>

<div class="grid gap-6">
	<!-- Header -->
	<div class="flex items-start gap-4">
		<Button variant="ghost" size="icon-sm" onclick={() => goto("/app/sessions")}>
			<ArrowLeftIcon class="size-4" />
			<span class="sr-only">Back</span>
		</Button>
		<div>
			{#if loading}
				<h1 class="text-2xl font-semibold tracking-tight">Loading...</h1>
			{:else if session}
				<h1 class="text-2xl font-semibold tracking-tight">{session.name}</h1>
				<p class="text-muted-foreground text-sm">
					Configure age groups, cabins, time slots, and activities for this session.
				</p>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load session details. Try refreshing the page.
		</div>
	{:else if session}
		<Tabs.Tabs value={activeTab} onValueChange={setActiveTab}>
			<Tabs.TabsList>
				<Tabs.TabsTrigger value="cabins">Cabins</Tabs.TabsTrigger>
				<Tabs.TabsTrigger value="activities">Activities</Tabs.TabsTrigger>
			</Tabs.TabsList>

			<!-- Cabins Tab -->
			<Tabs.TabsContent value="cabins">
				<div class="grid gap-4">
					<div class="flex items-center justify-between">
						<h2 class="text-lg font-semibold">Cabins</h2>
						<div class="flex items-center gap-2">
							{#if nonEmptyAgeGroups.length > 0}
								<Button
									variant="ghost"
									size="sm"
									onclick={toggleExpandAllAgeGroups}
									title={allAgeGroupsExpanded ? "Collapse All" : "Expand All"}
								>
									<ChevronsUpDownIcon class="mr-1 size-4" />
									{allAgeGroupsExpanded ? "Collapse All" : "Expand All"}
								</Button>
							{/if}
							<Button
								size="sm"
								disabled={disabled || availableCabinsForSession.length === 0}
								onclick={openAddCabin}
							>
								<PlusIcon class="mr-2 size-4" />
								Add Cabin
							</Button>
						</div>
					</div>

					{#if allCabins.length === 0}
						<div class="text-muted-foreground py-4 text-center text-sm">
							No cabins defined.
							<a href="/app/cabins" class="text-foreground underline">Create cabins</a>
							first.
						</div>
					{:else if nonEmptyAgeGroups.length === 0}
						<div class="text-muted-foreground py-4 text-center text-sm">
							No cabins assigned to this session yet. Click "Add Cabin" to assign one.
						</div>
					{:else}
						{#each nonEmptyAgeGroups as sag (sag.id)}
							{@const cabins = cabinsByAgeGroup.get(sag.id) ?? []}
							{@const isExpanded = expandedAgeGroups.has(sag.id)}
							<div class="border-border overflow-hidden rounded-lg border">
								<button
									type="button"
									class="bg-muted flex w-full items-center gap-2 px-4 py-3 text-left"
									onclick={() => {
										const next = new Set(expandedAgeGroups);
										if (next.has(sag.id)) next.delete(sag.id);
										else next.add(sag.id);
										expandedAgeGroups = next;
									}}
								>
									<ChevronDownIcon class="size-4 transition-transform {isExpanded ? '' : '-rotate-90'}" />
									<h3 class="font-medium">
										{ageGroupMap.get(sag.age_group_id) ?? "Unknown Age Group"}
									</h3>
									<span class="text-muted-foreground text-sm">
										— {cabins.length} {cabins.length === 1 ? "cabin" : "cabins"}
									</span>
								</button>

								{#if isExpanded}
									<Table.Table>
										<Table.TableHeader>
											<Table.TableRow>
												<Table.TableHead>Cabin</Table.TableHead>
												<Table.TableHead class="w-32">Group Size</Table.TableHead>
												<Table.TableHead class="w-40">Required Counselors</Table.TableHead>
												<Table.TableHead class="w-16">
													<span class="sr-only">Actions</span>
												</Table.TableHead>
											</Table.TableRow>
										</Table.TableHeader>
										<Table.TableBody>
											{#each cabins as sc (sc.id)}
												<Table.TableRow>
													<Table.TableCell>
														{cabinMap.get(sc.cabin_id)?.name ?? "Unknown"}
													</Table.TableCell>
													<Table.TableCell>
														{sc.group_size}
													</Table.TableCell>
													<Table.TableCell>
														{sc.required_counselors}
													</Table.TableCell>
													<Table.TableCell>
														<div class="flex justify-end gap-1">
															<Button
																variant="ghost"
																size="icon-sm"
																disabled={disabled}
																title="Edit"
																onclick={() => openEditCabin(sc)}
															>
																<PencilIcon class="size-4" />
																<span class="sr-only">Edit</span>
															</Button>
															<Button
																variant="ghost"
																size="icon-sm"
																disabled={disabled}
																title="Remove"
																onclick={() => confirmRemoveCabin(sc)}
															>
																<TrashIcon class="size-4" />
																<span class="sr-only">Remove</span>
															</Button>
														</div>
													</Table.TableCell>
												</Table.TableRow>
											{/each}
										</Table.TableBody>
									</Table.Table>
								{/if}
							</div>
						{/each}
					{/if}
				</div>
			</Tabs.TabsContent>

			<!-- Time Slots & Activities Tab -->
			<Tabs.TabsContent value="activities">
		<!-- Time Slots Section -->
		<div class="grid gap-4">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold">Time Slots</h2>
				<div class="flex items-center gap-2">
					{#if sortedSessionTimeSlots.length > 0}
						<Button
							variant="ghost"
							size="sm"
							onclick={toggleExpandAll}
							title={allExpanded ? "Collapse All" : "Expand All"}
						>
							<ChevronsUpDownIcon class="mr-1 size-4" />
							{allExpanded ? "Collapse All" : "Expand All"}
						</Button>
					{/if}
					<Button
						size="sm"
						disabled={disabled || availableTimeSlots.length === 0}
						onclick={openAddTimeSlot}
					>
						<PlusIcon class="mr-2 size-4" />
						Add Time Slot
					</Button>
				</div>
			</div>

			{#if allTimeSlots.length === 0}
				<div class="text-muted-foreground py-4 text-center text-sm">
					No time slots defined.
					<a href="/app/time-slots" class="text-foreground underline">Create time slots</a>
					first.
				</div>
			{:else if sortedSessionTimeSlots.length === 0}
				<div class="text-muted-foreground py-4 text-center text-sm">
					No time slots assigned to this session yet. Click "Add Time Slot" to assign one.
				</div>
			{:else}
				{#each sortedSessionTimeSlots as st (st.id)}
					{@const activities = [...(activitiesByTimeSlot.get(st.id) ?? [])].sort((a, b) =>
						(activityMap.get(a.activity_id) ?? "").localeCompare(activityMap.get(b.activity_id) ?? "")
					)}
					{@const availableActs = getAvailableActivities(st.id)}
					{@const isExpanded = expandedTimeSlots.has(st.id)}
				<div
					class="border-border relative rounded-lg border transition-opacity {draggedId === st.id ? 'opacity-50' : ''}"
					ondragover={(e) => handleDragOver(e, st.id)}
					ondragleave={(e) => handleDragLeave(e)}
					ondrop={(e) => handleDrop(e, st.id)}
				>
					{#if dropInsert?.id === st.id && dropInsert.position === "before"}
						<div class="bg-primary absolute -top-[9px] right-4 left-4 h-[2px] rounded-full"></div>
					{/if}
					{#if dropInsert?.id === st.id && dropInsert.position === "after"}
						<div class="bg-primary absolute -bottom-[9px] right-4 left-4 h-[2px] rounded-full"></div>
					{/if}
						<div class="bg-muted flex items-center justify-between rounded-t-lg px-4 py-3 {isExpanded ? '' : 'rounded-b-lg'}">
							<div class="flex items-center gap-2">
								{#if !disabled}
									<button
										type="button"
										class="text-muted-foreground hover:text-foreground cursor-grab active:cursor-grabbing"
										draggable="true"
										ondragstart={(e) => handleDragStart(e, st.id)}
										ondragend={handleDragEnd}
									>
										<GripVerticalIcon class="size-4" />
										<span class="sr-only">Drag to reorder</span>
									</button>
								{/if}
								<button
									type="button"
									class="flex items-center gap-2 text-left"
									onclick={() => {
										const next = new Set(expandedTimeSlots);
										if (next.has(st.id)) next.delete(st.id);
										else next.add(st.id);
										expandedTimeSlots = next;
									}}
								>
									<ChevronDownIcon class="size-4 transition-transform {isExpanded ? '' : '-rotate-90'}" />
									<h3 class="font-medium">
										{timeSlotMap.get(st.time_slot_id) ?? "Unknown Time Slot"}
									</h3>
									{#if !isExpanded && activities.length > 0}
										<span class="text-muted-foreground text-sm">
											({activities.length} {activities.length === 1 ? "activity" : "activities"})
										</span>
									{/if}
								</button>
							</div>
							<div class="flex items-center gap-2">
								<Button
									size="sm"
									variant="outline"
									disabled={disabled || availableActs.length === 0}
									onclick={() => openAddActivity(st.id)}
								>
									<PlusIcon class="mr-2 size-4" />
									Add Activity
								</Button>
								{#if activities.length > 0}
									<Button
										size="sm"
										variant="outline"
										disabled={disabled}
										onclick={() => openCopyActivities(st.id)}
									>
										<CopyIcon class="mr-2 size-4" />
										Copy to...
									</Button>
								{/if}
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Remove time slot"
									onclick={() => confirmRemoveTimeSlot(st)}
								>
									<TrashIcon class="size-4" />
									<span class="sr-only">Remove</span>
								</Button>
							</div>
						</div>

						{#if isExpanded}
							{#if activities.length === 0}
								<div class="text-muted-foreground px-4 py-4 text-center text-sm">
									No activities assigned to this time slot.
								</div>
							{:else}
								<Table.Table>
									<Table.TableHeader>
										<Table.TableRow>
											<Table.TableHead>Activity</Table.TableHead>
											<Table.TableHead class="w-32">Capacity</Table.TableHead>
											<Table.TableHead class="w-40">Required Counselors</Table.TableHead>
											<Table.TableHead class="w-16">
												<span class="sr-only">Actions</span>
										</Table.TableHead>
									</Table.TableRow>
								</Table.TableHeader>
								<Table.TableBody>
									{#each activities as activity (activity.id)}
										<Table.TableRow>
											<Table.TableCell>
												{activityMap.get(activity.activity_id) ?? "Unknown"}
											</Table.TableCell>
											<Table.TableCell>{activity.capacity}</Table.TableCell>
											<Table.TableCell>{activity.required_counselors}</Table.TableCell>
											<Table.TableCell>
												<div class="flex justify-end gap-1">
													<Button
														variant="ghost"
														size="icon-sm"
														disabled={disabled}
														title="Edit"
														onclick={() => openEditActivity(st, activity)}
													>
														<PencilIcon class="size-4" />
														<span class="sr-only">Edit</span>
													</Button>
													<Button
														variant="ghost"
														size="icon-sm"
														disabled={disabled}
														title="Remove"
														onclick={() => confirmRemoveActivity(st, activity)}
													>
														<TrashIcon class="size-4" />
														<span class="sr-only">Remove</span>
													</Button>
												</div>
											</Table.TableCell>
										</Table.TableRow>
									{/each}
								</Table.TableBody>
							</Table.Table>
							{/if}
						{/if}
					</div>
				{/each}
			{/if}
		</div>
			</Tabs.TabsContent>
		</Tabs.Tabs>
	{/if}
</div>

<!-- Add Time Slot Dialog -->
<Dialog.Dialog bind:open={addTimeSlotOpen}>
	<Dialog.DialogContent>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Add Time Slot</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Select a time slot to assign to this session.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<div class="grid gap-4">
			<div class="grid gap-2">
				<Label for="time-slot-select">Time Slot</Label>
				<Select.Select
					type="single"
					value={addTimeSlotId}
					onValueChange={(v) => (addTimeSlotId = v)}
				>
					<Select.SelectTrigger id="time-slot-select" class="w-full">
						{#if addTimeSlotId}
							{timeSlotMap.get(addTimeSlotId) ?? "Select time slot"}
						{:else}
							<span class="text-muted-foreground">Select time slot</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each availableTimeSlots as ts (ts.id)}
							<Select.SelectItem value={ts.id}>{ts.name}</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
			</div>
		</div>
		<Dialog.DialogFooter>
			<Button
				variant="outline"
				disabled={addingTimeSlot}
				onclick={() => (addTimeSlotOpen = false)}
			>
				Cancel
			</Button>
			<Button
				disabled={addingTimeSlot || !addTimeSlotId}
				onclick={handleAddTimeSlot}
			>
				{#if addingTimeSlot}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Add
			</Button>
		</Dialog.DialogFooter>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Remove Time Slot Confirmation -->
<AlertDialog.AlertDialog bind:open={removeTimeSlotOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Remove Time Slot</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to remove
				"{removeTimeSlotTarget ? timeSlotMap.get(removeTimeSlotTarget.time_slot_id) ?? 'this time slot' : ''}"
				from this session? All activity assignments within it will also be removed.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={removingTimeSlot}>
				Cancel
			</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				disabled={removingTimeSlot}
				onclick={handleRemoveTimeSlot}
			>
				{#if removingTimeSlot}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Remove
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>

<!-- Add Activity Dialog -->
<Dialog.Dialog bind:open={addActivityOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Add Activity</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Assign an activity to this time slot with capacity and staffing requirements.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleAddActivity} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="activity-select">Activity</Label>
				<Select.Select
					type="single"
					value={addActivityId}
					disabled={addingActivity}
					onValueChange={(v) => {
						addActivityId = v;
						activityError = "";
					}}
				>
					<Select.SelectTrigger id="activity-select" class="w-full">
						{#if addActivityId}
							{activityMap.get(addActivityId) ?? "Select activity"}
						{:else}
							<span class="text-muted-foreground">Select activity</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each getAvailableActivities(addActivityTimeSlotId) as act (act.id)}
							<Select.SelectItem value={act.id}>{act.name}</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if activityError}
					<p class="text-destructive text-sm">{activityError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="activity-capacity">Capacity</Label>
				<Input
					id="activity-capacity"
					type="number"
					min="1"
					placeholder="Max participants"
					bind:value={addActivityCapacity}
					disabled={addingActivity}
					oninput={() => (capacityError = "")}
				/>
				{#if capacityError}
					<p class="text-destructive text-sm">{capacityError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="activity-counselors">Required Counselors</Label>
				<Input
					id="activity-counselors"
					type="number"
					min="1"
					placeholder="Number of counselors needed"
					bind:value={addActivityCounselors}
					disabled={addingActivity}
					oninput={() => (counselorsError = "")}
				/>
				{#if counselorsError}
					<p class="text-destructive text-sm">{counselorsError}</p>
				{/if}
			</div>
			<Dialog.DialogFooter>
				<Button
					type="button"
					variant="outline"
					disabled={addingActivity}
					onclick={() => (addActivityOpen = false)}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={addingActivity}>
					{#if addingActivity}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Add
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Edit Activity Dialog -->
<Dialog.Dialog bind:open={editActivityOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Edit Activity</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Update capacity and staffing requirements for
				{editActivityTarget ? activityMap.get(editActivityTarget.activity.activity_id) ?? "this activity" : "this activity"}.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleEditActivity} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="edit-activity-capacity">Capacity</Label>
				<Input
					id="edit-activity-capacity"
					type="number"
					min="1"
					placeholder="Max participants"
					bind:value={editActivityCapacity}
					disabled={editingActivity}
					oninput={() => (editCapacityError = "")}
				/>
				{#if editCapacityError}
					<p class="text-destructive text-sm">{editCapacityError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="edit-activity-counselors">Required Counselors</Label>
				<Input
					id="edit-activity-counselors"
					type="number"
					min="1"
					placeholder="Number of counselors needed"
					bind:value={editActivityCounselors}
					disabled={editingActivity}
					oninput={() => (editCounselorsError = "")}
				/>
				{#if editCounselorsError}
					<p class="text-destructive text-sm">{editCounselorsError}</p>
				{/if}
			</div>
			<Dialog.DialogFooter>
				<Button
					type="button"
					variant="outline"
					disabled={editingActivity}
					onclick={() => (editActivityOpen = false)}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={editingActivity}>
					{#if editingActivity}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Save
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Copy Activities Dialog -->
<Dialog.Dialog bind:open={copyActivitiesOpen}>
	<Dialog.DialogContent>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Copy Activities</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Copy all activity assignments from
				"{timeSlotMap.get(sessionTimeSlots.find((st) => st.id === copySourceTimeSlotId)?.time_slot_id ?? '') ?? ''}"
				to another time slot.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<div class="grid gap-4">
			<div class="grid gap-2">
				<Label for="copy-target-select">Target Time Slot</Label>
				<Select.Select
					type="single"
					value={copyTargetValue}
					onValueChange={(v) => (copyTargetValue = v)}
				>
					<Select.SelectTrigger id="copy-target-select" class="w-full">
						{#if copyTargetValue}
							{#if copyTargetValue.startsWith("existing:")}
								{timeSlotMap.get(sessionTimeSlots.find((st) => st.id === copyTargetValue.slice("existing:".length))?.time_slot_id ?? '') ?? "Select time slot"}
							{:else}
								{timeSlotMap.get(copyTargetValue.slice("new:".length)) ?? "Select time slot"}
							{/if}
						{:else}
							<span class="text-muted-foreground">Select time slot</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each sortedSessionTimeSlots.filter((st) => st.id !== copySourceTimeSlotId) as st (st.id)}
							<Select.SelectItem value={`existing:${st.id}`}>
								{timeSlotMap.get(st.time_slot_id) ?? "Unknown"}
							</Select.SelectItem>
						{/each}
						{#if availableTimeSlots.length > 0}
							{#if sortedSessionTimeSlots.length > 1}
								<Select.SelectSeparator />
							{/if}
							{#each availableTimeSlots as ts (ts.id)}
								<Select.SelectItem value={`new:${ts.id}`}>
									{ts.name} <span class="text-muted-foreground ml-1">(add new)</span>
								</Select.SelectItem>
							{/each}
						{/if}
					</Select.SelectContent>
				</Select.Select>
			</div>
			{#if copyTargetValue && getCopyTargetActivityCount() > 0}
				{@const targetName = timeSlotMap.get(sessionTimeSlots.find((st) => st.id === copyTargetValue.slice("existing:".length))?.time_slot_id ?? '') ?? ''}
				<p class="text-destructive text-sm">
					This will replace {getCopyTargetActivityCount()} existing activity assignment{getCopyTargetActivityCount() === 1 ? "" : "s"} in "{targetName}".
				</p>
			{/if}
		</div>
		<Dialog.DialogFooter>
			<Button
				variant="outline"
				disabled={copyingActivities}
				onclick={() => (copyActivitiesOpen = false)}
			>
				Cancel
			</Button>
			<Button
				disabled={copyingActivities || !copyTargetValue}
				onclick={handleCopyActivities}
			>
				{#if copyingActivities}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Copy
			</Button>
		</Dialog.DialogFooter>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Remove Activity Confirmation -->
<AlertDialog.AlertDialog bind:open={removeActivityOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Remove Activity</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to remove
				"{removeActivityTarget ? activityMap.get(removeActivityTarget.activity.activity_id) ?? 'this activity' : ''}"
				from this time slot?
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={removingActivity}>
				Cancel
			</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				disabled={removingActivity}
				onclick={handleRemoveActivity}
			>
				{#if removingActivity}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Remove
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>

<!-- Add Cabin Dialog -->
<Dialog.Dialog bind:open={addCabinOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Add Cabin</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Assign a cabin to this session. The age group defaults to the cabin's
				default age group, but can be changed.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleAddCabin} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="cabin-select">Cabin</Label>
				<Select.Select
					type="single"
					value={addCabinId}
					disabled={addingCabin}
					onValueChange={onAddCabinSelected}
				>
					<Select.SelectTrigger id="cabin-select" class="w-full">
						{#if addCabinId}
							{cabinMap.get(addCabinId)?.name ?? "Select cabin"}
						{:else}
							<span class="text-muted-foreground">Select cabin</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each availableCabinsForSession as c (c.id)}
							<Select.SelectItem value={c.id}>{c.name}</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if addCabinError}
					<p class="text-destructive text-sm">{addCabinError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-age-group-select">Age Group</Label>
				<Select.Select
					type="single"
					value={addCabinAgeGroupId}
					disabled={addingCabin || !addCabinId}
					onValueChange={(v) => {
						addCabinAgeGroupId = v;
						addCabinAgeGroupError = "";
					}}
				>
					<Select.SelectTrigger id="cabin-age-group-select" class="w-full">
						{#if addCabinAgeGroupId}
							{ageGroupMap.get(addCabinAgeGroupId) ?? "Select age group"}
						{:else}
							<span class="text-muted-foreground">Select age group</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each allAgeGroups as ag (ag.id)}
							<Select.SelectItem value={ag.id}>{ag.name}</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if addCabinAgeGroupError}
					<p class="text-destructive text-sm">{addCabinAgeGroupError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-group-size">Group Size</Label>
				<Input
					id="cabin-group-size"
					type="number"
					min="1"
					required
					bind:value={addCabinGroupSize}
					disabled={addingCabin}
					oninput={() => (addCabinGroupSizeError = "")}
				/>
				{#if addCabinGroupSizeError}
					<p class="text-destructive text-sm">{addCabinGroupSizeError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-counselors">Required Counselors</Label>
				<Input
					id="cabin-counselors"
					type="number"
					min="1"
					required
					bind:value={addCabinCounselors}
					disabled={addingCabin}
					oninput={() => (addCabinCounselorsError = "")}
				/>
				{#if addCabinCounselorsError}
					<p class="text-destructive text-sm">{addCabinCounselorsError}</p>
				{/if}
			</div>
			<Dialog.DialogFooter>
				<Button
					type="button"
					variant="outline"
					disabled={addingCabin}
					onclick={() => (addCabinOpen = false)}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={addingCabin}>
					{#if addingCabin}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Add
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Edit Cabin Dialog -->
<Dialog.Dialog bind:open={editCabinOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Edit Cabin</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Update group size and required counselors for
				{editCabinTarget ? cabinMap.get(editCabinTarget.cabin_id)?.name ?? "this cabin" : "this cabin"}.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleEditCabin} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="edit-cabin-group-size">Group Size</Label>
				<Input
					id="edit-cabin-group-size"
					type="number"
					min="1"
					required
					bind:value={editCabinGroupSize}
					disabled={editingCabin}
					oninput={() => (editCabinGroupSizeError = "")}
				/>
				{#if editCabinGroupSizeError}
					<p class="text-destructive text-sm">{editCabinGroupSizeError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="edit-cabin-counselors">Required Counselors</Label>
				<Input
					id="edit-cabin-counselors"
					type="number"
					min="1"
					required
					bind:value={editCabinCounselors}
					disabled={editingCabin}
					oninput={() => (editCabinCounselorsError = "")}
				/>
				{#if editCabinCounselorsError}
					<p class="text-destructive text-sm">{editCabinCounselorsError}</p>
				{/if}
			</div>
			<Dialog.DialogFooter>
				<Button
					type="button"
					variant="outline"
					disabled={editingCabin}
					onclick={() => (editCabinOpen = false)}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={editingCabin}>
					{#if editingCabin}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Save
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Remove Cabin Confirmation -->
<AlertDialog.AlertDialog bind:open={removeCabinOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Remove Cabin</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to remove
				"{removeCabinTarget ? cabinMap.get(removeCabinTarget.cabin_id)?.name ?? 'this cabin' : ''}"
				from this age group?
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={removingCabin}>
				Cancel
			</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				disabled={removingCabin}
				onclick={handleRemoveCabin}
			>
				{#if removingCabin}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Remove
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
