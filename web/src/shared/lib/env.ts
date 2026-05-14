const defaultAdminAPIBaseURL = "/admin/api";

export const adminAPIBaseURL =
  import.meta.env.VITE_ADMIN_API_BASE_URL ?? defaultAdminAPIBaseURL;
