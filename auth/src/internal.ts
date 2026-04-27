import { Hono, type Context } from "hono";
import type { ContentfulStatusCode } from "hono/utils/http-status";
import { isAPIError } from "better-auth/api";
import { auth } from "./auth.js";
import { env } from "./env.js";

/**
 * Internal API used by the Go server for service-to-service operations that
 * BetterAuth doesn't expose as user-facing endpoints (notably: keeping the
 * `organization` table in sync with `camps`). All routes require the shared
 * service secret in the Authorization header.
 */
export const internalRouter = new Hono();

internalRouter.use("*", async (c, next) => {
  const header = c.req.header("Authorization") ?? "";
  const expected = `Bearer ${env.serviceSecret}`;
  if (header !== expected) {
    return c.json({ error: "unauthorized" }, 401);
  }
  await next();
});

/**
 * Surface BetterAuth APIErrors with their original status code and message
 * instead of collapsing every failure into a generic 500. Callers (the Go
 * seeders + admin handlers) need the underlying reason to give useful UX.
 */
function apiErrorResponse(c: Context, err: unknown, fallback: string) {
  if (isAPIError(err)) {
    const e = err as { statusCode?: number; body?: { message?: string; code?: string } };
    const status = (e.statusCode ?? 500) as ContentfulStatusCode;
    return c.json(
      { error: e.body?.message ?? fallback, code: e.body?.code },
      status,
    );
  }
  console.error(fallback, err);
  return c.json({ error: fallback }, 500);
}

internalRouter.post("/organizations", async (c) => {
  const body = (await c.req.json()) as {
    id: string;
    name: string;
    slug: string;
  };
  if (!body.id || !body.name || !body.slug) {
    return c.json({ error: "id, name, slug are required" }, 400);
  }

  // BetterAuth's createOrganization API generates IDs internally; we insert
  // directly so the org id matches the camps.id we control on the Go side.
  const { pool } = await import("./env.js");
  try {
    await pool.query(
      `INSERT INTO "organization" (id, name, slug, "createdAt")
       VALUES ($1, $2, $3, now())
       ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, slug = EXCLUDED.slug`,
      [body.id, body.name, body.slug],
    );
    return c.json({ ok: true });
  } catch (err) {
    return apiErrorResponse(c, err, "failed to create organization");
  }
});

internalRouter.delete("/organizations/:id", async (c) => {
  const id = c.req.param("id");
  const { pool } = await import("./env.js");
  try {
    await pool.query(`DELETE FROM "organization" WHERE id = $1`, [id]);
    return c.json({ ok: true });
  } catch (err) {
    return apiErrorResponse(c, err, "failed to delete organization");
  }
});

internalRouter.post("/members", async (c) => {
  const body = (await c.req.json()) as {
    userId: string;
    organizationId: string;
    role?: string;
  };
  if (!body.userId || !body.organizationId) {
    return c.json({ error: "userId and organizationId are required" }, 400);
  }

  try {
    await auth.api.addMember({
      body: {
        userId: body.userId,
        organizationId: body.organizationId,
        role: (body.role ?? "admin") as "admin" | "member" | "owner",
      },
    });
    return c.json({ ok: true });
  } catch (err) {
    return apiErrorResponse(c, err, "failed to add member");
  }
});

