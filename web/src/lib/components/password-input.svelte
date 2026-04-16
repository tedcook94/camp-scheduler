<script lang="ts">
	import type { HTMLInputAttributes } from "svelte/elements";
	import { cn } from "$lib/utils";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import EyeIcon from "@lucide/svelte/icons/eye";
	import EyeOffIcon from "@lucide/svelte/icons/eye-off";

	let {
		value = $bindable(),
		class: className,
		...restProps
	}: Omit<HTMLInputAttributes, "type" | "files"> & { value?: string } = $props();

	let visible = $state(false);
</script>

<div class="relative">
	<Input type={visible ? "text" : "password"} bind:value class={cn("pr-9", className)} {...restProps} />
	<Button
		type="button"
		variant="ghost"
		size="icon-sm"
		class="absolute top-1/2 right-1 -translate-y-1/2"
		onclick={() => (visible = !visible)}
		aria-label={visible ? "Hide password" : "Show password"}
	>
		{#if visible}
			<EyeOffIcon class="size-4" />
		{:else}
			<EyeIcon class="size-4" />
		{/if}
	</Button>
</div>
