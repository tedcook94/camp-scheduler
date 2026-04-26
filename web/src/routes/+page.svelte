<script lang="ts">
	import { goto } from "$app/navigation";
	import { browser } from "$app/environment";
	import { auth } from "$lib/stores/auth.svelte";

	$effect(() => {
		if (!browser || !auth.isInitialized) return;
		if (auth.isAuthenticated) {
			if (auth.role === "super_admin" && !auth.isImpersonating) {
				goto("/admin/camps", { replaceState: true });
			} else {
				goto("/app/dashboard", { replaceState: true });
			}
		} else {
			goto("/login", { replaceState: true });
		}
	});
</script>
