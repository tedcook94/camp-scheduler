<script lang="ts">
	import NameResourcePage from "$lib/components/name-resource-page.svelte";
	import { timeSlotApi } from "$lib/api";
</script>

<NameResourcePage
	title="Time Slots"
	description="Manage time slots that define scheduling periods for activities within sessions."
	resourceName="Time Slot"
	listFn={timeSlotApi.list}
	createFn={timeSlotApi.create}
	updateFn={timeSlotApi.update}
	deleteFn={timeSlotApi.delete}
/>
