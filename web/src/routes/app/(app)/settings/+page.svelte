<script lang="ts">
	import { onMount } from "svelte";
	import { campApi } from "$lib/api";
	import { ApiClientError } from "$lib/api/client";
	import { capitalizeFirst } from "$lib/utils";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import * as Card from "$lib/components/ui/card";
	import type { Camp } from "$lib/api/types";
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

	let camp = $state<Camp | null>(null);
	let loading = $state(true);
	let saving = $state(false);

	let formName = $state("");
	let formLocation = $state("");

	async function loadCamp() {
		loading = true;
		try {
			camp = await campApi.get();
			formName = camp.name;
			formLocation = camp.location ?? "";
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load camp";
			toast.error(capitalizeFirst(message));
		} finally {
			loading = false;
		}
	}

	async function handleSave(e: SubmitEvent) {
		e.preventDefault();
		if (!camp) return;
		saving = true;

		try {
			const location = formLocation.trim() || null;
			camp = await campApi.update({
				name: formName.trim(),
				location,
				enabled: camp.enabled,
			});
			formName = camp.name;
			formLocation = camp.location ?? "";
			toast.success("Camp settings updated");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to update camp";
			toast.error(capitalizeFirst(message));
		} finally {
			saving = false;
		}
	}

	onMount(() => {
		loadCamp();
	});
</script>

<div class="grid gap-6">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Camp Settings</h1>
		<p class="text-muted-foreground text-sm">View and edit your camp details.</p>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading camp settings...</div>
	{:else if camp}
		<Card.Card>
			<Card.CardHeader>
				<Card.CardTitle>Camp Details</Card.CardTitle>
			</Card.CardHeader>
			<Card.CardContent>
				<form onsubmit={handleSave} class="grid gap-4">
					<div class="grid gap-2">
						<Label for="name">Name</Label>
						<Input
							id="name"
							type="text"
							bind:value={formName}
							required
							disabled={saving}
						/>
					</div>
					<div class="grid gap-2">
						<Label for="location">Location</Label>
						<Input
							id="location"
							type="text"
							placeholder="Optional"
							bind:value={formLocation}
							disabled={saving}
						/>
					</div>
					<div class="flex items-center gap-2 text-sm">
						<span class="text-muted-foreground">Status:</span>
						<span class={camp.enabled ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}>
							{camp.enabled ? "Enabled" : "Disabled"}
						</span>
					</div>
					<div class="flex justify-end">
						<Button type="submit" disabled={saving}>
							{#if saving}
								<LoaderCircleIcon class="mr-2 size-4 animate-spin" />
							{/if}
							Save changes
						</Button>
					</div>
				</form>
			</Card.CardContent>
		</Card.Card>
	{:else}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No camp data available.
		</div>
	{/if}
</div>
