<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { ApiClientError } from "$lib/api/client";
	import { camperApi } from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import { Badge } from "$lib/components/ui/badge";
	import type { Camper, Camp, Gender } from "$lib/api/types";
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

	let campers = $state<Camper[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	let sortKey = $state<"name" | "gender">("name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedCampers = $derived(sortItems(campers, sortKey, sortDirection));

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
	let editingCamper = $state<Camper | null>(null);
	let formName = $state("");
	let formGender = $state<Gender | "">("");
	let submitting = $state(false);
	let nameError = $state("");
	let genderError = $state("");

	let dialogTitle = $derived(editingCamper ? "Edit Camper" : "Add Camper");
	let dialogDescription = $derived(
		editingCamper
			? "Update the camper's details."
			: "Enter a name for the new camper."
	);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<Camper | null>(null);
	let deleting = $state(false);

	onMount(async () => {
		try {
			campers = await camperApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load campers";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	function openCreate() {
		editingCamper = null;
		formName = "";
		formGender = "";
		nameError = "";
		genderError = "";
		dialogOpen = true;
	}

	function openEdit(camper: Camper) {
		editingCamper = camper;
		formName = camper.name;
		formGender = camper.gender;
		nameError = "";
		genderError = "";
		dialogOpen = true;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		nameError = "";
		genderError = "";

		const name = formName.trim();
		let valid = true;
		if (!name) {
			nameError = "Name is required.";
			valid = false;
		}
		if (formGender !== "male" && formGender !== "female") {
			genderError = "Gender is required.";
			valid = false;
		}
		if (!valid) return;

		submitting = true;

		try {
			if (editingCamper) {
				const updated = await camperApi.update(editingCamper.id, { name, gender: formGender as Gender });
				campers = campers.map((c) => (c.id === updated.id ? updated : c));
				toast.success("Camper updated");
			} else {
				const created = await camperApi.create({ name, gender: formGender as Gender });
				campers = [...campers, created];
				toast.success("Camper created");
			}
			dialogOpen = false;
		} catch (err) {
			const action = editingCamper ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} camper`;
			toast.error(message);
		} finally {
			submitting = false;
		}
	}

	function confirmDelete(camper: Camper) {
		deleteTarget = camper;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await camperApi.delete(deleteTarget.id);
			campers = campers.filter((c) => c.id !== deleteTarget!.id);
			toast.success("Camper deleted");
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete camper";
			toast.error(message);
		} finally {
			deleting = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Campers</h1>
			<p class="text-muted-foreground text-sm">Manage camper profiles, enrollments, and friend preferences.</p>
		</div>
		{#if camp}
			<Button size="sm" disabled={disabled || loading || loadError} onclick={openCreate}>
				<PlusIcon class="mr-2 size-4" />
				Add Camper
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load campers. Try refreshing the page.
		</div>
	{:else if campers.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No campers yet. Click "Add Camper" to create one.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<SortableTableHead label="Name" active={sortKey === "name"} direction={sortDirection} onclick={() => toggleSort("name")} />
					<SortableTableHead label="Gender" active={sortKey === "gender"} direction={sortDirection} onclick={() => toggleSort("gender")} />
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedCampers as camper (camper.id)}
					<Table.TableRow
						class="cursor-pointer hover:bg-muted/50"
						tabindex={0}
						role="button"
						onclick={() => goto(`/app/campers/${camper.id}`)}
						onkeydown={(e: KeyboardEvent) => {
							if (e.key === "Enter" || e.key === " ") {
								e.preventDefault();
								goto(`/app/campers/${camper.id}`);
							}
						}}
					>
						<Table.TableCell>{camper.name}</Table.TableCell>
						<Table.TableCell>
							<Badge variant={camper.gender === "female" ? "secondary" : "outline"}>
								{camper.gender === "female" ? "Female" : "Male"}
							</Badge>
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex items-center justify-end gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Edit"
									onclick={(e: MouseEvent) => { e.stopPropagation(); openEdit(camper); }}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Delete"
									onclick={(e: MouseEvent) => { e.stopPropagation(); confirmDelete(camper); }}
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
				<Label for="camper-name">Name</Label>
				<Input
					id="camper-name"
					type="text"
					placeholder="Camper name"
					bind:value={formName}
					disabled={submitting}
					oninput={() => (nameError = "")}
				/>
				{#if nameError}
					<p class="text-destructive text-sm">{nameError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="camper-gender">Gender</Label>
				<Select.Select type="single" bind:value={formGender} disabled={submitting} onValueChange={() => (genderError = "")}>
					<Select.SelectTrigger id="camper-gender" class="w-full">
						{#if formGender === "female"}
							Female
						{:else if formGender === "male"}
							Male
						{:else}
							<span class="text-muted-foreground">Select gender</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						<Select.SelectItem value="female">Female</Select.SelectItem>
						<Select.SelectItem value="male">Male</Select.SelectItem>
					</Select.SelectContent>
				</Select.Select>
				{#if genderError}
					<p class="text-destructive text-sm">{genderError}</p>
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
					{editingCamper ? "Save" : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete confirmation -->
<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Camper</AlertDialog.AlertDialogTitle>
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
