<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import { counselorApi } from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import { Badge } from "$lib/components/ui/badge";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import type { Counselor, Camp } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import { sortItems, type SortDirection } from "$lib/utils";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let counselors = $state<Counselor[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	let sortKey = $state<"name" | "junior_counselor" | "enabled">("name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedCounselors = $derived(
		sortItems(
			counselors,
			sortKey === "junior_counselor"
				? (c: Counselor) => (c.junior_counselor ? "Junior" : "Senior")
				: sortKey === "enabled"
					? (c: Counselor) => (c.enabled ? "Active" : "Inactive")
					: sortKey,
			sortDirection
		)
	);

	function toggleSort(key: typeof sortKey) {
		if (sortKey === key) {
			sortDirection = sortDirection === "asc" ? "desc" : "asc";
		} else {
			sortKey = key;
			sortDirection = "asc";
		}
	}

	// Create/edit dialog
	let dialogOpen = $state(false);
	let editingCounselor = $state<Counselor | null>(null);
	let formName = $state("");
	let formJunior = $state(false);
	let formEnabled = $state(true);
	let submitting = $state(false);
	let nameError = $state("");

	let dialogTitle = $derived(editingCounselor ? "Edit Counselor" : "Add Counselor");
	let dialogDescription = $derived(
		editingCounselor
			? "Update the counselor's details."
			: "Enter a name for the new counselor."
	);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<Counselor | null>(null);
	let deleting = $state(false);

	onMount(async () => {
		try {
			counselors = await counselorApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load counselors";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	function openCreate() {
		editingCounselor = null;
		formName = "";
		formJunior = false;
		formEnabled = true;
		nameError = "";
		dialogOpen = true;
	}

	function openEdit(counselor: Counselor) {
		editingCounselor = counselor;
		formName = counselor.name;
		formJunior = counselor.junior_counselor;
		formEnabled = counselor.enabled;
		nameError = "";
		dialogOpen = true;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		nameError = "";

		const name = formName.trim();
		if (!name) {
			nameError = "Name is required.";
			return;
		}

		submitting = true;

		try {
			if (editingCounselor) {
				const updated = await counselorApi.update(editingCounselor.id, {
					name,
					junior_counselor: formJunior,
					enabled: formEnabled,
				});
				counselors = counselors.map((c) => (c.id === updated.id ? updated : c));
				toast.success("Counselor updated");
			} else {
				const created = await counselorApi.create({
					name,
					junior_counselor: formJunior,
				});
				counselors = [...counselors, created];
				toast.success("Counselor created");
			}
			dialogOpen = false;
		} catch (err) {
			const action = editingCounselor ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} counselor`;
			toast.error(message);
		} finally {
			submitting = false;
		}
	}

	function confirmDelete(counselor: Counselor) {
		deleteTarget = counselor;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await counselorApi.delete(deleteTarget.id);
			counselors = counselors.filter((c) => c.id !== deleteTarget!.id);
			toast.success("Counselor deleted");
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete counselor";
			toast.error(message);
		} finally {
			deleting = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Counselors</h1>
			<p class="text-muted-foreground text-sm">Manage counselor profiles, preferences, and assignments.</p>
		</div>
		{#if camp}
			<Button size="sm" disabled={disabled || loading || loadError} onclick={openCreate}>
				<PlusIcon class="mr-2 size-4" />
				Add Counselor
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load counselors. Try refreshing the page.
		</div>
	{:else if counselors.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No counselors yet. Click "Add Counselor" to create one.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<SortableTableHead label="Name" active={sortKey === "name"} direction={sortDirection} onclick={() => toggleSort("name")} />
					<SortableTableHead label="Type" active={sortKey === "junior_counselor"} direction={sortDirection} onclick={() => toggleSort("junior_counselor")} />
					<SortableTableHead label="Status" active={sortKey === "enabled"} direction={sortDirection} onclick={() => toggleSort("enabled")} />
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedCounselors as counselor (counselor.id)}
					<Table.TableRow
						class="cursor-pointer hover:bg-muted/50"
						tabindex={0}
						role="button"
						onclick={() => goto(`/app/counselors/${counselor.id}`)}
						onkeydown={(e: KeyboardEvent) => {
							if (e.key === "Enter" || e.key === " ") {
								e.preventDefault();
								goto(`/app/counselors/${counselor.id}`);
							}
						}}
					>
						<Table.TableCell>{counselor.name}</Table.TableCell>
						<Table.TableCell>
							<Badge variant={counselor.junior_counselor ? "secondary" : "default"}>
								{counselor.junior_counselor ? "Junior" : "Senior"}
							</Badge>
						</Table.TableCell>
						<Table.TableCell>
							<Badge variant={counselor.enabled ? "default" : "outline"}>
								{counselor.enabled ? "Active" : "Inactive"}
							</Badge>
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex items-center justify-end gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Edit"
									onclick={(e: MouseEvent) => { e.stopPropagation(); openEdit(counselor); }}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Delete"
									onclick={(e: MouseEvent) => { e.stopPropagation(); confirmDelete(counselor); }}
								>
									<TrashIcon class="size-4" />
									<span class="sr-only">Delete</span>
								</Button>
								<ChevronRightIcon class="ml-1 size-4 text-muted-foreground" />
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
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>{dialogTitle}</Dialog.DialogTitle>
			<Dialog.DialogDescription>{dialogDescription}</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleSubmit} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="counselor-name">Name</Label>
				<Input
					id="counselor-name"
					type="text"
					placeholder="Counselor name"
					bind:value={formName}
					disabled={submitting}
					oninput={() => (nameError = "")}
				/>
				{#if nameError}
					<p class="text-destructive text-sm">{nameError}</p>
				{/if}
			</div>
			<div class="flex items-center gap-2">
				<input
					id="counselor-junior"
					type="checkbox"
					bind:checked={formJunior}
					disabled={submitting}
					class="size-4 rounded border-gray-300"
				/>
				<Label for="counselor-junior">Junior counselor</Label>
			</div>
			{#if editingCounselor}
				<div class="flex items-center gap-2">
					<input
						id="counselor-enabled"
						type="checkbox"
						bind:checked={formEnabled}
						disabled={submitting}
						class="size-4 rounded border-gray-300"
					/>
					<Label for="counselor-enabled">Enabled</Label>
				</div>
			{/if}
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" disabled={submitting} onclick={() => (dialogOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={submitting}>
					{#if submitting}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					{/if}
					{editingCounselor ? "Save" : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete confirmation -->
<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Counselor</AlertDialog.AlertDialogTitle>
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
