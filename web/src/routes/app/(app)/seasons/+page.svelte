<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { ApiClientError } from "$lib/api/client";
	import { seasonApi } from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import type { Season, Camp } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let seasons = $state<Season[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	// Create/edit dialog
	let dialogOpen = $state(false);
	let editingSeason = $state<Season | null>(null);
	let formName = $state("");
	let formStartDate = $state("");
	let formEndDate = $state("");
	let submitting = $state(false);
	let nameError = $state("");
	let startDateError = $state("");
	let endDateError = $state("");

	let dialogTitle = $derived(editingSeason ? "Edit Season" : "Add Season");
	let dialogDescription = $derived(
		editingSeason
			? "Update the season details."
			: "Enter a name and date range for the new season."
	);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<Season | null>(null);
	let deleting = $state(false);

	onMount(async () => {
		try {
			seasons = await seasonApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load seasons";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	function clearErrors() {
		nameError = "";
		startDateError = "";
		endDateError = "";
	}

	function openCreate() {
		editingSeason = null;
		formName = "";
		formStartDate = "";
		formEndDate = "";
		clearErrors();
		dialogOpen = true;
	}

	function openEdit(season: Season) {
		editingSeason = season;
		formName = season.name;
		formStartDate = season.start_date;
		formEndDate = season.end_date;
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
		if (!formStartDate) {
			startDateError = "Start date is required.";
			valid = false;
		}
		if (!formEndDate) {
			endDateError = "End date is required.";
			valid = false;
		}
		if (formStartDate && formEndDate && formStartDate > formEndDate) {
			endDateError = "End date must be on or after start date.";
			valid = false;
		}
		if (!valid) return;

		submitting = true;

		try {
			const payload = { name, start_date: formStartDate, end_date: formEndDate };
			if (editingSeason) {
				const updated = await seasonApi.update(editingSeason.id, payload);
				seasons = seasons.map((s) => (s.id === updated.id ? updated : s));
				toast.success("Season updated");
			} else {
				const created = await seasonApi.create(payload);
				seasons = [...seasons, created];
				toast.success("Season created");
			}
			dialogOpen = false;
		} catch (err) {
			const action = editingSeason ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} season`;
			toast.error(message);
		} finally {
			submitting = false;
		}
	}

	function confirmDelete(season: Season) {
		deleteTarget = season;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await seasonApi.delete(deleteTarget.id);
			seasons = seasons.filter((s) => s.id !== deleteTarget!.id);
			toast.success("Season deleted");
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete season";
			toast.error(message);
		} finally {
			deleting = false;
		}
	}

	function formatDate(dateStr: string): string {
		const [year, month, day] = dateStr.split("-");
		return `${month}/${day}/${year}`;
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Seasons</h1>
			<p class="text-muted-foreground text-sm">Manage camp seasons and their date ranges.</p>
		</div>
		{#if camp}
			<Button size="sm" disabled={disabled} onclick={openCreate}>
				<PlusIcon class="mr-2 size-4" />
				Add Season
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load seasons. Try refreshing the page.
		</div>
	{:else if seasons.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No seasons yet. Click "Add Season" to create one.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<Table.TableHead>Name</Table.TableHead>
					<Table.TableHead>Start Date</Table.TableHead>
					<Table.TableHead>End Date</Table.TableHead>
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each seasons as season (season.id)}
					<Table.TableRow>
						<Table.TableCell>{season.name}</Table.TableCell>
						<Table.TableCell>{formatDate(season.start_date)}</Table.TableCell>
						<Table.TableCell>{formatDate(season.end_date)}</Table.TableCell>
						<Table.TableCell>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									onclick={() => openEdit(season)}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									onclick={() => confirmDelete(season)}
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
				<Label for="season-name">Name</Label>
				<Input
					id="season-name"
					type="text"
					placeholder="Season name"
					bind:value={formName}
					disabled={submitting}
					oninput={() => (nameError = "")}
				/>
				{#if nameError}
					<p class="text-destructive text-sm">{nameError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="season-start-date">Start Date</Label>
				<Input
					id="season-start-date"
					type="date"
					bind:value={formStartDate}
					disabled={submitting}
					oninput={() => (startDateError = "")}
				/>
				{#if startDateError}
					<p class="text-destructive text-sm">{startDateError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="season-end-date">End Date</Label>
				<Input
					id="season-end-date"
					type="date"
					bind:value={formEndDate}
					disabled={submitting}
					oninput={() => (endDateError = "")}
				/>
				{#if endDateError}
					<p class="text-destructive text-sm">{endDateError}</p>
				{/if}
			</div>
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" disabled={submitting} onclick={() => (dialogOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={submitting}>
					{#if submitting}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					{editingSeason ? "Save" : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete confirmation -->
<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Season</AlertDialog.AlertDialogTitle>
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
