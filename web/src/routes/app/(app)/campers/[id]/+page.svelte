<script lang="ts">
	import { getContext, onMount, onDestroy } from "svelte";
	import { page } from "$app/state";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import {
		camperApi,
		enrollmentApi,
		sessionAgeGroupApi,
		camperFriendPreferenceApi,
		sessionApi,
		seasonApi,
		ageGroupApi,
	} from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Separator } from "$lib/components/ui/separator";
	import * as Table from "$lib/components/ui/table";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import * as Tabs from "$lib/components/ui/tabs";
	import type {
		Camper,
		CamperFriendPreference,
		Enrollment,
		Session,
		Season,
		AgeGroup,
		SessionAgeGroup,
	} from "$lib/api/types";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import ArrowUpIcon from "@lucide/svelte/icons/arrow-up";
	import ArrowDownIcon from "@lucide/svelte/icons/arrow-down";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

	const getCampDisabled = getContext<() => boolean>("campDisabled");

	let disabled = $derived.by(() => getCampDisabled());

	const camperId = page.params.id!;

	let camper = $state<Camper | null>(null);
	let loading = $state(true);
	let loadError = $state(false);

	let sessions = $state<Session[]>([]);
	let seasons = $state<Season[]>([]);
	let ageGroups = $state<AgeGroup[]>([]);

	// Tab state
	let activeTab = $state(page.url.searchParams.get("tab") ?? "enrollments");

	// Enrollments
	let enrollments = $state<Enrollment[]>([]);
	let enrollLoadFailed = $state(false);
	let enrollSessionId = $state("");
	let sessionAgeGroups = $state<SessionAgeGroup[]>([]);
	let enrollSessionAgeGroupId = $state("");
	let enrollLoading = $state(false);
	let enrolling = $state(false);
	let enrollAgeGroupController: AbortController | undefined;

	// Delete enrollment
	let deleteEnrollOpen = $state(false);
	let deleteEnrollTarget = $state<Enrollment | null>(null);
	let deletingEnroll = $state(false);

	// Friend preferences
	let allCampers = $state<Camper[]>([]);
	let prefSessionId = $state(page.url.searchParams.get("prefSession") ?? "");
	let prefLoading = $state(false);
	let prefAbortController: AbortController | null = null;
	let friendPrefs = $state<CamperFriendPreference[]>([]);
	let prefSaving = $state(false);
	let addFriendId = $state("");

	// IDs of campers enrolled in the currently selected pref session.
	let sessionEnrolledCamperIds = $state<Set<string>>(new Set());

	let campersInSession = $derived(
		allCampers.filter((c) => sessionEnrolledCamperIds.has(c.id))
	);

	let availableCampersForPref = $derived(
		campersInSession.filter(
			(c) => c.id !== camperId && !friendPrefs.some((p) => p.preferred_camper_id === c.id)
		)
	);

	// Derived: sessions the camper is NOT already enrolled in
	let availableSessionsForEnroll = $derived(
		sessions.filter((s) => !enrollments.some((e) => e.session_id === s.id))
	);

	let sessionSortLabelById = $derived.by(() => {
		const seasonNameById = new Map(seasons.map((season) => [season.id, season.name]));

		return new Map(
			sessions.map((session) => {
				const seasonName = seasonNameById.get(session.season_id);
				return [session.id, seasonName ? `${seasonName} - ${session.name}` : session.name];
			})
		);
	});

	let sortedAvailableSessionsForEnroll = $derived.by(() => {
		return [...availableSessionsForEnroll].sort((a, b) => {
			const aLabel = sessionSortLabelById.get(a.id) ?? a.name;
			const bLabel = sessionSortLabelById.get(b.id) ?? b.name;
			return aLabel.localeCompare(bLabel);
		});
	});

	let sortedSessions = $derived.by(() => {
		return [...sessions].sort((a, b) => {
			const aLabel = sessionSortLabelById.get(a.id) ?? a.name;
			const bLabel = sessionSortLabelById.get(b.id) ?? b.name;
			return aLabel.localeCompare(bLabel);
		});
	});

	// Derived: age groups available for the selected session
	let availableAgeGroupsForEnroll = $derived(
		sessionAgeGroups.map((sag) => {
			const ag = ageGroups.find((a) => a.id === sag.age_group_id);
			return { sessionAgeGroupId: sag.id, name: ag?.name ?? sag.age_group_id };
		})
	);

	onDestroy(() => {
		prefAbortController?.abort();
		enrollAgeGroupController?.abort();
		enrollAgeGroupController = undefined;
	});

	onMount(async () => {
		try {
			const [c, sess, seas, ag, campers] = await Promise.all([
				camperApi.get(camperId),
				sessionApi.list(),
				seasonApi.list(),
				ageGroupApi.list(),
				camperApi.list(),
			]);
			camper = c;
			sessions = sess;
			seasons = seas;
			ageGroups = ag;
			allCampers = campers;

			await loadAllEnrollments();

			if (prefSessionId && sessions.some((s) => s.id === prefSessionId)) {
				loadFriendPreferences(prefSessionId);
			} else {
				prefSessionId = "";
			}
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load camper";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	async function loadAllEnrollments() {
		try {
			enrollments = await enrollmentApi.listByCamper(camperId);
			enrollLoadFailed = false;
		} catch (err) {
			enrollments = [];
			enrollLoadFailed = true;
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load enrollment data";
			toast.warning(message);
		}
	}

	function handleTabChange(tab: string) {
		activeTab = tab;
		updateUrl();
	}

	function updateUrl() {
		const params = new URLSearchParams();
		if (activeTab !== "enrollments") params.set("tab", activeTab);
		if (prefSessionId) params.set("prefSession", prefSessionId);
		const qs = params.toString();
		goto(`?${qs}`, { replaceState: true, keepFocus: true, noScroll: true });
	}

	// Enrollment handlers
	async function handleEnrollSessionChange(sessionId: string | undefined) {
		enrollAgeGroupController?.abort();
		enrollSessionId = sessionId ?? "";
		enrollSessionAgeGroupId = "";
		sessionAgeGroups = [];

		if (!enrollSessionId) return;

		const controller = new AbortController();
		enrollAgeGroupController = controller;
		enrollLoading = true;
		try {
			sessionAgeGroups = await sessionAgeGroupApi.list(enrollSessionId, controller.signal);
		} catch (err) {
			if (controller.signal.aborted) return;
			const message = err instanceof ApiClientError ? err.message : "Failed to load age groups";
			toast.error(message);
		} finally {
			if (!controller.signal.aborted) {
				enrollLoading = false;
			}
		}
	}

	async function handleEnroll() {
		if (!enrollSessionId || !enrollSessionAgeGroupId) return;
		enrolling = true;

		try {
			const created = await enrollmentApi.create(enrollSessionId, {
				camper_id: camperId,
				session_age_group_id: enrollSessionAgeGroupId,
			});
			enrollments = [...enrollments, created];
			enrollSessionId = "";
			enrollSessionAgeGroupId = "";
			sessionAgeGroups = [];
			toast.success("Enrolled in session");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to enroll camper";
			toast.error(message);
		} finally {
			enrolling = false;
		}
	}

	function confirmDeleteEnroll(enrollment: Enrollment) {
		deleteEnrollTarget = enrollment;
		deleteEnrollOpen = true;
	}

	async function handleDeleteEnroll() {
		if (!deleteEnrollTarget) return;
		deletingEnroll = true;

		try {
			await enrollmentApi.delete(deleteEnrollTarget.session_id, deleteEnrollTarget.id);
			enrollments = enrollments.filter((e) => e.id !== deleteEnrollTarget!.id);
			toast.success("Enrollment removed");
			deleteEnrollOpen = false;
			deleteEnrollTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to remove enrollment";
			toast.error(message);
		} finally {
			deletingEnroll = false;
		}
	}

	// Friend preference handlers
	async function loadFriendPreferences(sessionId: string) {
		prefAbortController?.abort();
		prefAbortController = null;
		friendPrefs = [];

		if (!sessionId) {
			friendPrefs = [];
			sessionEnrolledCamperIds = new Set();
			prefLoading = false;
			return;
		}

		const controller = new AbortController();
		prefAbortController = controller;
		const { signal } = controller;
		prefLoading = true;

		try {
			const [prefs, sessEnrollments] = await Promise.all([
				camperFriendPreferenceApi.list(sessionId, camperId, signal),
				enrollmentApi.list(sessionId),
			]);
			if (signal.aborted) return;
			friendPrefs = prefs.sort((a, b) => a.rank - b.rank);
			sessionEnrolledCamperIds = new Set(sessEnrollments.map((e) => e.camper_id));
		} catch (err) {
			if (signal.aborted) return;
			const message = err instanceof ApiClientError ? err.message : "Failed to load friend preferences";
			toast.error(message);
		} finally {
			if (!signal.aborted) prefLoading = false;
		}
	}

	function handlePrefSessionChange(sessionId: string | undefined) {
		prefSessionId = sessionId ?? "";
		addFriendId = "";
		updateUrl();
		loadFriendPreferences(prefSessionId);
	}

	async function addFriendPref() {
		if (!addFriendId || !prefSessionId || prefSaving) return;
		prefSaving = true;
		const items = [
			...friendPrefs.map((p, i) => ({ preferred_camper_id: p.preferred_camper_id, rank: i + 1 })),
			{ preferred_camper_id: addFriendId, rank: friendPrefs.length + 1 },
		];
		try {
			friendPrefs = await camperFriendPreferenceApi.replaceAll(prefSessionId, camperId, items);
			friendPrefs = friendPrefs.sort((a, b) => a.rank - b.rank);
			addFriendId = "";
			toast.success("Friend preference added");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function moveFriendPref(index: number, direction: -1 | 1) {
		if (prefSaving) return;
		const newIndex = index + direction;
		const reordered = [...friendPrefs];
		[reordered[index], reordered[newIndex]] = [reordered[newIndex], reordered[index]];
		const items = reordered.map((p, i) => ({ preferred_camper_id: p.preferred_camper_id, rank: i + 1 }));
		prefSaving = true;
		try {
			friendPrefs = await camperFriendPreferenceApi.replaceAll(prefSessionId, camperId, items);
			friendPrefs = friendPrefs.sort((a, b) => a.rank - b.rank);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to reorder preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function removeFriendPref(index: number) {
		if (prefSaving) return;
		const remaining = friendPrefs.filter((_, i) => i !== index);
		const items = remaining.map((p, i) => ({ preferred_camper_id: p.preferred_camper_id, rank: i + 1 }));
		prefSaving = true;
		try {
			friendPrefs = await camperFriendPreferenceApi.replaceAll(prefSessionId, camperId, items);
			friendPrefs = friendPrefs.sort((a, b) => a.rank - b.rank);
			toast.success("Friend preference removed");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	function getCamperName(id: string): string {
		return allCampers.find((c) => c.id === id)?.name ?? id;
	}

	// Display helpers
	function getSessionLabel(session: Session): string {
		const season = seasons.find((s) => s.id === session.season_id);
		return season ? `${season.name} - ${session.name}` : session.name;
	}

	function getSessionName(sessionId: string): string {
		const session = sessions.find((s) => s.id === sessionId);
		return session ? getSessionLabel(session) : sessionId;
	}

	function getAgeGroupName(ageGroupId: string): string {
		return ageGroups.find((ag) => ag.id === ageGroupId)?.name ?? ageGroupId;
	}
</script>

<div class="grid gap-6">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon-sm" onclick={() => goto("/app/campers")}>
			<ArrowLeftIcon class="size-4" />
			<span class="sr-only">Back</span>
		</Button>
		<div class="flex-1">
			{#if loading}
				<h1 class="text-2xl font-semibold tracking-tight">Loading...</h1>
			{:else if camper}
				<h1 class="text-2xl font-semibold tracking-tight">{camper.name}</h1>
			{:else}
				<h1 class="text-2xl font-semibold tracking-tight">Camper not found</h1>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if loadError || !camper}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load camper. Try refreshing the page.
		</div>
	{:else}
		<Tabs.Root value={activeTab} onValueChange={handleTabChange}>
			<Tabs.List>
				<Tabs.Trigger value="enrollments">Session Enrollments</Tabs.Trigger>
				<Tabs.Trigger value="friends">Friend Preferences</Tabs.Trigger>
			</Tabs.List>

			<!-- Enrollments Tab -->
			<Tabs.Content value="enrollments">
				<div class="grid gap-4 pt-4">
					<p class="text-muted-foreground text-sm">
						Sessions this camper is enrolled in.
					</p>

					{#if enrollLoadFailed}
						<div class="text-warning-foreground bg-warning/10 border-warning/20 rounded-md border px-3 py-2 text-sm">
							Failed to load enrollment data. Try refreshing the page.
						</div>
					{/if}

					{#if enrollments.length === 0}
						<div class="text-muted-foreground py-8 text-center text-sm">
							Not enrolled in any sessions yet.
						</div>
					{:else}
						<Table.Table>
							<Table.TableHeader>
								<Table.TableRow>
									<Table.TableHead>Session</Table.TableHead>
									<Table.TableHead>Age Group</Table.TableHead>
									<Table.TableHead class="w-16">
										<span class="sr-only">Actions</span>
									</Table.TableHead>
								</Table.TableRow>
							</Table.TableHeader>
							<Table.TableBody>
								{#each enrollments as enrollment (enrollment.id)}
									<Table.TableRow>
										<Table.TableCell>{getSessionName(enrollment.session_id)}</Table.TableCell>
										<Table.TableCell>{getAgeGroupName(enrollment.age_group_id)}</Table.TableCell>
										<Table.TableCell>
											<Button
												variant="ghost"
												size="icon-sm"
												disabled={disabled || enrollLoadFailed}
												title="Remove enrollment"
												onclick={() => confirmDeleteEnroll(enrollment)}
											>
												<TrashIcon class="size-4" />
												<span class="sr-only">Remove</span>
											</Button>
										</Table.TableCell>
									</Table.TableRow>
								{/each}
							</Table.TableBody>
						</Table.Table>
					{/if}

					<Separator />

					<!-- Add enrollment -->
					<div class="grid gap-3">
						<h3 class="text-sm font-medium">Enroll in Session</h3>
						{#if availableSessionsForEnroll.length === 0 && !loading}
							<p class="text-muted-foreground text-sm">
								{#if sessions.length === 0}
									No sessions found. <a href="/app/sessions" class="text-foreground underline">Create a session</a> first.
								{:else}
									Already enrolled in all available sessions.
								{/if}
							</p>
						{:else}
							<div class="flex items-center gap-2">
								<Select.Root
									type="single"
									value={enrollSessionId}
									onValueChange={handleEnrollSessionChange}
								>
									<Select.Trigger class="w-64">
										{enrollSessionId
											? getSessionName(enrollSessionId)
											: "Select session"}
									</Select.Trigger>
									<Select.Content>
										{#each sortedAvailableSessionsForEnroll as session (session.id)}
											<Select.Item value={session.id}>{getSessionLabel(session)}</Select.Item>
										{/each}
									</Select.Content>
								</Select.Root>

								{#if enrollSessionId}
									{#if enrollLoading}
										<LoaderCircleIcon class="size-4 animate-spin text-muted-foreground" />
									{:else if availableAgeGroupsForEnroll.length > 0}
										<Select.Root
											type="single"
											value={enrollSessionAgeGroupId}
											onValueChange={(v) => (enrollSessionAgeGroupId = v ?? "")}
										>
											<Select.Trigger class="w-48">
												{enrollSessionAgeGroupId
													? availableAgeGroupsForEnroll.find((a) => a.sessionAgeGroupId === enrollSessionAgeGroupId)?.name ?? "Select age group"
													: "Select age group"}
											</Select.Trigger>
											<Select.Content>
												{#each availableAgeGroupsForEnroll as ag (ag.sessionAgeGroupId)}
													<Select.Item value={ag.sessionAgeGroupId}>{ag.name}</Select.Item>
												{/each}
											</Select.Content>
										</Select.Root>

										<Button
											size="sm"
											disabled={disabled || enrollLoadFailed || !enrollSessionAgeGroupId || enrolling}
											onclick={handleEnroll}
										>
											{#if enrolling}
												<LoaderCircleIcon class="mr-1 size-4 animate-spin" />
											{/if}
											<PlusIcon class="mr-1 size-4" />
											Enroll
										</Button>
									{:else}
										<p class="text-muted-foreground text-sm">
											No age groups configured for this session.
										</p>
									{/if}
								{/if}
							</div>
						{/if}
					</div>
				</div>
			</Tabs.Content>

			<!-- Friend Preferences Tab -->
			<Tabs.Content value="friends">
				<div class="grid gap-6 pt-4">
					<!-- Session selector -->
					<div class="flex items-center gap-4">
						<p class="text-muted-foreground text-sm">Manage ranked friend preferences for a session.</p>
						{#if sessions.length > 0}
							<Select.Root
								type="single"
								value={prefSessionId}
								onValueChange={handlePrefSessionChange}
								disabled={prefSaving}
							>
								<Select.Trigger class="w-64">
									{prefSessionId
										? getSessionName(prefSessionId)
										: "Select session"}
								</Select.Trigger>
								<Select.Content>
									{#each sortedSessions as session (session.id)}
										<Select.Item value={session.id}>{getSessionLabel(session)}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						{:else if !loading}
							<p class="text-muted-foreground text-sm">
								No sessions found. <a href="/app/sessions" class="text-foreground underline">Create a session</a> to manage preferences.
							</p>
						{/if}
					</div>

					{#if !prefSessionId}
						<div class="text-muted-foreground py-8 text-center text-sm">
							Select a session to manage friend preferences.
						</div>
					{:else if prefLoading}
						<div class="text-muted-foreground py-8 text-center text-sm">Loading preferences...</div>
					{:else}
						<div class="grid gap-3">
							{#if friendPrefs.length === 0}
								<div class="text-muted-foreground py-4 text-center text-sm">No friend preferences.</div>
							{:else}
								<Table.Table>
									<Table.TableHeader>
										<Table.TableRow>
											<Table.TableHead class="w-12">#</Table.TableHead>
											<Table.TableHead>Friend</Table.TableHead>
											<Table.TableHead class="w-28">
												<span class="sr-only">Actions</span>
											</Table.TableHead>
										</Table.TableRow>
									</Table.TableHeader>
									<Table.TableBody>
										{#each friendPrefs as pref, i (pref.id)}
											<Table.TableRow>
												<Table.TableCell class="text-muted-foreground">{pref.rank}</Table.TableCell>
												<Table.TableCell>{getCamperName(pref.preferred_camper_id)}</Table.TableCell>
												<Table.TableCell>
													<div class="flex justify-end gap-1">
														<Button variant="ghost" size="icon-sm" title="Move up" disabled={disabled || prefSaving || i === 0} onclick={() => moveFriendPref(i, -1)}>
															<ArrowUpIcon class="size-4" />
															<span class="sr-only">Move up</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Move down" disabled={disabled || prefSaving || i === friendPrefs.length - 1} onclick={() => moveFriendPref(i, 1)}>
															<ArrowDownIcon class="size-4" />
															<span class="sr-only">Move down</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Remove" disabled={disabled || prefSaving} onclick={() => removeFriendPref(i)}>
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
							{#if availableCampersForPref.length > 0}
								<div class="flex items-center gap-2">
									<Select.Root
										type="single"
										value={addFriendId}
										onValueChange={(v) => (addFriendId = v ?? "")}
									>
										<Select.Trigger class="w-48">
											{addFriendId
												? availableCampersForPref.find((c) => c.id === addFriendId)?.name ?? "Select camper"
												: "Select camper"}
										</Select.Trigger>
										<Select.Content>
											{#each availableCampersForPref as c (c.id)}
												<Select.Item value={c.id}>{c.name}</Select.Item>
											{/each}
										</Select.Content>
									</Select.Root>
									<Button size="sm" disabled={disabled || prefSaving || !addFriendId} onclick={addFriendPref}>
										<PlusIcon class="mr-1 size-4" />
										Add
									</Button>
								</div>
							{:else if allCampers.length <= 1}
								<p class="text-muted-foreground text-sm">
									No other campers available. <a href="/app/campers" class="text-foreground underline">Add more campers</a> to set friend preferences.
								</p>
							{/if}
						</div>
					{/if}
				</div>
			</Tabs.Content>
		</Tabs.Root>
	{/if}
</div>

<!-- Delete enrollment confirmation -->
<AlertDialog.AlertDialog bind:open={deleteEnrollOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Remove Enrollment</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to remove this camper from the session? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deletingEnroll}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				disabled={deletingEnroll}
				onclick={handleDeleteEnroll}
			>
				{#if deletingEnroll}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Remove
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
