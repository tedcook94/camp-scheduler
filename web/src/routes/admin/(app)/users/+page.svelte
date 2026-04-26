<script lang="ts">
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { campApi } from "$lib/api";
	import { authClient } from "$lib/auth-client";
	import { auth } from "$lib/stores/auth.svelte";
	import type { Camp } from "$lib/api/types";
	import { toast } from "svelte-sonner";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import PasswordInput from "$lib/components/password-input.svelte";
	import { Badge } from "$lib/components/ui/badge";
	import * as Table from "$lib/components/ui/table";
	import * as Dialog from "$lib/components/ui/dialog";
	import * as AlertDialog from "$lib/components/ui/alert-dialog";
	import * as Select from "$lib/components/ui/select";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import KeyIcon from "@lucide/svelte/icons/key-round";
	import UserCheckIcon from "@lucide/svelte/icons/user-check";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import { sortItems, type SortDirection, type SortAccessor } from "$lib/utils";

	interface AdminUser {
		id: string;
		username?: string | null;
		email: string;
		firstName?: string | null;
		lastName?: string | null;
		role?: string | null;
		// Org membership info merged client-side.
		campIds: string[];
	}

	let users = $state<AdminUser[]>([]);
	let camps = $state<Camp[]>([]);
	let loading = $state(true);

	type UserSortKey = "name" | "username" | "email" | "role" | "camp";
	let sortKey = $state<UserSortKey>("name");
	let sortDirection = $state<SortDirection>("asc");

	function userSortAccessor(key: UserSortKey): SortAccessor<AdminUser> {
		switch (key) {
			case "name":
				return (u: AdminUser) => `${u.firstName ?? ""} ${u.lastName ?? ""}`.trim();
			case "username":
				return (u: AdminUser) => u.username ?? "";
			case "email":
				return (u: AdminUser) => u.email;
			case "role":
				return (u: AdminUser) => u.role ?? "";
			case "camp":
				return (u: AdminUser) => u.campIds.map(campNameById).join(", ");
			default: {
				// Compile-time exhaustiveness check: adding a new UserSortKey
				// without handling it here will surface as a TypeScript error.
				const _exhaustive: never = key;
				throw new Error(`unhandled sort key: ${_exhaustive as string}`);
			}
		}
	}

	let sortedUsers = $derived(sortItems(users, userSortAccessor(sortKey), sortDirection));

	function toggleSort(key: UserSortKey) {
		if (sortKey === key) {
			sortDirection = sortDirection === "asc" ? "desc" : "asc";
		} else {
			sortKey = key;
			sortDirection = "asc";
		}
	}

	let dialogOpen = $state(false);
	let saving = $state(false);
	let formError = $state("");

	type AdminFormRole = "admin" | "super_admin";
	// BetterAuth's admin client typings don't see roles defined via our custom
	// access-control config (super_admin), so we describe the wire role union
	// ourselves and cast once at the createUser call site below.
	type BetterAuthAdminRole = "user" | "super_admin";

	let formUsername = $state("");
	let formEmail = $state("");
	let formFirstName = $state("");
	let formLastName = $state("");
	let formPassword = $state("");
	let formRole = $state<AdminFormRole>("admin");
	let formCampId = $state<string>("");

	let passwordDialogOpen = $state(false);
	let passwordUser = $state<AdminUser | null>(null);
	let passwordValue = $state("");
	let passwordError = $state("");
	let passwordSaving = $state(false);

	let deleteDialogOpen = $state(false);
	let deletingUser = $state<AdminUser | null>(null);
	let deleting = $state(false);

	function campNameById(id: string | null | undefined): string {
		if (!id) return "\u2014";
		const camp = camps.find((c) => c.id === id);
		return camp?.name ?? "Unknown";
	}

	async function loadData() {
		loading = true;
		try {
			const [usersRes, campsList] = await Promise.all([
				authClient.admin.listUsers({ query: { limit: 500 } }),
				campApi.list(),
			]);
			camps = campsList;

			if (usersRes.error) {
				toast.error(usersRes.error.message ?? "Failed to load users");
				return;
			}
			const rawUsers = (usersRes.data?.users ?? []) as unknown as Array<Record<string, unknown>>;

			// For each camp/org, fetch memberships and build a userId -> [campId] map.
			const userToCamps = new Map<string, string[]>();
			await Promise.all(
				campsList.map(async (camp) => {
					try {
						const res = await authClient.organization.listMembers({
							query: { organizationId: camp.id, limit: 500 },
						});
						const members = (res.data?.members ?? []) as Array<{ userId: string }>;
						for (const m of members) {
							const list = userToCamps.get(m.userId) ?? [];
							list.push(camp.id);
							userToCamps.set(m.userId, list);
						}
					} catch {
						// Skip camps whose member list can't be loaded.
					}
				}),
			);

			users = rawUsers.map((u) => ({
				id: String(u.id),
				username: (u.username as string) ?? null,
				email: String(u.email ?? ""),
				firstName: (u.firstName as string) ?? null,
				lastName: (u.lastName as string) ?? null,
				role: (u.role as string) ?? null,
				campIds: userToCamps.get(String(u.id)) ?? [],
			}));
		} catch (err) {
			const message = err instanceof Error ? err.message : "Failed to load data";
			toast.error(message);
		} finally {
			loading = false;
		}
	}

	function openCreate() {
		formUsername = "";
		formEmail = "";
		formFirstName = "";
		formLastName = "";
		formPassword = "";
		formRole = "admin";
		formCampId = "";
		formError = "";
		dialogOpen = true;
	}

	function openPassword(user: AdminUser) {
		passwordUser = user;
		passwordValue = "";
		passwordError = "";
		passwordDialogOpen = true;
	}

	function openDelete(user: AdminUser) {
		deletingUser = user;
		deleteDialogOpen = true;
	}

	async function handleSave(e: SubmitEvent) {
		e.preventDefault();
		formError = "";

		if (formRole === "admin" && !formCampId) {
			formError = "Camp is required.";
			return;
		}

		saving = true;

		try {
			const fullName = `${formFirstName.trim()} ${formLastName.trim()}`.trim();
			const baRole: BetterAuthAdminRole = formRole === "super_admin" ? "super_admin" : "user";

			const created = await authClient.admin.createUser({
				email: formEmail.trim(),
				password: formPassword,
				name: fullName,
				// Cast bridges the gap between our typed wire union and the
				// admin client's narrower default-roles typing.
				role: baRole as "user",
				data: {
					username: formUsername.trim(),
					displayUsername: formUsername.trim(),
					firstName: formFirstName.trim(),
					lastName: formLastName.trim(),
				},
			});

			if (created.error) {
				formError = created.error.message ?? "Failed to create user";
				return;
			}

			if (formRole === "admin" && formCampId) {
				toast.info("User created. Camp assignment must be done from the user's account for now.");
			} else {
				toast.success("User created.");
			}
			dialogOpen = false;
			await loadData();
		} catch (err) {
			formError = err instanceof Error ? err.message : "An unexpected error occurred.";
		} finally {
			saving = false;
		}
	}

	async function handlePasswordChange(e: SubmitEvent) {
		e.preventDefault();
		if (!passwordUser) return;
		passwordError = "";
		passwordSaving = true;

		try {
			const res = await authClient.admin.setUserPassword({
				userId: passwordUser.id,
				newPassword: passwordValue,
			});
			if (res.error) {
				passwordError = res.error.message ?? "Failed to update password";
				return;
			}
			toast.success("Password updated.");
			passwordDialogOpen = false;
			passwordUser = null;
		} catch (err) {
			passwordError = err instanceof Error ? err.message : "An unexpected error occurred.";
		} finally {
			passwordSaving = false;
		}
	}

	async function handleDelete() {
		if (!deletingUser) return;
		deleting = true;

		try {
			const res = await authClient.admin.removeUser({ userId: deletingUser.id });
			if (res.error) {
				toast.error(res.error.message ?? "Failed to delete user");
				deleteDialogOpen = false;
				return;
			}
			toast.success("User deleted.");
			deleteDialogOpen = false;
			deletingUser = null;
			await loadData();
		} catch (err) {
			const message = err instanceof Error ? err.message : "Failed to delete user";
			toast.error(message);
			deleteDialogOpen = false;
		} finally {
			deleting = false;
		}
	}

	async function handleImpersonate(user: AdminUser) {
		try {
			const res = await authClient.admin.impersonateUser({ userId: user.id });
			if (res.error) {
				toast.error(res.error.message ?? "Failed to impersonate user");
				return;
			}
			// New impersonation cookie is set; reload session + JWT.
			await auth.onSignedIn();
			goto("/app/dashboard");
		} catch (err) {
			const message = err instanceof Error ? err.message : "Failed to impersonate user";
			toast.error(message);
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<div class="grid gap-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Users</h1>
			<p class="text-muted-foreground text-sm">Manage user accounts.</p>
		</div>
		<Button onclick={openCreate} size="sm">
			<PlusIcon data-icon="inline-start" class="size-4" />
			Add User
		</Button>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-8 text-center text-sm">Loading users...</div>
	{:else if users.length === 0}
		<div class="text-muted-foreground py-8 text-center text-sm">
			No users yet. Create your first user to get started.
		</div>
	{:else}
		<Table.Table>
			<Table.TableHeader>
				<Table.TableRow>
					<SortableTableHead label="Name" active={sortKey === "name"} direction={sortDirection} onclick={() => toggleSort("name")} />
					<SortableTableHead label="Username" active={sortKey === "username"} direction={sortDirection} onclick={() => toggleSort("username")} />
					<SortableTableHead label="Email" active={sortKey === "email"} direction={sortDirection} onclick={() => toggleSort("email")} />
					<SortableTableHead label="Role" active={sortKey === "role"} direction={sortDirection} onclick={() => toggleSort("role")} />
					<SortableTableHead label="Camps" active={sortKey === "camp"} direction={sortDirection} onclick={() => toggleSort("camp")} />
					<Table.TableHead class="w-32">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedUsers as user (user.id)}
					<Table.TableRow>
						<Table.TableCell class="font-medium">
							{user.firstName ?? ""} {user.lastName ?? ""}
						</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">{user.username ?? "\u2014"}</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">{user.email}</Table.TableCell>
						<Table.TableCell>
							{#if user.role === "super_admin"}
								<Badge variant="default">Super Admin</Badge>
							{:else}
								<Badge variant="secondary">User</Badge>
							{/if}
						</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">
							{user.campIds.length === 0 ? "\u2014" : user.campIds.map(campNameById).join(", ")}
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex items-center justify-end gap-1">
								{#if user.role !== "super_admin"}
									<Button variant="ghost" size="icon-sm" onclick={() => handleImpersonate(user)} title="Impersonate" aria-label="Impersonate {user.username ?? user.email}">
										<UserCheckIcon class="size-4" />
									</Button>
								{/if}
								<Button variant="ghost" size="icon-sm" onclick={() => openPassword(user)} title="Change password" aria-label="Change password for {user.username ?? user.email}">
									<KeyIcon class="size-4" />
								</Button>
								<Button variant="ghost" size="icon-sm" onclick={() => openDelete(user)} title="Delete" aria-label="Delete {user.username ?? user.email}">
									<TrashIcon class="size-4" />
								</Button>
							</div>
						</Table.TableCell>
					</Table.TableRow>
				{/each}
			</Table.TableBody>
		</Table.Table>
	{/if}
</div>

<!-- Create User Dialog -->
<Dialog.Dialog bind:open={dialogOpen}>
	<Dialog.DialogContent class="max-w-md" onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Create User</Dialog.DialogTitle>
			<Dialog.DialogDescription>Add a new user account.</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handleSave} class="grid gap-4">
			{#if formError}
				<div class="bg-destructive/10 text-destructive rounded-lg px-3 py-2 text-sm">
					{formError}
				</div>
			{/if}
			<div class="grid grid-cols-2 gap-4">
				<div class="grid gap-2">
					<Label for="user-first-name">First Name</Label>
					<Input id="user-first-name" bind:value={formFirstName} placeholder="First name" required disabled={saving} />
				</div>
				<div class="grid gap-2">
					<Label for="user-last-name">Last Name</Label>
					<Input id="user-last-name" bind:value={formLastName} placeholder="Last name" required disabled={saving} />
				</div>
			</div>
			<div class="grid gap-2">
				<Label for="user-username">Username</Label>
				<Input
					id="user-username"
					bind:value={formUsername}
					placeholder="Username"
					required
					minlength={3}
					pattern="[a-zA-Z0-9._]+"
					title="Letters, numbers, dots, and underscores only"
					disabled={saving}
				/>
			</div>
			<div class="grid gap-2">
				<Label for="user-email">Email</Label>
				<Input id="user-email" type="email" bind:value={formEmail} placeholder="Email address" required disabled={saving} />
			</div>
			<div class="grid gap-2">
				<Label for="user-password">Password</Label>
				<PasswordInput id="user-password" bind:value={formPassword} placeholder="Minimum 8 characters" required minlength={8} disabled={saving} />
			</div>
			<div class="grid gap-2">
				<Label for="user-role">Role</Label>
				<Select.Select type="single" bind:value={formRole}>
					<Select.SelectTrigger id="user-role" class="w-full">
						{formRole === "super_admin" ? "Super Admin" : "Admin"}
					</Select.SelectTrigger>
					<Select.SelectContent>
						<Select.SelectItem value="admin" label="Admin" />
						<Select.SelectItem value="super_admin" label="Super Admin" />
					</Select.SelectContent>
				</Select.Select>
			</div>
			{#if formRole === "admin"}
				<div class="grid gap-2">
					<Label for="user-camp">Camp</Label>
					<Select.Select type="single" bind:value={formCampId}>
						<Select.SelectTrigger id="user-camp" class="w-full">
							{camps.find((c) => c.id === formCampId)?.name ?? "Select a camp"}
						</Select.SelectTrigger>
						<Select.SelectContent>
							{#each camps as camp (camp.id)}
								<Select.SelectItem value={camp.id} label={camp.name} />
							{/each}
						</Select.SelectContent>
					</Select.Select>
				</div>
			{/if}
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" onclick={() => (dialogOpen = false)} disabled={saving}>Cancel</Button>
				<Button type="submit" disabled={saving}>
					{saving ? "Saving..." : "Create"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Change Password Dialog -->
<Dialog.Dialog bind:open={passwordDialogOpen}>
	<Dialog.DialogContent class="max-w-sm" onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>Change Password</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				Set a new password for <strong>{passwordUser?.username ?? passwordUser?.email}</strong>.
			</Dialog.DialogDescription>
		</Dialog.DialogHeader>
		<form onsubmit={handlePasswordChange} class="grid gap-4">
			{#if passwordError}
				<div class="bg-destructive/10 text-destructive rounded-lg px-3 py-2 text-sm">
					{passwordError}
				</div>
			{/if}
			<div class="grid gap-2">
				<Label for="new-password">New Password</Label>
				<PasswordInput id="new-password" bind:value={passwordValue} placeholder="Minimum 8 characters" required minlength={8} disabled={passwordSaving} />
			</div>
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" onclick={() => (passwordDialogOpen = false)} disabled={passwordSaving}>Cancel</Button>
				<Button type="submit" disabled={passwordSaving}>
					{passwordSaving ? "Saving..." : "Update Password"}
				</Button>
			</Dialog.DialogFooter>
		</form>
	</Dialog.DialogContent>
</Dialog.Dialog>

<!-- Delete Confirmation -->
<AlertDialog.AlertDialog bind:open={deleteDialogOpen}>
	<AlertDialog.AlertDialogContent>
		<AlertDialog.AlertDialogHeader>
			<AlertDialog.AlertDialogTitle>Delete User</AlertDialog.AlertDialogTitle>
			<AlertDialog.AlertDialogDescription>
				Are you sure you want to delete <strong>{deletingUser?.username ?? deletingUser?.email}</strong>? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction onclick={handleDelete} disabled={deleting}>
				{deleting ? "Deleting..." : "Delete"}
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
