<script lang="ts">
	import { onMount } from "svelte";
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import { ApiClientError } from "$lib/api/client";
	import { assignmentApi, sessionApi, seasonApi } from "$lib/api";
	import type {
		RunResponse,
		TriggerRunRequest,
		Session,
		Season,
	} from "$lib/api/types";
	import { Button } from "$lib/components/ui/button";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Table from "$lib/components/ui/table";
	import * as Select from "$lib/components/ui/select";
	import { Label } from "$lib/components/ui/label";
	import { Badge } from "$lib/components/ui/badge";
	import PlayIcon from "@lucide/svelte/icons/play";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import CheckIcon from "@lucide/svelte/icons/check";
	import TentTreeIcon from "@lucide/svelte/icons/tent-tree";
	import UsersIcon from "@lucide/svelte/icons/users";
	import DumbbellIcon from "@lucide/svelte/icons/dumbbell";

	type RunType = "counselor_cabin" | "camper_cabin" | "activity_schedule";

	const RUN_TYPE_LABELS: Record<RunType, string> = {
		counselor_cabin: "Counselor-to-Cabin",
		camper_cabin: "Camper-to-Cabin",
		activity_schedule: "Activity Schedule",
	};

	let loading = $state(false);
	let loadError = $state<string | null>(null);
	let runs = $state<RunResponse[]>([]);
	let sessions = $state<Session[]>([]);
	let seasons = $state<Season[]>([]);
	let selectedSessionId = $state<string | null>(null);
	let sessionsLoaded = $state(false);

	let dialogOpen = $state(false);
	let triggering = $state(false);
	let triggerRunType = $state<RunType>("counselor_cabin");

	let deleteOpen = $state(false);
	let deleting = $state(false);
	let runToDelete = $state<string | null>(null);

	let errorDialogOpen = $state(false);
	let errorDialogMessage = $state("");

	let sessionLabelMap = $derived(new Map(
		sessions.map((s) => {
			const season = seasons.find((ss) => ss.id === s.season_id);
			const label = season ? `${season.name} - ${s.name}` : s.name;
			return [s.id, label];
		})
	));

	let sortedSessions = $derived.by(() => {
		return [...sessions].sort((a, b) => {
			const aLabel = sessionLabelMap.get(a.id) ?? a.name;
			const bLabel = sessionLabelMap.get(b.id) ?? b.name;
			return aLabel.localeCompare(bLabel);
		});
	});

	onMount(async () => {
		await loadSessions();
		sessionsLoaded = true;
	});

	$effect(() => {
		if (!sessionsLoaded) {
			return;
		}
		const sessionIdParam = $page.url.searchParams.get("session");
		if (!sessionIdParam) {
			selectedSessionId = null;
			runs = [];
			loading = false;
			loadError = null;
			return;
		}
		const exists = sessions.some((s) => s.id === sessionIdParam);
		if (!exists) {
			selectedSessionId = null;
			runs = [];
			loading = false;
			loadError = null;
			return;
		}
		if (sessionIdParam !== selectedSessionId) {
			selectedSessionId = sessionIdParam;
			loadRuns();
		}
	});

	async function loadSessions() {
		try {
			const [sessionsData, seasonsData] = await Promise.all([
				sessionApi.list(),
				seasonApi.list(),
			]);
			sessions = sessionsData;
			seasons = seasonsData;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load sessions";
			loadError = message;
			toast.error(message);
		}
	}

	function handleSessionChange(value: string) {
		selectedSessionId = value;
		updateSessionParam(value);
		loadRuns();
	}

	async function loadRuns() {
		if (!selectedSessionId) return;

		loading = true;
		loadError = null;

		try {
			runs = await assignmentApi.listRuns(selectedSessionId);
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load runs";
			loadError = message;
			toast.error(message);
		} finally {
			loading = false;
		}
	}

	function updateSessionParam(sessionId: string) {
		const url = new URL($page.url);
		url.searchParams.set("session", sessionId);
		goto(url, { replaceState: true, keepFocus: true, noScroll: true });
	}

	async function handleTriggerRun() {
		if (!selectedSessionId) return;

		triggering = true;
		try {
			const payload: TriggerRunRequest = {
				run_type: triggerRunType,
			};
			const result = await assignmentApi.triggerRun(selectedSessionId, payload);
			dialogOpen = false;
			toast.success("Solver run completed");
			runs = [result, ...runs.filter((r) => r.id !== result.id)];
			goto(`/app/assignment-runs/${result.id}`);
		} catch (err) {
			if (err instanceof ApiClientError && err.status === 422) {
				errorDialogMessage = err.message;
				dialogOpen = false;
				errorDialogOpen = true;
			} else {
				const message =
					err instanceof ApiClientError ? err.message : "Failed to trigger solver";
				toast.error(message);
			}
		} finally {
			triggering = false;
		}
	}

	function confirmDelete(runId: string) {
		runToDelete = runId;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!runToDelete || !selectedSessionId) return;

		deleting = true;
		try {
			await assignmentApi.deleteRun(selectedSessionId, runToDelete);
			runs = runs.filter((r) => r.id !== runToDelete);
			deleteOpen = false;
			runToDelete = null;
			toast.success("Run deleted");
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to delete run";
			toast.error(message);
		} finally {
			deleting = false;
		}
	}
