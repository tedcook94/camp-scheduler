import { Hono } from "hono";
import { serve } from "@hono/node-server";
import { getMigrations } from "better-auth/db/migration";
import { auth } from "./auth.js";
import { env, pool } from "./env.js";
import { internalRouter } from "./internal.js";

// Fail-fast on BetterAuth schema drift. The auth schema is owned by
// golang-migrate; if the BetterAuth config has gained a field/table the
// migrations don't yet reflect, runtime queries will fail in subtle ways.
// Refuse to start instead, prompting `mise run auth:migration` + apply.
async function assertSchemaUpToDate() {
  const { toBeAdded, toBeCreated } = await getMigrations(auth.options);
  if (toBeAdded.length === 0 && toBeCreated.length === 0) {
    return;
  }
  const summary = [...toBeCreated, ...toBeAdded]
    .map((t) => `  ${t.table}: ${Object.keys(t.fields).join(", ")}`)
    .join("\n");
  console.error(
    "auth schema is out of date. regenerate and apply migrations:\n" +
      "  mise run auth:migration <name>\n" +
      "  mise run migrate:auth\n\npending changes:\n" +
      summary,
  );
  process.exit(1);
}

await assertSchemaUpToDate();

const app = new Hono();

app.get("/health", (c) => c.text("ok"));

app.get("/ready", async (c) => {
  try {
    await pool.query("SELECT 1");
    return c.text("ok");
  } catch {
    return c.text("db unreachable", 503);
  }
});

app.route("/internal", internalRouter);

// Mount BetterAuth on every method so plugins that issue DELETE/PATCH/OPTIONS
// (e.g. CORS preflight, session revocation) aren't silently rejected with 404.
app.all("/api/auth/*", (c) => auth.handler(c.req.raw));

const server = serve({
  fetch: app.fetch,
  port: env.port,
  hostname: "0.0.0.0",
});

const shutdown = async () => {
  console.log("auth-server shutting down");
  server.close();
  await pool.end();
  process.exit(0);
};

process.on("SIGTERM", shutdown);
process.on("SIGINT", shutdown);

console.log(`auth-server listening on :${env.port}`);
