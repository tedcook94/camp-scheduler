import { betterAuth } from "better-auth";
import { APIError } from "better-auth/api";
import {
  USERNAME_ERROR_CODES,
  admin,
  jwt,
  organization,
  username,
} from "better-auth/plugins";
import { env, pool } from "./env.js";
import { adminAccessControl } from "./permissions.js";

/**
 * BetterAuth instance shared by HTTP handler and seed/admin scripts.
 *
 * Multi-tenancy maps directly: each camp is a BetterAuth organization
 * (organization.id = camps.id). The active organization on a session is
 * surfaced to the Go server via the `camp_id` JWT claim.
 *
 * Sessions are long-lived (30 days) with rolling extension on activity.
 * JWTs are short-lived (15m) and refreshed transparently by the client.
 */
export const auth = betterAuth({
  baseURL: env.baseURL,
  basePath: "/api/auth",
  secret: env.betterAuthSecret,
  database: pool,
  trustedOrigins: env.trustedOrigins,

  emailAndPassword: {
    enabled: true,
    autoSignIn: true,
    minPasswordLength: 8,
  },

  session: {
    expiresIn: 60 * 60 * 24 * 30, // 30 days
    updateAge: 60 * 60 * 24, // refresh session if older than 1 day
  },

  user: {
    additionalFields: {
      // BetterAuth requires `name`; we keep first/last name separately so the
      // Go side can render them without parsing.
      firstName: { type: "string", required: false, input: true },
      lastName: { type: "string", required: false, input: true },
      // Per-user revocation timestamp. The Go server rejects any JWT whose
      // `iat` is at or before this value, giving us a token-version-style
      // instant invalidation hook for membership removals, forced logouts,
      // etc. Defaults to row creation time so a brand-new user's first JWT
      // (issued strictly after) always passes.
      revokedAfter: {
        type: "date",
        required: false,
        input: false,
        defaultValue: () => new Date(),
      },
    },
  },

  databaseHooks: {
    user: {
      create: {
        before: async (user) => {
          // BetterAuth's admin plugin pre-checks email uniqueness but ignores
          // username (added by the username plugin). Without this, duplicate
          // usernames bubble up as raw Postgres unique-violation errors. We
          // also lowercase the username here so admin-created accounts are
          // findable by the username plugin's sign-in lookup, which always
          // normalizes the input.
          const candidate = (user as { username?: string | null }).username;
          if (typeof candidate !== "string" || candidate.length === 0) {
            return { data: user };
          }
          // Mirror the username plugin's defaults. `auth.api.createUser`
          // (admin/internal create paths) bypasses the plugin's own
          // /sign-up/email hook, so without these checks an admin or seeder
          // can mint usernames that later fail to sign in.
          if (candidate.length < 3) {
            throw new APIError("BAD_REQUEST", USERNAME_ERROR_CODES.USERNAME_TOO_SHORT);
          }
          if (candidate.length > 30) {
            throw new APIError("BAD_REQUEST", USERNAME_ERROR_CODES.USERNAME_TOO_LONG);
          }
          if (!/^[a-zA-Z0-9_.]+$/.test(candidate)) {
            throw new APIError("BAD_REQUEST", USERNAME_ERROR_CODES.INVALID_USERNAME);
          }
          const normalized = candidate.toLowerCase();
          const existing = await pool.query(
            `SELECT 1 FROM "user" WHERE username = $1 LIMIT 1`,
            [normalized],
          );
          if ((existing.rowCount ?? 0) > 0) {
            throw new APIError("BAD_REQUEST", {
              code: "USERNAME_IS_ALREADY_TAKEN",
              message: "Username is already taken. Please try another.",
            });
          }
          return { data: { ...user, username: normalized } };
        },
      },
    },
    session: {
      create: {
        before: async (session) => {
          // Auto-select an active organization on login so single-camp users
          // never see an "no active camp" state. Multi-camp users can switch
          // via the user dropdown.
          const result = await pool.query<{ organizationId: string }>(
            `SELECT "organizationId" FROM "member"
             WHERE "userId" = $1
             ORDER BY "createdAt" ASC
             LIMIT 1`,
            [session.userId],
          );
          const orgId = result.rows[0]?.organizationId;
          if (!orgId) {
            return { data: session };
          }
          return {
            data: { ...session, activeOrganizationId: orgId },
          };
        },
      },
    },
  },

  plugins: [
    username(),
    organization({
      // Each camp = one organization. We don't use teams.
      allowUserToCreateOrganization: false, // organizations are created server-side by Go
    }),
    admin({
      ...adminAccessControl,
      defaultRole: "user",
      adminRoles: ["super_admin"],
      impersonationSessionDuration: 60 * 60, // 1 hour
    }),
    jwt({
      jwks: {
        keyPairConfig: { alg: "EdDSA", crv: "Ed25519" },
      },
      jwt: {
        expirationTime: "15m",
        definePayload: ({ user, session }) => {
          const u = user as typeof user & {
            username?: string | null;
            displayUsername?: string | null;
            role?: string | null;
            firstName?: string | null;
            lastName?: string | null;
          };
          const s = session as typeof session & {
            activeOrganizationId?: string | null;
            activeOrganizationRole?: string | null;
            impersonatedBy?: string | null;
          };
          return {
            sub: user.id,
            username: u.username ?? user.email,
            email: user.email,
            role: u.role ?? "user",
            camp_id: s.activeOrganizationId ?? "",
            org_role: s.activeOrganizationRole ?? "",
            impersonated_by: s.impersonatedBy ?? "",
            first_name: u.firstName ?? "",
            last_name: u.lastName ?? "",
          };
        },
      },
    }),
  ],
});

export type Auth = typeof auth;
