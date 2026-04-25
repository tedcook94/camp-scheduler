<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { ApiClientError } from "$lib/api/client";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import { sortItems, type SortDirection } from "$lib/utils";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import ArchivedSection from "$lib/components/archived-section.svelte";
	import type { Camp } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import ArchiveIcon from "@lucide/svelte/icons/archive";
	import ArchiveRestoreIcon from "@lucide/svelte/icons/archive-restore";
	import CheckIcon from "@lucide/svelte/icons/check";
	import XIcon from "@lucide/svelte/icons/x";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

	interface NameResource {
		id: string;
		name: string;
		archived?: boolean;
	}

	interface Props {
		title: string;
		description: string;
		resourceName: string;
		resourceNamePlural?: string;
		listFn: () => Promise<NameResource[]>;
		createFn: (name: string) => Promise<NameResource>;
		updateFn: (id: string, name: string) => Promise<NameResource>;
		deleteFn: (id: string) => Promise<void>;
		archiveFn?: (id: string) => Promise<void>;
		unarchiveFn?: (id: string) => Promise<void>;
		listArchivedFn?: () => Promise<NameResource[]>;
	}

	let {
		title,
		description,
		resourceName,
		resourceNamePlural,
		listFn,
		createFn,
		updateFn,
		deleteFn,
		archiveFn,
		unarchiveFn,
		listArchivedFn,
	}: Props = $props();

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let items = $state<NameResource[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	let sortKey = $state<keyof NameResource>("name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedItems = $derived(sortItems(items, sortKey, sortDirection));

	function toggleSort(key: keyof NameResource) {
		if (sortKey === key) {
			sortDirection = sortDirection === "asc" ? "desc" : "asc";
		} else {
			sortKey = key;
			sortDirection = "asc";
		}
	}

	// Create dialog
	let createOpen = $state(false);
	let createName = $state("");
	let creating = $state(false);

	// Inline edit
	let editingId = $state<string | null>(null);
	let editingName = $state("");
	let saving = $state(false);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<NameResource | null>(null);
	let deleting = $state(false);

	let archivingId = $state<string | null>(null);
	let archivedSection = $state<ArchivedSection<NameResource> | null>(null);

	async function handleArchive(item: NameResource) {
		if (!archiveFn) return;
		archivingId = item.id;
		try {
			await archiveFn(item.id);
			items = items.filter((i) => i.id !== item.id);
			archivedSection?.addArchived({ ...item, archived: true });
			toast.success(`${resourceName} archived`);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : `Failed to archive ${resourceName}`;
			toast.error(message);
		} finally {
			archivingId = null;
		}
	}

	async function handleUnarchive(id: string) {
		if (!unarchiveFn) return;
		await unarchiveFn(id);
		// Refetch the active list so the restored item reappears.
		try {
			items = await listFn();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : `Failed to refresh ${resourceName.toLowerCase()} list`;
			toast.error(message);
		}
	}

	onMount(async () => {
		try {
			items = await listFn();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : `Failed to load ${resourceName}s`;
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	async function handleCreate(e: SubmitEvent) {
		e.preventDefault();
		const name = createName.trim();
		if (!name) return;
		creating = true;

		try {
			const created = await createFn(name);
			items = [...items, created];
			createName = "";
			createOpen = false;
			toast.success(`${resourceName} created`);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : `Failed to create ${resourceName}`;
			toast.error(message);
		} finally {
			creating = false;
		}
	}

	function startEdit(item: NameResource) {
		editingId = item.id;
		editingName = item.name;
	}

	function cancelEdit() {
		editingId = null;
		editingName = "";
	}

	async function handleSave(id: string) {
		const name = editingName.trim();
		if (!name) return;
		saving = true;

		try {
			const updated = await updateFn(id, name);
			items = items.map((i) => (i.id === id ? updated : i));
			editingId = null;
			editingName = "";
			toast.success(`${resourceName} updated`);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : `Failed to update ${resourceName}`;
			toast.error(message);
		} finally {
			saving = false;
		}
	}

	function confirmDelete(item: NameResource) {
		deleteTarget = item;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await deleteFn(deleteTarget.id);
			items = items.filter((i) => i.id !== deleteTarget!.id);
			toast.success(`${resourceName} deleted`);
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			if (err instanceof ApiClientError && err.status === 409 && archiveFn) {
				const target = deleteTarget;
				deleteOpen = false;
				deleteTarget = null;
				toast.message(`${resourceName} has dependent records and cannot be deleted.`, {
					description: "Archive it instead?",
					action: {
						label: "Archive",
						onClick: () => {
							if (target) handleArchive(target);
						},
					},
				});
			} else {
				const message = err instanceof ApiClientError ? err.message : `Failed to delete ${resourceName}`;
				toast.error(message);
			}
		} finally {
			deleting = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">{title}</h1>
			<p class="text-muted-foreground text-sm">{description}</p>
		</div>
		{#if camp}
			<Button size="sm" disabled={disabled} onclick={() => (createOpen = true)}>
				<PlusIcon class="mr-2 size-4" />
				Add {resourceName}
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load {resourceName.toLowerCase()}s. Try refreshing the page.
		</div>
	{:else if items.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No {resourceName.toLowerCase()}s yet. Click "Add {resourceName}" to create one.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<SortableTableHead label="Name" active={sortKey === "name"} direction={sortDirection} onclick={() => toggleSort("name")} />
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedItems as item (item.id)}
					<Table.TableRow>
						<Table.TableCell>
							{#if editingId === item.id}
								<form
									class="flex items-center gap-2"
									onsubmit={(e) => {
										e.preventDefault();
										handleSave(item.id);
									}}
								>
									<Input
										type="text"
										bind:value={editingName}
										class="h-7 max-w-xs"
										disabled={saving}
										autofocus
									/>
									<Button
										type="submit"
										variant="ghost"
										size="icon-sm"
										title="Save"
										disabled={saving || !editingName.trim()}
									>
										{#if saving}
											<LoaderCircleIcon class="size-4 animate-spin" />
										{:else}
											<CheckIcon class="size-4" />
										{/if}
									</Button>
									<Button
										type="button"
										variant="ghost"
										size="icon-sm"
										title="Cancel"
										disabled={saving}
										onclick={cancelEdit}
									>
										<XIcon class="size-4" />
									</Button>
								</form>
							{:else}
								{item.name}
							{/if}
						</Table.TableCell>
						<Table.TableCell>
							{#if editingId !== item.id}
								<div class="flex justify-end gap-1">
									<Button
										variant="ghost"
										size="icon-sm"
										title="Edit"
										disabled={disabled || editingId !== null}
										onclick={() => startEdit(item)}
									>
										<PencilIcon class="size-4" />
										<span class="sr-only">Edit</span>
									</Button>
									{#if archiveFn}
										<Button
											variant="ghost"
											size="icon-sm"
											title="Archive"
											disabled={disabled || editingId !== null || archivingId === item.id}
											onclick={() => handleArchive(item)}
										>
											{#if archivingId === item.id}
												<LoaderCircleIcon class="size-4 animate-spin" />
											{:else}
												<ArchiveIcon class="size-4" />
											{/if}
											<span class="sr-only">Archive</span>
										</Button>
									{/if}
									<Button
										variant="ghost"
										size="icon-sm"
										title="Delete"
										disabled={disabled || editingId !== null}
										onclick={() => confirmDelete(item)}
									>
										<TrashIcon class="size-4" />
										<span class="sr-only">Delete</span>
									</Button>
								</div>
							{/if}
						</Table.TableCell>
					</Table.TableRow>
				{/each}
			</Table.TableBody>
		</Table.Table>
	{/if}

	{#if camp && listArchivedFn && unarchiveFn}
		<ArchivedSection
			bind:this={archivedSection}
			{resourceName}
			{resourceNamePlural}
			listArchivedFn={listArchivedFn}
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

<!-- Create dialog -->
<Dialog.Dialog bind:open={createOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Add {resourceName}</Dialog.DialogTitle>
			<Dialog.DialogDescription>Enter a name for the new {resourceName.toLowerCase()}.</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleCreate} class="grid gap-4">
			<Input
				type="text"
				placeholder="Name"
				bind:value={createName}
				required
				disabled={creating}
			/>
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" disabled={creating} onclick={() => (createOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={creating || !createName.trim()}>
					{#if creating}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					Create
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete confirmation -->
<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete {resourceName}</AlertDialog.AlertDialogTitle>
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
