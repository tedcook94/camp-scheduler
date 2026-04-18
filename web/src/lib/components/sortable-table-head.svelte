<script lang="ts">
	import * as Table from "$lib/components/ui/table";
	import type { SortDirection } from "$lib/utils";
	import ChevronUpIcon from "@lucide/svelte/icons/chevron-up";
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";

	interface Props {
		label: string;
		active: boolean;
		direction: SortDirection;
		onclick: () => void;
	}

	let { label, active, direction, onclick }: Props = $props();

	const ariaSort = $derived(
		active ? (direction === "asc" ? "ascending" : "descending") : "none"
	);
	const sortButtonLabel = $derived(
		active
			? `${label}, sorted ${direction === "asc" ? "ascending" : "descending"}. Activate to sort ${direction === "asc" ? "descending" : "ascending"}.`
			: `Sort by ${label}`
	);
</script>

<Table.TableHead aria-sort={ariaSort}>
	<button
		type="button"
		class="inline-flex items-center gap-1 hover:text-foreground"
		aria-label={sortButtonLabel}
		{onclick}
	>
		{label}
		{#if active}
			{#if direction === "asc"}
				<ChevronUpIcon class="size-3.5" />
			{:else}
				<ChevronDownIcon class="size-3.5" />
			{/if}
		{:else}
			<ChevronsUpDownIcon class="text-muted-foreground/50 size-3.5" />
		{/if}
	</button>
</Table.TableHead>
