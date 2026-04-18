<script lang="ts">
	import * as Card from "$lib/components/ui/card";
	import SettingsIcon from "@lucide/svelte/icons/settings";
	import UsersIcon from "@lucide/svelte/icons/users";
	import CalendarIcon from "@lucide/svelte/icons/calendar";
	import TentTreeIcon from "@lucide/svelte/icons/tent-tree";
	import ClipboardListIcon from "@lucide/svelte/icons/clipboard-list";
	import DumbbellIcon from "@lucide/svelte/icons/dumbbell";
	import PlayIcon from "@lucide/svelte/icons/play";

	const sections = [
		{
			href: "/app/sessions",
			label: "Sessions",
			description: "Manage sessions and seasons for your camp",
			icon: CalendarIcon,
			enabled: true,
		},
		{
			href: "/app/cabins",
			label: "Cabins",
			description: "Manage camp cabins and their configurations",
			icon: TentTreeIcon,
			enabled: true,
		},
		{
			href: "/app/counselors",
			label: "Counselors",
			description: "Manage counselor profiles, preferences, and certifications",
			icon: UsersIcon,
			enabled: true,
		},
		{
			href: "/app/campers",
			label: "Campers",
			description: "Track camper registrations and session enrollments",
			icon: ClipboardListIcon,
			enabled: true,
		},
		{
			href: "/app/activities",
			label: "Activities",
			description: "Manage activities, time slots, and certification requirements",
			icon: DumbbellIcon,
			enabled: true,
		},
		{
			label: "Assignment Runs",
			description: "Run the solver and review cabin and activity assignments",
			icon: PlayIcon,
			enabled: false,
		},
		{
			href: "/app/settings",
			label: "Camp Settings",
			description: "View and edit your camp details",
			icon: SettingsIcon,
			enabled: true,
		},
	];
</script>

<div class="grid gap-6">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Dashboard</h1>
		<p class="text-muted-foreground text-sm">Welcome to Camp Scheduler.</p>
	</div>

	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each sections as section}
			{#if section.enabled}
				<a href={section.href} class="group">
					<Card.Card class="group-hover:border-primary/50 h-full transition-colors">
						<Card.CardHeader class="flex flex-row items-center gap-3 space-y-0 pb-2">
							<section.icon class="text-muted-foreground size-5" />
							<Card.CardTitle class="text-sm font-medium">{section.label}</Card.CardTitle>
						</Card.CardHeader>
						<Card.CardContent>
							<p class="text-muted-foreground text-sm">{section.description}</p>
						</Card.CardContent>
					</Card.Card>
				</a>
			{:else}
				<Card.Card class="h-full opacity-50">
					<Card.CardHeader class="flex flex-row items-center gap-3 space-y-0 pb-2">
						<section.icon class="text-muted-foreground size-5" />
						<Card.CardTitle class="text-sm font-medium">{section.label}</Card.CardTitle>
					</Card.CardHeader>
					<Card.CardContent>
						<p class="text-muted-foreground text-sm">{section.description}</p>
						<p class="mt-1 text-xs text-amber-600 dark:text-amber-400">Coming soon</p>
					</Card.CardContent>
				</Card.Card>
			{/if}
		{/each}
	</div>
</div>
