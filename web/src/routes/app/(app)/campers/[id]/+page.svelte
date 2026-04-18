<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { page } from "$app/state";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import {
		camperApi,
		enrollmentApi,
		sessionAgeGroupApi,
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
		Enrollment,
		Session,
		Season,
		AgeGroup,
		SessionAgeGroup,
	} from "$lib/api/types";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
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
	let enrollLoadPartial = $state(false);
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

	// Derived: sessions the camper is NOT already enrolled in
	let availableSessionsForEnroll = $derived(
		sessions.filter((s) => !enrollments.some((e) => e.session_id === s.id))
	);

	// Derived: age groups available for the selected session
	let availableAgeGroupsForEnroll = $derived(
		sessionAgeGroups.map((sag) => {
			const ag = ageGroups.find((a) => a.id === sag.age_group_id);
			return { sessionAgeGroupId: sag.id, name: ag?.name ?? sag.age_group_id };
		})
	);

	onMount(async () => {
		try {
			const [c, sess, seas, ag] = await Promise.all([
				camperApi.get(camperId),
				sessionApi.list(),
				seasonApi.list(),
				ageGroupApi.list(),
			]);
			camper = c;
			sessions = sess;
			seasons = seas;
			ageGroups = ag;

			// Load enrollments across all sessions
			await loadAllEnrollments();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load camper";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	async function loadAllEnrollments() {
		const allEnrollments: Enrollment[] = [];
		let hadError = false;
		const results = await Promise.all(
			sessions.map((s) =>
				enrollmentApi.list(s.id).catch(() => {
					hadError = true;
					return [] as Enrollment[];
				})
			)
		);
		for (const sessionEnrollments of results) {
			allEnrollments.push(...sessionEnrollments.filter((e) => e.camper_id === camperId));
		}
		enrollments = allEnrollments;
		enrollLoadPartial = hadError;
		if (hadError) {
			toast.warning("Some enrollment data could not be loaded");
		}
	}

	function handleTabChange(tab: string) {
		activeTab = tab;
		updateUrl();
	}

	function updateUrl() {
		const params = new URLSearchParams();
		if (activeTab !== "enrollments") params.set("tab", activeTab);
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

					{#if enrollLoadPartial}
						<div class="text-warning-foreground bg-warning/10 border-warning/20 rounded-md border px-3 py-2 text-sm">
							Some enrollment data could not be loaded. Enrollment changes are disabled until all data loads successfully.
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
												disabled={disabled || enrollLoadPartial}
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
										{#each availableSessionsForEnroll as session (session.id)}
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
											disabled={disabled || enrollLoadPartial || !enrollSessionAgeGroupId || enrolling}
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

			<!-- Friend Preferences Tab (placeholder for next commit) -->
			<Tabs.Content value="friends">
				<div class="text-muted-foreground py-8 text-center text-sm">
					Friend preferences coming soon.
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
