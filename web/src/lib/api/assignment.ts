import type {
	RunResponse,
	RunDetailResponse,
	SolutionDetailResponse,
	TriggerRunRequest,
} from "./types";
import { api } from "./client";

export const assignmentApi = {
	listRuns: async (sessionId: string): Promise<RunResponse[]> => {
		return api.get<RunResponse[]>(`/api/v1/sessions/${sessionId}/assignment-runs`);
	},

	getRun: async (sessionId: string, runId: string): Promise<RunDetailResponse> => {
		return api.get<RunDetailResponse>(
			`/api/v1/sessions/${sessionId}/assignment-runs/${runId}`,
		);
	},

	getRunById: async (runId: string): Promise<RunDetailResponse> => {
		return api.get<RunDetailResponse>(`/api/v1/assignment-runs/${runId}`);
	},

	triggerRun: async (
		sessionId: string,
		data: TriggerRunRequest,
	): Promise<RunDetailResponse> => {
		return api.post<RunDetailResponse>(
			`/api/v1/sessions/${sessionId}/assignment-runs`,
			data,
		);
	},

	getSolution: async (
		sessionId: string,
		runId: string,
		solutionId: string,
	): Promise<SolutionDetailResponse> => {
		return api.get<SolutionDetailResponse>(
			`/api/v1/sessions/${sessionId}/assignment-runs/${runId}/solutions/${solutionId}`,
		);
	},

	selectSolution: async (
		sessionId: string,
		runId: string,
		solutionId: string,
	): Promise<RunResponse> => {
		return api.post<RunResponse>(
			`/api/v1/sessions/${sessionId}/assignment-runs/${runId}/solutions/${solutionId}/select`,
		);
	},

	deleteRun: async (sessionId: string, runId: string): Promise<void> => {
		return api.delete<void>(`/api/v1/sessions/${sessionId}/assignment-runs/${runId}`);
	},
};
