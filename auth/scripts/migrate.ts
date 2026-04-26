/**
 * Run BetterAuth migrations against the configured database.
 *
 * BetterAuth's Kysely adapter generates and applies the SQL needed to support
 * core auth + the enabled plugins (organization, admin, jwt, username). We
 * call the library API directly so the auth-server stays on a single runtime
 * version and we don't need to keep the @better-auth/cli npm package — whose
 * stable line lags the runtime — installed.
 */
import { getMigrations } from "better-auth/db/migration";
import { auth } from "../src/auth.js";
import { pool } from "../src/env.js";

async function main() {
  const { toBeAdded, toBeCreated, runMigrations } = await getMigrations(auth.options);

  if (toBeAdded.length === 0 && toBeCreated.length === 0) {
    console.log("no migrations needed.");
    return;
  }

  console.log("the following migrations will be applied:");
  for (const t of [...toBeCreated, ...toBeAdded]) {
    console.log(`  ${t.table}: ${Object.keys(t.fields).join(", ")}`);
  }

  await runMigrations();
  console.log("migrations applied.");
}

main()
  .catch((err) => {
    console.error("migration failed:", err);
    process.exitCode = 1;
  })
  .finally(async () => {
    await pool.end();
  });
