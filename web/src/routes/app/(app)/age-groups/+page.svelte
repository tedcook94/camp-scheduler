<script lang="ts">
	import NameResourcePage from "$lib/components/name-resource-page.svelte";
	import { ageGroupApi } from "$lib/api";
</script>

<NameResourcePage
	title="Age Groups"
	description="Manage age groups used to organize campers and cabin assignments."
	resourceName="Age Group"
	listFn={ageGroupApi.list}
	createFn={ageGroupApi.create}
	updateFn={ageGroupApi.update}
	deleteFn={ageGroupApi.delete}
	archiveFn={ageGroupApi.archive}
	unarchiveFn={ageGroupApi.unarchive}
	listArchivedFn={ageGroupApi.listArchived}
/>
