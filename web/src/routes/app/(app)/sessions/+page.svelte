<script lang="ts">
	import { goto } from "$app/navigation";
	import { getContext, onMount } from "svelte";
	import { ApiClientError } from "$lib/api/client";
	import { sessionApi, seasonApi } from "$lib/api";

	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import type { Session, Season, Camp } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import CopyIcon from "@lucide/svelte/icons/copy";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import ArchiveIcon from "@lucide/svelte/icons/archive";
	import ArchiveRestoreIcon from "@lucide/svelte/icons/archive-restore";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
	import ArchivedSection from "$lib/components/archived-section.svelte";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let sessions = $state<Session[]>([]);
	let seasons = $state<Season[]>([]);
	let loading = $state(true);
	let loadError = $state(false);
	let noSeasons = $derived(!loading && seasons.length === 0);

	// Create/edit dialog
	let dialogOpen = $state(false);
	let editingSession = $state<Session | null>(null);
	let formName = $state("");
	let formSeasonId = $state("");
	let formPreviousSessionId = $state<string | null>(null);
	let submitting = $state(false);
	let nameError = $state("");
	let seasonError = $state("");

	let dialogTitle = $derived(editingSession ? "Edit Session" : "Add Session");
	let dialogDescription = $derived(
		editingSession
			? "Update the session details."
			: "Enter a name and select a season for the new session."
	);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<Session | null>(null);
	let deleting = $state(false);

	// Copy dialog
	let copyOpen = $state(false);
	let copySource = $state<Session | null>(null);
	let copyName = $state("");
	let copySeasonId = $state("");
	let copySetPrevious = $state(true);
	let copying = $state(false);
	let copyNameError = $state("");
	let copySeasonError = $state("");
	let copyPreviousAllowed = $derived(
		copySource !== null && copySeasonId === copySource.season_id
	);

	// Lookup helpers
	let seasonMap = $derived(new Map(seasons.map((s) => [s.id, s.name])));
	let archivedSeasons = $state<Season[]>([]);
	let archivedSeasonsLoaded = $state(false);
	let archivedSeasonMap = $derived(new Map(archivedSeasons.map((s) => [s.id, s.name])));
	let sessionMap = $derived(new Map(sessions.map((s) => [s.id, s.name])));

	// Archived sessions are loaded lazily so previous_session selectors can
	// preserve and label references to sessions that have since been archived.
	let archivedSessions = $state<Session[]>([]);
	let archivedSessionsLoaded = $state(false);
	let archivedSessionMap = $derived(new Map(archivedSessions.map((s) => [s.id, s.name])));

	// Group sessions by season; newest seasons first; sessions sorted by name asc.
	let sortedSeasons = $derived(
		[...seasons].sort((a, b) => b.start_date.localeCompare(a.start_date))
	);
	let sessionsBySeason = $derived.by(() => {
		const map = new Map<string, Session[]>();
		for (const s of sessions) {
			const list = map.get(s.season_id) ?? [];
			list.push(s);
			map.set(s.season_id, list);
		}
		for (const list of map.values()) {
			list.sort((a, b) => a.name.localeCompare(b.name));
		}
		return map;
	});

	let expandedSeasons = $state<Set<string>>(new Set());

	let nonEmptySeasons = $derived(
		sortedSeasons.filter((s) => (sessionsBySeason.get(s.id) ?? []).length > 0)
	);
	let allSeasonsExpanded = $derived(
		nonEmptySeasons.length > 0 &&
			nonEmptySeasons.every((s) => expandedSeasons.has(s.id))
	);

	function toggleExpandAllSeasons() {
		if (allSeasonsExpanded) {
			expandedSeasons = new Set();
		} else {
			expandedSeasons = new Set(nonEmptySeasons.map((s) => s.id));
		}
	}

	// Sessions available as "previous" (exclude the one being edited)
	let previousSessionOptions = $derived(
		sessions.filter((s) => s.id !== editingSession?.id && s.season_id === formSeasonId)
	);

	onMount(async () => {
		try {
			const [sessionList, seasonList] = await Promise.all([
				sessionApi.list(),
				seasonApi.list(),
			]);
			sessions = sessionList;
			seasons = seasonList;
			expandedSeasons = new Set(seasonList.map((s) => s.id));

			// If any session references a previous session that is not in the
			// active list, hydrate archived sessions so labels resolve.
			const activeIds = new Set(sessionList.map((s) => s.id));
			const referencesArchived = sessionList.some(
				(s) => s.previous_session_id && !activeIds.has(s.previous_session_id),
			);
			if (referencesArchived) {
				try {
					archivedSessions = await sessionApi.listArchived();
					archivedSessionsLoaded = true;
				} catch {
					// non-fatal; previous-session column will read "Unknown"
				}
			}
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load sessions";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	function clearErrors() {
		nameError = "";
		seasonError = "";
	}

	function openCreate() {
		editingSession = null;
		formName = "";
		formSeasonId = "";
		formPreviousSessionId = null;
		clearErrors();
		dialogOpen = true;
	}

	function openEdit(session: Session) {
		editingSession = session;
		formName = session.name;
		formSeasonId = session.season_id;
		formPreviousSessionId = session.previous_session_id;
		clearErrors();
		dialogOpen = true;
		ensureArchivedSeasonLoaded(session.season_id);
		if (session.previous_session_id) {
			void ensureArchivedPreviousLoaded(session.previous_session_id);
		}
	}

	async function ensureArchivedPreviousLoaded(id: string) {
		if (sessions.some((s) => s.id === id)) return;
		if (archivedSessions.some((s) => s.id === id)) return;
		if (archivedSessionsLoaded) return;
		try {
			archivedSessions = await sessionApi.listArchived();
			archivedSessionsLoaded = true;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load archived sessions";
			toast.error(message);
		}
	}

	async function ensureArchivedSeasonLoaded(seasonId: string) {
		if (!seasonId) return;
		if (seasons.some((s) => s.id === seasonId)) return;
		if (archivedSeasons.some((s) => s.id === seasonId)) return;
		if (archivedSeasonsLoaded) return;
		try {
			archivedSeasons = await seasonApi.listArchived();
			archivedSeasonsLoaded = true;
		} catch {
			// ignore
		}
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		clearErrors();

		const name = formName.trim();
		let valid = true;
		if (!name) {
			nameError = "Name is required.";
			valid = false;
		}
		if (!formSeasonId) {
			seasonError = "Season is required.";
			valid = false;
		}
		if (!valid) return;

		submitting = true;

		try {
			const payload = {
				name,
				season_id: formSeasonId,
				previous_session_id: formPreviousSessionId,
			};
			if (editingSession) {
				const updated = await sessionApi.update(editingSession.id, payload);
				sessions = sessions.map((s) => (s.id === updated.id ? updated : s));
				toast.success("Session updated");
			} else {
				const created = await sessionApi.create(payload);
				sessions = [...sessions, created];
				expandedSeasons = new Set([...expandedSeasons, created.season_id]);
				toast.success("Session created");
			}
			dialogOpen = false;
		} catch (err) {
			const action = editingSession ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} session`;
			toast.error(message);
		} finally {
			submitting = false;
		}
	}

	function confirmDelete(session: Session) {
		deleteTarget = session;
		deleteOpen = true;
	}

	let archivingId = $state<string | null>(null);
	let archivedSection = $state<ArchivedSection<Session> | null>(null);

	async function handleArchive(session: Session) {
		archivingId = session.id;
		try {
			await sessionApi.archive(session.id);
			sessions = sessions.filter((s) => s.id !== session.id);
			archivedSection?.addArchived({ ...session, archived: true });
			// Keep `archivedSessions` (and the derived `archivedSessionMap`) in
			// sync so any remaining session whose `previous_session_id` points
			// at the just-archived session still renders its label correctly.
			if (!archivedSessions.some((s) => s.id === session.id)) {
				archivedSessions = [...archivedSessions, { ...session, archived: true }];
			}
			toast.success("Session archived");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to archive session";
			toast.error(message);
		} finally {
			archivingId = null;
		}
	}

	async function handleUnarchive(id: string) {
		await sessionApi.unarchive(id);
		try {
			sessions = await sessionApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to refresh session list";
			toast.error(message);
		}
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await sessionApi.delete(deleteTarget.id);
			sessions = sessions.filter((s) => s.id !== deleteTarget!.id);
			toast.success("Session deleted");
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			if (err instanceof ApiClientError && err.status === 409) {
				const target = deleteTarget;
				deleteOpen = false;
				deleteTarget = null;
				toast.message("Session has dependent records and cannot be deleted.", {
					description: "Archive it instead?",
					action: {
						label: "Archive",
						onClick: () => {
							if (target) handleArchive(target);
						},
					},
				});
			} else {
				const message = err instanceof ApiClientError ? err.message : "Failed to delete session";
				toast.error(message);
			}
		} finally {
			deleting = false;
		}
	}

	function openCopy(session: Session) {
		copySource = session;
		copyName = `${session.name} (Copy)`;
		copySeasonId = session.season_id;
		copySetPrevious = true;
		copyNameError = "";
		copySeasonError = "";
		copyOpen = true;
	}

	async function handleCopy(e: SubmitEvent) {
		e.preventDefault();
		if (!copySource) return;
		copyNameError = "";
		copySeasonError = "";

		const name = copyName.trim();
		let valid = true;
		if (!name) {
			copyNameError = "Name is required.";
			valid = false;
		}
		if (!copySeasonId) {
			copySeasonError = "Season is required.";
			valid = false;
		}
		if (!valid) return;

		copying = true;
		try {
			const previousId = copySetPrevious && copyPreviousAllowed ? copySource.id : null;
			const created = await sessionApi.copy(copySource.id, {
				name,
				season_id: copySeasonId,
				previous_session_id: previousId,
			});
			sessions = [...sessions, created];
			expandedSeasons = new Set([...expandedSeasons, created.season_id]);
			toast.success("Session copied");
			copyOpen = false;
			copySource = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to copy session";
			toast.error(message);
		} finally {
			copying = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Sessions</h1>
			<p class="text-muted-foreground text-sm">Manage sessions within your camp seasons.</p>
		</div>
		{#if camp}
			<div class="flex items-center gap-2">
				{#if nonEmptySeasons.length > 0}
					<Button
						variant="ghost"
						size="sm"
						onclick={toggleExpandAllSeasons}
						title={allSeasonsExpanded ? "Collapse All" : "Expand All"}
					>
						<ChevronsUpDownIcon class="mr-1 size-4" />
						{allSeasonsExpanded ? "Collapse All" : "Expand All"}
					</Button>
				{/if}
				<Button size="sm" disabled={loading || disabled || noSeasons} onclick={openCreate}>
					<PlusIcon class="mr-2 size-4" />
					Add Session
				</Button>
			</div>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load sessions. Try refreshing the page.
		</div>
	{:else if noSeasons}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No seasons found. <a href="/app/seasons" class="text-foreground underline">Create a season</a> before adding sessions.
		</div>
	{:else if sessions.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No sessions yet. Click "Add Session" to create one.
		</div>
	{:else}
		<div class="grid gap-4">
			{#each sortedSeasons as season (season.id)}
				{@const seasonSessions = sessionsBySeason.get(season.id) ?? []}
				{#if seasonSessions.length > 0}
					{@const isExpanded = expandedSeasons.has(season.id)}
					<div class="border-border overflow-hidden rounded-lg border">
						<button
							type="button"
							class="bg-muted flex w-full items-center gap-2 px-4 py-3 text-left"
							onclick={() => {
								const next = new Set(expandedSeasons);
								if (next.has(season.id)) next.delete(season.id);
								else next.add(season.id);
								expandedSeasons = next;
							}}
						>
							<ChevronDownIcon class="size-4 transition-transform {isExpanded ? '' : '-rotate-90'}" />
							<h3 class="font-medium">{season.name}</h3>
							<span class="text-muted-foreground text-sm">
								— {seasonSessions.length} {seasonSessions.length === 1 ? "session" : "sessions"}
							</span>
						</button>

						{#if isExpanded}
							<Table.Table>
								<Table.TableHeader>
									<Table.TableRow>
										<Table.TableHead>Name</Table.TableHead>
										<Table.TableHead>Previous Session</Table.TableHead>
										<Table.TableHead class="w-32">
											<span class="sr-only">Actions</span>
										</Table.TableHead>
									</Table.TableRow>
								</Table.TableHeader>
								<Table.TableBody>
									{#each seasonSessions as session (session.id)}
										<Table.TableRow class="cursor-pointer hover:bg-muted/50" onclick={() => goto(`/app/sessions/${session.id}`)}>
											<Table.TableCell>{session.name}</Table.TableCell>
											<Table.TableCell>
												{#if session.previous_session_id}
													{@const prevName = sessionMap.get(session.previous_session_id) ?? archivedSessionMap.get(session.previous_session_id)}
													{#if prevName}
														{prevName}{#if archivedSessionMap.has(session.previous_session_id)}<span class="text-muted-foreground"> (archived)</span>{/if}
													{:else}
														Unknown
													{/if}
												{:else}
													<span class="text-muted-foreground">—</span>
												{/if}
											</Table.TableCell>
											<Table.TableCell>
												<div class="flex justify-end gap-1">
													<Button
														variant="ghost"
														size="icon-sm"
														disabled={disabled}
														title="Edit"
														onclick={(e: MouseEvent) => { e.stopPropagation(); openEdit(session); }}
													>
														<PencilIcon class="size-4" />
														<span class="sr-only">Edit</span>
													</Button>
													<Button
														variant="ghost"
														size="icon-sm"
														disabled={disabled}
														title="Copy"
														onclick={(e: MouseEvent) => { e.stopPropagation(); openCopy(session); }}
													>
														<CopyIcon class="size-4" />
														<span class="sr-only">Copy</span>
													</Button>
													<Button
														variant="ghost"
														size="icon-sm"
														title="Archive"
														disabled={disabled || archivingId === session.id}
														onclick={(e: MouseEvent) => { e.stopPropagation(); handleArchive(session); }}
													>
														{#if archivingId === session.id}
															<LoaderCircleIcon class="size-4 animate-spin" />
														{:else}
															<ArchiveIcon class="size-4" />
														{/if}
														<span class="sr-only">Archive</span>
													</Button>
													<Button
														variant="ghost"
														size="icon-sm"
														disabled={disabled}
														title="Delete"
														onclick={(e: MouseEvent) => { e.stopPropagation(); confirmDelete(session); }}
													>
														<TrashIcon class="size-4" />
														<span class="sr-only">Delete</span>
													</Button>
													<a
														href={`/app/sessions/${session.id}`}
														class="text-muted-foreground hover:text-foreground ml-1 inline-flex items-center"
														onclick={(e: MouseEvent) => e.stopPropagation()}
													>
														<ChevronRightIcon class="size-4" />
														<span class="sr-only">View details</span>
													</a>
												</div>
											</Table.TableCell>
										</Table.TableRow>
									{/each}
								</Table.TableBody>
							</Table.Table>
						{/if}
					</div>
				{/if}
			{/each}
		</div>
	{/if}

	{#if camp}
		<ArchivedSection
			bind:this={archivedSection}
			resourceName="Session"
			listArchivedFn={sessionApi.listArchived}
			unarchiveFn={handleUnarchive}
		>
			{#snippet row({ item, unarchive, busy })}
				<div class="flex items-center justify-between border-b py-2 last:border-b-0">
					<span class="text-sm">{item.name}</span>
					<Button
						variant="ghost"
						size="icon-sm"
						title="Restore"
						disabled={disabled || busy}
						onclick={unarchive}
					>
						{#if busy}
							<LoaderCircleIcon class="size-4 animate-spin" />
						{:else}
							<ArchiveRestoreIcon class="size-4" />
						{/if}
						<span class="sr-only">Restore</span>
					</Button>
				</div>
			{/snippet}
		</ArchivedSection>
	{/if}
</div>

<!-- Create/Edit dialog -->
<Dialog.Dialog bind:open={dialogOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>{dialogTitle}</Dialog.DialogTitle>
			<Dialog.DialogDescription>{dialogDescription}</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleSubmit} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="session-name">Name</Label>
				<Input
					id="session-name"
					type="text"
					placeholder="Session name"
					bind:value={formName}
					disabled={submitting}
					oninput={() => (nameError = "")}
				/>
				{#if nameError}
					<p class="text-destructive text-sm">{nameError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="session-season">Season</Label>
				<Select.Select
					type="single"
					value={formSeasonId}
					disabled={submitting}
					onValueChange={(v) => {
						if (v !== formSeasonId) {
							formSeasonId = v;
							formPreviousSessionId = null;
						}
						seasonError = "";
					}}
				>
					<Select.SelectTrigger id="session-season" class="w-full">
						{#if formSeasonId}
							{@const name = seasonMap.get(formSeasonId) ?? archivedSeasonMap.get(formSeasonId)}
							{#if name}
								{name}{#if archivedSeasonMap.has(formSeasonId)}<span class="text-muted-foreground"> (archived)</span>{/if}
							{:else}
								Select season
							{/if}
						{:else}
							<span class="text-muted-foreground">Select season</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each seasons as season (season.id)}
							<Select.SelectItem value={season.id}>{season.name}</Select.SelectItem>
						{/each}
						{#each archivedSeasons.filter((s) => s.id === formSeasonId) as season (season.id)}
							<Select.SelectItem value={season.id} disabled>{season.name} (archived)</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if seasonError}
					<p class="text-destructive text-sm">{seasonError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="session-previous">Previous Session</Label>
			<Select.Select
				type="single"
				value={formPreviousSessionId ?? ""}
				disabled={submitting || !formSeasonId}
				onValueChange={(v) => (formPreviousSessionId = v || null)}
				>
					<Select.SelectTrigger id="session-previous" class="w-full">
						{#if formPreviousSessionId}
							{@const name = sessionMap.get(formPreviousSessionId!) ?? archivedSessionMap.get(formPreviousSessionId!)}
							{#if name}
								{name}{#if archivedSessionMap.has(formPreviousSessionId!)}<span class="text-muted-foreground"> (archived)</span>{/if}
							{:else}
								Select session
							{/if}
						{:else}
							<span class="text-muted-foreground">None</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						<Select.SelectItem value="">None</Select.SelectItem>
						{#each previousSessionOptions as s (s.id)}
							<Select.SelectItem value={s.id}>{s.name}</Select.SelectItem>
						{/each}
						{#each archivedSessions.filter((s) => s.id === formPreviousSessionId) as s (s.id)}
							<Select.SelectItem value={s.id} disabled>{s.name} (archived)</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
			</div>
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" disabled={submitting} onclick={() => (dialogOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={submitting}>
					{#if submitting}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					{editingSession ? "Save" : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Copy dialog -->
<Dialog.Dialog bind:open={copyOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Copy Session</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Create a new session by copying the structural configuration (age groups, cabins, time slots, and activities) from "{copySource?.name}".
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleCopy} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="copy-session-name">Name</Label>
				<Input
					id="copy-session-name"
					type="text"
					placeholder="Session name"
					bind:value={copyName}
					disabled={copying}
					oninput={() => (copyNameError = "")}
				/>
				{#if copyNameError}
					<p class="text-destructive text-sm">{copyNameError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="copy-session-season">Season</Label>
				<Select.Select
					type="single"
					value={copySeasonId}
					disabled={copying}
					onValueChange={(v) => {
						copySeasonId = v;
						copySeasonError = "";
					}}
				>
					<Select.SelectTrigger id="copy-session-season" class="w-full">
						{#if copySeasonId}
							{seasonMap.get(copySeasonId) ?? "Select season"}
						{:else}
							<span class="text-muted-foreground">Select season</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each seasons as season (season.id)}
							<Select.SelectItem value={season.id}>{season.name}</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if copySeasonError}
					<p class="text-destructive text-sm">{copySeasonError}</p>
				{/if}
			</div>
			<div class="grid gap-1">
				<div class="flex items-center gap-2">
					<input
						id="copy-session-set-previous"
						type="checkbox"
						class="border-input size-4 rounded"
						checked={copySetPrevious && copyPreviousAllowed}
						disabled={copying || !copyPreviousAllowed}
						onchange={(e) => (copySetPrevious = (e.target as HTMLInputElement).checked)}
					/>
					<Label for="copy-session-set-previous" class="font-normal">
						Set "{copySource?.name}" as the previous session
					</Label>
				</div>
				{#if !copyPreviousAllowed}
					<p class="text-muted-foreground ml-6 text-xs">
						Only available when copying into the same season.
					</p>
				{/if}
			</div>
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" disabled={copying} onclick={() => (copyOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={copying}>
					{#if copying}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Copy
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete confirmation -->
<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Session</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to delete "{deleteTarget?.name}"? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				disabled={deleting}
				onclick={handleDelete}
			>
				{#if deleting}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Delete
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
