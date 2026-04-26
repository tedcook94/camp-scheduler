import { createAuthClient } from "better-auth/svelte";
import {
	adminClient,
	jwtClient,
	organizationClient,
	usernameClient,
} from "better-auth/client/plugins";

export const authClient = createAuthClient({
	baseURL: typeof window !== "undefined" ? window.location.origin : undefined,
	basePath: "/api/auth",
	plugins: [usernameClient(), organizationClient(), adminClient(), jwtClient()],
});
