<script lang="ts">
	import { getContext, onMount } from "svelte";
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
	} from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Badge } from "$lib/components/ui/badge";
	import { Label } from "$lib/components/ui/label";
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
	} from "$lib/api/types";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
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
	let historyFilterSeasonId = $state("");

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

	onMount(async () => {
		try {
			const [c, certs, allCerts, summary, s, sess, ag, cab] = await Promise.all([
				counselorApi.get(counselorId),
				counselorCertificationApi.list(counselorId),
				certificationApi.list(),
				sessionHistoryApi.summary(counselorId),
				seasonApi.list(),
				sessionApi.list(),
				ageGroupApi.list(),
				cabinApi.list(),
			]);
			counselor = c;
			counselorCerts = certs;
			allCertifications = allCerts;
			historySummary = summary;
			seasons = s;
			sessions = sess;
			ageGroups = ag;
			cabins = cab;
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
		<Tabs.Root value="certifications">
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
									onValueChange={(v) => (historyFilterSeasonId = v ?? "")}
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

			<!-- Preferences Tab (placeholder for next commit) -->
			<Tabs.Content value="preferences">
				<div class="text-muted-foreground py-8 text-center text-sm">
					Preferences management coming soon. Select a session to manage age group, co-counselor, and activity preferences.
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
						{#each sessions as session (session.id)}
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
