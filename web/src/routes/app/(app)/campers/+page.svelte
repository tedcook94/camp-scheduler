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
	import ArchiveIcon from "@lucide/svelte/icons/archive";
	import ArchiveRestoreIcon from "@lucide/svelte/icons/archive-restore";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import ArchivedSection from "$lib/components/archived-section.svelte";
	import { sortItems, type SortDirection } from "$lib/utils";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let campers = $state<Camper[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	let sortKey = $state<"last_name" | "gender">("last_name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedCampers = $derived.by(() => {
		const items = [...campers];
		const dir = sortDirection === "asc" ? 1 : -1;
		if (sortKey === "last_name") {
			items.sort((a, b) => {
				const byLast = a.last_name.localeCompare(b.last_name, undefined, { sensitivity: "base" });
				if (byLast !== 0) return byLast * dir;
				return a.first_name.localeCompare(b.first_name, undefined, { sensitivity: "base" }) * dir;
			});
			return items;
		}
		return sortItems(items, sortKey, sortDirection);
	});

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
	let formFirstName = $state("");
	let formLastName = $state("");
	let formGender = $state<Gender | "">("");
	let submitting = $state(false);
	let firstNameError = $state("");
	let lastNameError = $state("");
	let genderError = $state("");

	let dialogTitle = $derived(editingCamper ? "Edit Camper" : "Add Camper");
	let dialogDescription = $derived(
		editingCamper
			? "Update the camper's details."
			: "Enter the camper's first and last name."
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
		formFirstName = "";
		formLastName = "";
		formGender = "";
		firstNameError = "";
		lastNameError = "";
		genderError = "";
		dialogOpen = true;
	}

	function openEdit(camper: Camper) {
		editingCamper = camper;
		formFirstName = camper.first_name;
		formLastName = camper.last_name;
		formGender = camper.gender;
		firstNameError = "";
		lastNameError = "";
		genderError = "";
		dialogOpen = true;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		firstNameError = "";
		lastNameError = "";
		genderError = "";

		const first_name = formFirstName.trim();
		const last_name = formLastName.trim();
		let valid = true;
		if (!first_name) {
			firstNameError = "First name is required.";
			valid = false;
		}
		if (!last_name) {
			lastNameError = "Last name is required.";
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
				const updated = await camperApi.update(editingCamper.id, { first_name, last_name, gender: formGender as Gender });
				campers = campers.map((c) => (c.id === updated.id ? updated : c));
				toast.success("Camper updated");
			} else {
				const created = await camperApi.create({ first_name, last_name, gender: formGender as Gender });
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

	let archivingId = $state<string | null>(null);
	let archivedSection = $state<ArchivedSection<Camper> | null>(null);

	async function handleArchive(camper: Camper) {
		archivingId = camper.id;
		try {
			await camperApi.archive(camper.id);
			campers = campers.filter((c) => c.id !== camper.id);
			archivedSection?.addArchived({ ...camper, archived: true });
			toast.success("Camper archived");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to archive camper";
			toast.error(message);
		} finally {
			archivingId = null;
		}
	}

	async function handleUnarchive(id: string) {
		await camperApi.unarchive(id);
		try {
			campers = await camperApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to refresh camper list";
			toast.error(message);
		}
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
			if (err instanceof ApiClientError && err.status === 409) {
				const target = deleteTarget;
				deleteOpen = false;
				deleteTarget = null;
				toast.message("Camper has dependent records and cannot be deleted.", {
					description: "Archive it instead?",
					action: {
						label: "Archive",
						onClick: () => {
							if (target) handleArchive(target);
						},
					},
				});
			} else {
				const message = err instanceof ApiClientError ? err.message : "Failed to delete camper";
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
					<SortableTableHead label="Name" active={sortKey === "last_name"} direction={sortDirection} onclick={() => toggleSort("last_name")} />
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
									title="Archive"
									disabled={disabled || archivingId === camper.id}
									onclick={(e: MouseEvent) => { e.stopPropagation(); handleArchive(camper); }}
								>
									{#if archivingId === camper.id}
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

	{#if camp}
		<ArchivedSection
			bind:this={archivedSection}
			resourceName="Camper"
			listArchivedFn={camperApi.listArchived}
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
				<Label for="camper-first-name">First name</Label>
				<Input
					id="camper-first-name"
					type="text"
					placeholder="First name"
					bind:value={formFirstName}
					disabled={submitting}
					oninput={() => (firstNameError = "")}
				/>
				{#if firstNameError}
					<p class="text-destructive text-sm">{firstNameError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="camper-last-name">Last name</Label>
				<Input
					id="camper-last-name"
					type="text"
					placeholder="Last name"
					bind:value={formLastName}
					disabled={submitting}
					oninput={() => (lastNameError = "")}
				/>
				{#if lastNameError}
					<p class="text-destructive text-sm">{lastNameError}</p>
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
