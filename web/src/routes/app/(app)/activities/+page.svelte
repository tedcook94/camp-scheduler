<script lang="ts">
	import { getContext, onMount } from "svelte";
	import { ApiClientError } from "$lib/api/client";
	import { activityApi, activityCertificationApi, certificationApi } from "$lib/api";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import type { Activity, ActivityCertification, Certification, Camp } from "$lib/api/types";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import { sortItems, type SortDirection } from "$lib/utils";

	const getCampDisabled = getContext<() => boolean>("campDisabled");
	const getCamp = getContext<() => Camp | null>("camp");

	let disabled = $derived.by(() => getCampDisabled());
	let camp = $derived.by(() => getCamp());

	let activities = $state<Activity[]>([]);
	let allCertifications = $state<Certification[]>([]);
	let certsByActivity = $state<Map<string, ActivityCertification[]>>(new Map());
	let loading = $state(true);
	let loadError = $state(false);
	let syncingActivityId = $state<string | null>(null);

	let sortKey = $state<"name">("name");
	let sortDirection = $state<SortDirection>("asc");
	let sortedActivities = $derived(sortItems(activities, sortKey, sortDirection));

	function toggleSort(key: typeof sortKey) {
		if (sortKey === key) {
			sortDirection = sortDirection === "asc" ? "desc" : "asc";
		} else {
			sortKey = key;
			sortDirection = "asc";
		}
	}

	let dialogOpen = $state(false);
	let editingActivity = $state<Activity | null>(null);
	let formName = $state("");
	let submitting = $state(false);
	let nameError = $state("");

	let dialogTitle = $derived(editingActivity ? "Edit Activity" : "Add Activity");
	let dialogDescription = $derived(
		editingActivity
			? "Update the activity name."
			: "Enter a name for the new activity."
	);

	let deleteOpen = $state(false);
	let deleteTarget = $state<Activity | null>(null);
	let deleting = $state(false);

	// Certification name lookup
	let certNameMap = $derived(new Map(allCertifications.map((c) => [c.id, c.name])));

	function getSelectedCertIds(activityId: string): string[] {
		return (certsByActivity.get(activityId) ?? []).map((ac) => ac.certification_id);
	}

	onMount(async () => {
		try {
			const [actList, certList] = await Promise.all([
				activityApi.list(),
				certificationApi.list(),
			]);
			activities = actList;
			allCertifications = certList;

			const certsMap = new Map<string, ActivityCertification[]>();
			await Promise.all(
				actList.map(async (a) => {
					const certs = await activityCertificationApi.list(a.id);
					certsMap.set(a.id, certs);
				})
			);
			certsByActivity = certsMap;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load activities";
			toast.error(message);
			loadError = true;
		} finally {
			loading = false;
		}
	});

	async function handleCertsChange(activityId: string, newCertIds: string[]) {
		if (syncingActivityId) return;
		syncingActivityId = activityId;

		try {
			const current = certsByActivity.get(activityId) ?? [];
			const currentIds = new Set(current.map((ac) => ac.certification_id));
			const newIds = new Set(newCertIds);

			const toAdd = newCertIds.filter((id) => !currentIds.has(id));
			const toRemove = current.filter((ac) => !newIds.has(ac.certification_id));

			for (const certId of toAdd) {
				try {
					const created = await activityCertificationApi.add(activityId, certId);
					const updated = [...(certsByActivity.get(activityId) ?? []), created];
					certsByActivity = new Map([...certsByActivity, [activityId, updated]]);
				} catch (err) {
					const message = err instanceof ApiClientError ? err.message : "Failed to add certification";
					toast.error(message);
				}
			}

			for (const ac of toRemove) {
				try {
					await activityCertificationApi.remove(activityId, ac.id);
					const updated = (certsByActivity.get(activityId) ?? []).filter((c) => c.id !== ac.id);
					certsByActivity = new Map([...certsByActivity, [activityId, updated]]);
				} catch (err) {
					const message = err instanceof ApiClientError ? err.message : "Failed to remove certification";
					toast.error(message);
				}
			}
		} finally {
			syncingActivityId = null;
		}
	}

	function openCreate() {
		editingActivity = null;
		formName = "";
		nameError = "";
		dialogOpen = true;
	}

	function openEdit(activity: Activity) {
		editingActivity = activity;
		formName = activity.name;
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
			if (editingActivity) {
				const updated = await activityApi.update(editingActivity.id, name);
				activities = activities.map((a) => (a.id === updated.id ? updated : a));
				toast.success("Activity updated");
			} else {
				const created = await activityApi.create(name);
				activities = [...activities, created];
				certsByActivity = new Map([...certsByActivity, [created.id, []]]);
				toast.success("Activity created");
			}
			dialogOpen = false;
		} catch (err) {
			const action = editingActivity ? "update" : "create";
			const message = err instanceof ApiClientError ? err.message : `Failed to ${action} activity`;
			toast.error(message);
		} finally {
			submitting = false;
		}
	}

	function confirmDelete(activity: Activity) {
		deleteTarget = activity;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;

		try {
			await activityApi.delete(deleteTarget.id);
			activities = activities.filter((a) => a.id !== deleteTarget!.id);
			const updated = new Map(certsByActivity);
			updated.delete(deleteTarget.id);
			certsByActivity = updated;
			toast.success("Activity deleted");
			deleteOpen = false;
			deleteTarget = null;
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete activity";
			toast.error(message);
		} finally {
			deleting = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Activities</h1>
			<p class="text-muted-foreground text-sm">Manage activities and their certification requirements.</p>
		</div>
		{#if camp}
			<Button size="sm" disabled={disabled || loading || loadError} onclick={openCreate}>
				<PlusIcon class="mr-2 size-4" />
				Add Activity
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading...</div>
	{:else if !camp}
		<div class="text-muted-foreground py-8 text-center text-sm">No camp data available.</div>
	{:else if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Failed to load activities. Try refreshing the page.
		</div>
	{:else if activities.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No activities yet. Click "Add Activity" to create one.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<SortableTableHead label="Name" active={sortKey === "name"} direction={sortDirection} onclick={() => toggleSort("name")} />
					<Table.TableHead class="w-96">Required Certifications</Table.TableHead>
					<Table.TableHead class="w-24">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedActivities as activity (activity.id)}
					{@const selectedCertIds = getSelectedCertIds(activity.id)}
					<Table.TableRow>
						<Table.TableCell>{activity.name}</Table.TableCell>
						<Table.TableCell>
							<Select.Root
								type="multiple"
								value={selectedCertIds}
								disabled={disabled || allCertifications.length === 0 || syncingActivityId !== null}
								onValueChange={(v) => handleCertsChange(activity.id, v)}
							>
								<Select.Trigger class="h-auto min-h-9 w-full">
									{#if selectedCertIds.length === 0}
										<span class="text-muted-foreground">None</span>
									{:else}
										<span class="truncate">
											{selectedCertIds.map((id) => certNameMap.get(id) ?? id).sort((a, b) => a.localeCompare(b)).join(", ")}
										</span>
									{/if}
								</Select.Trigger>
								<Select.Content>
									{#each allCertifications as cert (cert.id)}
										<Select.Item value={cert.id}>{cert.name}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Edit"
									onclick={() => openEdit(activity)}
								>
									<PencilIcon class="size-4" />
									<span class="sr-only">Edit</span>
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									disabled={disabled}
									title="Delete"
									onclick={() => confirmDelete(activity)}
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

<Dialog.Dialog bind:open={dialogOpen}>
	<Dialog.DialogContent onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>{dialogTitle}</Dialog.DialogTitle>
			<Dialog.DialogDescription>{dialogDescription}</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleSubmit} class="grid gap-4">
			<div class="grid gap-2">
				<Label for="activity-name">Name</Label>
				<Input
					id="activity-name"
					type="text"
					placeholder="Activity name"
					bind:value={formName}
					disabled={submitting}
					oninput={() => (nameError = "")}
				/>
				{#if nameError}
					<p class="text-destructive text-sm">{nameError}</p>
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
					{editingActivity ? "Save" : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Activity</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to delete "{deleteTarget?.name}"? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction disabled={deleting} onclick={handleDelete}>
				{#if deleting}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Delete
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