</script>

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Assignment Runs</h1>
			<p class="text-muted-foreground text-sm">
				Trigger solver runs and review assignment solutions.
			</p>
		</div>
		<div class="flex items-center gap-3">
			<Select.Select
				type="single"
				value={selectedSessionId ?? undefined}
				onValueChange={handleSessionChange}
			>
				<Select.SelectTrigger class="w-[200px]">
					{#if selectedSessionId}
						{sessionLabelMap.get(selectedSessionId)}
					{:else}
						Select session...
					{/if}
				</Select.SelectTrigger>
				<Select.SelectContent>
					{#each sortedSessions as session}
						<Select.SelectItem value={session.id}>
							{sessionLabelMap.get(session.id)}
						</Select.SelectItem>
					{/each}
				</Select.SelectContent>
			</Select.Select>
			<Button
				size="sm"
				disabled={!selectedSessionId}
				onclick={() => (dialogOpen = true)}
			>
				<PlayIcon class="mr-2 size-4" />
				New Run
			</Button>
		</div>
	</div>

	{#if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">{loadError}</div>
	{:else if !selectedSessionId}
		<div class="text-muted-foreground py-8 text-center text-sm">
			Select a session above to view assignment runs.
		</div>
	{:else if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading runs...</div>
	{:else if runs.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No runs yet. Click "New Run" to trigger the solver.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<Table.TableHead>Type</Table.TableHead>
					<Table.TableHead>Status</Table.TableHead>
					<Table.TableHead>Created</Table.TableHead>
					<Table.TableHead class="w-[100px]">Actions</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each runs as run}
					<Table.TableRow
						class="cursor-pointer hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
						role="link"
						tabindex="0"
						onclick={() => goto(`/app/assignment-runs/${run.id}`)}
						onkeydown={(e: KeyboardEvent) => {
							if (e.key === "Enter" || e.key === " ") {
								e.preventDefault();
								goto(`/app/assignment-runs/${run.id}`);
							}
						}}
					>
						<Table.TableCell>
							<div class="flex items-center gap-2">
								{#if run.run_type === "counselor_cabin"}
									<UsersIcon class="size-4" />
								{:else if run.run_type === "camper_cabin"}
									<TentTreeIcon class="size-4" />
								{:else}
									<DumbbellIcon class="size-4" />
								{/if}
								{RUN_TYPE_LABELS[run.run_type]}
							</div>
						</Table.TableCell>
						<Table.TableCell>
							{#if run.status === "selected"}
								<Badge variant="default" class="bg-green-600">
									<CheckIcon class="mr-1 size-3" />
									Selected
								</Badge>
							{:else}
								<Badge variant="secondary">Completed</Badge>
							{/if}
						</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">
							{new Date(run.created_at).toLocaleString()}
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex gap-1">
								<a
									href={`/app/assignment-runs/${run.id}`}
									class="inline-flex items-center text-muted-foreground hover:text-foreground"
									onclick={(e: MouseEvent) => e.stopPropagation()}
								>
									<ChevronRightIcon class="size-4" />
									<span class="sr-only">View</span>
								</a>
								{#if run.status !== "selected"}
									<Button
										variant="ghost"
										size="icon-sm"
										title="Delete"
										onclick={(e: MouseEvent) => {
											e.stopPropagation();
											confirmDelete(run.id);
										}}
									>
										<TrashIcon class="size-4" />
										<span class="sr-only">Delete</span>
									</Button>
								{/if}
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
			<Dialog.DialogTitle>Trigger New Solver Run</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Select the type of assignment to run.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleTriggerRun();
			}}
			class="grid gap-4"
		>
			<div class="grid gap-2">
				<Label for="run-type">Run Type</Label>
				<Select.Select
					type="single"
					value={triggerRunType}
					onValueChange={(v: string) => (triggerRunType = v as RunType)}
				>
					<Select.SelectTrigger id="run-type" class="w-full">
						{#if triggerRunType === "counselor_cabin"}
							Counselor-to-Cabin
						{:else if triggerRunType === "camper_cabin"}
							Camper-to-Cabin
						{:else}
							Activity Schedule
						{/if}
					</Select.SelectTrigger>
					<Select.SelectContent>
						<Select.SelectItem value="counselor_cabin">
							Counselor-to-Cabin
						</Select.SelectItem>
						<Select.SelectItem value="camper_cabin">
							Camper-to-Cabin
						</Select.SelectItem>
						<Select.SelectItem value="activity_schedule">
							Activity Schedule
						</Select.SelectItem>
					</Select.SelectContent>
				</Select.Select>
			</div>
			<Dialog.DialogFooter>
				<Button
					type="button"
					variant="outline"
					disabled={triggering}
					onclick={() => (dialogOpen = false)}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={triggering}>
					{#if triggering}
						<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
						Running...
					{:else}
						<PlayIcon class="mr-2 size-4" />
						Run Solver
					{/if}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<AlertDialog.AlertDialog bind:open={errorDialogOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Solver Run Failed</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				{errorDialogMessage}
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogAction onclick={() => errorDialogOpen = false}>
				Close
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>

<AlertDialog.AlertDialog bind:open={deleteOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete Run</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to delete this assignment run? This action cannot be
				undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting}>
				Cancel
			</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction onclick={handleDelete} disabled={deleting}>
				{#if deleting}
					<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
					Deleting...
				{:else}
					<TrashIcon class="mr-2 size-4" />
					Delete
				{/if}
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
