<script lang="ts">
	import { goto } from "$app/navigation";
	import { browser } from "$app/environment";
	import { auth } from "$lib/stores/auth.svelte";
	import { authClient } from "$lib/auth-client";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import PasswordInput from "$lib/components/password-input.svelte";
	import * as Card from "$lib/components/ui/card";

	let identifier = $state("");
	let password = $state("");
	let error = $state("");
	let loading = $state(false);

	function redirectForRole(role: string | null) {
		if (role === "super_admin") {
			goto("/admin/camps", { replaceState: true });
		} else {
			goto("/app/dashboard", { replaceState: true });
		}
	}

	$effect(() => {
		if (browser && auth.isInitialized && auth.isAuthenticated) {
			redirectForRole(auth.role);
		}
	});

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		error = "";
		loading = true;

		try {
			// Identifier is dispatched to the matching BetterAuth endpoint: an "@" picks
			// email/password sign-in, anything else hits the username plugin.
			const result = identifier.includes("@")
				? await authClient.signIn.email({ email: identifier, password })
				: await authClient.signIn.username({ username: identifier, password });
			if (result.error) {
				error = result.error.message ?? "Invalid credentials";
				return;
			}
			await auth.onSignedIn();
			redirectForRole(auth.role);
		} catch (err) {
			error = err instanceof Error ? err.message : "An unexpected error occurred.";
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-svh items-center justify-center p-4">
	<Card.Card class="w-full max-w-sm">
		<Card.CardHeader>
			<Card.CardTitle class="text-xl">Camp Scheduler</Card.CardTitle>
			<Card.CardDescription>Sign in to your account</Card.CardDescription>
		</Card.CardHeader>
		<Card.CardContent>
			<form onsubmit={handleLogin} class="grid gap-4">
				{#if error}
					<div class="bg-destructive/10 text-destructive rounded-lg px-3 py-2 text-sm">
						{error}
					</div>
				{/if}
				<div class="grid gap-2">
					<Label for="identifier">Username or email</Label>
					<Input
						id="identifier"
						type="text"
						placeholder="Enter your username or email"
						bind:value={identifier}
						required
						disabled={loading}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="password">Password</Label>
					<PasswordInput
						id="password"
						placeholder="Enter your password"
						bind:value={password}
						required
						disabled={loading}
					/>
				</div>
				<Button type="submit" class="w-full" disabled={loading}>
					{loading ? "Signing in..." : "Sign in"}
				</Button>
			</form>
		</Card.CardContent>
	</Card.Card>
</div>