internalRouter.delete("/members", async (c) => {
  const userId = c.req.query("userId");
  const organizationId = c.req.query("organizationId");
  if (!userId || !organizationId) {
    return c.json({ error: "userId and organizationId are required" }, 400);
  }

  // The BetterAuth removeMember endpoint requires a session-bound caller,
  // which we don't have for service-to-service calls. Delete the join row
  // directly to mirror the organization endpoint's approach. We then:
  //   1. Re-target sessions whose activeOrganizationId pointed at the
  //      removed org to the user's next remaining org (or null), keeping
  //      activeOrganizationRole consistent with the new org so the next
  //      JWT mint can't carry a stale org_role from the removed camp.
  //   2. Bump revokedAfter to invalidate any JWTs already in flight; the
  //      Go server checks this on every request.
  // All four mutations run inside a single transaction so we never end up
  // with a half-removed membership and a still-valid session/JWT pointing
  // at the removed camp.
  const { pool } = await import("./env.js");
  const client = await pool.connect();
  try {
    await client.query("BEGIN");

    await client.query(
      `DELETE FROM "member" WHERE "userId" = $1 AND "organizationId" = $2`,
      [userId, organizationId],
    );

    const next = await client.query<{ organizationId: string; role: string }>(
      `SELECT "organizationId", role FROM "member"
       WHERE "userId" = $1
       ORDER BY "createdAt" ASC
       LIMIT 1`,
      [userId],
    );
    const nextOrgId = next.rows[0]?.organizationId ?? null;
    const nextRole = next.rows[0]?.role ?? null;
    await client.query(
      `UPDATE "session"
       SET "activeOrganizationId" = $3, "activeOrganizationRole" = $4
       WHERE "userId" = $1 AND "activeOrganizationId" = $2`,
      [userId, organizationId, nextOrgId, nextRole],
    );

    await client.query(
      `UPDATE "user" SET "revokedAfter" = now() WHERE id = $1`,
      [userId],
    );

    await client.query("COMMIT");
    return c.json({ ok: true });
  } catch (err) {
    await client.query("ROLLBACK").catch(() => {});
    return apiErrorResponse(c, err, "failed to remove member");
  } finally {
    client.release();
  }
});

/**
 * Returns the user's current revocation timestamp. The Go server caches this
 * per-user and rejects any JWT whose `iat` is at or before the value, giving
 * us instant invalidation on membership removal (or any future "kick this
 * user" operation that bumps the column).
 */
internalRouter.get("/users/:id/revoked-after", async (c) => {
  const id = c.req.param("id");
  const { pool } = await import("./env.js");
  try {
    const result = await pool.query<{ revokedAfter: Date | null }>(
      `SELECT "revokedAfter" FROM "user" WHERE id = $1`,
      [id],
    );
    if (result.rowCount === 0) {
      return c.json({ error: "user not found" }, 404);
    }
    const value = result.rows[0]?.revokedAfter;
    return c.json({
      revokedAfter: value ? new Date(value).toISOString() : null,
    });
  } catch (err) {
    return apiErrorResponse(c, err, "failed to fetch revocation timestamp");
  }
});

internalRouter.post("/users", async (c) => {
  const body = (await c.req.json()) as {
    email: string;
    password: string;
    username?: string;
    firstName?: string;
    lastName?: string;
    role?: string;
  };
  if (!body.email || !body.password) {
    return c.json({ error: "email and password are required" }, 400);
  }

  const fullName = `${body.firstName ?? ""} ${body.lastName ?? ""}`.trim() || body.username || body.email;
  try {
    const created = await auth.api.createUser({
      body: {
        email: body.email,
        password: body.password,
        name: fullName,
        role: (body.role ?? "user") as "user" | "super_admin",
        data: {
          username: body.username,
          displayUsername: body.username,
          firstName: body.firstName,
          lastName: body.lastName,
        },
      },
    });
    return c.json({ ok: true, userId: created.user.id });
  } catch (err) {
    return apiErrorResponse(c, err, "failed to create user");
  }
});

/**
 * Updates a user's mutable profile fields (first/last name, email). The admin
 * plugin does not expose a generic "update arbitrary user" route, so we patch
 * the underlying tables directly. An email change clears emailVerified
 * (mirroring sign-up semantics) and bumps revokedAfter so any in-flight JWT
 * with the old email claim becomes invalid within one cache TTL.
 */
