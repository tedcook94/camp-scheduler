import { Pool } from "pg";

function required(name: string): string {
  const v = process.env[name];
  if (!v) {
    throw new Error(`required env var ${name} is not set`);
  }
  return v;
}

function optional(name: string, fallback: string): string {
  return process.env[name] ?? fallback;
}

export const env = {
  port: parseInt(optional("AUTH_PORT", "9101"), 10),
  baseURL: required("AUTH_BASE_URL"),
  trustedOrigins: optional("AUTH_TRUSTED_ORIGINS", "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean),
  betterAuthSecret: required("BETTER_AUTH_SECRET"),
  serviceSecret: required("AUTH_SHARED_SECRET"),
  // Runtime URL for the BetterAuth auth-server. Connects as the restricted
  // `auth_user` role whose search_path is fixed to `auth, public` by the
  // database bootstrap, so unqualified identifiers resolve in the auth schema.
  databaseUrl: required("AUTH_DATABASE_URL"),
};

export const pool = new Pool({ connectionString: env.databaseUrl });
