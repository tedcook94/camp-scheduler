import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			// BetterAuth auth-server (must come before /api catch-all)
			'/api/auth': 'http://localhost:9101',
			// Go server
			'/api': 'http://localhost:9100',
			'/health': 'http://localhost:9100',
			'/ready': 'http://localhost:9100'
		}
	}
});
