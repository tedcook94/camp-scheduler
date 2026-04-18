<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { page } from "$app/state";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import {
		sessionApi,
		sessionTimeSlotApi,
		sessionActivityApi,
		timeSlotApi,
		activityApi,
	} from "$lib/api";
	import type {
		Session,
		SessionTimeSlot,
		SessionActivity,
		TimeSlot,
		Activity,
	} from "$lib/api/types";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
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

	let session = $state<Session | null>(null);
	let sessionTimeSlots = $state<SessionTimeSlot[]>([]);
	let allTimeSlots = $state<TimeSlot[]>([]);
	let allActivities = $state<Activity[]>([]);
	// Map from session_time_slot id to its activities
	let activitiesByTimeSlot = $state<Map<string, SessionActivity[]>>(new Map());
	let loading = $state(true);
	let loadError = $state(false);

	// Lookup maps
	let timeSlotMap = $derived(new Map(allTimeSlots.map((t) => [t.id, t.name])));
	let activityMap = $derived(new Map(allActivities.map((a) => [a.id, a.name])));

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
			const [s, stSlots, ts, acts, allSessionActs] = await Promise.all([
				sessionApi.get(sessionId),
				sessionTimeSlotApi.list(sessionId),
				timeSlotApi.list(),
				activityApi.list(),
				sessionActivityApi.listAll(sessionId),
			]);
			session = s;
			sessionTimeSlots = stSlots;
			allTimeSlots = ts;
			allActivities = acts;

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
		const capacity = parseInt(addActivityCapacity);
		if (!addActivityCapacity || isNaN(capacity) || capacity < 1) {
			capacityError = "Capacity must be at least 1.";
			valid = false;
		}
		const counselors = parseInt(addActivityCounselors);
		if (!addActivityCounselors || isNaN(counselors) || counselors < 1) {
			counselorsError = "Required counselors must be at least 1.";
			valid = false;
		}
		if (valid && counselors > capacity) {
			counselorsError = "Required counselors cannot exceed capacity.";
			valid = false;
		}
		if (!valid) return;

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
		const capacity = parseInt(editActivityCapacity);
		if (!editActivityCapacity || isNaN(capacity) || capacity < 1) {
			editCapacityError = "Capacity must be at least 1.";
			valid = false;
		}
		const counselors = parseInt(editActivityCounselors);
		if (!editActivityCounselors || isNaN(counselors) || counselors < 1) {
			editCounselorsError = "Required counselors must be at least 1.";
			valid = false;
		}
		if (valid && counselors > capacity) {
			editCounselorsError = "Required counselors cannot exceed capacity.";
			valid = false;
		}
		if (!valid) return;

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
					Manage time slots and activity assignments for this session.
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
		<!-- Time Slots Section -->
		<div class="grid gap-4">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-2">
					<h2 class="text-lg font-semibold">Time Slots</h2>
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
				</div>
				<Button
					size="sm"
					disabled={disabled || availableTimeSlots.length === 0}
					onclick={openAddTimeSlot}
				>
					<PlusIcon class="mr-2 size-4" />
					Add Time Slot
				</Button>
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
					{@const activities = activitiesByTimeSlot.get(st.id) ?? []}
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
						<div class="bg-muted flex items-center justify-between px-4 py-3">
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
