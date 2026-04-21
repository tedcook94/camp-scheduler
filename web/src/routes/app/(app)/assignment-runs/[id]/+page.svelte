<script lang="ts">
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import { ApiClientError } from "$lib/api/client";
	import { assignmentApi } from "$lib/api";
	import type {
		RunDetailResponse,
		SolutionDetailResponse,
	} from "$lib/api/types";
	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Table from "$lib/components/ui/table";
	import { Badge } from "$lib/components/ui/badge";
	import { Separator } from "$lib/components/ui/separator";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import CheckIcon from "@lucide/svelte/icons/check";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import TentTreeIcon from "@lucide/svelte/icons/tent-tree";
	import UsersIcon from "@lucide/svelte/icons/users";
	import DumbbellIcon from "@lucide/svelte/icons/dumbbell";
	import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";

	type RunType = "counselor_cabin" | "camper_cabin" | "activity_schedule";

	const RUN_TYPE_LABELS: Record<RunType, string> = {
		counselor_cabin: "Counselor-to-Cabin",
		camper_cabin: "Camper-to-Cabin",
		activity_schedule: "Activity Schedule",
	};

	const runId = $derived($page.params.id);

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let run = $state<RunDetailResponse | null>(null);
	let sessionId = $state<string | null>(null);

	let expandedSolutions = $state<Set<string>>(new Set());
	let solutionDetails = $state<Map<string, SolutionDetailResponse>>(new Map());
	let loadingSolution = $state<string | null>(null);

	let deleteOpen = $state(false);
	let deleting = $state(false);

	let selecting = $state<string | null>(null);

	let prevRunId = $state<string | null>(null);

	$effect(() => {
		const currentRunId = runId;
		if (currentRunId && currentRunId !== prevRunId) {
			prevRunId = currentRunId;
			run = null;
			sessionId = null;
			expandedSolutions = new Set();
			solutionDetails = new Map();
			loadRun();
		}
	});

	async function loadRun() {
		if (!runId) return;

		loading = true;
		loadError = null;

		try {
			run = await assignmentApi.getRunById(runId);
			sessionId = run.session_id;
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load run";
			loadError = message;
			toast.error(message);
		} finally {
			loading = false;
		}
	}

	function handleBack() {
		const runSessionId = run?.session_id;
		const url = runSessionId
			? `/app/assignment-runs?session=${runSessionId}`
			: "/app/assignment-runs";
		goto(url);
	}

	async function toggleSolution(solutionId: string) {
		if (expandedSolutions.has(solutionId)) {
			expandedSolutions = new Set([...expandedSolutions].filter((id) => id !== solutionId));
			return;
		}

		if (solutionDetails.has(solutionId)) {
			expandedSolutions = new Set(expandedSolutions).add(solutionId);
			return;
		}

		if (!run || !sessionId) return;

		loadingSolution = solutionId;
		try {
			const details = await assignmentApi.getSolution(
				sessionId,
				run.id,
				solutionId,
			);
			solutionDetails = new Map(solutionDetails).set(solutionId, details);
			expandedSolutions = new Set(expandedSolutions).add(solutionId);
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load solution details";
			toast.error(message);
		} finally {
			loadingSolution = null;
		}
	}

	async function handleSelectSolution(solutionId: string) {
		if (!run || !sessionId) return;

		selecting = solutionId;
		try {
			await assignmentApi.selectSolution(
				sessionId,
				run.id,
				solutionId,
			);
			run = { ...run, status: "selected", selected_solution_id: solutionId };
			toast.success("Solution selected");
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to select solution";
			toast.error(message);
		} finally {
			selecting = null;
		}
	}

	function confirmDelete() {
		if (!run) return;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!run || !sessionId) return;

		deleting = true;
		try {
			await assignmentApi.deleteRun(sessionId, run.id);
			deleteOpen = false;
			toast.success("Run deleted");
			handleBack();
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
	{#if loadError}
		<div class="text-muted-foreground py-8 text-center text-sm">{loadError}</div>
	{:else if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading run...</div>
	{:else if run}
		<div class="grid gap-6">
			<div class="flex items-center gap-4">
				<Button variant="ghost" size="icon-sm" onclick={handleBack}>
					<ArrowLeftIcon class="size-4" />
					<span class="sr-only">Back</span>
				</Button>
				<div class="flex items-center gap-2">
					{#if run.run_type === "counselor_cabin"}
						<UsersIcon class="size-4" />
					{:else if run.run_type === "camper_cabin"}
						<TentTreeIcon class="size-4" />
					{:else}
						<DumbbellIcon class="size-4" />
					{/if}
					<span class="font-medium">{RUN_TYPE_LABELS[run.run_type]}</span>
					{#if run.status === "selected"}
						<Badge variant="default" class="bg-green-600">
							<CheckIcon class="mr-1 size-3" />
							Selected
						</Badge>
					{:else}
						<Badge variant="secondary">Completed</Badge>
					{/if}
				</div>
			</div>

			<div class="text-muted-foreground text-sm">
				Created: {new Date(run.created_at).toLocaleString()}
			</div>

			<div class="grid gap-4">
				<h2 class="text-lg font-semibold">Solutions</h2>
				{#each run.solutions as solution, index}
					<Card.Card>
						<div
							class="flex w-full cursor-pointer items-center justify-between bg-transparent p-4 text-left hover:bg-muted/50"
							role="button"
							tabindex="0"
							aria-expanded={expandedSolutions.has(solution.id)}
							onclick={() => toggleSolution(solution.id)}
							onkeydown={(e) => {
								if (e.key === "Enter" || e.key === " ") {
									e.preventDefault();
									toggleSolution(solution.id);
								}
							}}
						>
							<div class="flex items-center gap-3">
								{#if expandedSolutions.has(solution.id)}
									<ChevronDownIcon class="size-5" />
								{:else}
									<ChevronRightIcon class="size-5" />
								{/if}
								<div>
									<div class="flex items-center gap-2">
										<span class="font-medium">Solution {index + 1}</span>
										{#if run.selected_solution_id === solution.id}
											<Badge variant="default" class="bg-green-600">
												<CheckIcon class="mr-1 size-3" />
												Selected
											</Badge>
										{/if}
									</div>
									<div class="text-muted-foreground text-sm">
										Score: {solution.score.toFixed(2)}
									</div>
								</div>
							</div>
							<div class="flex items-center gap-2">
								{#if run.status !== "selected"}
									<Button
										variant={run.selected_solution_id === solution.id ? "default" : "outline"}
										size="sm"
										disabled={selecting !== null}
										onclick={(e) => {
											e.stopPropagation();
											handleSelectSolution(solution.id);
										}}
									>
										{#if selecting === solution.id}
											<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
										{:else if run.selected_solution_id === solution.id}
											<CheckIcon class="mr-2 size-4" />
										{/if}
										{run.selected_solution_id === solution.id ? "Selected" : "Select"}
									</Button>
								{/if}
							</div>
						</div>

						{#if expandedSolutions.has(solution.id)}
							<Separator />
							<div class="p-4">
								{#if loadingSolution === solution.id}
									<div class="text-muted-foreground py-4 text-center text-sm">
										Loading details...
									</div>
								{:else}
									{@const details = solutionDetails.get(solution.id)}
									{#if details}
										<div class="grid gap-6">
											<div>
												<h3 class="mb-2 text-sm font-medium">Score Breakdown</h3>
												<div class="grid gap-1">
													{#each details.score_breakdown as breakdown}
														<div class="flex justify-between text-sm">
															<span class="text-muted-foreground">{breakdown.Constraint}</span>
															<span class="font-medium">+{breakdown.Score}</span>
														</div>
													{/each}
												</div>
											</div>

											<div>
												<h3 class="mb-2 text-sm font-medium">Assignments</h3>
												<Table.Table>
													<Table.TableHeader>
														<Table.TableRow>
															<Table.TableHead>
																{#if run.run_type === "counselor_cabin"}
																	Counselor
																{:else if run.run_type === "camper_cabin"}
																	Camper
																{:else}
																	Counselor
																{/if}
															</Table.TableHead>
															<Table.TableHead>
																{#if run.run_type === "activity_schedule"}
																	Activity
																{:else}
																	Cabin
																{/if}
															</Table.TableHead>
														</Table.TableRow>
													</Table.TableHeader>
													<Table.TableBody>
														{#each details.assignments as assignment}
															<Table.TableRow>
																<Table.TableCell>
																	{#if run.run_type === "counselor_cabin"}
																		{assignment.counselor_id ?? "Unknown"}
																	{:else if run.run_type === "camper_cabin"}
																		{assignment.camper_id ?? "Unknown"}
																	{:else}
																		{assignment.counselor_id ?? "Unknown"}
																	{/if}
																</Table.TableCell>
																<Table.TableCell>
																	{#if run.run_type === "activity_schedule"}
																		{assignment.session_activity_id ?? "Unknown"}
																	{:else}
																		{assignment.cabin_id ?? "Unknown"}
																	{/if}
																</Table.TableCell>
															</Table.TableRow>
														{/each}
													</Table.TableBody>
												</Table.Table>
											</div>

											{#if details.explanations.length > 0}
												<div>
													<h3 class="mb-2 text-sm font-medium">Explanations</h3>
													<div class="grid gap-2">
														{#each details.explanations as explanation}
															<div
																class="rounded-lg border p-3 {explanation.explanation_type === 'unmet_preference'
																	? 'bg-amber-50 dark:bg-amber-950/20'
																	: 'bg-muted'}"
															>
																<div class="flex items-center gap-2">
																	{#if explanation.explanation_type === "reason"}
																		<CheckIcon class="text-green-600 size-4" />
																	{:else}
																		<TriangleAlertIcon class="text-amber-600 size-4" />
																	{/if}
																	<span class="text-sm">{explanation.message}</span>
																</div>
															</div>
														{/each}
													</div>
												</div>
											{/if}
										</div>
									{:else}
										<div class="text-muted-foreground py-4 text-center text-sm">
											Failed to load solution details
										</div>
									{/if}
								{/if}
							</div>
						{/if}
					</Card.Card>
				{/each}
			</div>

			{#if run.status !== "selected"}
			<div class="flex justify-end">
				<Button
					variant="destructive"
					size="sm"
					onclick={confirmDelete}
				>
					<TrashIcon class="mr-2 size-4" />
					Delete Run
				</Button>
			</div>
			{/if}
		</div>
	{/if}
</div>

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
