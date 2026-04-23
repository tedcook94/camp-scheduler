<script lang="ts">
	import { getContext, onMount, onDestroy } from "svelte";
	import { page } from "$app/state";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import {
		counselorApi,
		counselorCertificationApi,
		certificationApi,
		sessionHistoryApi,
		seasonApi,
		sessionApi,
		ageGroupApi,
		cabinApi,
		activityApi,
		ageGroupPreferenceApi,
		cocounselorPreferenceApi,
		activityPreferenceApi,
		sessionAgeGroupApi,
		sessionActivityApi,
		sessionCounselorApi,
	} from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Badge } from "$lib/components/ui/badge";
	import { Label } from "$lib/components/ui/label";
	import { Separator } from "$lib/components/ui/separator";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import * as Tabs from "$lib/components/ui/tabs";
	import type {
		Counselor,
		CounselorCertification,
		Certification,
		HistorySummary,
		SessionHistory,
		Season,
		Session,
		AgeGroup,
		Cabin,
		Activity,
		AgeGroupPreference,
		CocounselorPreference,
		ActivityPreference,
	} from "$lib/api/types";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import ArrowUpIcon from "@lucide/svelte/icons/arrow-up";
	import ArrowDownIcon from "@lucide/svelte/icons/arrow-down";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import { sortItems, type SortDirection } from "$lib/utils";

	const getCampDisabled = getContext<() => boolean>("campDisabled");

	let disabled = $derived.by(() => getCampDisabled());

	const counselorId = page.params.id!;

	// Core data
	let counselor = $state<Counselor | null>(null);
	let loading = $state(true);
	let loadError = $state(false);

	// Certifications
	let counselorCerts = $state<CounselorCertification[]>([]);
	let allCertifications = $state<Certification[]>([]);
	let addCertOpen = $state(false);
	let addCertId = $state("");
	let addingCert = $state(false);
	let deleteCertOpen = $state(false);
	let deleteCertTarget = $state<CounselorCertification | null>(null);
	let deletingCert = $state(false);

	let availableCerts = $derived(
		allCertifications.filter(
			(c) => !counselorCerts.some((cc) => cc.certification_id === c.id)
		)
	);

	// Session history
	let historySummary = $state<HistorySummary[]>([]);
	let seasons = $state<Season[]>([]);
	let sessions = $state<Session[]>([]);
	let ageGroups = $state<AgeGroup[]>([]);
	let cabins = $state<Cabin[]>([]);
	let historyFilterSeasonId = $state(page.url.searchParams.get("season") ?? "");

	let seasonNamesById = $derived(new Map(seasons.map((s) => [s.id, s.name])));

	let sortedSessions = $derived.by(() => {
		return [...sessions].sort((a, b) => {
			const aSeasonName = seasonNamesById.get(a.season_id);
			const bSeasonName = seasonNamesById.get(b.season_id);
			const aLabel = aSeasonName ? `${aSeasonName} - ${a.name}` : a.name;
			const bLabel = bSeasonName ? `${bSeasonName} - ${b.name}` : b.name;
			return aLabel.localeCompare(bLabel);
		});
	});

	let filteredHistory = $derived(
		historyFilterSeasonId
			? historySummary.filter((h) => h.season_id === historyFilterSeasonId)
			: historySummary
	);

	let historySortKey = $state<"session_name" | "season_name" | "age_group_name" | "cabin_name">("session_name");
	let historySortDir = $state<SortDirection>("asc");
	let sortedHistory = $derived(sortItems(filteredHistory, historySortKey, historySortDir));

	function toggleHistorySort(key: typeof historySortKey) {
		if (historySortKey === key) {
			historySortDir = historySortDir === "asc" ? "desc" : "asc";
		} else {
			historySortKey = key;
			historySortDir = "asc";
		}
	}

	// History create/edit dialog
	let historyDialogOpen = $state(false);
	let editingHistory = $state<HistorySummary | null>(null);
	let editingHistoryRecord = $state<SessionHistory | null>(null);
	let formSessionId = $state("");
	let formAgeGroupId = $state("");
	let formCabinId = $state("");
	let submittingHistory = $state(false);
	let historyFormError = $state("");

	let historyDialogTitle = $derived(editingHistory ? "Edit Session History" : "Add Session History");

	// History delete
	let deleteHistoryOpen = $state(false);
	let deleteHistoryTarget = $state<HistorySummary | null>(null);
	let deletingHistory = $state(false);

	// Preferences
	let allActivities = $state<Activity[]>([]);
	let allCounselors = $state<Counselor[]>([]);
	let activeTab = $state(page.url.searchParams.get("tab") ?? "certifications");
	let prefSessionId = $state(page.url.searchParams.get("prefSession") ?? "");
	let prefLoading = $state(false);
	let prefAbortController: AbortController | null = null;

	let ageGroupPrefs = $state<AgeGroupPreference[]>([]);
	let cocounselorPrefs = $state<CocounselorPreference[]>([]);
	let activityPrefs = $state<ActivityPreference[]>([]);
	let prefSaving = $state(false);

	// Session-scoped option sources, populated when a pref session is selected.
	let sessionAgeGroupIds = $state<Set<string>>(new Set());
	let sessionActivityIds = $state<Set<string>>(new Set());
	let sessionCounselorIds = $state<Set<string>>(new Set());

	let addAgeGroupId = $state("");
	let addCocounselorId = $state("");
	let addActivityId = $state("");

	let ageGroupsInSession = $derived(
		ageGroups.filter((ag) => sessionAgeGroupIds.has(ag.id))
	);
	let counselorsInSession = $derived(
		allCounselors.filter((c) => sessionCounselorIds.has(c.id))
	);
	let activitiesInSession = $derived(
		allActivities.filter((a) => sessionActivityIds.has(a.id))
	);

	let availableAgeGroupsForPref = $derived(
		ageGroupsInSession.filter((ag) => !ageGroupPrefs.some((p) => p.age_group_id === ag.id))
	);
	let availableCounselorsForPref = $derived(
		counselorsInSession.filter(
			(c) => c.id !== counselorId && !cocounselorPrefs.some((p) => p.preferred_counselor_id === c.id)
		)
	);
	let availableActivitiesForPref = $derived(
		activitiesInSession.filter((a) => !activityPrefs.some((p) => p.activity_id === a.id))
	);

	onDestroy(() => {
		prefAbortController?.abort();
	});

	onMount(async () => {
		try {
			const [c, certs, allCerts, summary, s, sess, ag, cab, acts, couns] = await Promise.all([
				counselorApi.get(counselorId),
				counselorCertificationApi.list(counselorId),
				certificationApi.list(),
				sessionHistoryApi.summary(counselorId),
				seasonApi.list(),
				sessionApi.list(),
				ageGroupApi.list(),
				cabinApi.list(),
				activityApi.list(),
				counselorApi.list(),
			]);
			counselor = c;
			counselorCerts = certs;
			allCertifications = allCerts;
			historySummary = summary;
			seasons = s;
			sessions = sess;
			ageGroups = ag;
			cabins = cab;
			allActivities = acts;
			allCounselors = couns;

			if (prefSessionId && sessions.some((s) => s.id === prefSessionId)) {
				loadPreferences(prefSessionId);
			} else {
				prefSessionId = "";
			}
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load counselor";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	// Certification handlers
	function openAddCert() {
		addCertId = availableCerts.length > 0 ? availableCerts[0].id : "";
		addCertOpen = true;
	}

	async function handleAddCert() {
		if (!addCertId) return;
		addingCert = true;
		try {
			const created = await counselorCertificationApi.add(counselorId, {
				certification_id: addCertId,
			});
			counselorCerts = [...counselorCerts, created];
			toast.success("Certification added");
			addCertOpen = false;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to add certification";
			toast.error(message);
		} finally {
			addingCert = false;
		}
	}

	function confirmDeleteCert(cert: CounselorCertification) {
		deleteCertTarget = cert;
		deleteCertOpen = true;
	}

	async function handleDeleteCert() {
		if (!deleteCertTarget) return;
		deletingCert = true;
		try {
			await counselorCertificationApi.remove(counselorId, deleteCertTarget.id);
			counselorCerts = counselorCerts.filter((c) => c.id !== deleteCertTarget!.id);
			toast.success("Certification removed");
			deleteCertOpen = false;
			deleteCertTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to remove certification";
			toast.error(message);
		} finally {
			deletingCert = false;
		}
	}

	// Session history handlers
	function openAddHistory() {
		editingHistory = null;
		editingHistoryRecord = null;
		formSessionId = sessions.length > 0 ? sessions[0].id : "";
		formAgeGroupId = ageGroups.length > 0 ? ageGroups[0].id : "";
		formCabinId = "";
		historyFormError = "";
		historyDialogOpen = true;
	}

	async function openEditHistory(entry: HistorySummary) {
		editingHistory = entry;
		historyFormError = "";
		try {
			const record = await sessionHistoryApi.get(counselorId, entry.id);
			editingHistoryRecord = record;
			formSessionId = record.session_id;
			formAgeGroupId = record.age_group_id;
			formCabinId = record.cabin_id ?? "";
			historyDialogOpen = true;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load history record";
			toast.error(message);
		}
	}

	async function handleHistorySubmit(e: SubmitEvent) {
		e.preventDefault();
		historyFormError = "";

		if (!formSessionId || !formAgeGroupId) {
			historyFormError = "Session and age group are required.";
			return;
		}

		submittingHistory = true;
		const data = {
			session_id: formSessionId,
			age_group_id: formAgeGroupId,
			cabin_id: formCabinId || undefined,
		};

		try {
			if (editingHistory && editingHistoryRecord) {
				await sessionHistoryApi.update(counselorId, editingHistory.id, data);
				toast.success("Session history updated");
			} else {
				await sessionHistoryApi.create(counselorId, data);
				toast.success("Session history added");
			}
			// Refresh summary
			historySummary = await sessionHistoryApi.summary(counselorId);
			historyDialogOpen = false;
		} catch (err) {
			const action = editingHistory ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} session history`;
			toast.error(message);
		} finally {
			submittingHistory = false;
		}
	}

	function confirmDeleteHistory(entry: HistorySummary) {
		deleteHistoryTarget = entry;
		deleteHistoryOpen = true;
	}

	async function handleDeleteHistory() {
		if (!deleteHistoryTarget) return;
		deletingHistory = true;
		try {
			await sessionHistoryApi.delete(counselorId, deleteHistoryTarget.id);
			historySummary = historySummary.filter((h) => h.id !== deleteHistoryTarget!.id);
			toast.success("Session history deleted");
			deleteHistoryOpen = false;
			deleteHistoryTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete session history";
			toast.error(message);
		} finally {
			deletingHistory = false;
		}
	}

	// Preference handlers
	async function loadPreferences(sessionId: string) {
		prefAbortController?.abort();
		prefAbortController = null;

		if (!sessionId) {
			ageGroupPrefs = [];
			cocounselorPrefs = [];
			activityPrefs = [];
			sessionAgeGroupIds = new Set();
			sessionActivityIds = new Set();
			sessionCounselorIds = new Set();
			prefLoading = false;
			return;
		}

		const controller = new AbortController();
		prefAbortController = controller;
		const { signal } = controller;
		prefLoading = true;

		try {
			const [agp, cop, acp, sessAg, sessAct, sessCoun] = await Promise.all([
				ageGroupPreferenceApi.list(sessionId, counselorId, signal),
				cocounselorPreferenceApi.list(sessionId, counselorId, signal),
				activityPreferenceApi.list(sessionId, counselorId, signal),
				sessionAgeGroupApi.list(sessionId, signal),
				sessionActivityApi.listAll(sessionId),
				sessionCounselorApi.list(sessionId),
			]);
			if (signal.aborted) return;
			ageGroupPrefs = agp.sort((a, b) => a.rank - b.rank);
			cocounselorPrefs = cop.sort((a, b) => a.rank - b.rank);
			activityPrefs = acp.sort((a, b) => a.rank - b.rank);
			sessionAgeGroupIds = new Set(sessAg.map((s) => s.age_group_id));
			sessionActivityIds = new Set(sessAct.map((s) => s.activity_id));
			sessionCounselorIds = new Set(sessCoun.map((s) => s.counselor_id));
		} catch (err) {
			if (signal.aborted) return;
			const message = err instanceof ApiClientError ? err.message : "Failed to load preferences";
			toast.error(message);
		} finally {
			if (!signal.aborted) prefLoading = false;
		}
	}

	function handlePrefSessionChange(sessionId: string | undefined) {
		prefSessionId = sessionId ?? "";
		addAgeGroupId = "";
		addCocounselorId = "";
		addActivityId = "";
		updateUrl();
		loadPreferences(prefSessionId);
	}

	function handleTabChange(tab: string) {
		activeTab = tab;
		updateUrl();
	}

	function updateUrl() {
		const params = new URLSearchParams();
		if (activeTab !== "certifications") params.set("tab", activeTab);
		if (prefSessionId) params.set("prefSession", prefSessionId);
		if (historyFilterSeasonId) params.set("season", historyFilterSeasonId);
		const qs = params.toString();
		goto(`?${qs}`, { replaceState: true, keepFocus: true, noScroll: true });
	}

	async function addAgeGroupPref() {
		if (!addAgeGroupId || !prefSessionId || prefSaving) return;
		prefSaving = true;
		const items = [
			...ageGroupPrefs.map((p, i) => ({ age_group_id: p.age_group_id, rank: i + 1 })),
			{ age_group_id: addAgeGroupId, rank: ageGroupPrefs.length + 1 },
		];
		try {
			ageGroupPrefs = await ageGroupPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			ageGroupPrefs = ageGroupPrefs.sort((a, b) => a.rank - b.rank);
			addAgeGroupId = "";
			toast.success("Age group preference added");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function addCocounselorPref() {
		if (!addCocounselorId || !prefSessionId || prefSaving) return;
		prefSaving = true;
		const items = [
			...cocounselorPrefs.map((p, i) => ({ preferred_counselor_id: p.preferred_counselor_id, rank: i + 1 })),
			{ preferred_counselor_id: addCocounselorId, rank: cocounselorPrefs.length + 1 },
		];
		try {
			cocounselorPrefs = await cocounselorPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			cocounselorPrefs = cocounselorPrefs.sort((a, b) => a.rank - b.rank);
			addCocounselorId = "";
			toast.success("Co-counselor preference added");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function addActivityPref() {
		if (!addActivityId || !prefSessionId || prefSaving) return;
		prefSaving = true;
		const items = [
			...activityPrefs.map((p, i) => ({ activity_id: p.activity_id, rank: i + 1 })),
			{ activity_id: addActivityId, rank: activityPrefs.length + 1 },
		];
		try {
			activityPrefs = await activityPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			activityPrefs = activityPrefs.sort((a, b) => a.rank - b.rank);
			addActivityId = "";
			toast.success("Activity preference added");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function moveAgeGroupPref(index: number, direction: -1 | 1) {
		if (prefSaving) return;
		const newIndex = index + direction;
		const reordered = [...ageGroupPrefs];
		[reordered[index], reordered[newIndex]] = [reordered[newIndex], reordered[index]];
		const items = reordered.map((p, i) => ({ age_group_id: p.age_group_id, rank: i + 1 }));
		prefSaving = true;
		try {
			ageGroupPrefs = await ageGroupPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			ageGroupPrefs = ageGroupPrefs.sort((a, b) => a.rank - b.rank);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to reorder preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function moveCocounselorPref(index: number, direction: -1 | 1) {
		if (prefSaving) return;
		const newIndex = index + direction;
		const reordered = [...cocounselorPrefs];
		[reordered[index], reordered[newIndex]] = [reordered[newIndex], reordered[index]];
		const items = reordered.map((p, i) => ({ preferred_counselor_id: p.preferred_counselor_id, rank: i + 1 }));
		prefSaving = true;
		try {
			cocounselorPrefs = await cocounselorPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			cocounselorPrefs = cocounselorPrefs.sort((a, b) => a.rank - b.rank);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to reorder preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function moveActivityPref(index: number, direction: -1 | 1) {
		if (prefSaving) return;
		const newIndex = index + direction;
		const reordered = [...activityPrefs];
		[reordered[index], reordered[newIndex]] = [reordered[newIndex], reordered[index]];
		const items = reordered.map((p, i) => ({ activity_id: p.activity_id, rank: i + 1 }));
		prefSaving = true;
		try {
			activityPrefs = await activityPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			activityPrefs = activityPrefs.sort((a, b) => a.rank - b.rank);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to reorder preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function removeAgeGroupPref(index: number) {
		if (prefSaving) return;
		const remaining = ageGroupPrefs.filter((_, i) => i !== index);
		const items = remaining.map((p, i) => ({ age_group_id: p.age_group_id, rank: i + 1 }));
		prefSaving = true;
		try {
			ageGroupPrefs = await ageGroupPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			ageGroupPrefs = ageGroupPrefs.sort((a, b) => a.rank - b.rank);
			toast.success("Age group preference removed");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function removeCocounselorPref(index: number) {
		if (prefSaving) return;
		const remaining = cocounselorPrefs.filter((_, i) => i !== index);
		const items = remaining.map((p, i) => ({ preferred_counselor_id: p.preferred_counselor_id, rank: i + 1 }));
		prefSaving = true;
		try {
			cocounselorPrefs = await cocounselorPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			cocounselorPrefs = cocounselorPrefs.sort((a, b) => a.rank - b.rank);
			toast.success("Co-counselor preference removed");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	async function removeActivityPref(index: number) {
		if (prefSaving) return;
		const remaining = activityPrefs.filter((_, i) => i !== index);
		const items = remaining.map((p, i) => ({ activity_id: p.activity_id, rank: i + 1 }));
		prefSaving = true;
		try {
			activityPrefs = await activityPreferenceApi.replaceAll(prefSessionId, counselorId, items);
			activityPrefs = activityPrefs.sort((a, b) => a.rank - b.rank);
			toast.success("Activity preference removed");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update preferences";
			toast.error(message);
		} finally {
			prefSaving = false;
		}
	}

	function getSessionLabel(session: Session | undefined): string {
		if (!session) return "Select session";
		const season = seasons.find((s) => s.id === session.season_id);
		return season ? `${season.name} - ${session.name}` : session.name;
	}

	function getAgeGroupName(id: string) {
		return ageGroups.find((ag) => ag.id === id)?.name ?? id;
	}

	function getCounselorName(id: string) {
		return allCounselors.find((c) => c.id === id)?.name ?? id;
	}

	function getActivityName(id: string) {
		return allActivities.find((a) => a.id === id)?.name ?? id;
	}
</script>

<div class="grid gap-6">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon-sm" onclick={() => goto("/app/counselors")}>
			<ArrowLeftIcon class="size-4" />
			<span class="sr-only">Back</span>
		</Button>
		<div class="flex-1">
			{#if loading}
				<h1 class="text-2xl font-semibold tracking-tight">Loading...</h1>
			{:else if counselor}
				<div class="flex items-center gap-3">
					<h1 class="text-2xl font-semibold tracking-tight">{counselor.name}</h1>
					<Badge variant={counselor.junior_counselor ? "secondary" : "default"}>
						{counselor.junior_counselor ? "Junior" : "Senior"}
					</Badge>
					<Badge variant={counselor.gender === "female" ? "secondary" : "outline"}>
						{counselor.gender === "female" ? "Female" : "Male"}
					</Badge>
					<Badge variant={counselor.enabled ? "default" : "outline"}>
						{counselor.enabled ? "Active" : "Inactive"}
					</Badge>
				</div>
			{:else}
				<h1 class="text-2xl font-semibold tracking-tight">Counselor not found</h1>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if loadError || !counselor}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load counselor. Try refreshing the page.
		</div>
	{:else}
		<Tabs.Root value={activeTab} onValueChange={handleTabChange}>
			<Tabs.List>
				<Tabs.Trigger value="certifications">Certifications</Tabs.Trigger>
				<Tabs.Trigger value="history">Session History</Tabs.Trigger>
				<Tabs.Trigger value="preferences">Preferences</Tabs.Trigger>
			</Tabs.List>

			<!-- Certifications Tab -->
			<Tabs.Content value="certifications">
				<div class="grid gap-4 pt-4">
					<div class="flex items-center justify-between">
						<p class="text-muted-foreground text-sm">
							Certifications held by this counselor.
						</p>
						<Button
							size="sm"
							disabled={disabled || availableCerts.length === 0}
							onclick={openAddCert}
						>
							<PlusIcon class="mr-2 size-4" />
							Add Certification
						</Button>
					</div>

					{#if counselorCerts.length === 0}
						<div class="text-muted-foreground py-8 text-center text-sm">
							No certifications yet.
						</div>
					{:else}
						<Table.Table>
							<Table.TableHeader>
								<Table.TableRow>
									<Table.TableHead>Certification</Table.TableHead>
									<Table.TableHead class="w-16">
										<span class="sr-only">Actions</span>
									</Table.TableHead>
								</Table.TableRow>
							</Table.TableHeader>
							<Table.TableBody>
								{#each counselorCerts as cert (cert.id)}
									<Table.TableRow>
										<Table.TableCell>
											{cert.certification_name ?? allCertifications.find((c) => c.id === cert.certification_id)?.name ?? cert.certification_id}
										</Table.TableCell>
										<Table.TableCell>
											<Button
												variant="ghost"
												size="icon-sm"
												disabled={disabled}
												title="Remove"
												onclick={() => confirmDeleteCert(cert)}
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
				</div>
			</Tabs.Content>

			<!-- Session History Tab -->
			<Tabs.Content value="history">
				<div class="grid gap-4 pt-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-4">
							<p class="text-muted-foreground text-sm">Past session assignments.</p>
							{#if seasons.length > 0}
								<Select.Root
									type="single"
									value={historyFilterSeasonId}
									onValueChange={(v) => {
									historyFilterSeasonId = v ?? "";
									updateUrl();
								}}
								>
									<Select.Trigger class="w-48">
										{historyFilterSeasonId
											? seasons.find((s) => s.id === historyFilterSeasonId)?.name ?? "Filter by season"
											: "All seasons"}
									</Select.Trigger>
									<Select.Content>
										<Select.Item value="">All seasons</Select.Item>
										{#each seasons as season (season.id)}
											<Select.Item value={season.id}>{season.name}</Select.Item>
										{/each}
									</Select.Content>
								</Select.Root>
							{/if}
						</div>
						<Button size="sm" disabled={disabled} onclick={openAddHistory}>
							<PlusIcon class="mr-2 size-4" />
							Add History
						</Button>
					</div>

					{#if sortedHistory.length === 0}
						<div class="text-muted-foreground py-8 text-center text-sm">
							No session history yet.
						</div>
					{:else}
						<Table.Table>
							<Table.TableHeader>
								<Table.TableRow>
									<SortableTableHead label="Session" active={historySortKey === "session_name"} direction={historySortDir} onclick={() => toggleHistorySort("session_name")} />
									<SortableTableHead label="Season" active={historySortKey === "season_name"} direction={historySortDir} onclick={() => toggleHistorySort("season_name")} />
									<SortableTableHead label="Age Group" active={historySortKey === "age_group_name"} direction={historySortDir} onclick={() => toggleHistorySort("age_group_name")} />
									<SortableTableHead label="Cabin" active={historySortKey === "cabin_name"} direction={historySortDir} onclick={() => toggleHistorySort("cabin_name")} />
									<Table.TableHead class="w-24">
										<span class="sr-only">Actions</span>
									</Table.TableHead>
								</Table.TableRow>
							</Table.TableHeader>
							<Table.TableBody>
								{#each sortedHistory as entry (entry.id)}
									<Table.TableRow>
										<Table.TableCell>{entry.session_name}</Table.TableCell>
										<Table.TableCell>{entry.season_name}</Table.TableCell>
										<Table.TableCell>{entry.age_group_name}</Table.TableCell>
										<Table.TableCell>{entry.cabin_name ?? "—"}</Table.TableCell>
										<Table.TableCell>
											<div class="flex justify-end gap-1">
												<Button
													variant="ghost"
													size="icon-sm"
													disabled={disabled}
													title="Edit"
													onclick={() => openEditHistory(entry)}
												>
													<PencilIcon class="size-4" />
													<span class="sr-only">Edit</span>
												</Button>
												<Button
													variant="ghost"
													size="icon-sm"
													disabled={disabled}
													title="Delete"
													onclick={() => confirmDeleteHistory(entry)}
												>
													<TrashIcon class="size-4" />
													<span class="sr-only">Delete</span>
												</Button>
											</div>
										</Table.TableCell>
									</Table.TableRow>
								{/each}
							</Table.TableBody>
						</Table.Table>
					{/if}
				</div>
			</Tabs.Content>

			<!-- Preferences Tab -->
			<Tabs.Content value="preferences">
				<div class="grid gap-6 pt-4">
					<!-- Session selector -->
					<div class="flex items-center gap-4">
						<p class="text-muted-foreground text-sm">Manage ranked preferences for a session.</p>
						{#if sessions.length > 0}
							<Select.Root
								type="single"
								value={prefSessionId}
								onValueChange={handlePrefSessionChange}
								disabled={prefSaving}
							>
								<Select.Trigger class="w-64">
									{prefSessionId
										? getSessionLabel(sessions.find((s) => s.id === prefSessionId))
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
							Select a session to manage preferences.
						</div>
					{:else if prefLoading}
						<div class="text-muted-foreground py-8 text-center text-sm">Loading preferences...</div>
					{:else}
						<!-- Age Group Preferences -->
						<div class="grid gap-3">
							<h3 class="text-sm font-medium">Age Group Preferences</h3>
							{#if ageGroupPrefs.length === 0}
								<div class="text-muted-foreground py-4 text-center text-sm">No age group preferences.</div>
							{:else}
								<Table.Table>
									<Table.TableHeader>
										<Table.TableRow>
											<Table.TableHead class="w-12">#</Table.TableHead>
											<Table.TableHead>Age Group</Table.TableHead>
											<Table.TableHead class="w-28">
												<span class="sr-only">Actions</span>
											</Table.TableHead>
										</Table.TableRow>
									</Table.TableHeader>
									<Table.TableBody>
										{#each ageGroupPrefs as pref, i (pref.id)}
											<Table.TableRow>
												<Table.TableCell class="text-muted-foreground">{pref.rank}</Table.TableCell>
												<Table.TableCell>
													<div class="flex items-center gap-2">
														<span>{getAgeGroupName(pref.age_group_id)}</span>
														{#if !sessionAgeGroupIds.has(pref.age_group_id)}
															<Badge variant="outline" class="text-muted-foreground" title="This age group is no longer configured for the selected session and will be ignored by the solver.">no longer in session</Badge>
														{/if}
													</div>
												</Table.TableCell>
												<Table.TableCell>
													<div class="flex justify-end gap-1">
														<Button variant="ghost" size="icon-sm" title="Move up" disabled={disabled || prefSaving || i === 0} onclick={() => moveAgeGroupPref(i, -1)}>
															<ArrowUpIcon class="size-4" />
															<span class="sr-only">Move up</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Move down" disabled={disabled || prefSaving || i === ageGroupPrefs.length - 1} onclick={() => moveAgeGroupPref(i, 1)}>
															<ArrowDownIcon class="size-4" />
															<span class="sr-only">Move down</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Remove" disabled={disabled || prefSaving} onclick={() => removeAgeGroupPref(i)}>
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
							{#if availableAgeGroupsForPref.length > 0}
								<div class="flex items-center gap-2">
									<Select.Root
										type="single"
										value={addAgeGroupId}
										onValueChange={(v) => (addAgeGroupId = v ?? "")}
									>
										<Select.Trigger class="w-48">
											{addAgeGroupId
												? availableAgeGroupsForPref.find((ag) => ag.id === addAgeGroupId)?.name ?? "Select age group"
												: "Select age group"}
										</Select.Trigger>
										<Select.Content>
											{#each availableAgeGroupsForPref as ag (ag.id)}
												<Select.Item value={ag.id}>{ag.name}</Select.Item>
											{/each}
										</Select.Content>
									</Select.Root>
									<Button size="sm" disabled={disabled || prefSaving || !addAgeGroupId} onclick={addAgeGroupPref}>
										<PlusIcon class="mr-1 size-4" />
										Add
									</Button>
								</div>
							{:else if ageGroups.length === 0}
								<p class="text-muted-foreground text-sm">
									No age groups found. <a href="/app/age-groups" class="text-foreground underline">Create age groups</a> to add preferences.
								</p>
							{/if}
						</div>

						<Separator />

						<!-- Co-Counselor Preferences -->
						<div class="grid gap-3">
							<h3 class="text-sm font-medium">Co-Counselor Preferences</h3>
							{#if cocounselorPrefs.length === 0}
								<div class="text-muted-foreground py-4 text-center text-sm">No co-counselor preferences.</div>
							{:else}
								<Table.Table>
									<Table.TableHeader>
										<Table.TableRow>
											<Table.TableHead class="w-12">#</Table.TableHead>
											<Table.TableHead>Counselor</Table.TableHead>
											<Table.TableHead class="w-28">
												<span class="sr-only">Actions</span>
											</Table.TableHead>
										</Table.TableRow>
									</Table.TableHeader>
									<Table.TableBody>
										{#each cocounselorPrefs as pref, i (pref.id)}
											<Table.TableRow>
												<Table.TableCell class="text-muted-foreground">{pref.rank}</Table.TableCell>
												<Table.TableCell>
													<div class="flex items-center gap-2">
														<span>{getCounselorName(pref.preferred_counselor_id)}</span>
														{#if !sessionCounselorIds.has(pref.preferred_counselor_id)}
															<Badge variant="outline" class="text-muted-foreground" title="This counselor is no longer on the selected session's roster and will be ignored by the solver.">no longer in session</Badge>
														{/if}
													</div>
												</Table.TableCell>
												<Table.TableCell>
													<div class="flex justify-end gap-1">
														<Button variant="ghost" size="icon-sm" title="Move up" disabled={disabled || prefSaving || i === 0} onclick={() => moveCocounselorPref(i, -1)}>
															<ArrowUpIcon class="size-4" />
															<span class="sr-only">Move up</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Move down" disabled={disabled || prefSaving || i === cocounselorPrefs.length - 1} onclick={() => moveCocounselorPref(i, 1)}>
															<ArrowDownIcon class="size-4" />
															<span class="sr-only">Move down</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Remove" disabled={disabled || prefSaving} onclick={() => removeCocounselorPref(i)}>
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
							{#if availableCounselorsForPref.length > 0}
								<div class="flex items-center gap-2">
									<Select.Root
										type="single"
										value={addCocounselorId}
										onValueChange={(v) => (addCocounselorId = v ?? "")}
									>
										<Select.Trigger class="w-48">
											{addCocounselorId
												? availableCounselorsForPref.find((c) => c.id === addCocounselorId)?.name ?? "Select counselor"
												: "Select counselor"}
										</Select.Trigger>
										<Select.Content>
											{#each availableCounselorsForPref as c (c.id)}
												<Select.Item value={c.id}>{c.name}</Select.Item>
											{/each}
										</Select.Content>
									</Select.Root>
									<Button size="sm" disabled={disabled || prefSaving || !addCocounselorId} onclick={addCocounselorPref}>
										<PlusIcon class="mr-1 size-4" />
										Add
									</Button>
								</div>
							{:else if allCounselors.length <= 1}
								<p class="text-muted-foreground text-sm">
									No other counselors found. <a href="/app/counselors" class="text-foreground underline">Create counselors</a> to add co-counselor preferences.
								</p>
							{/if}
						</div>

						<Separator />

						<!-- Activity Preferences -->
						<div class="grid gap-3">
							<h3 class="text-sm font-medium">Activity Preferences</h3>
							{#if activityPrefs.length === 0}
								<div class="text-muted-foreground py-4 text-center text-sm">No activity preferences.</div>
							{:else}
								<Table.Table>
									<Table.TableHeader>
										<Table.TableRow>
											<Table.TableHead class="w-12">#</Table.TableHead>
											<Table.TableHead>Activity</Table.TableHead>
											<Table.TableHead class="w-28">
												<span class="sr-only">Actions</span>
											</Table.TableHead>
										</Table.TableRow>
									</Table.TableHeader>
									<Table.TableBody>
										{#each activityPrefs as pref, i (pref.id)}
											<Table.TableRow>
												<Table.TableCell class="text-muted-foreground">{pref.rank}</Table.TableCell>
												<Table.TableCell>
													<div class="flex items-center gap-2">
														<span>{getActivityName(pref.activity_id)}</span>
														{#if !sessionActivityIds.has(pref.activity_id)}
															<Badge variant="outline" class="text-muted-foreground" title="This activity is no longer configured for the selected session and will be ignored by the solver.">no longer in session</Badge>
														{/if}
													</div>
												</Table.TableCell>
												<Table.TableCell>
													<div class="flex justify-end gap-1">
														<Button variant="ghost" size="icon-sm" title="Move up" disabled={disabled || prefSaving || i === 0} onclick={() => moveActivityPref(i, -1)}>
															<ArrowUpIcon class="size-4" />
															<span class="sr-only">Move up</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Move down" disabled={disabled || prefSaving || i === activityPrefs.length - 1} onclick={() => moveActivityPref(i, 1)}>
															<ArrowDownIcon class="size-4" />
															<span class="sr-only">Move down</span>
														</Button>
														<Button variant="ghost" size="icon-sm" title="Remove" disabled={disabled || prefSaving} onclick={() => removeActivityPref(i)}>
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
							{#if availableActivitiesForPref.length > 0}
								<div class="flex items-center gap-2">
									<Select.Root
										type="single"
										value={addActivityId}
										onValueChange={(v) => (addActivityId = v ?? "")}
									>
										<Select.Trigger class="w-48">
											{addActivityId
												? availableActivitiesForPref.find((a) => a.id === addActivityId)?.name ?? "Select activity"
												: "Select activity"}
										</Select.Trigger>
										<Select.Content>
											{#each availableActivitiesForPref as a (a.id)}
												<Select.Item value={a.id}>{a.name}</Select.Item>
											{/each}
										</Select.Content>
									</Select.Root>
									<Button size="sm" disabled={disabled || prefSaving || !addActivityId} onclick={addActivityPref}>
										<PlusIcon class="mr-1 size-4" />
										Add
									</Button>
								</div>
							{:else if allActivities.length === 0}
								<p class="text-muted-foreground text-sm">
									No activities found. <a href="/app/activities" class="text-foreground underline">Create activities</a> to add preferences.
								</p>
							{/if}
						</div>
					{/if}
				</div>
			</Tabs.Content>
		</Tabs.Root>
	{/if}
</div>

<!-- Add Certification Dialog -->
<Dialog.Dialog bind:open={addCertOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Add Certification</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Select a certification to add for this counselor.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<div class="grid gap-4">
			<div class="grid gap-2">
				<Label for="cert-select">Certification</Label>
				<Select.Root
					type="single"
					value={addCertId}
					onValueChange={(v) => (addCertId = v ?? "")}
				>
					<Select.Trigger id="cert-select">
						{availableCerts.find((c) => c.id === addCertId)?.name ?? "Select certification"}
					</Select.Trigger>
					<Select.Content>
						{#each availableCerts as cert (cert.id)}
							<Select.Item value={cert.id}>{cert.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<Dialog.DialogFooter>
				<Button variant="outline" disabled={addingCert} onclick={() => (addCertOpen = false)}>
					Cancel
				</Button>
				<Button disabled={addingCert || !addCertId} onclick={handleAddCert}>
					{#if addingCert}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Add
				</Button>
			</Dialog.DialogFooter>
		</div>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete Certification Confirmation -->
<AlertDialog.AlertDialog bind:open={deleteCertOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Remove Certification</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to remove this certification? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deletingCert}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction disabled={deletingCert} onclick={handleDeleteCert}>
				{#if deletingCert}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Remove
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>

<!-- Session History Create/Edit Dialog -->
<Dialog.Dialog bind:open={historyDialogOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>{historyDialogTitle}</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Record a past session assignment for this counselor.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleHistorySubmit} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="history-session">Session</Label>
				<Select.Root
					type="single"
					value={formSessionId}
					onValueChange={(v) => (formSessionId = v ?? "")}
				>
					<Select.Trigger id="history-session">
						{sessions.find((s) => s.id === formSessionId)?.name ?? "Select session"}
					</Select.Trigger>
					<Select.Content>
						{#each sortedSessions as session (session.id)}
							<Select.Item value={session.id}>{session.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<div class="grid gap-2">
				<Label for="history-age-group">Age Group</Label>
				<Select.Root
					type="single"
					value={formAgeGroupId}
					onValueChange={(v) => (formAgeGroupId = v ?? "")}
				>
					<Select.Trigger id="history-age-group">
						{ageGroups.find((a) => a.id === formAgeGroupId)?.name ?? "Select age group"}
					</Select.Trigger>
					<Select.Content>
						{#each ageGroups as ag (ag.id)}
							<Select.Item value={ag.id}>{ag.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<div class="grid gap-2">
				<Label for="history-cabin">Cabin (optional)</Label>
				<Select.Root
					type="single"
					value={formCabinId}
					onValueChange={(v) => (formCabinId = v ?? "")}
				>
					<Select.Trigger id="history-cabin">
						{formCabinId
							? cabins.find((c) => c.id === formCabinId)?.name ?? "Select cabin"
							: "None"}
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="">None</Select.Item>
						{#each cabins as cabin (cabin.id)}
							<Select.Item value={cabin.id}>{cabin.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			{#if historyFormError}
				<p class="text-destructive text-sm">{historyFormError}</p>
			{/if}
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" disabled={submittingHistory} onclick={() => (historyDialogOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={submittingHistory}>
					{#if submittingHistory}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					{editingHistory ? "Save" : "Add"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete History Confirmation -->
<AlertDialog.AlertDialog bind:open={deleteHistoryOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Session History</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to delete this session history entry? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deletingHistory}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction disabled={deletingHistory} onclick={handleDeleteHistory}>
				{#if deletingHistory}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Delete
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
