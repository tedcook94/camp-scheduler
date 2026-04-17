<script lang="ts">
	import { onMount } from "svelte";
	import { campApi } from "$lib/api";
	import { ApiClientError } from "$lib/api/client";
	import type { Camp, CreateCampRequest, UpdateCampRequest } from "$lib/api/types";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import { Badge } from "$lib/components/ui/badge";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";

	let camps = $state<Camp[]>([]);
	let loading = $state(true);

	// Dialog state
	let dialogOpen = $state(false);
	let dialogMode = $state<"create" | "edit">("create");
	let editingCamp = $state<Camp | null>(null);
	let saving = $state(false);

	// Form fields
	let formName = $state("");
	let formLocation = $state("");
	let formEnabled = $state(true);
	let formError = $state("");

	// Delete state
	let deleteDialogOpen = $state(false);
	let deletingCamp = $state<Camp | null>(null);
	let deleting = $state(false);

	async function loadCamps() {
		loading = true;
		try {
			camps = await campApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load camps";
			toast.error(message);
		} finally {
			loading = false;
		}
	}

	function openCreate() {
		dialogMode = "create";
		editingCamp = null;
		formName = "";
		formLocation = "";
		formEnabled = true;
		formError = "";
		dialogOpen = true;
	}

	function openEdit(camp: Camp) {
		dialogMode = "edit";
		editingCamp = camp;
		formName = camp.name;
		formLocation = camp.location ?? "";
		formEnabled = camp.enabled;
		formError = "";
		dialogOpen = true;
	}

	function openDelete(camp: Camp) {
		deletingCamp = camp;
		deleteDialogOpen = true;
	}

	async function handleSave(e: SubmitEvent) {
		e.preventDefault();
		formError = "";
		saving = true;

		try {
			const location = formLocation.trim() || null;

			if (dialogMode === "create") {
				const req: CreateCampRequest = { name: formName.trim(), location };
				await campApi.create(req);
				toast.success("Camp created.");
			} else if (editingCamp) {
				const req: UpdateCampRequest = {
					name: formName.trim(),
					location,
					enabled: formEnabled,
				};
				await campApi.updateById(editingCamp.id, req);
				toast.success("Camp updated.");
			}

			dialogOpen = false;
			await loadCamps();
		} catch (err) {
			if (err instanceof ApiClientError) {
				formError = err.message;
			} else {
				formError = "An unexpected error occurred.";
			}
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!deletingCamp) return;
		deleting = true;

		try {
			await campApi.delete(deletingCamp.id);
			toast.success("Camp deleted.");
			deleteDialogOpen = false;
			deletingCamp = null;
			await loadCamps();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete camp";
			toast.error(message);
			deleteDialogOpen = false;
		} finally {
			deleting = false;
		}
	}

	onMount(() => {
		loadCamps();
	});
</script>

<div class="grid gap-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Camps</h1>
			<p class="text-muted-foreground text-sm">Manage summer camps.</p>
		</div>
		<Button onclick={openCreate} size="sm">
			<PlusIcon data-icon="inline-start" class="size-4" />
			Add Camp
		</Button>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading camps...</div>
	{:else if camps.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No camps yet. Create your first camp to get started.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<Table.TableHead>Name</Table.TableHead>
					<Table.TableHead>Location</Table.TableHead>
					<Table.TableHead>Status</Table.TableHead>
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each camps as camp (camp.id)}
					<Table.TableRow>
						<Table.TableCell class="font-medium">{camp.name}</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">
							{camp.location ?? "\u2014"}
						</Table.TableCell>
						<Table.TableCell>
							{#if camp.enabled}
								<Badge variant="default">Enabled</Badge>
							{:else}
								<Badge variant="secondary">Disabled</Badge>
							{/if}
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex items-center justify-end gap-1">
							<Button variant="ghost" size="icon-sm" onclick={() => openEdit(camp)} title="Edit" aria-label="Edit {camp.name}">
								<PencilIcon class="size-4" />
							</Button>
							<Button variant="ghost" size="icon-sm" onclick={() => openDelete(camp)} title="Delete" aria-label="Delete {camp.name}">
									<TrashIcon class="size-4" />
								</Button>
							</div>
						</Table.TableCell>
					</Table.TableRow>
				{/each}
			</Table.TableBody>
		</Table.Table>
	{/if}
</div>

<!-- Create / Edit Dialog -->
<Dialog.Dialog bind:open={dialogOpen}>
	<Dialog.DialogContent>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>
				{dialogMode === "create" ? "Create Camp" : "Edit Camp"}
			</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				{dialogMode === "create"
					? "Add a new summer camp."
					: "Update camp details."}
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleSave} class="grid gap-4">
			{#if formError}
				<div class="bg-destructive/10 text-destructive rounded-lg px-3 py-2 text-sm">
					{formError}
				</div>
			{/if}
			<div class="grid gap-2">
				<Label for="camp-name">Name</Label>
				<Input
					id="camp-name"
					bind:value={formName}
					placeholder="Camp name"
					required
					disabled={saving}
				/>
			</div>
			<div class="grid gap-2">
				<Label for="camp-location">Location</Label>
				<Input
					id="camp-location"
					bind:value={formLocation}
					placeholder="Optional location"
					disabled={saving}
				/>
			</div>
			{#if dialogMode === "edit"}
				<div class="flex items-center gap-2">
					<input
						id="camp-enabled"
						type="checkbox"
						bind:checked={formEnabled}
						disabled={saving}
						class="border-input size-4 rounded"
					/>
					<Label for="camp-enabled">Enabled</Label>
				</div>
			{/if}
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" onclick={() => (dialogOpen = false)} disabled={saving}>
					Cancel
				</Button>
				<Button type="submit" disabled={saving}>
					{saving ? "Saving..." : dialogMode === "create" ? "Create" : "Save"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete Confirmation -->
<AlertDialog.AlertDialog bind:open={deleteDialogOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Camp</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to delete <strong>{deletingCamp?.name}</strong>? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				variant="destructive"
				onclick={handleDelete}
				disabled={deleting}
			>
				{deleting ? "Deleting..." : "Delete"}
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