internalRouter.put("/users/:id", async (c) => {
  const id = c.req.param("id");
  const body = (await c.req.json()) as {
    firstName?: string;
    lastName?: string;
    email?: string;
  };
  const wantsEmail = typeof body.email === "string" && body.email.length > 0;
  const wantsFirst = typeof body.firstName === "string";
  const wantsLast = typeof body.lastName === "string";
  if (!wantsEmail && !wantsFirst && !wantsLast) {
    return c.json({ error: "no fields to update" }, 400);
  }

  const { pool } = await import("./env.js");
  const client = await pool.connect();
  try {
    await client.query("BEGIN");

    const current = await client.query<{
      firstName: string | null;
      lastName: string | null;
      email: string;
    }>(
      `SELECT "firstName", "lastName", email FROM "user" WHERE id = $1 FOR UPDATE`,
      [id],
    );
    if (current.rowCount === 0) {
      await client.query("ROLLBACK");
      return c.json({ error: "user not found" }, 404);
    }
    const row = current.rows[0]!;

    let newEmail = row.email;
    let emailChanged = false;
    if (wantsEmail) {
      newEmail = body.email!.toLowerCase();
      emailChanged = newEmail !== row.email;
      if (emailChanged) {
        const dup = await client.query(
          `SELECT 1 FROM "user" WHERE email = $1 AND id <> $2 LIMIT 1`,
          [newEmail, id],
        );
        if ((dup.rowCount ?? 0) > 0) {
          await client.query("ROLLBACK");
          return c.json(
            {
              error: "User with this email already exists.",
              code: "USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL",
            },
            400,
          );
        }
      }
    }

    const newFirst = wantsFirst ? body.firstName! : row.firstName ?? "";
    const newLast = wantsLast ? body.lastName! : row.lastName ?? "";
    const newName = `${newFirst} ${newLast}`.trim() || newEmail;

    await client.query(
      `UPDATE "user"
       SET "firstName" = $2,
           "lastName" = $3,
           email = $4,
           name = $5,
           "emailVerified" = CASE WHEN $6::boolean THEN false ELSE "emailVerified" END,
           "revokedAfter" = CASE WHEN $6::boolean THEN now() ELSE "revokedAfter" END,
           "updatedAt" = now()
       WHERE id = $1`,
      [id, newFirst, newLast, newEmail, newName, emailChanged],
    );

    await client.query("COMMIT");
    return c.json({ ok: true });
  } catch (err) {
    await client.query("ROLLBACK").catch(() => {});
    return apiErrorResponse(c, err, "failed to update user");
  } finally {
    client.release();
  }
});

/**
 * Sets a user's global role. The role lives in the JWT, so we also bump
 * revokedAfter to invalidate any outstanding tokens carrying the previous
 * role within one cache TTL. We update the column directly because
 * `auth.api.setRole` requires a session-bound caller.
 *
 * Promoting to super_admin enforces the invariant that super-admins are
 * not members of any camp: existing memberships are deleted and any active
 * org pointer on their sessions is cleared, all in the same transaction.
 * Without this, a promoted user could mint JWTs with role=super_admin and
 * a non-empty camp_id, bypassing the "super-admins must impersonate to
 * use camp routes" model. Demotion to "user" leaves memberships untouched
 * (they are simply re-assigned via the manage-camps flow).
 */
internalRouter.post("/users/:id/role", async (c) => {
  const id = c.req.param("id");
  const body = (await c.req.json()) as { role?: string };
  if (!body.role) {
    return c.json({ error: "role is required" }, 400);
  }
  if (body.role !== "user" && body.role !== "super_admin") {
    return c.json({ error: "role must be 'user' or 'super_admin'" }, 400);
  }

  const { pool } = await import("./env.js");
  const client = await pool.connect();
  try {
    await client.query("BEGIN");
    const result = await client.query(
      `UPDATE "user"
       SET role = $2, "revokedAfter" = now(), "updatedAt" = now()
       WHERE id = $1`,
      [id, body.role],
    );
    if (result.rowCount === 0) {
      await client.query("ROLLBACK");
      return c.json({ error: "user not found" }, 404);
    }
    if (body.role === "super_admin") {
      await client.query(`DELETE FROM "member" WHERE "userId" = $1`, [id]);
      await client.query(
        `UPDATE "session"
         SET "activeOrganizationId" = NULL, "activeOrganizationRole" = NULL
         WHERE "userId" = $1`,
        [id],
      );
    }
    await client.query("COMMIT");
    return c.json({ ok: true });
  } catch (err) {
    await client.query("ROLLBACK").catch(() => {});
    return apiErrorResponse(c, err, "failed to update user role");
  } finally {
    client.release();
  }
});

internalRouter.delete("/users", async (c) => {
  const email = c.req.query("email");
  if (!email) {
    return c.json({ error: "email is required" }, 400);
  }
  const { pool } = await import("./env.js");
  try {
    await pool.query(`DELETE FROM "user" WHERE email = $1`, [email]);
    return c.json({ ok: true });
  } catch (err) {
    return apiErrorResponse(c, err, "failed to delete user");
  }
});
