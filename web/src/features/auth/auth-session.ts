export const adminTokenStorageKey = "goroute.admin-token";
export const authRedirectEvent = "goroute:auth-redirect";

export function readStoredAdminToken() {
  if (typeof window === "undefined") {
    return null;
  }

  const token = window.localStorage.getItem(adminTokenStorageKey);

  if (!token) {
    return null;
  }

  const trimmedToken = token.trim();
  return trimmedToken.length > 0 ? trimmedToken : null;
}

export function persistAdminToken(token: string) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(adminTokenStorageKey, token);
}

export function clearStoredAdminToken() {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.removeItem(adminTokenStorageKey);
}

export function redirectToLogin() {
  if (typeof window === "undefined") {
    return;
  }

  window.dispatchEvent(new CustomEvent(authRedirectEvent));
}
