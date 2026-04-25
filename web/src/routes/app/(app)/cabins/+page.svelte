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
	import { Badge } from "$lib/components/ui/badge";
	import type { AgeGroup, Cabin, Camp, Gender } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import ArchiveIcon from "@lucide/svelte/icons/archive";
	import ArchiveRestoreIcon from "@lucide/svelte/icons/archive-restore";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import ArchivedSection from "$lib/components/archived-section.svelte";
	import { sortItems, parsePositiveInt, type SortDirection } from "$lib/utils";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let cabins = $state<Cabin[]>([]);
	let ageGroups = $state<AgeGroup[]>([]);
	let loading = $state(true);
	let loadError = $state(false);
	let noAgeGroups = $derived(!loading && ageGroups.length === 0);

	let sortKey = $state<"name" | "default_age_group_name" | "gender">("name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedCabins = $derived(sortItems(cabins, sortKey, sortDirection));

	function toggleSort(key: "name" | "default_age_group_name" | "gender") {
		if (sortKey === key) {
			sortDirection = sortDirection === "asc" ? "desc" : "asc";
		} else {
			sortKey = key;
			sortDirection = "asc";
		}
	}

	// Create/edit dialog
	let dialogOpen = $state(false);
	let editingCabin = $state<Cabin | null>(null);
	let formName = $state("");
	let formAgeGroupId = $state("");
	let formGroupSize = $state("");
	let formRequiredCounselors = $state("");
	let formGender = $state<Gender | "">("");
	let submitting = $state(false);
	let nameError = $state("");
	let ageGroupError = $state("");
	let groupSizeError = $state("");
	let requiredCounselorsError = $state("");
	let genderError = $state("");

	let dialogTitle = $derived(editingCabin ? "Edit Cabin" : "Add Cabin");
	let dialogDescription = $derived(
		editingCabin
			? "Update the cabin details."
			: "Enter the details for the new cabin."
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
		groupSizeError = "";
		requiredCounselorsError = "";
		genderError = "";
	}

	function openCreate() {
		editingCabin = null;
		formName = "";
		formAgeGroupId = "";
		formGroupSize = "";
		formRequiredCounselors = "";
		formGender = "";
		clearErrors();
		dialogOpen = true;
	}

	function openEdit(cabin: Cabin) {
		editingCabin = cabin;
		formName = cabin.name;
		formAgeGroupId = cabin.default_age_group_id;
		formGroupSize = String(cabin.default_group_size);
		formRequiredCounselors = String(cabin.default_required_counselors);
		formGender = cabin.gender;
		clearErrors();
		dialogOpen = true;
		ensureArchivedParentLoaded(cabin.default_age_group_id);
	}

	let archivedAgeGroups = $state<AgeGroup[]>([]);
	let archivedAgeGroupsLoaded = $state(false);

	async function ensureArchivedParentLoaded(ageGroupId: string) {
		if (!ageGroupId) return;
		if (ageGroups.some((ag) => ag.id === ageGroupId)) return;
		if (archivedAgeGroups.some((ag) => ag.id === ageGroupId)) return;
		if (archivedAgeGroupsLoaded) return;
		try {
			archivedAgeGroups = await ageGroupApi.listArchived();
			archivedAgeGroupsLoaded = true;
		} catch {
			// ignore — dropdown will just show id
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
		if (!formAgeGroupId) {
			ageGroupError = "Age group is required.";
			valid = false;
		}
		const groupSize = parsePositiveInt(formGroupSize);
		if (!groupSize.ok) {
			groupSizeError = groupSize.error;
			valid = false;
		}
		const requiredCounselors = parsePositiveInt(formRequiredCounselors);
		if (!requiredCounselors.ok) {
			requiredCounselorsError = requiredCounselors.error;
			valid = false;
		}
		if (formGender !== "male" && formGender !== "female") {
			genderError = "Gender is required.";
			valid = false;
		}
		if (!valid || !groupSize.ok || !requiredCounselors.ok) return;

		submitting = true;

		try {
			if (editingCabin) {
				const updated = await cabinApi.update(editingCabin.id, {
					name,
					default_age_group_id: formAgeGroupId,
					default_group_size: groupSize.value,
					default_required_counselors: requiredCounselors.value,
					gender: formGender as Gender,
				});
				cabins = cabins.map((c) => (c.id === updated.id ? updated : c));
				toast.success("Cabin updated");
			} else {
				const created = await cabinApi.create({
					name,
					default_age_group_id: formAgeGroupId,
					default_group_size: groupSize.value,
					default_required_counselors: requiredCounselors.value,
					gender: formGender as Gender,
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

	let archivingId = $state<string | null>(null);
	let archivedSection = $state<ArchivedSection<Cabin> | null>(null);

	async function handleArchive(cabin: Cabin) {
		archivingId = cabin.id;
		try {
			await cabinApi.archive(cabin.id);
			cabins = cabins.filter((c) => c.id !== cabin.id);
			archivedSection?.addArchived({ ...cabin, archived: true });
			toast.success("Cabin archived");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to archive cabin";
			toast.error(message);
		} finally {
			archivingId = null;
		}
	}

	async function handleUnarchive(id: string) {
		await cabinApi.unarchive(id);
		try {
			cabins = await cabinApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to refresh cabin list";
			toast.error(message);
		}
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
			if (err instanceof ApiClientError && err.status === 409) {
				const target = deleteTarget;
				deleteOpen = false;
				deleteTarget = null;
				toast.message("Cabin has dependent records and cannot be deleted.", {
					description: "Archive it instead?",
					action: {
						label: "Archive",
						onClick: () => {
							if (target) handleArchive(target);
						},
					},
				});
			} else {
				const message = err instanceof ApiClientError ? err.message : "Failed to delete cabin";
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
					<SortableTableHead label="Name" active={sortKey === "name"} direction={sortDirection} onclick={() => toggleSort("name")} />
					<SortableTableHead label="Default Age Group" active={sortKey === "default_age_group_name"} direction={sortDirection} onclick={() => toggleSort("default_age_group_name")} />
					<Table.TableHead>Default Group Size</Table.TableHead>
					<Table.TableHead>Default Required Counselors</Table.TableHead>
					<SortableTableHead label="Gender" active={sortKey === "gender"} direction={sortDirection} onclick={() => toggleSort("gender")} />
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedCabins as cabin (cabin.id)}
					<Table.TableRow>
						<Table.TableCell>{cabin.name}</Table.TableCell>
						<Table.TableCell>{cabin.default_age_group_name}</Table.TableCell>
						<Table.TableCell>{cabin.default_group_size}</Table.TableCell>
						<Table.TableCell>{cabin.default_required_counselors}</Table.TableCell>
						<Table.TableCell>
							<Badge variant={cabin.gender === "female" ? "secondary" : "outline"}>
								{cabin.gender === "female" ? "Female" : "Male"}
							</Badge>
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Edit"
									onclick={() => openEdit(cabin)}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									title="Archive"
									disabled={disabled || archivingId === cabin.id}
									onclick={() => handleArchive(cabin)}
								>
									{#if archivingId === cabin.id}
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

	{#if camp}
		<ArchivedSection
			bind:this={archivedSection}
			resourceName="Cabin"
			listArchivedFn={cabinApi.listArchived}
			unarchiveFn={handleUnarchive}
		>
			{#snippet row({ item, unarchive, busy })}
				<div class="flex items-center justify-between border-b py-2 last:border-b-0">
					<span class="text-sm">
						{item.name}
						<span class="text-muted-foreground ml-2">{item.default_age_group_name}</span>
					</span>
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
							{@const selected = ageGroups.find((ag) => ag.id === formAgeGroupId) ?? archivedAgeGroups.find((ag) => ag.id === formAgeGroupId)}
							{#if selected}
								{selected.name}{#if selected.archived}<span class="text-muted-foreground"> (archived)</span>{/if}
							{:else}
								Select age group
							{/if}
						{:else}
							<span class="text-muted-foreground">Select age group</span>
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						{#each ageGroups as ag (ag.id)}
							<Select.SelectItem value={ag.id}>{ag.name}</Select.SelectItem>
						{/each}
						{#each archivedAgeGroups.filter((ag) => ag.id === formAgeGroupId) as ag (ag.id)}
							<Select.SelectItem value={ag.id} disabled>{ag.name} (archived)</Select.SelectItem>
						{/each}
					</Select.SelectContent>
				</Select.Select>
				{#if ageGroupError}
					<p class="text-destructive text-sm">{ageGroupError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-group-size">Default Group Size</Label>
				<Input
					id="cabin-group-size"
					type="number"
					min="1"
					placeholder="e.g. 8"
					bind:value={formGroupSize}
					disabled={submitting}
					oninput={() => (groupSizeError = "")}
				/>
				{#if groupSizeError}
					<p class="text-destructive text-sm">{groupSizeError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-required-counselors">Default Required Counselors</Label>
				<Input
					id="cabin-required-counselors"
					type="number"
					min="1"
					placeholder="e.g. 1"
					bind:value={formRequiredCounselors}
					disabled={submitting}
					oninput={() => (requiredCounselorsError = "")}
				/>
				{#if requiredCounselorsError}
					<p class="text-destructive text-sm">{requiredCounselorsError}</p>
				{/if}
			</div>
			<div class="grid gap-2">
				<Label for="cabin-gender">Gender</Label>
				<Select.Select type="single" bind:value={formGender} disabled={submitting} onValueChange={() => (genderError = "")}>
					<Select.SelectTrigger id="cabin-gender" class="w-full">
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
			>
				{#if deleting}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Delete
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
