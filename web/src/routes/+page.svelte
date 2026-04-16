<script lang="ts">
	import { goto } from "$app/navigation";
	import { browser } from "$app/environment";
	import { auth } from "$lib/stores/auth.svelte";
	import { authApi } from "$lib/api";
	import { ApiClientError } from "$lib/api/client";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import PasswordInput from "$lib/components/password-input.svelte";
	import * as Card from "$lib/components/ui/card";

	let username = $state("");
	let password = $state("");
	let error = $state("");
	let loading = $state(false);

	function redirectForRole(role: string | null) {
		if (role === "super_admin") {
			goto("/admin/camps");
		} else {
			goto("/app/dashboard");
		}
	}

	if (browser && auth.isAuthenticated) {
		redirectForRole(auth.role);
	}

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		error = "";
		loading = true;

		try {
			const tokens = await authApi.login(username, password);
			auth.setTokens(tokens);
			redirectForRole(auth.role);
		} catch (err) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else {
				error = "An unexpected error occurred.";
			}
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
					<Label for="username">Username</Label>
					<Input
						id="username"
						type="text"
						placeholder="Enter your username"
						bind:value={username}
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
