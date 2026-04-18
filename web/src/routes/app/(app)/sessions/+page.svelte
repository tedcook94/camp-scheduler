<script lang="ts">
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
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

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

	// Lookup helpers
	let seasonMap = $derived(new Map(seasons.map((s) => [s.id, s.name])));
	let sessionMap = $derived(new Map(sessions.map((s) => [s.id, s.name])));

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
			const message = err instanceof ApiClientError ? err.message : "Failed to delete session";
			toast.error(message);
		} finally {
			deleting = false;
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
			<Button size="sm" disabled={loading || disabled || noSeasons} onclick={openCreate}>
				<PlusIcon class="mr-2 size-4" />
				Add Session
			</Button>
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
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<Table.TableHead>Name</Table.TableHead>
					<Table.TableHead>Season</Table.TableHead>
					<Table.TableHead>Previous Session</Table.TableHead>
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sessions as session (session.id)}
					<Table.TableRow>
						<Table.TableCell>{session.name}</Table.TableCell>
						<Table.TableCell>{seasonMap.get(session.season_id) ?? "Unknown"}</Table.TableCell>
						<Table.TableCell>
							{#if session.previous_session_id}
								{sessionMap.get(session.previous_session_id) ?? "Unknown"}
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
									onclick={() => openEdit(session)}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									onclick={() => confirmDelete(session)}
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

<!-- Create/Edit dialog -->
<Dialog.Dialog bind:open={dialogOpen}>
	<Dialog.DialogContent>
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
							{seasonMap.get(formSeasonId) ?? "Select season"}
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
							{sessionMap.get(formPreviousSessionId!) ?? "Select session"}
						{:else}
							<span class="text-muted-foreground">None</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						<Select.SelectItem value="">None</Select.SelectItem>
						{#each previousSessionOptions as s (s.id)}
							<Select.SelectItem value={s.id}>{s.name}</Select.SelectItem>
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
				class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
			>
				{#if deleting}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Delete
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
