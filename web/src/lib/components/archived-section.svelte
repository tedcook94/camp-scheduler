<script lang="ts" generics="T extends { id: string }">
	import { ApiClientError } from "$lib/api/client";
	import { toast } from "svelte-sonner";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import type { Snippet } from "svelte";

	interface Props {
		resourceName: string;
		resourceNamePlural?: string;
		listArchivedFn: () => Promise<T[]>;
		unarchiveFn: (id: string) => Promise<void>;
		row: Snippet<[{ item: T; unarchive: () => Promise<void>; busy: boolean }]>;
		header?: Snippet;
	}

	let {
		resourceName,
		resourceNamePlural = `${resourceName.toLowerCase()}s`,
		listArchivedFn,
		unarchiveFn,
		row,
		header,
	}: Props = $props();

	let open = $state(false);
	let loaded = $state(false);
	let loading = $state(false);
	let items = $state<T[]>([]);
	let busyId = $state<string | null>(null);
	let loadPromise: Promise<void> | null = null;

	// Stable per-instance id so the disclosure button can reference its panel
	// via aria-controls.
	const uid = $props.id();
	const panelId = `archived-panel-${uid}`;

	async function toggle() {
		if (open) {
			open = false;
			return;
		}
		open = true;
		if (loaded) return;
		// Coalesce concurrent toggles into a single in-flight fetch so closing
		// and re-opening quickly does not trigger duplicate requests.
		if (!loadPromise) {
			loading = true;
			loadPromise = (async () => {
				try {
					items = await listArchivedFn();
					loaded = true;
				} catch (err) {
					const message =
						err instanceof ApiClientError
							? err.message
							: `Failed to load archived ${resourceNamePlural.toLowerCase()}`;
					toast.error(message);
				} finally {
					loading = false;
					loadPromise = null;
				}
			})();
		}
		await loadPromise;
	}

	async function handleUnarchive(id: string) {
		busyId = id;
		try {
			await unarchiveFn(id);
			items = items.filter((i) => i.id !== id);
			toast.success(`${resourceName} restored`);
		} catch (err) {
			const message =
				err instanceof ApiClientError ? err.message : `Failed to restore ${resourceName.toLowerCase()}`;
			toast.error(message);
		} finally {
			busyId = null;
		}
	}

	export function addArchived(item: T) {
		// Called by parent when an item is freshly archived so it shows up
		// without needing to refetch.
		if (loaded) {
			items = [...items, item];
		}
	}
</script>

<div class="border-t pt-4">
	<button
		type="button"
		class="text-muted-foreground hover:text-foreground flex items-center gap-2 text-sm font-medium"
		onclick={toggle}
		aria-expanded={open}
		aria-controls={panelId}
	>
		{#if open}
			<ChevronDownIcon class="size-4" />
		{:else}
			<ChevronRightIcon class="size-4" />
		{/if}
		Archived {resourceNamePlural.toLowerCase()}
		{#if loaded}
			<span class="text-muted-foreground">({items.length})</span>
		{/if}
	</button>

	<div id={panelId} class="mt-3" hidden={!open}>
		{#if open}
			{#if loading}
				<div class="text-muted-foreground py-6 text-center text-sm">
					<LoaderCircleIcon class="mx-auto size-5 animate-spin" />
				</div>
			{:else if items.length === 0}
				<div class="text-muted-foreground py-6 text-center text-sm">
					No archived {resourceNamePlural.toLowerCase()}.
				</div>
			{:else}
				{#if header}{@render header()}{/if}
				{#each items as item (item.id)}
					{@render row({
						item,
						unarchive: () => handleUnarchive(item.id),
						busy: busyId === item.id,
					})}
				{/each}
			{/if}
		{/if}
	</div>
</div>
