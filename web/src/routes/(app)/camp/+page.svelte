<script lang="ts">
	import { onMount } from "svelte";
	import { myCampApi } from "$lib/api";
	import { ApiClientError } from "$lib/api/client";
	import { capitalizeFirst } from "$lib/utils";
	import { toast } from "svelte-sonner";
	import * as Card from "$lib/components/ui/card";
	import type { Camp } from "$lib/api/types";

	let camp = $state<Camp | null>(null);
	let loading = $state(true);

	async function loadCamp() {
		loading = true;
		try {
			camp = await myCampApi.get();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load camp";
			toast.error(capitalizeFirst(message));
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadCamp();
	});
</script>

<div class="grid gap-6">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Camp</h1>
		<p class="text-muted-foreground text-sm">Camp details for the current user.</p>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading camp...</div>
	{:else if camp}
		<Card.Card>
			<Card.CardHeader>
				<Card.CardTitle>{camp.name}</Card.CardTitle>
				{#if camp.location}
					<Card.CardDescription>{camp.location}</Card.CardDescription>
				{/if}
			</Card.CardHeader>
			<Card.CardContent>
				<dl class="grid gap-2 text-sm">
					<div class="flex gap-2">
						<dt class="text-muted-foreground font-medium">Status</dt>
						<dd>{camp.enabled ? "Enabled" : "Disabled"}</dd>
					</div>
				</dl>
			</Card.CardContent>
		</Card.Card>
	{:else}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No camp data available.
		</div>
	{/if}
</div>
