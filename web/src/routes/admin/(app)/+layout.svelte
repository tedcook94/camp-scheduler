<script lang="ts">
	import { goto } from "$app/navigation";
	import { browser } from "$app/environment";
	import { page } from "$app/stores";
	import { auth } from "$lib/stores/auth.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Separator } from "$lib/components/ui/separator";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
	import { toggleMode } from "mode-watcher";
	import TentTreeIcon from "@lucide/svelte/icons/tent-tree";
	import UsersIcon from "@lucide/svelte/icons/users";
	import MoonIcon from "@lucide/svelte/icons/moon";
	import SunIcon from "@lucide/svelte/icons/sun";
	import LogOutIcon from "@lucide/svelte/icons/log-out";
	import CircleUserIcon from "@lucide/svelte/icons/circle-user";

	let { children } = $props();

	if (browser && !auth.isAuthenticated) {
		auth.clear();
		goto("/");
	}

	if (browser && auth.isAuthenticated && auth.role !== "super_admin") {
		auth.clear();
		goto("/");
	}

	function handleLogout() {
		auth.clear();
		goto("/");
	}

	const adminNavItems = [
		{ href: "/admin/camps", label: "Camps", icon: TentTreeIcon },
		{ href: "/admin/users", label: "Users", icon: UsersIcon },
	];
</script>

{#if auth.isAuthenticated}
	{@const navItems = adminNavItems}
	<div class="flex min-h-svh">
		<div class="flex flex-1">
			<!-- Sidebar -->
			<aside class="bg-sidebar text-sidebar-foreground border-sidebar-border flex w-56 flex-col border-r">
				<div class="flex h-12 items-center gap-2 px-4">
					<TentTreeIcon class="text-sidebar-primary size-5" />
					<span class="text-sm font-semibold">Camp Scheduler</span>
				</div>
				<Separator />
				<nav class="flex-1 p-2">
					<ul class="grid gap-0.5">
						{#each navItems as item}
							{@const active = $page.url.pathname.startsWith(item.href)}
							<li>
								<a
									href={item.href}
									class="flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm transition-colors {active
										? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
										: 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground'}"
								>
									<item.icon class="size-4" />
									{item.label}
								</a>
							</li>
						{/each}
					</ul>
				</nav>
				<Separator />
				<div class="flex items-center justify-between p-2">
					<DropdownMenu.DropdownMenu>
						<DropdownMenu.Trigger>
							{#snippet children()}
								<Button variant="ghost" size="sm" class="gap-2 px-2">
									<CircleUserIcon class="size-4" />
									<span class="truncate text-xs">{auth.username}</span>
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="start" class="w-40">
							<DropdownMenu.Item onclick={handleLogout}>
								<LogOutIcon class="mr-2 size-4" />
								Sign out
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.DropdownMenu>
					<Button variant="ghost" size="icon-sm" onclick={toggleMode} aria-label="Toggle dark mode">
						<SunIcon class="size-4 scale-100 rotate-0 transition-transform dark:scale-0 dark:-rotate-90" />
						<MoonIcon class="absolute size-4 scale-0 rotate-90 transition-transform dark:scale-100 dark:rotate-0" />
					</Button>
				</div>
			</aside>

			<!-- Main content -->
			<main class="flex-1 overflow-auto">
				<div class="mx-auto max-w-5xl p-6">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
