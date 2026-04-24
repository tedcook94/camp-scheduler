import { api } from "./client";

export interface ExportOptions {
	includeUnassigned: boolean;
}

function buildQuery(format: "csv" | "pdf", opts: ExportOptions): string {
	const params = new URLSearchParams();
	params.set("format", format);
	params.set("include_unassigned", opts.includeUnassigned ? "true" : "false");
	return params.toString();
}

export const reportApi = {
	downloadCabinCSV: (sessionId: string, opts: ExportOptions) =>
		api.download(`/api/v1/sessions/${sessionId}/reports/cabin?${buildQuery("csv", opts)}`),
	downloadActivityCSV: (sessionId: string, opts: ExportOptions) =>
		api.download(`/api/v1/sessions/${sessionId}/reports/activities?${buildQuery("csv", opts)}`),
	downloadCabinPDF: (sessionId: string, opts: ExportOptions) =>
		api.download(`/api/v1/sessions/${sessionId}/reports/cabin?${buildQuery("pdf", opts)}`),
	downloadActivityPDF: (sessionId: string, opts: ExportOptions) =>
		api.download(`/api/v1/sessions/${sessionId}/reports/activities?${buildQuery("pdf", opts)}`),
};
