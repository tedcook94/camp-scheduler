<script lang="ts">
	import { goto } from "$app/navigation";
	import { getContext, onMount } from "svelte";

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
	import * as Select from "$lib/components/ui/select";
	import type { Counselor, Camp, Gender } from "$lib/api/types";
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

	let counselors = $state<Counselor[]>([]);
	let loading = $state(true);
	let loadError = $state(false);

	let sortKey = $state<"last_name" | "junior_counselor" | "gender">("last_name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedCounselors = $derived.by(() => {
		const items = [...counselors];
		const dir = sortDirection === "asc" ? 1 : -1;
		if (sortKey === "last_name") {
			items.sort((a, b) => {
				const byLast = a.last_name.localeCompare(b.last_name, undefined, { sensitivity: "base" });
				if (byLast !== 0) return byLast * dir;
				return a.first_name.localeCompare(b.first_name, undefined, { sensitivity: "base" }) * dir;
			});
			return items;
		}
		return sortItems(
			items,
			sortKey === "junior_counselor"
				? (c: Counselor) => (c.junior_counselor ? "Junior" : "Senior")
				: sortKey,
			sortDirection
		);
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
	let editingCounselor = $state<Counselor | null>(null);
	let formFirstName = $state("");
	let formLastName = $state("");
	let formJunior = $state(false);
	let formGender = $state<Gender | "">("");
	let submitting = $state(false);
	let firstNameError = $state("");
	let lastNameError = $state("");
	let genderError = $state("");

	let dialogTitle = $derived(editingCounselor ? "Edit Counselor" : "Add Counselor");
	let dialogDescription = $derived(
		editingCounselor
			? "Update the counselor's details."
			: "Enter the counselor's first and last name."
	);

	// Delete confirmation
	let deleteOpen = $state(false);
	let deleteTarget = $state<Counselor | null>(null);
	let deleting = $state(false);

	let archivingId = $state<string | null>(null);
	let archivedSection = $state<ArchivedSection<Counselor> | null>(null);

	async function handleArchive(counselor: Counselor) {
		archivingId = counselor.id;
		try {
			await counselorApi.archive(counselor.id);
			counselors = counselors.filter((c) => c.id !== counselor.id);
			archivedSection?.addArchived({ ...counselor, archived: true });
			toast.success("Counselor archived");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to archive counselor";
			toast.error(message);
		} finally {
			archivingId = null;
		}
	}

	async function handleUnarchive(id: string) {
		await counselorApi.unarchive(id);
		try {
			counselors = await counselorApi.list();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to refresh counselor list";
			toast.error(message);
		}
	}

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
		formFirstName = "";
		formLastName = "";
		formJunior = false;
		formGender = "";
		firstNameError = "";
		lastNameError = "";
		genderError = "";
		dialogOpen = true;
	}

	function openEdit(counselor: Counselor) {
		editingCounselor = counselor;
		formFirstName = counselor.first_name;
		formLastName = counselor.last_name;
		formJunior = counselor.junior_counselor;
		formGender = counselor.gender;
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
			if (editingCounselor) {
				const updated = await counselorApi.update(editingCounselor.id, {
					first_name,
					last_name,
					junior_counselor: formJunior,
					gender: formGender as Gender,
				});
				counselors = counselors.map((c) => (c.id === updated.id ? updated : c));
				toast.success("Counselor updated");
			} else {
				const created = await counselorApi.create({
					first_name,
					last_name,
					junior_counselor: formJunior,
					gender: formGender as Gender,
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
			if (err instanceof ApiClientError && err.status === 409) {
				const target = deleteTarget;
				deleteOpen = false;
				deleteTarget = null;
				toast.message("Counselor has dependent records and cannot be deleted.", {
					description: "Archive it instead?",
					action: {
						label: "Archive",
						onClick: () => {
							if (target) handleArchive(target);
						},
					},
				});
			} else {
				const message = err instanceof ApiClientError ? err.message : "Failed to delete counselor";
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
					<SortableTableHead label="Name" active={sortKey === "last_name"} direction={sortDirection} onclick={() => toggleSort("last_name")} />
					<SortableTableHead label="Type" active={sortKey === "junior_counselor"} direction={sortDirection} onclick={() => toggleSort("junior_counselor")} />
					<SortableTableHead label="Gender" active={sortKey === "gender"} direction={sortDirection} onclick={() => toggleSort("gender")} />
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedCounselors as counselor (counselor.id)}
					<Table.TableRow class="cursor-pointer hover:bg-muted/50" onclick={() => goto(`/app/counselors/${counselor.id}`)}>
						<Table.TableCell>{counselor.name}</Table.TableCell>
						<Table.TableCell>
							<Badge variant={counselor.junior_counselor ? "secondary" : "default"}>
								{counselor.junior_counselor ? "Junior" : "Senior"}
							</Badge>
						</Table.TableCell>
						<Table.TableCell>
							<Badge variant={counselor.gender === "female" ? "secondary" : "outline"}>
								{counselor.gender === "female" ? "Female" : "Male"}
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
									title="Archive"
									disabled={disabled || archivingId === counselor.id}
									onclick={(e: MouseEvent) => { e.stopPropagation(); handleArchive(counselor); }}
								>
									{#if archivingId === counselor.id}
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
									onclick={(e: MouseEvent) => { e.stopPropagation(); confirmDelete(counselor); }}
								>
									<TrashIcon class="size-4" />
									<span class="sr-only">Delete</span>
								</Button>
								<a
									href={`/app/counselors/${counselor.id}`}
									class="ml-1 inline-flex items-center text-muted-foreground hover:text-foreground"
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

	{#if camp}
		<ArchivedSection
			bind:this={archivedSection}
			resourceName="Counselor"
			listArchivedFn={counselorApi.listArchived}
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
				<Label for="counselor-first-name">First name</Label>
				<Input
					id="counselor-first-name"
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
				<Label for="counselor-last-name">Last name</Label>
				<Input
					id="counselor-last-name"
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
			<div class="grid gap-2">
				<Label for="counselor-gender">Gender</Label>
				<Select.Select type="single" bind:value={formGender} disabled={submitting} onValueChange={() => (genderError = "")}>
					<Select.SelectTrigger id="counselor-gender" class="w-full">
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
