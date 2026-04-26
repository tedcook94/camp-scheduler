import { Hono } from "hono";
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
    console.error("error creating organization", err);
    return c.json({ error: "failed to create organization" }, 500);
  }
});

internalRouter.delete("/organizations/:id", async (c) => {
  const id = c.req.param("id");
  const { pool } = await import("./env.js");
  try {
    await pool.query(`DELETE FROM "organization" WHERE id = $1`, [id]);
    return c.json({ ok: true });
  } catch (err) {
    console.error("error deleting organization", err);
    return c.json({ error: "failed to delete organization" }, 500);
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
    console.error("error adding member", err);
    return c.json({ error: "failed to add member" }, 500);
  }
});
