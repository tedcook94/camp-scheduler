import { createAccessControl } from "better-auth/plugins/access";
import {
  adminAc,
  defaultStatements as adminDefaultStatements,
  userAc,
} from "better-auth/plugins/admin/access";

/**
 * Access-control configuration for the admin plugin.
 *
 * `super_admin` is the global role used to be admins of the whole system —
 * it carries all default admin permissions plus the ability to impersonate
 * other admins (rarely needed for us, but matches the prior behaviour).
 *
 * `user` is the default role assigned to newly created accounts; it has no
 * admin permissions.
 */
const ac = createAccessControl({ ...adminDefaultStatements });

const user = ac.newRole({ ...userAc.statements });
const super_admin = ac.newRole({
  ...adminAc.statements,
  user: ["impersonate-admins", ...adminAc.statements.user],
});

export const adminAccessControl = {
  ac,
  roles: { user, super_admin },
};
