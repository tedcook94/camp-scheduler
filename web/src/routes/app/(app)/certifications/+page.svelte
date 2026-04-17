<script lang="ts">
	import NameResourcePage from "$lib/components/name-resource-page.svelte";
	import { certificationApi } from "$lib/api";
</script>

<NameResourcePage
	title="Certifications"
	description="Manage certifications that can be assigned to counselors and required by activities."
	resourceName="Certification"
	listFn={certificationApi.list}
	createFn={certificationApi.create}
	updateFn={certificationApi.update}
	deleteFn={certificationApi.delete}
/>
