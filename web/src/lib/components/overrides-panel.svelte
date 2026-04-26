<script lang="ts">
	import { toast } from "svelte-sonner";
	import { ApiClientError } from "$lib/api/client";
	import {
		overrideApi,
		sessionCounselorApi,
		sessionCabinApi,
		enrollmentApi,
		sessionTimeSlotApi,
		sessionActivityApi,
		cabinApi,
		activityApi,
		timeSlotApi,
	} from "$lib/api";
	import type {
		Cabin,
		Activity,
		TimeSlot,
		SessionCabin,
		SessionCounselor,
		SessionTimeSlot,
		SessionActivity,
		Enrollment,
		SessionOverrides,
	} from "$lib/api/types";
	import { Button } from "$lib/components/ui/button";
	import * as Select from "$lib/components/ui/select";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import PlusIcon from "@lucide/svelte/icons/plus";

	interface Props {
		sessionId: string;
		onMutate?: () => void;
	}

	let { sessionId, onMutate }: Props = $props();

	let overrides = $state<SessionOverrides>({
		counselor_cabin: [],
		camper_cabin: [],
		counselor_activity: [],
	});
	let counselors = $state<SessionCounselor[]>([]);
	let sessionCabins = $state<SessionCabin[]>([]);
	let cabins = $state<Cabin[]>([]);
	let enrollments = $state<Enrollment[]>([]);
	let sessionTimeSlots = $state<SessionTimeSlot[]>([]);
	let sessionActivities = $state<SessionActivity[]>([]);
	let activities = $state<Activity[]>([]);
	let timeSlots = $state<TimeSlot[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let newCounselorCabinCounselor = $state<string | null>(null);
	let newCounselorCabinSAGC = $state<string | null>(null);
	let creatingCounselorCabin = $state(false);

	let newCamperCabinCamper = $state<string | null>(null);
	let newCamperCabinSAGC = $state<string | null>(null);
	let creatingCamperCabin = $state(false);

	let newCounselorActivityCounselor = $state<string | null>(null);
	let newCounselorActivitySA = $state<string | null>(null);
	let creatingCounselorActivity = $state(false);

	let deleting = $state<string | null>(null);
	let deleteOpen = $state(false);
	let deleteTarget = $state<{ type: "counselor_cabin" | "camper_cabin" | "counselor_activity"; id: string; label: string } | null>(null);

	const cabinNameById = $derived(new Map(cabins.map((c) => [c.id, c.name])));
	const activityNameById = $derived(new Map(activities.map((a) => [a.id, a.name])));
	const timeSlotNameById = $derived(new Map(timeSlots.map((t) => [t.id, t.name])));

	const sortedSessionCabins = $derived(
		[...sessionCabins].sort((a, b) => {
			const an = cabinNameById.get(a.cabin_id) ?? a.cabin_id;
			const bn = cabinNameById.get(b.cabin_id) ?? b.cabin_id;
			return an.localeCompare(bn);
		}),
	);

	const sessionCabinLabel = $derived((id: string) => {
		const sc = sessionCabins.find((c) => c.id === id);
		if (!sc) return id.slice(0, 8);
		return cabinNameById.get(sc.cabin_id) ?? id.slice(0, 8);
	});

	const sessionActivityLabel = $derived((id: string) => {
		const sa = sessionActivities.find((a) => a.id === id);
		if (!sa) return id.slice(0, 8);
		const actName = activityNameById.get(sa.activity_id) ?? "?";
		const sts = sessionTimeSlots.find((s) => s.id === sa.session_time_slot_id);
		const tsName = sts ? (timeSlotNameById.get(sts.time_slot_id) ?? "?") : "?";
		return `${tsName} — ${actName}`;
	});

	const pinnedCounselorIDs = $derived(new Set(overrides.counselor_cabin.map((o) => o.counselor_id)));
	const pinnedCamperIDs = $derived(new Set(overrides.camper_cabin.map((o) => o.camper_id)));
	// Activity overrides allow a counselor to be pinned to multiple time slots
	// (one per slot). We don't filter them out from the dropdown here; the
	// backend rejects double-bookings within the same time slot.

	const availableCounselorsForCabin = $derived(
		counselors.filter((c) => !c.archived && !pinnedCounselorIDs.has(c.counselor_id)),
	);
	const availableCampersForCabin = $derived(
		enrollments.filter((e) => !pinnedCamperIDs.has(e.camper_id)),
	);
	const activeCounselors = $derived(counselors.filter((c) => !c.archived));

	$effect(() => {
		if (sessionId) {
			// Reset selection + dialog state so values from a prior session
			// can't be submitted against the new one.
			newCounselorCabinCounselor = null;
			newCounselorCabinSAGC = null;
			newCamperCabinCamper = null;
			newCamperCabinSAGC = null;
			newCounselorActivityCounselor = null;
			newCounselorActivitySA = null;
			deleteOpen = false;
			deleteTarget = null;
			void loadAll();
		}
	});

	async function loadAll() {
		loading = true;
		loadError = null;
		try {
			const [ov, cs, scs, en, sts, sas, cb, ac, ts] = await Promise.all([
				overrideApi.list(sessionId),
				sessionCounselorApi.list(sessionId),
				sessionCabinApi.list(sessionId),
				enrollmentApi.list(sessionId),
				sessionTimeSlotApi.list(sessionId),
				sessionActivityApi.listAll(sessionId),
				cabinApi.list(),
				activityApi.list(),
				timeSlotApi.list(),
			]);
			overrides = ov;
			counselors = cs;
			sessionCabins = scs;
			enrollments = en;
			sessionTimeSlots = sts;
			sessionActivities = sas;
			cabins = cb;
			activities = ac;
			timeSlots = ts;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load overrides";
			loadError = message;
			toast.error(message);
		} finally {
			loading = false;
		}
	}

	async function addCounselorCabin() {
		if (!newCounselorCabinCounselor || !newCounselorCabinSAGC) return;
		creatingCounselorCabin = true;
		try {
			const created = await overrideApi.createCounselorCabin(sessionId, {
				counselor_id: newCounselorCabinCounselor,
				session_age_group_cabin_id: newCounselorCabinSAGC,
			});
			overrides = {
				...overrides,
				counselor_cabin: [...overrides.counselor_cabin, created],
			};
			newCounselorCabinCounselor = null;
			newCounselorCabinSAGC = null;
			toast.success("Counselor pinned to cabin");
			onMutate?.();
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to add override";
			toast.error(message);
		} finally {
			creatingCounselorCabin = false;
		}
	}

	async function addCamperCabin() {
		if (!newCamperCabinCamper || !newCamperCabinSAGC) return;
		creatingCamperCabin = true;
		try {
			const created = await overrideApi.createCamperCabin(sessionId, {
				camper_id: newCamperCabinCamper,
				session_age_group_cabin_id: newCamperCabinSAGC,
			});
			overrides = {
				...overrides,
				camper_cabin: [...overrides.camper_cabin, created],
			};
			newCamperCabinCamper = null;
			newCamperCabinSAGC = null;
			toast.success("Camper pinned to cabin");
			onMutate?.();
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to add override";
			toast.error(message);
		} finally {
			creatingCamperCabin = false;
		}
	}

	async function addCounselorActivity() {
		if (!newCounselorActivityCounselor || !newCounselorActivitySA) return;
		creatingCounselorActivity = true;
		try {
			const created = await overrideApi.createCounselorActivity(sessionId, {
				counselor_id: newCounselorActivityCounselor,
				session_activity_id: newCounselorActivitySA,
			});
			overrides = {
				...overrides,
				counselor_activity: [...overrides.counselor_activity, created],
			};
			newCounselorActivityCounselor = null;
			newCounselorActivitySA = null;
			toast.success("Counselor pinned to activity");
			onMutate?.();
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to add override";
			toast.error(message);
		} finally {
			creatingCounselorActivity = false;
		}
	}

	function confirmDelete(
		type: "counselor_cabin" | "camper_cabin" | "counselor_activity",
		id: string,
		label: string,
	) {
		deleteTarget = { type, id, label };
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		const target = deleteTarget;
		deleting = target.id;
		try {
			if (target.type === "counselor_cabin") {
				await overrideApi.deleteCounselorCabin(sessionId, target.id);
				overrides = {
					...overrides,
					counselor_cabin: overrides.counselor_cabin.filter((o) => o.id !== target.id),
				};
			} else if (target.type === "camper_cabin") {
				await overrideApi.deleteCamperCabin(sessionId, target.id);
				overrides = {
					...overrides,
					camper_cabin: overrides.camper_cabin.filter((o) => o.id !== target.id),
				};
			} else {
				await overrideApi.deleteCounselorActivity(sessionId, target.id);
				overrides = {
					...overrides,
					counselor_activity: overrides.counselor_activity.filter((o) => o.id !== target.id),
				};
			}
			toast.success("Override removed");
			deleteOpen = false;
			deleteTarget = null;
			onMutate?.();
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to remove override";
			toast.error(message);
		} finally {
			deleting = null;
		}
	}
</script>

{#if loading}
	<div class="text-muted-foreground py-4 text-center text-sm">Loading overrides...</div>
{:else if loadError}
	<div class="text-muted-foreground py-4 text-center text-sm">{loadError}</div>
{:else}
	<div class="grid gap-6">
		<p class="text-muted-foreground text-sm">
			Pin counselors and campers to specific cabins or activities before triggering a run.
			The solver treats overrides as forced placements. Adding or removing an override
			marks any existing run for this session as stale.
		</p>

		<!-- Counselor → Cabin -->
		<section class="grid gap-3">
			<h3 class="text-sm font-semibold">Counselor → Cabin</h3>

			{#if overrides.counselor_cabin.length === 0}
				<p class="text-muted-foreground text-sm">No counselor cabin overrides.</p>
			{:else}
				<div class="grid gap-2">
					{#each overrides.counselor_cabin as o (o.id)}
						<div class="flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2">
							<div class="text-sm">
								<span class="font-medium">{o.counselor_name}</span>
								<span class="text-muted-foreground"> → </span>
								<span>{o.cabin_name}</span>
								<span class="text-muted-foreground"> ({o.age_group_name})</span>
							</div>
							<Button
								variant="ghost"
								size="sm"
								aria-label="Remove counselor→cabin override"
								title="Remove counselor→cabin override"
								disabled={deleting === o.id}
								onclick={() => confirmDelete("counselor_cabin", o.id, `${o.counselor_name} → ${o.cabin_name}`)}
							>
								<TrashIcon class="size-4" />
								<span class="sr-only">Remove counselor→cabin override</span>
							</Button>
						</div>
					{/each}
				</div>
			{/if}

			<div class="flex flex-wrap items-end gap-2">
				<div class="grid gap-1">
					<label for="cc-counselor" class="text-xs text-muted-foreground">Counselor</label>
					<Select.Select
						type="single"
						value={newCounselorCabinCounselor ?? undefined}
						onValueChange={(v) => (newCounselorCabinCounselor = v)}
					>
						<Select.SelectTrigger id="cc-counselor" class="w-[220px]">
							{#if newCounselorCabinCounselor}
								{counselors.find((c) => c.counselor_id === newCounselorCabinCounselor)?.counselor_name ?? "?"}
							{:else}
								Select counselor...
							{/if}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each availableCounselorsForCabin as c (c.counselor_id)}
								<Select.SelectItem value={c.counselor_id}>{c.counselor_name}</Select.SelectItem>
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
				<div class="grid gap-1">
					<label for="cc-cabin" class="text-xs text-muted-foreground">Cabin</label>
					<Select.Select
						type="single"
						value={newCounselorCabinSAGC ?? undefined}
						onValueChange={(v) => (newCounselorCabinSAGC = v)}
					>
						<Select.SelectTrigger id="cc-cabin" class="w-[220px]">
							{#if newCounselorCabinSAGC}
								{sessionCabinLabel(newCounselorCabinSAGC)}
							{:else}
								Select cabin...
							{/if}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each sortedSessionCabins as sc (sc.id)}
								<Select.SelectItem value={sc.id}>
									{cabinNameById.get(sc.cabin_id) ?? sc.cabin_id}
								</Select.SelectItem>
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
				<Button
					size="sm"
					disabled={!newCounselorCabinCounselor || !newCounselorCabinSAGC || creatingCounselorCabin}
					onclick={addCounselorCabin}
				>
					{#if creatingCounselorCabin}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{:else}
						<PlusIcon class="mr-2 size-4" />
					{/if}
					Pin
				</Button>
			</div>
		</section>

		<!-- Camper → Cabin -->
		<section class="grid gap-3">
			<h3 class="text-sm font-semibold">Camper → Cabin</h3>

			{#if overrides.camper_cabin.length === 0}
				<p class="text-muted-foreground text-sm">No camper cabin overrides.</p>
			{:else}
				<div class="grid gap-2">
					{#each overrides.camper_cabin as o (o.id)}
						<div class="flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2">
							<div class="text-sm">
								<span class="font-medium">{o.camper_name}</span>
								<span class="text-muted-foreground"> → </span>
								<span>{o.cabin_name}</span>
								<span class="text-muted-foreground"> ({o.age_group_name})</span>
							</div>
							<Button
								variant="ghost"
								size="sm"
								aria-label="Remove camper→cabin override"
								title="Remove camper→cabin override"
								disabled={deleting === o.id}
								onclick={() => confirmDelete("camper_cabin", o.id, `${o.camper_name} → ${o.cabin_name}`)}
							>
								<TrashIcon class="size-4" />
								<span class="sr-only">Remove camper→cabin override</span>
							</Button>
						</div>
					{/each}
				</div>
			{/if}

			<div class="flex flex-wrap items-end gap-2">
				<div class="grid gap-1">
					<label for="kc-camper" class="text-xs text-muted-foreground">Camper</label>
					<Select.Select
						type="single"
						value={newCamperCabinCamper ?? undefined}
						onValueChange={(v) => (newCamperCabinCamper = v)}
					>
						<Select.SelectTrigger id="kc-camper" class="w-[220px]">
							{#if newCamperCabinCamper}
								{enrollments.find((e) => e.camper_id === newCamperCabinCamper)?.camper_name ?? "?"}
							{:else}
								Select camper...
							{/if}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each availableCampersForCabin as e (e.camper_id)}
								<Select.SelectItem value={e.camper_id}>{e.camper_name}</Select.SelectItem>
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
				<div class="grid gap-1">
					<label for="kc-cabin" class="text-xs text-muted-foreground">Cabin</label>
					<Select.Select
						type="single"
						value={newCamperCabinSAGC ?? undefined}
						onValueChange={(v) => (newCamperCabinSAGC = v)}
					>
						<Select.SelectTrigger id="kc-cabin" class="w-[220px]">
							{#if newCamperCabinSAGC}
								{sessionCabinLabel(newCamperCabinSAGC)}
							{:else}
								Select cabin...
							{/if}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each sortedSessionCabins as sc (sc.id)}
								<Select.SelectItem value={sc.id}>
									{cabinNameById.get(sc.cabin_id) ?? sc.cabin_id}
								</Select.SelectItem>
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
				<Button
					size="sm"
					disabled={!newCamperCabinCamper || !newCamperCabinSAGC || creatingCamperCabin}
					onclick={addCamperCabin}
				>
					{#if creatingCamperCabin}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{:else}
						<PlusIcon class="mr-2 size-4" />
					{/if}
					Pin
				</Button>
			</div>
		</section>

		<!-- Counselor → Activity -->
		<section class="grid gap-3">
			<h3 class="text-sm font-semibold">Counselor → Activity</h3>

			{#if overrides.counselor_activity.length === 0}
				<p class="text-muted-foreground text-sm">No counselor activity overrides.</p>
			{:else}
				<div class="grid gap-2">
					{#each overrides.counselor_activity as o (o.id)}
						<div class="flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2">
							<div class="text-sm">
								<span class="font-medium">{o.counselor_name}</span>
								<span class="text-muted-foreground"> → </span>
								<span>{o.time_slot_name} — {o.activity_name}</span>
							</div>
							<Button
								variant="ghost"
								size="sm"
								aria-label="Remove counselor→activity override"
								title="Remove counselor→activity override"
								disabled={deleting === o.id}
								onclick={() => confirmDelete("counselor_activity", o.id, `${o.counselor_name} → ${o.activity_name}`)}
							>
								<TrashIcon class="size-4" />
								<span class="sr-only">Remove counselor→activity override</span>
							</Button>
						</div>
					{/each}
				</div>
			{/if}

			<div class="flex flex-wrap items-end gap-2">
				<div class="grid gap-1">
					<label for="ca-counselor" class="text-xs text-muted-foreground">Counselor</label>
					<Select.Select
						type="single"
						value={newCounselorActivityCounselor ?? undefined}
						onValueChange={(v) => (newCounselorActivityCounselor = v)}
					>
						<Select.SelectTrigger id="ca-counselor" class="w-[220px]">
							{#if newCounselorActivityCounselor}
								{counselors.find((c) => c.counselor_id === newCounselorActivityCounselor)?.counselor_name ?? "?"}
							{:else}
								Select counselor...
							{/if}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each activeCounselors as c (c.counselor_id)}
								<Select.SelectItem value={c.counselor_id}>{c.counselor_name}</Select.SelectItem>
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
				<div class="grid gap-1">
					<label for="ca-activity" class="text-xs text-muted-foreground">Activity slot</label>
					<Select.Select
						type="single"
						value={newCounselorActivitySA ?? undefined}
						onValueChange={(v) => (newCounselorActivitySA = v)}
					>
						<Select.SelectTrigger id="ca-activity" class="w-[260px]">
							{#if newCounselorActivitySA}
								{sessionActivityLabel(newCounselorActivitySA)}
							{:else}
								Select activity slot...
							{/if}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each sessionActivities as sa (sa.id)}
								<Select.SelectItem value={sa.id}>{sessionActivityLabel(sa.id)}</Select.SelectItem>
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
				<Button
					size="sm"
					disabled={!newCounselorActivityCounselor || !newCounselorActivitySA || creatingCounselorActivity}
					onclick={addCounselorActivity}
				>
					{#if creatingCounselorActivity}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{:else}
						<PlusIcon class="mr-2 size-4" />
					{/if}
					Pin
				</Button>
			</div>
		</section>
	</div>
{/if}

<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Remove Override</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Remove the override for {deleteTarget?.label ?? ""}? Any existing run for this session
				will be marked as stale.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting !== null}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction onclick={handleDelete} disabled={deleting !== null}>
				{#if deleting !== null}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{:else}
					<TrashIcon class="mr-2 size-4" />
				{/if}
				Remove
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
