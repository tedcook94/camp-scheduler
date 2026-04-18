<script lang="ts">
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { campApi, userApi } from "$lib/api";
	import { ApiClientError } from "$lib/api/client";
	import { auth } from "$lib/stores/auth.svelte";
	import type { Camp, User, CreateUserRequest, UpdateUserRequest } from "$lib/api/types";
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
	import PencilIcon from "@lucide/svelte/icons/pencil";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import KeyIcon from "@lucide/svelte/icons/key-round";
	import UserCheckIcon from "@lucide/svelte/icons/user-check";
	import SortableTableHead from "$lib/components/sortable-table-head.svelte";
	import { sortItems, type SortDirection, type SortAccessor } from "$lib/utils";

	let users = $state<User[]>([]);
	let camps = $state<Camp[]>([]);
	let loading = $state(true);

	type UserSortKey = "name" | "username" | "email" | "role" | "camp";
	let sortKey = $state<UserSortKey>("name");
	let sortDirection = $state<SortDirection>("asc");

	function userSortAccessor(key: UserSortKey): SortAccessor<User> {
		switch (key) {
			case "name":
				return (u: User) => `${u.first_name} ${u.last_name}`;
			case "camp":
				return (u: User) => campNameById(u.camp_id);
			default:
				return key;
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

	// User form dialog
	let dialogOpen = $state(false);
	let dialogMode = $state<"create" | "edit">("create");
	let editingUser = $state<User | null>(null);
	let saving = $state(false);
	let formError = $state("");

	// User form fields
	let formUsername = $state("");
	let formEmail = $state("");
	let formFirstName = $state("");
	let formLastName = $state("");
	let formPassword = $state("");
	let formRole = $state<string>("admin");
	let formCampId = $state<string>("");

	// Password dialog
	let passwordDialogOpen = $state(false);
	let passwordUser = $state<User | null>(null);
	let passwordValue = $state("");
	let passwordError = $state("");
	let passwordSaving = $state(false);

	// Delete dialog
	let deleteDialogOpen = $state(false);
	let deletingUser = $state<User | null>(null);
	let deleting = $state(false);

	async function loadData() {
		loading = true;
		try {
			[users, camps] = await Promise.all([userApi.list(), campApi.list()]);
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to load data";
			toast.error(message);
		} finally {
			loading = false;
		}
	}

	function campNameById(id: string | null): string {
		if (!id) return "\u2014";
		const camp = camps.find((c) => c.id === id);
		return camp?.name ?? "Unknown";
	}

	function openCreate() {
		dialogMode = "create";
		editingUser = null;
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

	function openEdit(user: User) {
		dialogMode = "edit";
		editingUser = user;
		formUsername = user.username;
		formEmail = user.email;
		formFirstName = user.first_name;
		formLastName = user.last_name;
		formPassword = "";
		formRole = user.role;
		formCampId = user.camp_id ?? "";
		formError = "";
		dialogOpen = true;
	}

	function openPassword(user: User) {
		passwordUser = user;
		passwordValue = "";
		passwordError = "";
		passwordDialogOpen = true;
	}

	function openDelete(user: User) {
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
			const campId = formRole === "super_admin" ? null : formCampId;

			if (dialogMode === "create") {
				const req: CreateUserRequest = {
					username: formUsername.trim(),
					email: formEmail.trim(),
					password: formPassword,
					first_name: formFirstName.trim(),
					last_name: formLastName.trim(),
					role: formRole as "admin" | "super_admin",
					camp_id: campId,
				};
				await userApi.create(req);
				toast.success("User created.");
			} else if (editingUser) {
				const req: UpdateUserRequest = {
					username: formUsername.trim(),
					email: formEmail.trim(),
					first_name: formFirstName.trim(),
					last_name: formLastName.trim(),
					role: formRole as "admin" | "super_admin",
					camp_id: campId,
				};
				await userApi.update(editingUser.id, req);
				toast.success("User updated.");
			}

			dialogOpen = false;
			await loadData();
		} catch (err) {
			if (err instanceof ApiClientError) {
				formError = err.message;
			} else {
				formError = "An unexpected error occurred.";
			}
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
			await userApi.updatePassword(passwordUser.id, { password: passwordValue });
			toast.success("Password updated.");
			passwordDialogOpen = false;
			passwordUser = null;
		} catch (err) {
			if (err instanceof ApiClientError) {
				passwordError = err.message;
			} else {
				passwordError = "An unexpected error occurred.";
			}
		} finally {
			passwordSaving = false;
		}
	}

	async function handleDelete() {
		if (!deletingUser) return;
		deleting = true;

		try {
			await userApi.delete(deletingUser.id);
			toast.success("User deleted.");
			deleteDialogOpen = false;
			deletingUser = null;
			await loadData();
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to delete user";
			toast.error(message);
			deleteDialogOpen = false;
		} finally {
			deleting = false;
		}
	}

	async function handleImpersonate(user: User) {
		try {
			const tokens = await userApi.impersonate(user.id);
			auth.startImpersonation(tokens);
			goto("/app/dashboard");
		} catch (err) {
			const message = err instanceof ApiClientError ? err.message : "Failed to impersonate user";
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
					<SortableTableHead label="Camp" active={sortKey === "camp"} direction={sortDirection} onclick={() => toggleSort("camp")} />
					<Table.TableHead class="w-32">
						<span class="sr-only">Actions</span>
					</Table.TableHead>
				</Table.TableRow>
			</Table.TableHeader>
			<Table.TableBody>
				{#each sortedUsers as user (user.id)}
					<Table.TableRow>
						<Table.TableCell class="font-medium">
							{user.first_name} {user.last_name}
						</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">{user.username}</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">{user.email}</Table.TableCell>
						<Table.TableCell>
							{#if user.role === "super_admin"}
								<Badge variant="default">Super Admin</Badge>
							{:else}
								<Badge variant="secondary">Admin</Badge>
							{/if}
						</Table.TableCell>
						<Table.TableCell class="text-muted-foreground">
							{campNameById(user.camp_id)}
						</Table.TableCell>
						<Table.TableCell>
							<div class="flex items-center justify-end gap-1">
								{#if user.role !== "super_admin"}
									<Button variant="ghost" size="icon-sm" onclick={() => handleImpersonate(user)} title="Impersonate" aria-label="Impersonate {user.username}">
										<UserCheckIcon class="size-4" />
									</Button>
								{/if}
								<Button variant="ghost" size="icon-sm" onclick={() => openPassword(user)} title="Change password" aria-label="Change password for {user.username}">
									<KeyIcon class="size-4" />
								</Button>
								<Button variant="ghost" size="icon-sm" onclick={() => openEdit(user)} title="Edit" aria-label="Edit {user.username}">
									<PencilIcon class="size-4" />
								</Button>
								<Button variant="ghost" size="icon-sm" onclick={() => openDelete(user)} title="Delete" aria-label="Delete {user.username}">
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

<!-- Create / Edit User Dialog -->
<Dialog.Dialog bind:open={dialogOpen}>
	<Dialog.DialogContent class="max-w-md" onInteractOutside={(e) => e.preventDefault()}>
		<Dialog.DialogHeader>
			<Dialog.DialogTitle>
				{dialogMode === "create" ? "Create User" : "Edit User"}
			</Dialog.DialogTitle>
			<Dialog.DialogDescription>
				{dialogMode === "create"
					? "Add a new user account."
					: "Update user details."}
			</Dialog.DialogDescription>
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
					<Input
						id="user-first-name"
						bind:value={formFirstName}
						placeholder="First name"
						required
						disabled={saving}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="user-last-name">Last Name</Label>
					<Input
						id="user-last-name"
						bind:value={formLastName}
						placeholder="Last name"
						required
						disabled={saving}
					/>
				</div>
			</div>
			<div class="grid gap-2">
				<Label for="user-username">Username</Label>
				<Input
					id="user-username"
					bind:value={formUsername}
					placeholder="Username"
					required
					minlength={8}
					pattern="[a-zA-Z0-9._\\-]+"
					title="Letters, numbers, dots, hyphens, and underscores only"
					disabled={saving}
				/>
			</div>
			<div class="grid gap-2">
				<Label for="user-email">Email</Label>
				<Input
					id="user-email"
					type="email"
					bind:value={formEmail}
					placeholder="Email address"
					required
					disabled={saving}
				/>
			</div>
		{#if dialogMode === "create"}
			<div class="grid gap-2">
				<Label for="user-password">Password</Label>
				<PasswordInput
					id="user-password"
					bind:value={formPassword}
					placeholder="Minimum 8 characters"
					required
					minlength={8}
					disabled={saving}
				/>
			</div>
		{/if}
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
				<Button type="button" variant="outline" onclick={() => (dialogOpen = false)} disabled={saving}>
					Cancel
				</Button>
				<Button type="submit" disabled={saving}>
					{saving ? "Saving..." : dialogMode === "create" ? "Create" : "Save"}
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
				Set a new password for <strong>{passwordUser?.username}</strong>.
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
			<PasswordInput
				id="new-password"
				bind:value={passwordValue}
				placeholder="Minimum 8 characters"
				required
				minlength={8}
				disabled={passwordSaving}
			/>
		</div>
			<Dialog.DialogFooter>
				<Button type="button" variant="outline" onclick={() => (passwordDialogOpen = false)} disabled={passwordSaving}>
					Cancel
				</Button>
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
				Are you sure you want to delete <strong>{deletingUser?.username}</strong>? This action cannot be undone.
			</AlertDialog.AlertDialogDescription>
		</AlertDialog.AlertDialogHeader>
		<AlertDialog.AlertDialogFooter>
			<AlertDialog.AlertDialogCancel disabled={deleting}>Cancel</AlertDialog.AlertDialogCancel>
			<AlertDialog.AlertDialogAction
				variant="destructive"
				onclick={handleDelete}
				disabled={deleting}
			>
				{deleting ? "Deleting..." : "Delete"}
			</AlertDialog.AlertDialogAction>
		</AlertDialog.AlertDialogFooter>
	</AlertDialog.AlertDialogContent>
</AlertDialog.AlertDialog>
