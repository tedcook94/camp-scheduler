<script lang="ts">
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import { ApiClientError } from "$lib/api/client";
	import { assignmentApi } from "$lib/api";
	import type {
		RunDetailResponse,
		SolutionDetailResponse,
		ExplanationDetail,
		AssignmentDetail,
	} from "$lib/api/types";
	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
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
	import BanIcon from "@lucide/svelte/icons/ban";

	type RunType = "counselor_cabin" | "camper_cabin" | "activity_schedule";

	const RUN_TYPE_LABELS: Record<RunType, string> = {
		counselor_cabin: "Counselor Cabin",
		camper_cabin: "Camper Cabin",
		activity_schedule: "Activity Schedule",
	};

	const constraintNameOverrides: Record<string, string> = {
		cocounselor_preference: "Co-counselor Preference",
	};

	function formatConstraintName(name: string | null): string {
		if (!name) return "";
		if (constraintNameOverrides[name]) return constraintNameOverrides[name];
		return name
			.split("_")
			.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
			.join(" ");
	}

	function entityLabel(explanation: ExplanationDetail, runType: RunType): string {
		if (runType === "camper_cabin") {
			return explanation.camper_name || explanation.camper_id?.slice(0, 8) || "";
		}
		return explanation.counselor_name || explanation.counselor_id?.slice(0, 8) || "";
	}

	interface ActivityGroup {
		timeSlotName: string;
		activities: { activityName: string; counselors: string[] }[];
	}

	interface CabinGroup {
		ageGroupName: string;
		cabins: { cabinName: string; people: string[] }[];
	}

	function groupByTimeSlotActivity(assignments: AssignmentDetail[]): ActivityGroup[] {
		const groups: ActivityGroup[] = [];
		let currentSlot: ActivityGroup | null = null;
		let currentActivity: { activityName: string; counselors: string[] } | null = null;

		for (const a of assignments) {
			const slotName = a.time_slot_name || "Unknown";
			const actName = a.activity_name || "Unknown";
			const person = a.counselor_name || a.counselor_id?.slice(0, 8) || "Unknown";

			if (!currentSlot || currentSlot.timeSlotName !== slotName) {
				currentSlot = { timeSlotName: slotName, activities: [] };
				currentActivity = null;
				groups.push(currentSlot);
			}

			if (!currentActivity || currentActivity.activityName !== actName) {
				currentActivity = { activityName: actName, counselors: [] };
				currentSlot.activities.push(currentActivity);
			}

			currentActivity.counselors.push(person);
		}

		return groups;
	}

	function groupByAgeGroupCabin(assignments: AssignmentDetail[]): CabinGroup[] {
		const groups: CabinGroup[] = [];
		let currentGroup: CabinGroup | null = null;
		let currentCabin: { cabinName: string; people: string[] } | null = null;

		for (const a of assignments) {
			const groupName = a.age_group_name || "Unknown";
			const cabName = a.cabin_name || "Unknown";
			const person =
				a.camper_name || a.counselor_name ||
				a.camper_id?.slice(0, 8) || a.counselor_id?.slice(0, 8) ||
				"Unknown";

			if (!currentGroup || currentGroup.ageGroupName !== groupName) {
				currentGroup = { ageGroupName: groupName, cabins: [] };
				currentCabin = null;
				groups.push(currentGroup);
			}

			if (!currentCabin || currentCabin.cabinName !== cabName) {
				currentCabin = { cabinName: cabName, people: [] };
				currentGroup.cabins.push(currentCabin);
			}

			currentCabin.people.push(person);
		}

		return groups;
	}

	const runId = $derived($page.params.id);

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let run = $state<RunDetailResponse | null>(null);
	let sessionId = $state<string | null>(null);

	let expandedSolutions = $state<Set<string>>(new Set());
	let solutionDetails = $state<Map<string, SolutionDetailResponse>>(new Map());
	let loadingSolution = $state<string | null>(null);

	let expandedMet = $state<Set<string>>(new Set());
	let expandedUnmet = $state<Set<string>>(new Set());
	let expandedIneligible = $state<Set<string>>(new Set());

	function toggleMet(solutionId: string) {
		const next = new Set(expandedMet);
		if (next.has(solutionId)) next.delete(solutionId);
		else next.add(solutionId);
		expandedMet = next;
	}

	function toggleUnmet(solutionId: string) {
		const next = new Set(expandedUnmet);
		if (next.has(solutionId)) next.delete(solutionId);
		else next.add(solutionId);
		expandedUnmet = next;
	}

	function toggleIneligible(solutionId: string) {
		const next = new Set(expandedIneligible);
		if (next.has(solutionId)) next.delete(solutionId);
		else next.add(solutionId);
		expandedIneligible = next;
	}

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
			await loadRun();
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
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold">Solutions</h2>
					<Button
						variant="destructive"
						size="sm"
						onclick={confirmDelete}
						disabled={deleting}
					>
						<TrashIcon class="mr-2 size-4" />
						Delete Run
					</Button>
				</div>
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
								{#if run.selected_solution_id === solution.id}
									<Badge variant="default" class="bg-green-600">
										<CheckIcon class="mr-1 size-3" />
										Selected
									</Badge>
								{:else}
									<Button
										variant="outline"
										size="sm"
										disabled={selecting !== null}
										onclick={(e) => {
											e.stopPropagation();
											handleSelectSolution(solution.id);
										}}
									>
										{#if selecting === solution.id}
											<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
										{/if}
										Select
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
												<h3 class="mb-3 text-sm font-medium">Assignments</h3>
												{#if run.run_type === "activity_schedule"}
													{@const activityGroups = groupByTimeSlotActivity(details.assignments)}
													<div class="grid gap-4">
														{#each activityGroups as slotGroup}
															<div>
																<h4 class="mb-2 border-b pb-1 text-sm font-semibold">{slotGroup.timeSlotName}</h4>
																<div class="grid grid-cols-1 gap-3 pl-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
																	{#each slotGroup.activities as actGroup}
																		<div class="rounded-lg border bg-muted/30 p-3">
																			<h5 class="mb-1 text-sm font-medium text-muted-foreground">{actGroup.activityName}</h5>
																			<div class="grid gap-0.5">
																				{#each actGroup.counselors as counselor}
																					<div class="text-sm">{counselor}</div>
																				{/each}
																			</div>
																		</div>
																	{/each}
																</div>
															</div>
														{/each}
													</div>
												{:else}
													{@const cabinGroups = groupByAgeGroupCabin(details.assignments)}
													<div class="grid gap-4">
														{#each cabinGroups as ageGroup}
															<div>
																<h4 class="mb-2 border-b pb-1 text-sm font-semibold">{ageGroup.ageGroupName}</h4>
																<div class="grid grid-cols-1 gap-3 pl-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
																	{#each ageGroup.cabins as cabin}
																		<div class="rounded-lg border bg-muted/30 p-3">
																			<h5 class="mb-1 text-sm font-medium text-muted-foreground">{cabin.cabinName}</h5>
																			<div class="grid gap-0.5">
																				{#each cabin.people as person}
																					<div class="text-sm">{person}</div>
																				{/each}
																			</div>
																		</div>
																	{/each}
																</div>
															</div>
														{/each}
													</div>
												{/if}
											</div>

											{#if details.explanations.length > 0}
												{@const metPreferences = details.explanations.filter((e) => e.explanation_type === "reason")}
												{@const unmetPreferences = details.explanations.filter((e) => e.explanation_type === "unmet_preference")}
												{@const ineligiblePreferences = details.explanations.filter((e) => e.explanation_type === "ineligible_preference")}

												{#if metPreferences.length > 0}
													<div>
														<button
															type="button"
															class="mb-2 flex w-full items-center gap-2 text-left text-sm font-medium text-green-700 hover:opacity-80 dark:text-green-400"
															aria-expanded={expandedMet.has(solution.id)}
															onclick={() => toggleMet(solution.id)}
														>
															{#if expandedMet.has(solution.id)}
																<ChevronDownIcon class="size-4" />
															{:else}
																<ChevronRightIcon class="size-4" />
															{/if}
															Met Preferences ({metPreferences.length})
														</button>
														{#if expandedMet.has(solution.id)}
															<div class="grid gap-2">
																{#each metPreferences as explanation}
																	<div class="rounded-lg border bg-muted p-3">
																		<div class="flex items-start gap-2">
																			<CheckIcon class="mt-0.5 size-4 shrink-0 text-green-600" />
																			<div>
																				{#if entityLabel(explanation, run.run_type)}
																					<span class="font-medium">{entityLabel(explanation, run.run_type)}</span>
																					{#if explanation.constraint_name}
																						<span class="text-muted-foreground"> — {formatConstraintName(explanation.constraint_name)}</span>
																					{/if}
																					<br />
																				{/if}
																				<span class="text-sm">{explanation.message}</span>
																			</div>
																		</div>
																	</div>
																{/each}
															</div>
														{/if}
													</div>
												{/if}

												{#if unmetPreferences.length > 0}
													<div>
														<button
															type="button"
															class="mb-2 flex w-full items-center gap-2 text-left text-sm font-medium text-amber-700 hover:opacity-80 dark:text-amber-400"
															aria-expanded={expandedUnmet.has(solution.id)}
															onclick={() => toggleUnmet(solution.id)}
														>
															{#if expandedUnmet.has(solution.id)}
																<ChevronDownIcon class="size-4" />
															{:else}
																<ChevronRightIcon class="size-4" />
															{/if}
															Unmet Preferences ({unmetPreferences.length})
														</button>
														{#if expandedUnmet.has(solution.id)}
															<div class="grid gap-2">
																{#each unmetPreferences as explanation}
																	<div class="rounded-lg border bg-amber-50 p-3 dark:bg-amber-950/20">
																		<div class="flex items-start gap-2">
																			<TriangleAlertIcon class="mt-0.5 size-4 shrink-0 text-amber-600" />
																			<div>
																				{#if entityLabel(explanation, run.run_type)}
																					<span class="font-medium">{entityLabel(explanation, run.run_type)}</span>
																					{#if explanation.constraint_name}
																						<span class="text-muted-foreground"> — {formatConstraintName(explanation.constraint_name)}</span>
																					{/if}
																					<br />
																				{/if}
																				<span class="text-sm">{explanation.message}</span>
																			</div>
																		</div>
																	</div>
																{/each}
															</div>
														{/if}
													</div>
												{/if}

												{#if ineligiblePreferences.length > 0}
													<div>
														<button
															type="button"
															class="mb-2 flex w-full items-center gap-2 text-left text-sm font-medium text-red-700 hover:opacity-80 dark:text-red-400"
															aria-expanded={expandedIneligible.has(solution.id)}
															onclick={() => toggleIneligible(solution.id)}
														>
															{#if expandedIneligible.has(solution.id)}
																<ChevronDownIcon class="size-4" />
															{:else}
																<ChevronRightIcon class="size-4" />
															{/if}
															Cannot Satisfy ({ineligiblePreferences.length})
														</button>
														{#if expandedIneligible.has(solution.id)}
															<div class="grid gap-2">
																{#each ineligiblePreferences as explanation}
																	<div class="rounded-lg border bg-red-50 p-3 dark:bg-red-950/20">
																		<div class="flex items-start gap-2">
																			<BanIcon class="mt-0.5 size-4 shrink-0 text-red-600" />
																			<div>
																				{#if entityLabel(explanation, run.run_type)}
																					<span class="font-medium">{entityLabel(explanation, run.run_type)}</span>
																					{#if explanation.constraint_name}
																						<span class="text-muted-foreground"> — {formatConstraintName(explanation.constraint_name)}</span>
																					{/if}
																					<br />
																				{/if}
																				<span class="text-sm">{explanation.message}</span>
																			</div>
																		</div>
																	</div>
																{/each}
															</div>
														{/if}
													</div>
												{/if}
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