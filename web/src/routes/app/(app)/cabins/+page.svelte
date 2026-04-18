<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { ApiClientError } from "$lib/api/client";
	import { cabinApi, ageGroupApi } from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import type { AgeGroup, Cabin, Camp } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let cabins = $state<Cabin[]>([]);
	let ageGroups = $state<AgeGroup[]>([]);
	let loading = $state(true);
	let loadError = $state(false);
	let noAgeGroups = $derived(!loading && ageGroups.length === 0);

	// Create/edit dialog
	let dialogOpen = $state(false);
	let editingCabin = $state<Cabin | null>(null);
	let formName = $state("");
	let formAgeGroupId = $state("");
	let submitting = $state(false);
	let nameError = $state("");
	let ageGroupError = $state("");

	let dialogTitle = $derived(editingCabin ? "Edit Cabin" : "Add Cabin");
	let dialogDescription = $derived(
		editingCabin
			? "Update the cabin name and default age group."
			: "Enter a name and select a default age group for the new cabin."
	);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<Cabin | null>(null);
	let deleting = $state(false);

	onMount(async () => {
		try {
			const [cabinList, ageGroupList] = await Promise.all([
				cabinApi.list(),
				ageGroupApi.list(),
			]);
			cabins = cabinList;
			ageGroups = ageGroupList;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load cabins";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	function clearErrors() {
		nameError = "";
		ageGroupError = "";
	}

	function openCreate() {
		editingCabin = null;
		formName = "";
		formAgeGroupId = "";
		clearErrors();
		dialogOpen = true;
	}

	function openEdit(cabin: Cabin) {
		editingCabin = cabin;
		formName = cabin.name;
		formAgeGroupId = cabin.default_age_group_id;
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
		if (!formAgeGroupId) {
			ageGroupError = "Age group is required.";
			valid = false;
		}
		if (!valid) return;

		submitting = true;

		try {
			if (editingCabin) {
				const updated = await cabinApi.update(editingCabin.id, {
					name,
					default_age_group_id: formAgeGroupId,
				});
				cabins = cabins.map((c) => (c.id === updated.id ? updated : c));
				toast.success("Cabin updated");
			} else {
				const created = await cabinApi.create({
					name,
					default_age_group_id: formAgeGroupId,
				});
				cabins = [...cabins, created];
				toast.success("Cabin created");
			}
			dialogOpen = false;
		} catch (err) {
			const action = editingCabin ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} cabin`;
			toast.error(message);
		} finally {
			submitting = false;
		}
	}

	function confirmDelete(cabin: Cabin) {
		deleteTarget = cabin;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await cabinApi.delete(deleteTarget.id);
			cabins = cabins.filter((c) => c.id !== deleteTarget!.id);
			toast.success("Cabin deleted");
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete cabin";
			toast.error(message);
		} finally {
			deleting = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Cabins</h1>
			<p class="text-muted-foreground text-sm">Manage camp cabins and their default age group assignments.</p>
		</div>
		{#if camp}
			<Button size="sm" disabled={disabled || loading || loadError || noAgeGroups} onclick={openCreate}>
				<PlusIcon class="mr-2 size-4" />
				Add Cabin
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load cabins. Try refreshing the page.
		</div>
	{:else if noAgeGroups}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No age groups found. <a href="/app/age-groups" class="text-foreground underline">Create an age group</a> before adding cabins.
		</div>
	{:else if cabins.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No cabins yet. Click "Add Cabin" to create one.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<Table.TableHead>Name</Table.TableHead>
					<Table.TableHead>Default Age Group</Table.TableHead>
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each cabins as cabin (cabin.id)}
					<Table.TableRow>
						<Table.TableCell>{cabin.name}</Table.TableCell>
						<Table.TableCell>{cabin.default_age_group_name}</Table.TableCell>
						<Table.TableCell>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									onclick={() => openEdit(cabin)}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									onclick={() => confirmDelete(cabin)}
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
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>{dialogTitle}</Dialog.DialogTitle>
			<Dialog.DialogDescription>{dialogDescription}</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleSubmit} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="cabin-name">Name</Label>
				<Input
					id="cabin-name"
					type="text"
					placeholder="Cabin name"
					bind:value={formName}
					disabled={submitting}
					oninput={() => (nameError = "")}
				/>
				{#if nameError}
					<p class="text-destructive text-sm">{nameError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-age-group">Default Age Group</Label>
				<Select.Select type="single" bind:value={formAgeGroupId} disabled={submitting} onValueChange={() => (ageGroupError = "")}>
					<Select.SelectTrigger id="cabin-age-group" class="w-full">
						{#if formAgeGroupId}
							{ageGroups.find((ag) => ag.id === formAgeGroupId)?.name ?? "Select age group"}
						{:else}
							<span class="text-muted-foreground">Select age group</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each ageGroups as ag (ag.id)}
							<Select.SelectItem value={ag.id}>{ag.name}</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if ageGroupError}
					<p class="text-destructive text-sm">{ageGroupError}</p>
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
					{editingCabin ? "Save" : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete confirmation -->
<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Cabin</AlertDialog.AlertDialogTitle>
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
