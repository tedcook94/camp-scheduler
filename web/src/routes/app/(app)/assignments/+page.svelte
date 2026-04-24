<script lang="ts">
	import { onMount } from "svelte";
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import { ApiClientError } from "$lib/api/client";
	import { assignmentApi, sessionApi, seasonApi } from "$lib/api";
	import {
		groupByTimeSlotActivity,
		groupByAgeGroupCabin,
	} from "$lib/report-grouping";
	import type {
		RunResponse,
		RunDetailResponse,
		RunType,
		Session,
		Season,
		SolutionResponse,
		SolutionDetailResponse,
		ExplanationDetail,
	} from "$lib/api/types";
	import { Button } from "$lib/components/ui/button";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Card from "$lib/components/ui/card";
	import * as Select from "$lib/components/ui/select";
	import { Badge } from "$lib/components/ui/badge";
	import { Separator } from "$lib/components/ui/separator";
	import PlayIcon from "@lucide/svelte/icons/play";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import CheckIcon from "@lucide/svelte/icons/check";
	import TentTreeIcon from "@lucide/svelte/icons/tent-tree";
	import DumbbellIcon from "@lucide/svelte/icons/dumbbell";
	import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
	import BanIcon from "@lucide/svelte/icons/ban";

	interface CardSpec {
		runType: RunType;
		title: string;
		description: string;
		icon: typeof TentTreeIcon;
	}

	const CARDS: CardSpec[] = [
		{
			runType: "cabin",
			title: "Cabin Assignments",
			description: "Assign counselors and campers to cabins.",
			icon: TentTreeIcon,
		},
		{
			runType: "activity_schedule",
			title: "Activity Assignments",
			description: "Assign counselors to activity time slots.",
			icon: DumbbellIcon,
		},
	];

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

	type Population = "counselor" | "camper";

	const seenMissingEntity = new Set<string>();

	function entityLabel(explanation: ExplanationDetail, pop: Population): string {
		if (pop === "camper") {
			const label =
				explanation.camper_name || explanation.camper_id?.slice(0, 8) || "";
			if (!label) {
				const key = `camper:${explanation.id}`;
				if (!seenMissingEntity.has(key)) {
					seenMissingEntity.add(key);
					console.warn(
						"camper explanation missing both camper_name and camper_id",
						explanation,
					);
				}
			}
			return label;
		}
		const label =
			explanation.counselor_name || explanation.counselor_id?.slice(0, 8) || "";
		if (!label) {
			const key = `counselor:${explanation.id}`;
			if (!seenMissingEntity.has(key)) {
				seenMissingEntity.add(key);
				console.warn(
					"counselor explanation missing both counselor_name and counselor_id",
					explanation,
				);
			}
		}
		return label;
	}

	function isCounselorExplanation(e: ExplanationDetail): boolean {
		return Boolean(e.counselor_id || e.counselor_name);
	}

	function isCamperExplanation(e: ExplanationDetail): boolean {
		return Boolean(e.camper_id || e.camper_name);
	}

	function sortSolutions(detail: RunDetailResponse): SolutionResponse[] {
		const selectedId = detail.selected_solution_id;
		if (!selectedId) return [...detail.solutions];
		const selected = detail.solutions.find((s) => s.id === selectedId);
		if (!selected) return [...detail.solutions];
		return [selected, ...detail.solutions.filter((s) => s.id !== selectedId)];
	}

	let loading = $state(false);
	let loadError = $state<string | null>(null);
	let runs = $state<RunResponse[]>([]);
	let sessions = $state<Session[]>([]);
	let seasons = $state<Season[]>([]);
	let selectedSessionId = $state<string | null>(null);
	let sessionsLoaded = $state(false);

	let expandedCards = $state<Set<RunType>>(new Set());
	let runDetails = $state<Map<string, RunDetailResponse>>(new Map());
	let loadingRuns = $state<Set<string>>(new Set());

	let expandedSolutions = $state<Set<string>>(new Set());
	let solutionDetails = $state<Map<string, SolutionDetailResponse>>(new Map());
	let loadingSolution = $state<string | null>(null);
	let selecting = $state<string | null>(null);

	let expandedMet = $state<Set<string>>(new Set());
	let expandedUnmet = $state<Set<string>>(new Set());
	let expandedIneligible = $state<Set<string>>(new Set());

	let triggering = $state<RunType | null>(null);
	let confirmRunOpen = $state(false);
	let pendingRunType = $state<RunType | null>(null);

	let deleteOpen = $state(false);
	let deleting = $state(false);
	let runToDelete = $state<RunResponse | null>(null);

	let errorDialogOpen = $state(false);
	let errorDialogMessage = $state("");
	type CamperShortage = {
		age_group_id: string;
		age_group_name: string;
		gender: string;
		count: number;
		reason: "no_matching_cabin" | "over_capacity" | string;
	};
	let errorDialogShortages = $state<CamperShortage[]>([]);

	let runsByType = $derived.by(() => {
		const map = new Map<RunType, RunResponse | null>();
		for (const c of CARDS) map.set(c.runType, null);
		for (const run of runs) map.set(run.run_type, run);
		return map;
	});

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
		if (!sessionsLoaded) return;
		const sessionIdParam = $page.url.searchParams.get("session");
		if (!sessionIdParam) {
			selectedSessionId = null;
			resetRunState();
			return;
		}
		const exists = sessions.some((s) => s.id === sessionIdParam);
		if (!exists) {
			selectedSessionId = null;
			resetRunState();
			return;
		}
		if (sessionIdParam !== selectedSessionId) {
			selectedSessionId = sessionIdParam;
			resetRunState();
			loadRuns();
		}
	});

	function resetRunState() {
		runs = [];
		loading = false;
		loadError = null;
		expandedCards = new Set();
		runDetails = new Map();
		loadingRuns = new Set();
		expandedSolutions = new Set();
		solutionDetails = new Map();
		expandedMet = new Set();
		expandedUnmet = new Set();
		expandedIneligible = new Set();
	}

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
		const url = new URL($page.url);
		url.searchParams.set("session", value);
		goto(url, { replaceState: true, keepFocus: true, noScroll: true });
		resetRunState();
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

	async function fetchRunDetails(runId: string) {
		if (!selectedSessionId || runDetails.has(runId)) return;
		loadingRuns = new Set(loadingRuns).add(runId);
		try {
			const detail = await assignmentApi.getRun(selectedSessionId, runId);
			runDetails = new Map(runDetails).set(runId, detail);
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to load run";
			toast.error(message);
		} finally {
			const next = new Set(loadingRuns);
			next.delete(runId);
			loadingRuns = next;
		}
	}

	function toggleCard(runType: RunType) {
		const next = new Set(expandedCards);
		const isExpanding = !next.has(runType);
		if (isExpanding) {
			next.add(runType);
			const run = runsByType.get(runType);
			if (run) void fetchRunDetails(run.id);
		} else {
			next.delete(runType);
		}
		expandedCards = next;
	}

	function startRun(runType: RunType) {
		if (!selectedSessionId) return;
		const existing = runsByType.get(runType);
		if (existing) {
			pendingRunType = runType;
			confirmRunOpen = true;
		} else {
			void triggerRun(runType);
		}
	}

	async function triggerRun(runType: RunType) {
		if (!selectedSessionId) return;
		triggering = runType;
		try {
			const result = await assignmentApi.triggerRun(selectedSessionId, {
				run_type: runType,
			});
			toast.success("Solver run completed");
			// Replace cached state for this run type and auto-expand the card.
			runs = [result, ...runs.filter((r) => r.run_type !== runType)];
			runDetails = new Map(runDetails).set(result.id, result);
			// Drop any stale solution details (old run is gone).
			solutionDetails = new Map();
			expandedSolutions = new Set();
			expandedMet = new Set();
			expandedUnmet = new Set();
			expandedIneligible = new Set();
			expandedCards = new Set(expandedCards).add(runType);
		} catch (err) {
			if (err instanceof ApiClientError && err.status === 422) {
				errorDialogMessage = err.message;
				const raw = err.data?.camper_shortages;
				errorDialogShortages = Array.isArray(raw) ? (raw as CamperShortage[]) : [];
				errorDialogOpen = true;
			} else {
				const message =
					err instanceof ApiClientError ? err.message : "Failed to trigger solver";
				toast.error(message);
			}
		} finally {
			triggering = null;
		}
	}

	function confirmDelete(run: RunResponse) {
		runToDelete = run;
		deleteOpen = true;
	}

	async function handleDelete() {
		if (!runToDelete || !selectedSessionId) return;
		const targetId = runToDelete.id;
		deleting = true;
		try {
			await assignmentApi.deleteRun(selectedSessionId, targetId);
			runs = runs.filter((r) => r.id !== targetId);
			const nextDetails = new Map(runDetails);
			nextDetails.delete(targetId);
			runDetails = nextDetails;
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

	async function toggleSolution(runId: string, solutionId: string) {
		if (expandedSolutions.has(solutionId)) {
			const next = new Set(expandedSolutions);
			next.delete(solutionId);
			expandedSolutions = next;
			return;
		}
		if (solutionDetails.has(solutionId)) {
			expandedSolutions = new Set(expandedSolutions).add(solutionId);
			return;
		}
		if (!selectedSessionId) return;

		loadingSolution = solutionId;
		try {
			const details = await assignmentApi.getSolution(selectedSessionId, runId, solutionId);
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

	async function handleSelectSolution(runId: string, solutionId: string) {
		if (!selectedSessionId) return;
		selecting = solutionId;
		try {
			await assignmentApi.selectSolution(selectedSessionId, runId, solutionId);
			const fresh = await assignmentApi.getRun(selectedSessionId, runId);
			runDetails = new Map(runDetails).set(runId, fresh);
			runs = runs.map((r) => (r.id === runId ? { ...r, status: fresh.status, selected_solution_id: fresh.selected_solution_id } : r));
			toast.success("Solution selected");
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : "Failed to select solution";
			toast.error(message);
		} finally {
			selecting = null;
		}
	}

	function toggleSet(s: Set<string>, key: string): Set<string> {
		const next = new Set(s);
		if (next.has(key)) next.delete(key);
		else next.add(key);
		return next;
	}
</script>

{#snippet explanationGroups(
	solutionId: string,
	explanations: ExplanationDetail[],
	pop: Population,
	heading: string | null,
)}
	{@const met = explanations.filter((e) => e.explanation_type === "reason")}
	{@const unmet = explanations.filter((e) => e.explanation_type === "unmet_preference")}
	{@const ineligible = explanations.filter((e) => e.explanation_type === "ineligible_preference")}
	{@const metKey = `${solutionId}:${pop}:met`}
	{@const unmetKey = `${solutionId}:${pop}:unmet`}
	{@const ineligibleKey = `${solutionId}:${pop}:ineligible`}

	{#if met.length + unmet.length + ineligible.length > 0}
		<div class="grid gap-3">
			{#if heading}
				<h4 class="text-sm font-semibold">{heading}</h4>
			{/if}

			{#if met.length > 0}
				<div>
					<button
						type="button"
						class="mb-2 flex w-full items-center gap-2 text-left text-sm font-medium text-green-700 hover:opacity-80 dark:text-green-400"
						aria-expanded={expandedMet.has(metKey)}
						onclick={() => (expandedMet = toggleSet(expandedMet, metKey))}
					>
						{#if expandedMet.has(metKey)}
							<ChevronDownIcon class="size-4" />
						{:else}
							<ChevronRightIcon class="size-4" />
						{/if}
						Met Preferences ({met.length})
					</button>
					{#if expandedMet.has(metKey)}
						<div class="grid gap-2">
							{#each met as explanation}
								<div class="rounded-lg border bg-muted p-3">
									<div class="flex items-start gap-2">
										<CheckIcon class="mt-0.5 size-4 shrink-0 text-green-600" />
										<div>
											{#if entityLabel(explanation, pop)}
												<span class="font-medium">{entityLabel(explanation, pop)}</span>
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

			{#if unmet.length > 0}
				<div>
					<button
						type="button"
						class="mb-2 flex w-full items-center gap-2 text-left text-sm font-medium text-amber-700 hover:opacity-80 dark:text-amber-400"
						aria-expanded={expandedUnmet.has(unmetKey)}
						onclick={() => (expandedUnmet = toggleSet(expandedUnmet, unmetKey))}
					>
						{#if expandedUnmet.has(unmetKey)}
							<ChevronDownIcon class="size-4" />
						{:else}
							<ChevronRightIcon class="size-4" />
						{/if}
						Unmet Preferences ({unmet.length})
					</button>
					{#if expandedUnmet.has(unmetKey)}
						<div class="grid gap-2">
							{#each unmet as explanation}
								<div class="rounded-lg border bg-amber-50 p-3 dark:bg-amber-950/20">
									<div class="flex items-start gap-2">
										<TriangleAlertIcon class="mt-0.5 size-4 shrink-0 text-amber-600" />
										<div>
											{#if entityLabel(explanation, pop)}
												<span class="font-medium">{entityLabel(explanation, pop)}</span>
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

			{#if ineligible.length > 0}
				<div>
					<button
						type="button"
						class="mb-2 flex w-full items-center gap-2 text-left text-sm font-medium text-red-700 hover:opacity-80 dark:text-red-400"
						aria-expanded={expandedIneligible.has(ineligibleKey)}
						onclick={() => (expandedIneligible = toggleSet(expandedIneligible, ineligibleKey))}
					>
						{#if expandedIneligible.has(ineligibleKey)}
							<ChevronDownIcon class="size-4" />
						{:else}
							<ChevronRightIcon class="size-4" />
						{/if}
						Cannot Satisfy ({ineligible.length})
					</button>
					{#if expandedIneligible.has(ineligibleKey)}
						<div class="grid gap-2">
							{#each ineligible as explanation}
								<div class="rounded-lg border bg-red-50 p-3 dark:bg-red-950/20">
									<div class="flex items-start gap-2">
										<BanIcon class="mt-0.5 size-4 shrink-0 text-red-600" />
										<div>
											{#if entityLabel(explanation, pop)}
												<span class="font-medium">{entityLabel(explanation, pop)}</span>
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
		</div>
	{/if}
{/snippet}

{#snippet solutionCard(runType: RunType, runId: string, solution: SolutionResponse, isSelected: boolean)}
	<Card.Card>
		<div
			class="flex w-full cursor-pointer items-center justify-between bg-transparent p-4 text-left hover:bg-muted/50"
			role="button"
			tabindex="0"
			aria-expanded={expandedSolutions.has(solution.id)}
			onclick={() => toggleSolution(runId, solution.id)}
			onkeydown={(e) => {
				if (e.key === "Enter" || e.key === " ") {
					e.preventDefault();
					toggleSolution(runId, solution.id);
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
					<div class="font-medium">Solution {solution.solution_index + 1}</div>
					<div class="text-muted-foreground text-sm">
						Score: {solution.score.toFixed(2)}
					</div>
				</div>
			</div>
			<div class="flex items-center gap-2">
				{#if isSelected}
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
							handleSelectSolution(runId, solution.id);
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
								{#if runType === "activity_schedule"}
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
															<h5 class="mb-2 text-sm font-medium text-muted-foreground">{cabin.cabinName}</h5>
															{#if cabin.counselors.length > 0}
																<div class="text-muted-foreground text-xs uppercase tracking-wide">Counselors</div>
																<div class="mb-2 grid gap-0.5">
																	{#each cabin.counselors as counselor}
																		<div class="text-sm font-medium">{counselor}</div>
																	{/each}
																</div>
															{/if}
															{#if cabin.campers.length > 0}
																<div class="text-muted-foreground text-xs uppercase tracking-wide">Campers</div>
																<div class="grid gap-0.5">
																	{#each cabin.campers as camper}
																		<div class="text-sm">{camper}</div>
																	{/each}
																</div>
															{/if}
														</div>
													{/each}
												</div>
											</div>
										{/each}
									</div>
								{/if}
							</div>

							{#if details.unassigned_counselors && details.unassigned_counselors.length > 0}
								<div class="rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950/30">
									<h3 class="mb-2 text-sm font-medium text-amber-900 dark:text-amber-200">Unassigned counselors</h3>
									{#if runType === "cabin"}
										<p class="mb-2 text-xs text-amber-800 dark:text-amber-300">These counselors were not assigned to any cabin.</p>
										<ul class="grid gap-1 pl-2">
											{#each details.unassigned_counselors as uc}
												<li class="text-sm">{uc.counselor_name}</li>
											{/each}
										</ul>
									{:else}
										<p class="mb-2 text-xs text-amber-800 dark:text-amber-300">These counselors were not assigned for one or more time slots.</p>
										<div class="grid gap-2 pl-2">
											{#each details.unassigned_counselors as uc}
												<div>
													<div class="text-sm font-medium">{uc.counselor_name}</div>
													{#if uc.missing_time_slots && uc.missing_time_slots.length > 0}
														<div class="pl-3 text-xs text-muted-foreground">
															Missing: {uc.missing_time_slots.map((s) => s.time_slot_name).join(", ")}
														</div>
													{/if}
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{/if}

							{#if details.explanations.length > 0}
								{#if runType === "cabin"}
									{@const counselorExplanations = details.explanations.filter(isCounselorExplanation)}
									{@const camperExplanations = details.explanations.filter(isCamperExplanation)}
									{@render explanationGroups(solution.id, counselorExplanations, "counselor", "Counselor preferences")}
									{@render explanationGroups(solution.id, camperExplanations, "camper", "Camper preferences")}
								{:else}
									{@render explanationGroups(solution.id, details.explanations, "counselor", null)}
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
{/snippet}

<div class="grid gap-6">
	<div class="flex items-start justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Assignments</h1>
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
	{:else}
		<div class="grid gap-4">
			{#each CARDS as card (card.runType)}
				{@const Icon = card.icon}
				{@const run = runsByType.get(card.runType) ?? null}
				{@const isExpanded = expandedCards.has(card.runType)}
				{@const isTriggering = triggering === card.runType}
				{@const detail = run ? runDetails.get(run.id) : null}
				{@const isLoadingRun = run ? loadingRuns.has(run.id) : false}
				<div class="border-border overflow-hidden rounded-lg border">
					<div class="bg-muted hover:bg-muted/80 flex w-full items-center">
						<button
							type="button"
							class="flex flex-1 items-center gap-2 px-4 py-3 text-left"
							onclick={() => toggleCard(card.runType)}
						>
							<ChevronDownIcon
								class="size-4 transition-transform {isExpanded ? '' : '-rotate-90'}"
							/>
							<Icon class="size-4" />
							<h3 class="font-medium">{card.title}</h3>
							<span class="text-muted-foreground text-sm">
								{#if run}
									— {run.status === "selected" ? "selected" : "completed"} {new Date(run.created_at).toLocaleDateString()}
								{:else}
									— no runs yet
								{/if}
							</span>
							{#if run?.status !== "selected"}
								<span
									title="No assignments selected"
									class="ml-auto inline-flex items-center text-amber-600 dark:text-amber-400"
								>
									<TriangleAlertIcon class="size-5" />
									<span class="sr-only">No assignments selected</span>
								</span>
							{/if}
						</button>
						<Button
							size="sm"
							class="mr-4"
							disabled={isTriggering || triggering !== null}
							onclick={() => startRun(card.runType)}
						>
							{#if isTriggering}
								<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
								Running...
							{:else}
								<PlayIcon class="mr-2 size-4" />
								Run
							{/if}
						</Button>
					</div>

					{#if isExpanded}
						<div class="bg-background px-4 py-3">
							<p class="text-muted-foreground mb-3 text-sm">{card.description}</p>
							{#if !run}
								<p class="text-muted-foreground py-2 text-sm">
									No runs yet — click Run to start.
								</p>
							{:else if isLoadingRun || !detail}
								<div class="text-muted-foreground py-4 text-center text-sm">
									Loading solutions...
								</div>
							{:else}
								<div class="grid gap-4">
									<div class="flex items-center justify-between gap-3">
										<div class="text-muted-foreground text-sm">
											{detail.solutions.length} solution{detail.solutions.length === 1 ? "" : "s"} · created {new Date(detail.created_at).toLocaleString()}
										</div>
										<Button
											variant="outline"
											size="sm"
											onclick={() => confirmDelete(run)}
										>
											<TrashIcon class="mr-2 size-4" />
											Delete Run
										</Button>
									</div>
									{#if detail.solutions.length === 0}
										<p class="text-muted-foreground py-2 text-sm">
											The solver produced no solutions for this run.
										</p>
									{:else}
										<div class="grid gap-3">
											{#each sortSolutions(detail) as solution (solution.id)}
												{@render solutionCard(card.runType, run.id, solution, detail.selected_solution_id === solution.id)}
											{/each}
										</div>
									{/if}
								</div>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<AlertDialog.AlertDialog bind:open={confirmRunOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Replace existing run?</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				A run for this session already exists. Triggering a new run will
				replace it and discard its solutions{runsByType.get(pendingRunType ?? "cabin")?.status === "selected"
					? ", including the currently selected solution."
					: "."}
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={triggering !== null}>
				Cancel
			</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				disabled={triggering !== null}
				onclick={() => {
					const rt = pendingRunType;
					confirmRunOpen = false;
					if (rt) void triggerRun(rt);
				}}
			>
				<PlayIcon class="mr-2 size-4" />
				Replace and Run
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>

<AlertDialog.AlertDialog bind:open={errorDialogOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Solver Run Failed</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				{errorDialogMessage}
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		{#if errorDialogShortages.length > 0}
			<div class="max-h-60 overflow-y-auto rounded-md border bg-muted/30 p-3">
				<div class="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Camper shortages</div>
				<ul class="grid gap-0.5">
					{#each errorDialogShortages as s}
						<li class="text-sm">
							{s.age_group_name} ({s.gender}) —
							{#if s.reason === "no_matching_cabin"}
								no matching cabin ({s.count} camper{s.count === 1 ? "" : "s"})
							{:else if s.reason === "over_capacity"}
								over capacity by {s.count}
							{:else}
								{s.count} camper{s.count === 1 ? "" : "s"}
							{/if}
						</li>
					{/each}
				</ul>
			</div>
		{/if}
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogAction onclick={() => (errorDialogOpen = false)}>
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
				{#if runToDelete?.status === "selected"}
					<span class="block font-medium text-amber-600 dark:text-amber-400">
						This is the currently selected solution. Deleting it will clear that selection.
					</span>
					<span class="mt-2 block">
						Are you sure you want to delete this assignment run? This action cannot be undone.
					</span>
				{:else}
					Are you sure you want to delete this assignment run? This action cannot be
					undone.
				{/if}
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
