import type { AdminSession } from './api';
import { api } from './api';

export type SessionState = AdminSession;

function guestSession(): SessionState {
  return { authenticated: false };
}

function normalizeSession(session: SessionState): SessionState {
  if (session.authenticated === false) {
    return guestSession();
  }
  return {
    authenticated: true,
    username: session.username
  };
}

export function isAuthenticated(session: SessionState | null): boolean {
  return Boolean(session?.authenticated);
}

export async function fetchSession(): Promise<SessionState> {
  try {
    const session = await api.getSession();
    return normalizeSession(session);
  } catch {
    return guestSession();
  }
}

export async function loginAndFetchSession(username: string, password: string): Promise<SessionState> {
  await api.login({ username, password });
  return fetchSession();
}

export async function logoutAndClearSession(): Promise<SessionState> {
  try {
    await api.logout();
  } catch {
    // Ignore logout errors and clear local state.
  }
  return guestSession();
}
