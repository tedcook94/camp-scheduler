<script lang="ts">
	import { goto } from "$app/navigation";
	import { browser } from "$app/environment";
	import { page } from "$app/stores";
	import { auth } from "$lib/stores/auth.svelte";

	if (browser && $page.status === 404) {
		if (!auth.isAuthenticated) {
			goto("/login", { replaceState: true });
		} else if (auth.role === "super_admin" && !auth.isImpersonating) {
			goto("/admin/camps", { replaceState: true });
		} else {
			goto("/app/dashboard", { replaceState: true });
		}
	}
</script>

{#if $page.status !== 404}
	<div class="flex min-h-svh items-center justify-center p-4">
		<div class="text-center">
			<h1 class="text-4xl font-bold">{$page.status}</h1>
			<p class="text-muted-foreground mt-2 text-sm">{$page.error?.message ?? "Something went wrong"}</p>
		</div>
	</div>
{/if}
